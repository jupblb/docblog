package main

import (
	"encoding/json"
	"os"

	"google.golang.org/api/option"
)

const (
	DefaultCredentialsFilePath = ".gcloud/application_default_credentials.json"
	DefaultPostsOutput         = "hugo/content/posts"
	DefaultAssetsOutput        = "hugo/static"
)

type Config struct {
	DriveDirId string `json:"drive_dir_id"`

	AssetsOutput        string `json:"assets_output,omitempty"`
	CredentialsFilePath string `json:"credentials_file_path,omitempty"`
	PostsOutput         string `json:"posts_output,omitempty"`
}

func ReadConfig() (*Config, error) {
	file, err := os.Open("config.json")
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(file)
	config := &Config{}
	if err := decoder.Decode(config); err != nil {
		return nil, err
	}

	if config.CredentialsFilePath == "" {
		config.CredentialsFilePath = DefaultCredentialsFilePath
	}
	if config.PostsOutput == "" {
		config.PostsOutput = DefaultPostsOutput
	}
	if config.AssetsOutput == "" {
		config.AssetsOutput = DefaultAssetsOutput
	}

	return config, nil
}

func (config *Config) DriveOpts() []option.ClientOption {
	return []option.ClientOption{
		option.WithCredentialsFile(config.CredentialsFilePath),
	}
}

func (config *Config) SetupDirs() error {
	if err := os.MkdirAll(config.PostsOutput, 0o750); err != nil {
		return err
	}
	if err := os.MkdirAll(config.AssetsOutput, 0o750); err != nil {
		return err
	}
	return nil
}
