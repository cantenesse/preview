package config

import (
	"encoding/json"
	"github.com/ngerakines/preview/util"
	"io/ioutil"
	"path/filepath"
)

type DocumentConfig struct {
	Source string `json:"-"`

	Common struct {
		LocalAssetStoragePath string `json:"localAssetStoragePath"`
	} `json:"common"`

	Http struct {
		Listen string `json:"listen"`
	} `json:"http"`

	Conversion struct {
		Enabled            bool     `json:"enabled"`
		MaxWork            int      `json:"maxWork"`
		BasePath           string   `json:"basePath"`
		SupportedFileTypes []string `json:"supportedFileTypes"`
	} `json:"conversion"`

	S3 struct {
		Key           string   `json:"key"`
		Secret        string   `json:"secret"`
		Host          string   `json:"host"`
		Buckets       []string `json:"buckets"`
		VerifySsl     bool     `json:"verifySsl"`
		UrlCompatMode bool     `json:"urlCompatMode"`
	} `json:"s3"`

	Downloader struct {
		BasePath string `json:"basePath"`
	} `json:"downloader"`
}

func LoadDocumentConfig(givenPath string) (*DocumentConfig, error) {
	configPath := determineDocumentConfigPath(givenPath)
	if configPath == "" {
		return NewDocumentConfig(NewDefaultDocumentConfig())
	}
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	return NewDocumentConfig(data)
}

func NewDocumentConfig(data []byte) (*DocumentConfig, error) {
	var docConfig DocumentConfig
	err := json.Unmarshal(data, &docConfig)
	if err != nil {
		return nil, err
	}
	docConfig.Source = string(data)
	return &docConfig, nil
}

func determineDocumentConfigPath(givenPath string) string {
	paths := []string{
		givenPath,
		filepath.Join(util.Cwd(), "preview_document.config"),
		filepath.Join(userHomeDir(), ".preview_document.config"),
		"/etc/preview_document.config",
	}
	for _, path := range paths {
		if util.CanLoadFile(path) {
			return path
		}
	}
	return ""
}
