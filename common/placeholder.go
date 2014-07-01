package common

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
