package main

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/net/html"
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

	if err := os.MkdirAll("posts", 0o750); err != nil {
		log.Fatalf("Error creating 'posts' directory: %v", err)
	}
	if err := os.MkdirAll("hugo/static", 0o750); err != nil {
		log.Fatalf("Error creating 'hugo/static' directory: %v", err)
	}

	srv, err := drive.NewService(ctx, DriveOpts...)
	if err != nil {
		panic(err)
	}

	files, err := listFiles(srv)
	if err != nil {
		panic(err)
	}

	for _, file := range files {
		log.Printf("| %s (%s)", file.Name, file.Id)
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

		for _, zipFile := range zipReader.File {
			unzippedFile, err := readZipFile(zipFile)
			if err != nil {
				log.Println(err)
				continue
			}

			if strings.HasSuffix(zipFile.Name, ".html") {
				handleHtml(file, zipFile.Name, unzippedFile)
				continue
			}
			if strings.HasSuffix(zipFile.Name, ".png") {
				modifiedName := fmt.Sprintf("%s-%s", file.Id, filepath.Base(zipFile.Name))
				handlePng(modifiedName, unzippedFile)
				continue
			}
			log.Printf("Skipping unsupported file: %s\n", zipFile.Name)
		}
	}
}

func handlePng(name string, fileContent []byte) {
	outputPath := fmt.Sprintf("hugo/static/%s", name)
	f, err := os.Create(outputPath)
	if err != nil {
		log.Println(err)
		return
	}
	defer f.Close()

	log.Printf("Writing file: %s\n", outputPath)
	if _, err = f.Write(fileContent); err != nil {
		log.Println(err)
	}
}

func handleHtml(file *drive.File, name string, fileContent []byte) {
	modifiedFile, err := modifyHtml(file.Id, fileContent)
	if err != nil {
		log.Println(err)
		return
	}

	outputPath := fmt.Sprintf("posts/%s", name)
	f, err := os.Create(outputPath)
	if err != nil {
		log.Println(err)
		return
	}
	defer f.Close()

	frontmatter := DocumentFrontmatter{
		Date:  time.Now(),
		Title: file.Name,
	}

	content, err := frontmatter.PrependTo(modifiedFile)
	if err != nil {
		fmt.Println(err)
		return
	}

	log.Printf("Writing file: %s\n", outputPath)
	if _, err = f.Write(content); err != nil {
		log.Println(err)
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

func modifyHtml(fileId string, content []byte) ([]byte, error) {
	rootNode, err := html.Parse(bytes.NewReader(content))
	if err != nil {
		return nil, err
	}

	htmlNode := rootNode.FirstChild

	for el0 := htmlNode.FirstChild; el0 != nil; el0 = el0.NextSibling {
		if el0.Type == html.ElementNode && el0.Data == "body" {
			for i, attr := range el0.Attr {
				if attr.Key == "style" {
					el0.Attr[i].Val = ""
				}
			}

			nodesToRemove := []*html.Node{}
			for el1 := el0.FirstChild; el1 != nil; el1 = el1.NextSibling {
				for _, attr := range el1.Attr {
					if attr.Key == "class" &&
						(attr.Val == "title" || attr.Val == "subtitle") {
						nodesToRemove = append(nodesToRemove, el1)
					}
				}
			}
			for _, node := range nodesToRemove {
				el0.RemoveChild(node)
			}

			modifyImg(fileId, el0)
		}
	}

	var b bytes.Buffer
	if err := html.Render(&b, rootNode); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func modifyImg(fileId string, node *html.Node) {
	for el0 := node.FirstChild; el0 != nil; el0 = el0.NextSibling {
		if el0.Type == html.ElementNode && el0.Data == "img" {
			for i, attr := range el0.Attr {
				if attr.Key == "src" {
					el0.Attr[i].Val = fmt.Sprintf("/%s-%s", fileId, filepath.Base(attr.Val))
				}
			}
		} else {
			modifyImg(fileId, el0)
		}
	}
}
