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

	indexMetadata, err := srv.GetIndexMetadata(config.DriveDirId)
	if err != nil {
		panic(err)
	}

	filesMetadata, err := srv.ListGoogleDocs(config.DriveDirId)
	if err != nil {
		panic(err)
	}

	metadatas := combineIndexAndFilesMetadata(indexMetadata, filesMetadata)

	for _, fileMetadata := range metadatas {
		log.Printf("| %s (%s)", fileMetadata.Name, fileMetadata.Id)

		unzippedFiles, err := srv.ExportGoogleDocToZippedHtml(fileMetadata)
		if err != nil {
			log.Printf("Error exporting file: %s\n", fileMetadata.Name)
			continue
		}

		for _, unzippedFile := range unzippedFiles {
			switch filepath.Ext(unzippedFile.Name) {
			case ".html":
				log.Printf("Processing HTML document: %s\n", unzippedFile.Name)
				outputPath := fmt.Sprintf("%s/%s", config.PostsOutput, unzippedFile.Name)
				if err := processHtml(outputPath, fileMetadata, unzippedFile.Content); err != nil {
					log.Printf("Error processing HTML file: %v\n", err)
				}
			case ".png":
				log.Printf("Processing PNG asset: %s\n", unzippedFile.Name)
				modifiedName := drive.NormalizedAssetPath(fileMetadata.Id, unzippedFile.Name)
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

func combineIndexAndFilesMetadata(
	indexMetadata map[string]drive.GoogleDocMetadata,
	filesMetadata []*drive.GoogleDocMetadata,
) []*drive.GoogleDocMetadata {
	for i, fileMetadata := range filesMetadata {
		if metadata, ok := indexMetadata[fileMetadata.Id]; ok {
			log.Printf("Found metadata for file: %s\n", fileMetadata.Name)

			if !metadata.CreatedTime.IsZero() {
				filesMetadata[i].CreatedTime = metadata.CreatedTime
			}
			if metadata.Description != "" {
				filesMetadata[i].Description = metadata.Description
			}
			if !metadata.ModifiedTime.IsZero() {
				filesMetadata[i].ModifiedTime = metadata.ModifiedTime
			}
			if metadata.Name != "" {
				filesMetadata[i].Name = metadata.Name
			}
		}
	}
	return filesMetadata
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
