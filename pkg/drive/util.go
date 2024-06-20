// Copyright 2024 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
