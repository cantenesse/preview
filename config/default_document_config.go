package config

import (
	"os"
	"path/filepath"
)

func NewDefaultDocumentConfig() []byte {
	return buildDefaultDocumentConfig(defaultBasePath)
}

func NewDefaultDocumentConfigWithBaseDirectory(root string) []byte {
	return buildDefaultDocumentConfig(func(section string) string {
		cacheDirectory := filepath.Join(root, ".cache", section)
		os.MkdirAll(cacheDirectory, 00777)
		return cacheDirectory
	})
}

func buildDefaultDocumentConfig(basePathFunc basePath) []byte {
	return []byte(`{
   "common": {
      "localAssetStoragePath":"` + basePathFunc("assets") + `"
   },
   "http":{
      "listen":":8081"
   },
   "conversion":{
      "enabled":true,
      "maxWork":4,
      "basePath":"` + basePathFunc("conversionTmp") + `",
      "supportedFileTypes":["doc", "docx", "ppt", "pptx"]
   },
   "downloader":{
      "basePath":"` + basePathFunc("cache") + `",
      "tramEnabled": false
   },
   "host":"http://127.0.0.1:8081"
}`)
}
