package config

import (
	"github.com/ngerakines/preview/util"
	"os"
	"path/filepath"
)

func NewDefaultAppConfig() []byte {
	return buildDefaultConfig(defaultBasePath)
}

func NewDefaultAppConfigWithBaseDirectory(root string) []byte {
	return buildDefaultConfig(func(section string) string {
		cacheDirectory := filepath.Join(root, ".cache", section)
		os.MkdirAll(cacheDirectory, 00777)
		return cacheDirectory
	})
}

func buildDefaultConfig(basePathFunc basePath) []byte {
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
                "placeholderSize":["small"]
            }
        },
	{
            "id":"9B17C6CE-7B09-4FD5-92AD-D85DD218D6D7",
            "renderAgent":"documentRenderAgent",
            "group":"A907",
            "attributes":{
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

type basePath func(string) string

func defaultBasePath(section string) string {
	cacheDirectory := filepath.Join(util.Cwd(), ".cache", section)
	os.MkdirAll(cacheDirectory, 00777)
	return cacheDirectory
}
