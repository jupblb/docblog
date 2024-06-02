package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"

	"github.com/google/docblog/pkg/config"
	"github.com/google/docblog/pkg/drive"
)

func main() {
	config, err := config.ReadConfig()
	if err != nil {
		panic(err)
	}
	if err := config.SetupDirs(); err != nil {
		panic(err)
	}

	ctx := context.Background()
	srv, err := drive.NewDriveService(ctx, config.DriveOpts())
	if err != nil {
		panic(err)
	}

	_, err = srv.GetOrCreateIndexSheet(config.DriveDirId)
	if err != nil {
		panic(err)
	}

	files, err := srv.ListGoogleDocs(config.DriveDirId)
	if err != nil {
		panic(err)
	}

	for _, file := range files {
		log.Printf("| %s (%s)", file.Name, file.Id)

		unzippedFiles, err := srv.ExportGoogleDocToZippedHtml(file)
		if err != nil {
			log.Printf("Error exporting file: %s\n", file.Name)
			continue
		}

		for _, unzippedFile := range unzippedFiles {
			switch filepath.Ext(unzippedFile.Name) {
			case ".html":
				log.Printf("Processing HTML document: %s\n", unzippedFile.Name)
				outputPath := fmt.Sprintf("%s/%s", config.PostsOutput, unzippedFile.Name)
				if err := processHtml(outputPath, file, unzippedFile.Content); err != nil {
					log.Printf("Error processing HTML file: %v\n", err)
				}
			case ".png":
				log.Printf("Processing PNG asset: %s\n", unzippedFile.Name)
				modifiedName := drive.NormalizedAssetPath(file.Id, unzippedFile.Name)
				outputPath := fmt.Sprintf("%s/%s", config.AssetsOutput, modifiedName)
				if err := drive.WriteFile(outputPath, unzippedFile.Content); err != nil {
					log.Printf("Error processing PNG file: %v\n", err)
				}
			default:
				log.Printf("Skipping unsupported file: %s\n", unzippedFile.Name)
			}
		}
	}
}

func processHtml(outputPath string, file *drive.GoogleDocMetadata, fileContent []byte) error {
	htmlDoc, err := drive.NewHtmlDoc(file, fileContent)
	if err != nil {
		return err
	}

	htmlDoc, err = htmlDoc.WithFixedContent()
	if err != nil {
		return err
	}

	htmlDoc, err = htmlDoc.WithFrontmatter()
	if err != nil {
		return err
	}

	return drive.WriteFile(outputPath, htmlDoc.Content)
}
