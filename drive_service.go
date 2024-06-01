package main

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

const (
	FileListFields     = "nextPageToken, files(id, createdTime, mimeType, name)"
	GoogleDocListQuery = "'%s' in parents and trashed=false and " +
		"mimeType='application/vnd.google-apps.document'"
)

type DriveService struct {
	srv *drive.Service
}

type DriveFile struct {
	CreatedTime string
	Id          string
	MimeType    string
	Name        string
}

type unzippedFile struct {
	Name    string
	Content []byte
}

func NewDriveService(
	ctx context.Context,
	opts []option.ClientOption,
) (*DriveService, error) {
	srv, err := drive.NewService(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return &DriveService{srv: srv}, nil
}

func (ds *DriveService) ListFiles(driveDirId string) ([]*DriveFile, error) {
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

	driveFiles := []*DriveFile{}
	for _, file := range files {
		driveFiles = append(driveFiles, &DriveFile{
			CreatedTime: file.CreatedTime,
			Id:          file.Id,
			MimeType:    file.MimeType,
			Name:        file.Name,
		})
	}

	return driveFiles, nil
}

func (ds *DriveService) ExportGoogleDocToZippedHtml(
	file *DriveFile,
) ([]*unzippedFile, error) {
	if file.MimeType != "application/vnd.google-apps.document" {
		return nil,
			fmt.Errorf("file %s (%s) is not a Google Doc", file.Name, file.Id)
	}

	resp, err := ds.srv.Files.Export(file.Id, "application/zip").Download()
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
	call := ds.srv.Files.List().
		Fields(FileListFields).
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
