package drive

import (
	"os"
	"path/filepath"
	"strings"
)

// WriteFile writes the provided file content to the output path.
func WriteFile(outputPath string, fileContent []byte) error {
	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(fileContent)
	return err
}

// NormalizedAssetPath returns a normalized asset path based on the document ID
// and the asset's relative path. So `images/image1.png` asset of document with
// ID `123` will be normalized to `123-image1.png`.
func NormalizedAssetPath(
	assetPathPrefix string,
	docId string,
	assetRelativePath string,
) string {
	var sb strings.Builder

	if assetPathPrefix != "" {
		sb.WriteString(assetPathPrefix)
		sb.WriteByte('/')
	}

	sb.WriteString(docId)
	sb.WriteByte('-')
	sb.WriteString(filepath.Base(assetRelativePath))

	return sb.String()
}
