package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/alexflint/go-arg"
	"github.com/google/docblog/pkg/ai"
	"github.com/google/docblog/pkg/drive"
	"google.golang.org/api/option"
)

var args struct {
	ai.GeminiOptions

	DriveDirId string `arg:"positional,required" help:"Google Drive directory with blog posts." placeholder:"DRIVE-DIR-ID"`

	AssetsOutputPath          string `arg:"--assets-output,env:DOCBLOG_ASSETS_OUTPUT" default:"assets" help:"asset output path"`
	AssetsPathPrefix          string `arg:"--assets-prefix,env:DOCBLOG_ASSETS_PREFIX" help:"asset path prefix (html)"`
	GcloudCredentialsFilePath string `arg:"--credentials,env:DOCBLOG_GCLOUD_CREDENTIALS" default:".gcloud/application_default_credentials.json" help:"file with Google Cloud credentials"`
	PostsOutputPath           string `arg:"--posts-output,env:DOCBLOG_POSTS_OUTPUT" default:"posts" help:"HTML output path"`
}

func main() {
	arg.MustParse(&args)

	if err := os.MkdirAll(args.PostsOutputPath, 0o750); err != nil {
		panic(err)
	}
	if err := os.MkdirAll(args.AssetsOutputPath, 0o750); err != nil {
		panic(err)
	}

	ctx := context.Background()
	srv, err := drive.NewDriveService(ctx, []option.ClientOption{
		option.WithCredentialsFile(args.GcloudCredentialsFilePath),
	})
	if err != nil {
		panic(err)
	}

	indexMetadata, err := srv.GetIndexMetadata(args.DriveDirId)
	if err != nil {
		panic(err)
	}

	filesMetadata, err := srv.ListGoogleDocs(args.DriveDirId)
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
				outputPath := fmt.Sprintf(
					"%s/%s", args.PostsOutputPath, fileMetadata.FileName())
				err := processHtml(ctx, outputPath, fileMetadata, unzippedFile.Content)
				if err != nil {
					log.Printf("Error processing HTML file: %v\n", err)
				}
			case ".png":
				log.Printf("Processing PNG asset: %s\n", unzippedFile.Name)
				modifiedName := drive.NormalizedAssetPath(
					args.AssetsPathPrefix, fileMetadata.Id, unzippedFile.Name)
				outputPath := fmt.Sprintf("%s/%s", args.AssetsOutputPath, modifiedName)
				if err := drive.WriteFile(outputPath, unzippedFile.Content); err != nil {
					log.Printf("Error processing PNG file: %v\n", err)
				}
			default:
				log.Printf("Skipping unsupported file: %s\n", unzippedFile.Name)
			}
		}
	}

	if err = srv.UpdateIndexMetadata(args.DriveDirId, filesMetadata); err != nil {
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
		description, err := ai.DescribeContent(ctx, args.GeminiOptions, fileContent)
		if err == nil {
			metadata.Description = description
		} else {
			log.Printf("Error generating description: %v\n", err)
		}
	}

	htmlDoc, err := drive.NewHtmlDoc(metadata, fileContent)
	if err != nil {
		return fmt.Errorf("failed to parse input HTML document: %v", err)
	}

	htmlDoc, err = htmlDoc.WithFixedContent(args.AssetsPathPrefix)
	if err != nil {
		return fmt.Errorf("failed to fix assets: %v", err)
	}

	htmlDoc, err = htmlDoc.WithFrontmatter()
	if err != nil {
		return fmt.Errorf("failed to add frontmatter: %v", err)
	}

	return drive.WriteFile(outputPath, htmlDoc.Content)
}
