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
)

func main() {
	config, err := ReadConfig()
	if err != nil {
		panic(err)
	}
	if err := config.SetupDirs(); err != nil {
		panic(err)
	}

	ctx := context.Background()
	srv, err := NewDriveService(ctx, config.DriveOpts())
	if err != nil {
		panic(err)
	}

	files, err := srv.ListFiles(config.DriveDirId)
	if err != nil {
		panic(err)
	}

	for _, file := range files {
		log.Printf("| %s (%s)", file.Name, file.Id)
		zippedFiles, err := srv.ExportGoogleDocToZippedHtml(file)
		if err != nil {
			log.Println(err)
		}

		for _, zipFile := range zippedFiles {
			unzippedFile, err := readZipFile(zipFile)
			if err != nil {
				log.Println(err)
				continue
			}

			if strings.HasSuffix(zipFile.Name, ".html") {
				handleHtml(config.PostsOutput, file, zipFile.Name, unzippedFile)
				continue
			}
			if strings.HasSuffix(zipFile.Name, ".png") {
				modifiedName := fmt.Sprintf("%s-%s", file.Id, filepath.Base(zipFile.Name))
				handlePng(config.AssetsOutput, modifiedName, unzippedFile)
				continue
			}
			log.Printf("Skipping unsupported file: %s\n", zipFile.Name)
		}
	}
}

func handlePng(outputDir string, name string, fileContent []byte) {
	outputPath := fmt.Sprintf("%s/%s", outputDir, name)
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

func handleHtml(outputDir string, file *drive.File, name string, fileContent []byte) {
	modifiedFile, err := modifyHtml(file.Id, fileContent)
	if err != nil {
		log.Println(err)
		return
	}

	outputPath := fmt.Sprintf("%s/%s", outputDir, name)
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
