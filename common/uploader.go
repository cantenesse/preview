package common

type Uploader interface {
	Upload(destination string, path string) error
	Url(sourceAsset *SourceAsset, template *Template, page int32) string
}
