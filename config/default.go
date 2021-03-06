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
   "documentRenderAgent":{
      "enabled":true,
      "count":16,
      "basePath":"` + basePathFunc("documentRenderAgentTmp") + `",
      "supportedFileTypes":["doc", "docx", "ppt", "pptx"]
   },
   "videoRenderAgent":{
      "enabled":false,
      "count":16,
      "supportedFileTypes":["mp4"]
   },
   "imageMagickRenderAgent":{
      "enabled":true,
      "count":16,
      "supportedFileTypes":["jpg", "jpeg", "png", "gif", "pdf"]
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

type basePath func(string) string

func defaultBasePath(section string) string {
	cacheDirectory := filepath.Join(util.Cwd(), ".cache", section)
	os.MkdirAll(cacheDirectory, 00777)
	return cacheDirectory
}
