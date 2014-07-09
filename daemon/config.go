package daemon

import (
	"encoding/json"
	"github.com/ngerakines/preview/common"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
)

type configBasePath func(string) string

type daemonConfig struct {
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

	RenderAgents map[string]struct {
		Enabled            bool              `json:"enabled"`
		Count              int               `json:"count"`
		SupportedFileTypes []string          `json:"supportedFileTypes"`
		RendererParams     map[string]string `json:"rendererParams"`
	} `json:"renderAgents"`

	Zencoder struct {
		Enabled bool   `json:"enabled"`
		Key     string `json:"key"`
	} `json:"zencoder"`

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

func loadDaemonConfig(givenPath string) (*daemonConfig, error) {
	configPath := determineConfigPath(givenPath)
	if configPath == "" {
		return newDaemonConfig(newDefaultDaemonConfig())
	}
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	return newDaemonConfig(data)
}

func newDaemonConfig(data []byte) (*daemonConfig, error) {
	var appConfig daemonConfig
	err := json.Unmarshal(data, &appConfig)
	if err != nil {
		return nil, err
	}
	appConfig.Source = string(data)
	return &appConfig, nil
}

func determineConfigPath(givenPath string) string {
	paths := []string{
		givenPath,
		filepath.Join(common.Cwd(), "preview.config"),
		filepath.Join(userHomeDir(), ".preview.config"),
		"/etc/preview.config",
	}
	for _, path := range paths {
		if common.CanLoadFile(path) {
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

func newDefaultDaemonConfig() []byte {
	return buildDefaultDaemonConfig(defaultBasePath)
}

func NewDefaultAppConfigWithBaseDirectory(root string) []byte {
	return buildDefaultDaemonConfig(func(section string) string {
		cacheDirectory := filepath.Join(root, ".cache", section)
		os.MkdirAll(cacheDirectory, 00777)
		return cacheDirectory
	})
}

func buildDefaultDaemonConfig(basePathFunc configBasePath) []byte {
	return []byte(`{
   "common": {
      "placeholderBasePath":"` + basePathFunc("placeholders") + `",
      "placeholderGroups": {
         "image":["jpg", "jpeg", "png", "gif", "pdf"],
         "document":["doc", "docx"],
         "video":["mp4"]
      },
      "localAssetStoragePath":"` + basePathFunc("assets") + `",
      "nodeId":"E876F147E331",
      "workDispatcherEnabled":true
   },
   "http":{
      "listen":":8080"
   },
   "storage":{
      "engine":"memory"
   },
   "renderAgents": {
      "documentRenderAgent":{
         "enabled":true,
         "count":16,
         "supportedFileTypes":["doc", "docx", "ppt", "pptx"],
         "rendererParams":{
             "basePath":"` + basePathFunc("documentRenderAgentTmp") + `"
         }
      },
      "videoRenderAgent":{
         "enabled":false,
         "count":16,
         "supportedFileTypes":["mp4"],
         "engine":"zencoder",
         "rendererParams":{
               "zencoderNotificationUrl":"http://zencoderfetcher"
          }         
      },
      "imageMagickRenderAgent":{
         "enabled":true,
         "count":16,
         "supportedFileTypes":["jpg", "jpeg", "png", "gif", "pdf"],
         "rendererParams":{
         }
      }
   },
   "zencoder":{
      "enabled":false,
      "key":"YOUR_KEY_HERE"
   },
   "simpleApi":{
      "enabled":true,
      "baseUrl":"/api",
      "edgeBaseUrl":"http://localhost:8080"
   },
   "assetApi":{
      "enabled":true
   },
   "uploader":{
      "engine":"local"
   },
   "downloader":{
      "basePath":"` + basePathFunc("cache") + `",
      "tramEnabled": false
   }
}`)
}

func defaultBasePath(section string) string {
	cacheDirectory := filepath.Join(common.Cwd(), ".cache", section)
	os.MkdirAll(cacheDirectory, 00777)
	return cacheDirectory
}
