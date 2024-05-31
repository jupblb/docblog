package main

import (
	"encoding/json"
	"time"
)

type DocumentFrontmatter struct {
	Date  time.Time `json:"date,omitempty"`
	Title string    `json:"title,omitempty"`
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
