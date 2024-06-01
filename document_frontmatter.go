package main

import (
	"encoding/json"
	"time"

	"google.golang.org/api/drive/v3"
)

type DocumentFrontmatter struct {
	Date        time.Time `json:"date,omitempty"`
	Title       string    `json:"title,omitempty"`
	Description string    `json:"description,omitempty"`
}

func NewDocumentFrontmatter(file *drive.File) (DocumentFrontmatter, error) {
	date, err := time.Parse(time.RFC3339, file.CreatedTime)
	if err != nil {
		return DocumentFrontmatter{}, err
	}

	return DocumentFrontmatter{
		Date:  date,
		Title: file.Name,
	}, nil
}

func (d *DocumentFrontmatter) PrependTo(content []byte) ([]byte, error) {
	jsonBytes, err := json.Marshal(d)
	if err != nil {
		return nil, err
	}
	jsonBytes = append(jsonBytes, '\n')
	jsonBytes = append(jsonBytes, content...)
	return jsonBytes, nil
}
