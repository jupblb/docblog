package drive

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"time"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

const (
	GoogleDocListFields = "nextPageToken, files(id, createdTime, modifiedTime, name)"
	GoogleDocListQuery  = "'%s' in parents and trashed=false and " +
		"mimeType='application/vnd.google-apps.document'"

	GoogleSheetIndexListQuery = "'%s' in parents and trashed=false and " +
		"mimeType='application/vnd.google-apps.spreadsheet' and name='index'"
	GoogleSheetIndexTitle = "Docblog configuration"

	PostDayFormat = "02/01/2006"
)

var GoogleSheetIndexColumnMetadata = [...]configColumnMetadata{
	{"Id", 350},
	{"Name", 300},
	{"Date", 100},
	{"Last modified", 100},
	{"Description", 800},
}

var (
	// https://developers.google.com/sheets/api/guides/formats#about_date_time_values
	GoogleSheetEpoch0 = time.Date(1899, time.December, 30, 0, 0, 0, 0, time.UTC)
	CellDateFormat    = sheets.CellFormat{
		HorizontalAlignment: "LEFT",
		NumberFormat: &sheets.NumberFormat{
			Pattern: "dd/mm/yyyy",
			Type:    "DATE",
		},
	}
)

type DriveService struct {
	driveSrv *drive.Service
	sheetSrv *sheets.Service
}

type GoogleDocMetadata struct {
	CreatedTime  time.Time
	Description  string
	Id           string
	ModifiedTime time.Time
	Name         string
}

type unzippedFile struct {
	Name    string
	Content []byte
}

type configColumnMetadata struct {
	name       string
	pixelWidth int64
}

func NewDriveService(
	ctx context.Context,
	opts []option.ClientOption,
) (*DriveService, error) {
	driveSrv, err := drive.NewService(ctx, opts...)
	if err != nil {
		return nil, err
	}
	sheetSrv, err := sheets.NewService(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return &DriveService{driveSrv: driveSrv, sheetSrv: sheetSrv}, nil
}

func (ds *DriveService) ListGoogleDocs(
	driveDirId string,
) ([]*GoogleDocMetadata, error) {
	fileList, err := ds.listGoogleDocs(driveDirId, "")
	if err != nil {
		return nil, err
	}

	files := fileList.Files
	pageToken := fileList.NextPageToken
	for pageToken != "" {
		fileList, err = ds.listGoogleDocs(driveDirId, pageToken)
		if err != nil {
			return nil, err
		}
		files = append(files, fileList.Files...)
		pageToken = fileList.NextPageToken
	}

	driveFiles := []*GoogleDocMetadata{}
	for _, file := range files {
		createdDate, err := time.Parse(time.RFC3339, file.CreatedTime)
		if err != nil {
			return driveFiles, err
		}

		modifiedDate, err := time.Parse(time.RFC3339, file.ModifiedTime)
		if err != nil {
			return driveFiles, err
		}

		driveFiles = append(driveFiles, &GoogleDocMetadata{
			CreatedTime:  createdDate,
			ModifiedTime: modifiedDate,
			Id:           file.Id,
			Name:         file.Name,
		})
	}

	return driveFiles, nil
}

func (ds *DriveService) GetIndexMetadata(
	driveDirId string,
) (map[string]GoogleDocMetadata, error) {
	sheet, err := ds.getOrCreateIndexSheet(driveDirId)
	if err != nil {
		return nil, fmt.Errorf("error getting or creating index sheet: %w", err)
	}

	output := map[string]GoogleDocMetadata{}
	rows := sheet.
		Sheets[0].
		Data[0].
		RowData[1:]
	for _, row := range rows {
		metadata := GoogleDocMetadata{}
		if errs := metadata.ParseRowData(row); errs != nil {
			for _, err := range errs {
				log.Printf("error parsing metadata (%s): %v", metadata.Id, err)
			}
		}
		output[metadata.Id] = metadata
	}
	return output, nil
}

func (ds *DriveService) UpdateIndexMetadata(
	driveDirId string,
	metadata []*GoogleDocMetadata,
) error {
	sheet, err := ds.getOrCreateIndexSheet(driveDirId)
	if err != nil {
		return fmt.Errorf("error getting or creating index sheet: %w", err)
	}
	var rows []*sheets.RowData
	for _, fileMetadata := range metadata {
		rows = append(rows, fileMetadata.ToRowData())
	}

	_, err = ds.sheetSrv.Spreadsheets.BatchUpdate(
		sheet.SpreadsheetId, &sheets.BatchUpdateSpreadsheetRequest{
			Requests: []*sheets.Request{{
				UpdateSheetProperties: &sheets.UpdateSheetPropertiesRequest{
					Fields: "*",
					Properties: &sheets.SheetProperties{
						GridProperties: &sheets.GridProperties{
							ColumnCount: int64(len(GoogleSheetIndexColumnMetadata)),
							RowCount:    1 + int64(len(rows)),
						},
						SheetId: sheet.Sheets[0].Properties.SheetId,
						Title:   GoogleSheetIndexTitle,
					},
				},
			}, {
				UpdateCells: &sheets.UpdateCellsRequest{
					Fields: "*",
					Rows:   rows,
					Start: &sheets.GridCoordinate{
						SheetId:     sheet.Sheets[0].Properties.SheetId,
						RowIndex:    1,
						ColumnIndex: 0,
					},
				},
			}},
		}).Do()
	if err != nil {
		return fmt.Errorf("error updating index metadata: %v", err)
	}
	return nil
}

func (ds *DriveService) ExportGoogleDocToZippedHtml(
	file *GoogleDocMetadata,
) ([]*unzippedFile, error) {
	resp, err := ds.driveSrv.Files.Export(file.Id, "application/zip").Download()
	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	zipReader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return nil, err
	}

	var unzippedFiles []*unzippedFile
	for _, zipFile := range zipReader.File {
		content, err := readZipFile(zipFile)
		if err != nil {
			return unzippedFiles, err
		}
		unzippedFiles = append(unzippedFiles, &unzippedFile{
			Name:    zipFile.Name,
			Content: content,
		})
	}

	return unzippedFiles, nil
}

func (m1 *GoogleDocMetadata) UpdateWith(m2 GoogleDocMetadata) {
	if !m2.CreatedTime.IsZero() {
		m1.CreatedTime = m2.CreatedTime
	}
	if m2.Description != "" {
		m1.Description = m2.Description
	}
}

func (m *GoogleDocMetadata) ToRowData() *sheets.RowData {
	docUrl := fmt.Sprintf("https://docs.google.com/document/d/%s", m.Id)
	hyperlink := fmt.Sprintf("=HYPERLINK(\"%s\", \"%s\")", docUrl, m.Id)
	createdDate := float64(
		m.CreatedTime.Sub(GoogleSheetEpoch0).Hours() / 24)
	modifiedDate := float64(
		m.ModifiedTime.Sub(GoogleSheetEpoch0).Hours() / 24)

	return &sheets.RowData{
		Values: []*sheets.CellData{
			{
				UserEnteredFormat: &sheets.CellFormat{
					HyperlinkDisplayType: "LINKED",
					TextFormat: &sheets.TextFormat{
						Link: &sheets.Link{Uri: docUrl},
					},
				},
				UserEnteredValue: &sheets.ExtendedValue{
					FormulaValue: &hyperlink,
				},
			},
			{UserEnteredValue: &sheets.ExtendedValue{StringValue: &m.Name}},
			{
				UserEnteredValue:  &sheets.ExtendedValue{NumberValue: &createdDate},
				UserEnteredFormat: &CellDateFormat,
			},
			{
				UserEnteredValue:  &sheets.ExtendedValue{NumberValue: &modifiedDate},
				UserEnteredFormat: &CellDateFormat,
			},
			{
				UserEnteredValue: &sheets.ExtendedValue{StringValue: &m.Description},
				UserEnteredFormat: &sheets.CellFormat{
					WrapStrategy: "WRAP",
				},
			},
		},
	}
}

func (m *GoogleDocMetadata) ParseRowData(row *sheets.RowData) []error {
	errors := []error{}

	createdDate, err := time.Parse(PostDayFormat, row.Values[2].FormattedValue)
	if err != nil {
		createdDate = time.Time{}
		errors = append(errors, fmt.Errorf("error parsing created date: %w", err))
	}

	modifiedDate, err := time.Parse(PostDayFormat, row.Values[3].FormattedValue)
	if err != nil {
		modifiedDate = time.Time{}
		errors = append(errors, fmt.Errorf("error parsing modified date: %w", err))
	}

	m.Id = row.Values[0].FormattedValue
	m.Name = row.Values[1].FormattedValue
	m.CreatedTime = createdDate
	m.ModifiedTime = modifiedDate

	if len(row.Values) >= len(GoogleSheetIndexColumnMetadata) {
		m.Description = row.Values[4].FormattedValue
	} else {
		errors = append(errors, fmt.Errorf("missing description value"))
	}

	return errors
}

func (ds *DriveService) listGoogleDocs(
	driveDirId string,
	pageToken string,
) (*drive.FileList, error) {
	call := ds.driveSrv.Files.List().
		Fields(GoogleDocListFields).
		Q(fmt.Sprintf(GoogleDocListQuery, driveDirId))

	if pageToken != "" {
		call.PageToken(pageToken)
	}

	return call.Do()
}

func readZipFile(zf *zip.File) ([]byte, error) {
	f, err := zf.Open()
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}

func (ds *DriveService) getOrCreateIndexSheet(
	driveDirId string,
) (*sheets.Spreadsheet, error) {
	fileList, err := ds.driveSrv.Files.List().
		Fields("files(id)").
		Q(fmt.Sprintf(GoogleSheetIndexListQuery, driveDirId)).
		Do()
	if err != nil {
		return nil, err
	}

	files := fileList.Files
	if len(files) > 1 {
		return nil, fmt.Errorf("multiple index sheets found")
	}
	if len(files) == 0 {
		sheet, err := ds.sheetSrv.Spreadsheets.Create(&sheets.Spreadsheet{
			Properties: &sheets.SpreadsheetProperties{
				Title: "index",
			},
			Sheets: []*sheets.Sheet{{
				Data: []*sheets.GridData{{
					ColumnMetadata: ds.getColumnMetadata(),
					RowData: []*sheets.RowData{
						{Values: ds.getHeaders()},
					},
				}},
				Properties: &sheets.SheetProperties{
					GridProperties: &sheets.GridProperties{
						ColumnCount: int64(len(GoogleSheetIndexColumnMetadata)),
						RowCount:    1,
					},
					Title: GoogleSheetIndexTitle,
				},
			}},
		}).Do()
		if err != nil {
			return sheet, err
		}

		_, err = ds.driveSrv.Files.
			Update(sheet.SpreadsheetId, nil).
			AddParents(driveDirId).
			Do()
		if err != nil {
			return sheet, err
		}

		return ds.sheetSrv.Spreadsheets.
			Get(sheet.SpreadsheetId).
			IncludeGridData(true).
			Do()
	}

	return ds.sheetSrv.Spreadsheets.Get(files[0].Id).IncludeGridData(true).Do()
}

func (ds *DriveService) getColumnMetadata() []*sheets.DimensionProperties {
	var columnMetadata []*sheets.DimensionProperties
	for _, metadata := range GoogleSheetIndexColumnMetadata {
		columnMetadata = append(columnMetadata, &sheets.DimensionProperties{
			PixelSize: metadata.pixelWidth,
		})
	}
	return columnMetadata
}

func (ds *DriveService) getHeaders() []*sheets.CellData {
	var headers []*sheets.CellData
	for _, metadata := range GoogleSheetIndexColumnMetadata {
		headers = append(headers, &sheets.CellData{
			UserEnteredValue: &sheets.ExtendedValue{
				StringValue: &metadata.name,
			},
			UserEnteredFormat: &sheets.CellFormat{
				TextFormat: &sheets.TextFormat{
					Bold: true,
				},
			},
		})
	}
	return headers
}
