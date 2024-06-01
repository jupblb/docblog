package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
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
					log.Printf("Error processing HTML file: %s\n", unzippedFile.Name)
				}
			case ".png":
				log.Printf("Processing PNG asset: %s\n", unzippedFile.Name)
				modifiedName := NormalizedAssetPath(file.Id, unzippedFile.Name)
				outputPath := fmt.Sprintf("%s/%s", config.AssetsOutput, modifiedName)
				if err := WriteFile(outputPath, unzippedFile.Content); err != nil {
					log.Printf("Error processing PNG file: %s\n", unzippedFile.Name)
				}
			default:
				log.Printf("Skipping unsupported file: %s\n", unzippedFile.Name)
			}
		}
	}
}

func processHtml(outputPath string, file *DriveFile, fileContent []byte) error {
	htmlDoc, err := NewHtmlDoc(file, fileContent)
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

	return WriteFile(outputPath, htmlDoc.Content)
}
