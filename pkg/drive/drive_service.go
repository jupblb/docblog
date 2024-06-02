package drive

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"

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
)

type DriveService struct {
	driveSrv *drive.Service
	sheetSrv *sheets.Service
}

type GoogleDocMetadata struct {
	CreatedTime  string
	Id           string
	ModifiedTime string
	Name         string
}

type unzippedFile struct {
	Name    string
	Content []byte
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
		driveFiles = append(driveFiles, &GoogleDocMetadata{
			CreatedTime:  file.CreatedTime,
			ModifiedTime: file.ModifiedTime,
			Id:           file.Id,
			Name:         file.Name,
		})
	}

	return driveFiles, nil
}

func (ds *DriveService) GetOrCreateIndexSheet(
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
			Sheets: []*sheets.Sheet{{}},
		}).Do()
		if err != nil {
			return nil, err
		}

		_, err = ds.driveSrv.Files.
			Update(sheet.SpreadsheetId, nil).
			AddParents(driveDirId).
			Do()
		if err != nil {
			return nil, err
		}

		return sheet, nil
	}

	return ds.sheetSrv.Spreadsheets.Get(files[0].Id).Do()
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
