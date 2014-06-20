package config

import (
	"encoding/json"
	"github.com/ngerakines/preview/util"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
)

type appConfigError struct {
	message string
}

type AppConfig struct {
	Source string `json:"-"`

	Common struct {
		PlaceholderBasePath   string              `json:"placeholderBasePath"`
		PlaceholderGroups     map[string][]string `json:"placeholderGroups"`
		LocalAssetStoragePath string              `json:"localAssetStoragePath"`
		NodeId                string              `json:"nodeId"`
		WorkDispatcherEnabled bool                `json:"workDispatcherEnabled"`
	} `json:"common"`

	Http struct {
		Listen string `json:"listen"`
	} `json:"http"`

	Storage struct {
		Engine            string   `json:"engine"`
		CassandraNodes    []string `json:"cassandraNodes"`
		CassandraKeyspace string   `json:"cassandraKeyspace"`
		MysqlHost         string   `json:"mysqlHost"`
		MysqlUser         string   `json:"mysqlUser"`
		MysqlPassword     string   `json:"mysqlPassword"`
		MysqlDatabase     string   `json:"mysqlDatabase"`
	} `json:"storage"`

	ImageMagickRenderAgent struct {
		Enabled            bool     `json:"enabled"`
		Count              int      `json:"count"`
		SupportedFileTypes []string `json:"supportedFileTypes"`
	} `json:"imageMagickRenderAgent"`

	DocumentRenderAgent struct {
		Enabled            bool     `json:"enabled"`
		Count              int      `json:"count"`
		BasePath           string   `json:"basePath"`
		SupportedFileTypes []string `json:"supportedFileTypes"`
	} `json:"documentRenderAgent"`

	VideoRenderAgent struct {
		Enabled                 bool     `json:"enabled"`
		Count                   int      `json:"count"`
		BasePath                string   `json:"basePath"`
		ZencoderKey             string   `json:"zencoderKey"`
		ZencoderS3Bucket        string   `json:"zencoderS3Bucket"`
		ZencoderNotificationUrl string   `json:"zencoderNotificationUrl"`
		SupportedFileTypes      []string `json:"supportedFileTypes"`
	} `json:"videoRenderAgent"`

	SimpleApi struct {
		Enabled     bool   `json:"enabled"`
		EdgeBaseUrl string `json:"edgeBaseUrl"`
		BaseUrl     string `json:"baseUrl"`
	} `json:"simpleApi"`

	AssetApi struct {
		Enabled bool `json:"enabled"`
	} `json:"assetApi"`

	Uploader struct {
		Engine string `json:"engine"`
	} `json:"uploader"`

	S3 struct {
		Key           string   `json:"key"`
		Secret        string   `json:"secret"`
		Host          string   `json:"host"`
		Buckets       []string `json:"buckets"`
		VerifySsl     bool     `json:"verifySsl"`
		UrlCompatMode bool     `json:"urlCompatMode"`
	} `json:"s3"`

	Downloader struct {
		BasePath    string   `json:"basePath"`
		TramEnabled bool     `json:"tramEnabled"`
		TramHosts   []string `json:"tramHosts"`
	} `json:"downloader"`
}

func LoadAppConfig(givenPath string) (*AppConfig, error) {
	configPath := determineConfigPath(givenPath)
	if configPath == "" {
		return NewAppConfig(NewDefaultAppConfig())
	}
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	return NewAppConfig(data)
}

func NewAppConfig(data []byte) (*AppConfig, error) {
	var appConfig AppConfig
	err := json.Unmarshal(data, &appConfig)
	if err != nil {
		return nil, err
	}
	appConfig.Source = string(data)
	return &appConfig, nil
}

func (err appConfigError) Error() string {
	return err.message
}

func determineConfigPath(givenPath string) string {
	paths := []string{
		givenPath,
		filepath.Join(util.Cwd(), "preview.config"),
		filepath.Join(userHomeDir(), ".preview.config"),
		"/etc/preview.config",
	}
	for _, path := range paths {
		if util.CanLoadFile(path) {
			return path
		}
	}
	return ""
}

func userHomeDir() string {
	if runtime.GOOS == "windows" {
		home := filepath.Join(os.Getenv("HOMEDRIVE"), os.Getenv("HOMEPATH"))
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
		return home
	}
	return os.Getenv("HOME")
}
