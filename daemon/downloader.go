package daemon

import (
	"fmt"
	"github.com/ngerakines/ketama"
	"github.com/ngerakines/preview/common"
	"io"
	"io/ioutil"
	"log"
	neturl "net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Downloader structures retreive remote files and make them available locally.
type Downloader interface {
	// Download attempts to retreive a file with a given url and store it to a temporary file that is managed by a TemporaryFileManager.
	Download(url, source string) (common.TemporaryFile, error)
}

type defaultDownloader struct {
	basePath         string
	localStoragePath string
	tfm              common.TemporaryFileManager
	tramEnabled      bool
	tramHostRing     ketama.HashRing
	s3Client         common.S3Client
}

// NewDownloader creates, configures and returns a new defaultDownloader.
func newDownloader(basePath, localStoragePath string, tfm common.TemporaryFileManager, tramEnabled bool, tramHosts []string, s3Client common.S3Client) Downloader {
	downloader := new(defaultDownloader)
	downloader.basePath = basePath
	downloader.localStoragePath = localStoragePath
	downloader.tfm = tfm
	downloader.tramEnabled = tramEnabled
	downloader.s3Client = s3Client

	log.Println("downloader", tramEnabled, tramHosts)

	if downloader.tramEnabled {
		hashRing := ketama.NewRing(180)
		for _, tramHost := range tramHosts {
			hashRing.Add(tramHost, 1)
		}
		hashRing.Bake()
		downloader.tramHostRing = hashRing
	}
	return downloader
}

// Download attempts to retreive a file with a given url and store it to a temporary file that is managed by a TemporaryFileManager.
func (downloader *defaultDownloader) Download(url, source string) (common.TemporaryFile, error) {
	log.Println("Attempting to download", url)
	if strings.HasPrefix(url, "file://") {
		return downloader.handleFile(url)
	}
	if strings.HasPrefix(url, "local://") {
		return downloader.handleLocal(url)
	}
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		return downloader.handleHttp(url, source)
	}
	if downloader.s3Client != nil && strings.HasPrefix(url, "s3://") {
		return downloader.handleS3Object(url, source)
	}
	return nil, common.ErrorNotImplemented
}

func (downloader *defaultDownloader) handleLocal(url string) (common.TemporaryFile, error) {
	log.Println("Attempting to download file", url[8:])
	path := filepath.Join(downloader.localStoragePath, url[8:])

	uuid, err := common.NewUuid()
	if err != nil {
		return nil, err
	}
	newPath := filepath.Join(downloader.basePath, uuid)

	newPathDir := filepath.Dir(newPath)
	err = os.MkdirAll(newPathDir, 0777)
	if err != nil {
		log.Println("error copying file:", err.Error())
		return nil, err
	}

	err = common.CopyFile(path, newPath)
	if err != nil {
		log.Println("error copying file:", err.Error())
		return nil, err
	}
	log.Println("File", path, "copied to", newPath)

	return downloader.tfm.Create(newPath), nil
}

func (downloader *defaultDownloader) handleFile(url string) (common.TemporaryFile, error) {
	log.Println("Attempting to download file", url[7:])
	log.Println("downloading file url", url)
	path := url[7:]
	log.Println("actual path", path)

	uuid, err := common.NewUuid()
	if err != nil {
		return nil, err
	}

	newPath := filepath.Join(downloader.basePath, uuid)

	newPathDir := filepath.Dir(newPath)
	err = os.MkdirAll(newPathDir, 0777)
	if err != nil {
		log.Println("error copying file:", err.Error())
		return nil, err
	}

	err = common.CopyFile(path, newPath)
	if err != nil {
		log.Println("error copying file:", err.Error())
		return nil, err
	}
	log.Println("File", path, "copied to", newPath)

	return downloader.tfm.Create(newPath), nil
}

func (downloader *defaultDownloader) handleS3Object(url, source string) (common.TemporaryFile, error) {
	uuid, err := common.NewUuid()
	if err != nil {
		return nil, err
	}
	newPath := filepath.Join(downloader.basePath, uuid)

	usableData := url[5:]
	// NKG: The url will have the following format: `s3://[bucket][path]`
	// where path will begin with a `/` character.
	parts := strings.SplitN(usableData, "/", 2)

	s3Object, err := downloader.s3Client.Get(parts[0], parts[1])
	if err != nil {
		return nil, err
	}

	newPathDir := filepath.Dir(newPath)
	os.MkdirAll(newPathDir, 0777)

	out, err := os.Create(newPath)
	if err != nil {
		return nil, err
	}
	defer out.Close()

	err = ioutil.WriteFile(newPath, s3Object.Payload(), 0777)
	if err != nil {
		return nil, err
	}

	return downloader.tfm.Create(newPath), nil
}

func (downloader *defaultDownloader) handleHttp(url, source string) (common.TemporaryFile, error) {
	uuid, err := common.NewUuid()
	if err != nil {
		return nil, err
	}
	newPath := filepath.Join(downloader.basePath, uuid)

	newPathDir := filepath.Dir(newPath)
	os.MkdirAll(newPathDir, 0777)

	out, err := os.Create(newPath)
	if err != nil {
		return nil, err
	}
	defer out.Close()

	httpClient := common.NewHttpClient(false, 30*time.Second)

	resp, err := httpClient.Get(downloader.getHttpUrl(url, source))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	n, err := io.Copy(out, resp.Body)
	if err != nil {
		return nil, err
	}
	log.Println("Downloaded", n, "bytes to file", newPath)

	return downloader.tfm.Create(newPath), nil
}

func (downloader *defaultDownloader) getHttpUrl(url, source string) string {
	if downloader.tramEnabled {
		tramHost := downloader.tramHostRing.Hash(source)
		return fmt.Sprintf("http://%s/?url=%s&alias=%s", tramHost, neturl.QueryEscape(url), neturl.QueryEscape(source))
	}
	return url
}
