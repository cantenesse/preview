package common

// Downloader structures retreive remote files and make them available locally.
type Downloader interface {
	// Download attempts to retreive a file with a given url and store it to a temporary file that is managed by a TemporaryFileManager.
	Download(url, source string) (TemporaryFile, error)
}
