package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"

	"github.com/google/docblog/pkg/ai"
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

	for i, fileMetadata := range filesMetadata {
		if metadata, ok := indexMetadata[fileMetadata.Id]; ok {
			log.Printf("Found metadata for file: %s\n", fileMetadata.Name)
			filesMetadata[i].UpdateWith(metadata)
		}
	}

	for _, fileMetadata := range filesMetadata {
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
				err := processHtml(ctx, outputPath, fileMetadata, unzippedFile.Content)
				if err != nil {
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

	if err = srv.UpdateIndexMetadata(config.DriveDirId, filesMetadata); err != nil {
		panic(err)
	}
}

func processHtml(
	ctx context.Context,
	outputPath string,
	metadata *drive.GoogleDocMetadata,
	fileContent []byte,
) error {
	if metadata.Description == "" {
		description, err := ai.DescribeContent(ctx, fileContent)
		if err != nil {
			return fmt.Errorf("failed to describe content using AI: %v", err)
		}
		metadata.Description = description
	}

	htmlDoc, err := drive.NewHtmlDoc(metadata, fileContent)
	if err != nil {
		return fmt.Errorf("failed to parse input HTML document: %v", err)
	}

	htmlDoc, err = htmlDoc.WithFixedContent()
	if err != nil {
		return fmt.Errorf("failed to fix assets: %v", err)
	}

	htmlDoc, err = htmlDoc.WithFrontmatter()
	if err != nil {
		return fmt.Errorf("failed to add frontmatter: %v", err)
	}

	return drive.WriteFile(outputPath, htmlDoc.Content)
}
