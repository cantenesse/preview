package common

import (
	"github.com/bmizerany/pat"
)

type Blueprint interface {
	AddRoutes(p *pat.PatternServeMux)
}

// Downloader structures retreive remote files and make them available locally.
type Downloader interface {
	// Download attempts to retreive a file with a given url and store it to a temporary file that is managed by a TemporaryFileManager.
	Download(url, source string) (TemporaryFile, error)
}

type Uploader interface {
	Upload(destination string, path string) error
	Url(sourceAsset *SourceAsset, template *Template, page int32) string
}

type PlaceholderManager interface {
	Url(fileType, placeholderSize string) *Placeholder
	AllFileTypes() []string
}

type Placeholder struct {
	Url      string
	Path     string
	Height   int
	Width    int
	FileSize int64
}

var (
	PlaceholderSizeJumbo  = "jumbo"
	PlaceholderSizeLarge  = "large"
	PlaceholderSizeMedium = "medium"
	PlaceholderSizeSmall  = "small"

	DefaultPlaceholderSizes = []string{PlaceholderSizeJumbo, PlaceholderSizeLarge, PlaceholderSizeMedium, PlaceholderSizeSmall}

	DefaultPlaceholderType = "unknown"
)
