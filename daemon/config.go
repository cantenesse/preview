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
		OnDemandEnabled       bool                `json:"onDemandEnabled"`
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

	Templates []struct {
		Id          string              `json:"id"`
		RenderAgent string              `json:"renderAgent"`
		Group       string              `json:"group"`
		Attributes  map[string][]string `json:"attributes"`
	}
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
      "onDemandEnabled":true
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
   },
   "templates": [
        {
            "id":"04a2c710-8872-4c88-9c75-a67175d3a8e7",
            "renderAgent":"imageMagickRenderAgent",
            "group":"4C96",
            "attributes":{
                "width":["1040"],
                "height":["780"],
                "output":["jpg"],
                "density": ["144"],
                "placeholderSize":["jumbo"]
            }
        },
	{
            "id":"2eee7c27-75e2-4682-9920-9a4e14caa433",
            "renderAgent":"imageMagickRenderAgent",
            "group":"4C96",
            "attributes":{
                "width":["520"],
                "height":["390"],
                "output":["jpg"],
                "density": ["144"],
                "placeholderSize":["large"]
            }
        },
	{
            "id":"a89a6a0d-51d9-4d99-b278-0c5dfc538984",
            "renderAgent":"imageMagickRenderAgent",
            "group":"4C96",
            "attributes":{
                "width":["500"],
                "height":["376"],
                "output":["jpg"],
                "density": ["144"],
                "placeholderSize":["medium"]
            }
        },
	{
            "id":"eaa7be0e-354f-482c-ac75-75cbdafecb6e",
            "renderAgent":"imageMagickRenderAgent",
            "group":"4C96",
            "attributes":{
                "width":["250"],
                "height":["188"],
                "output":["jpg"],
                "density": ["144"],
                "placeholderSize":["small"]
            }
        },
	{
            "id":"9B17C6CE-7B09-4FD5-92AD-D85DD218D6D7",
            "renderAgent":"documentRenderAgent",
            "group":"A907",
            "attributes":{
                "density": ["144"],
                "output":["pdf"]
            }
        },
	{
            "id":"4128966B-9F69-4E56-AD5C-1FDB3C24F910",
            "renderAgent":"videoRenderAgent",
            "group":"7A96",
            "attributes":{
                "output":["m3u8"],
		             "forceS3Location":["true"],
                "zencoderNotificationUrl":["http://example.com/zencoderhandler"]
            }
        }
    ]
}`)
}

func defaultBasePath(section string) string {
	cacheDirectory := filepath.Join(common.Cwd(), ".cache", section)
	os.MkdirAll(cacheDirectory, 00777)
	return cacheDirectory
}
