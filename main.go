package main

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

const (
	FileListDriveDir = "180SMxnqxvPk6B52OnIZrsVakLvnh3nyc"
	FileListFields   = "nextPageToken, files(id, createdTime, mimeType, name)"
	FileListQuery    = "'%s' in parents and trashed=false and " +
		"mimeType='application/vnd.google-apps.document'"
)

var DriveOpts = []option.ClientOption{
	option.WithCredentialsFile(".gcloud/application_default_credentials.json"),
}

func main() {
	ctx := context.Background()

	srv, err := drive.NewService(ctx, DriveOpts...)
	if err != nil {
		panic(err)
	}

	files, err := listFiles(srv)
	if err != nil {
		panic(err)
	}

	for _, file := range files {
		log.Printf("| %s -- %s (%s)", file.Name, file.Id, file.MimeType)
		resp, err := srv.Files.Export(file.Id, "application/zip").Download()
		if err != nil {
			log.Printf("Error downloading file: %v", err)
		}
		defer resp.Body.Close()

		// https://stackoverflow.com/a/50539327/2900417
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Println(err.Error())
		}

		zipReader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
		if err != nil {
			log.Println(err.Error())
		}

		if err := os.MkdirAll("output", 0o700); err != nil {
			log.Fatalf("Error creating output directory: %v", err)
		}
		for _, zipFile := range zipReader.File {
			if strings.HasSuffix(zipFile.Name, ".html") {
				log.Printf("Writing file: %s\n", zipFile.Name)
				unzippedFileBytes, err := readZipFile(zipFile)
				if err != nil {
					log.Println(err)
					continue
				}

				f, err := os.Create(fmt.Sprintf("output/%s", zipFile.Name))
				if err != nil {
					log.Println(err)
					continue
				}
				defer f.Close()

				if _, err = f.Write(unzippedFileBytes); err != nil {
					log.Println(err)
				}
			}
		}
	}
}

func listFiles(srv *drive.Service) ([]*drive.File, error) {
	fileList, err := srv.Files.List().
		Fields(FileListFields).
		Q(fmt.Sprintf(FileListQuery, FileListDriveDir)).
		Do()
	if err != nil {
		log.Fatalf("Unable to retrieve files: %v", err)
	}

	files := fileList.Files
	pageToken := fileList.NextPageToken
	for pageToken != "" {
		fileList, err = srv.Files.List().
			Fields(FileListFields).
			Q(fmt.Sprintf(FileListQuery, FileListDriveDir)).
			PageToken(pageToken).
			Do()
		if err != nil {
			return nil, err
		}
		files = append(files, fileList.Files...)
		pageToken = fileList.NextPageToken
	}

	return files, nil
}

func readZipFile(zf *zip.File) ([]byte, error) {
	f, err := zf.Open()
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}
