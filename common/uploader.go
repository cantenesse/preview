package common

import (
	"fmt"
	"github.com/ngerakines/ketama"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type Uploader interface {
	Upload(destination string, path string) error
	Url(sourceAsset *SourceAsset, template *Template, page int32) string
}

type s3Uploader struct {
	bucketRing ketama.HashRing
	s3Client   S3Client
}

type localUploader struct {
	basePath string
}

func NewUploader(buckets []string, s3Client S3Client) Uploader {
	hashRing := ketama.NewRing(180)
	for _, bucket := range buckets {
		hashRing.Add(bucket, 1)
	}
	hashRing.Bake()

	uploader := new(s3Uploader)
	uploader.bucketRing = hashRing
	uploader.s3Client = s3Client
	return uploader
}

func NewLocalUploader(basePath string) Uploader {
	return &localUploader{basePath}
}

func (uploader *s3Uploader) Upload(destination, path string) error {
	log.Println("Uploading", path, "to", destination)
	if strings.HasPrefix(destination, "s3://") {
		usableData := destination[5:]
		// NKG: The url will have the following format: `s3://[bucket][path]`
		// where path will begin with a `/` character.
		parts := strings.SplitN(usableData, "/", 2)
		log.Println("parts", parts)
		object, err := uploader.s3Client.NewObject(parts[1], parts[0], "application/octet-stream")
		if err != nil {
			log.Println("Could not create object", err)
			return err
		}

		payload, err := ioutil.ReadFile(path)
		if err != nil {
			log.Println("Could not read file", path, "because", err)
			return err
		}
		err = uploader.s3Client.Put(object, payload)
		if err != nil {
			log.Println("Could not PUT file", err)
			return err
		}

		return nil
	}
	return ErrorUploaderDoesNotSupportUrl
}

func (uploader *s3Uploader) Url(sourceAsset *SourceAsset, template *Template, page int32) string {
	bucket := uploader.bucketRing.Hash(sourceAsset.Id)
	if template.Id == DocumentConversionTemplateId {
		return fmt.Sprintf("s3://%s/%s-pdf", bucket, sourceAsset.Id)
	}
	placeholderSize := template.GetAttribute(TemplateAttributePlaceholderSize)[0]
	return fmt.Sprintf("s3://%s/%s-%s-%d", bucket, sourceAsset.Id, placeholderSize, page)
}

func (uploader *localUploader) Upload(destination, existingFile string) error {
	log.Println("Uploading", existingFile, "to", destination)
	if strings.HasPrefix(destination, "local://") {
		path := destination[8:]
		newPath := filepath.Join(uploader.basePath, path)
		newPathDir := filepath.Dir(newPath)
		os.MkdirAll(newPathDir, 0777)
		log.Println("uploading to", newPath)
		err := copyFile(existingFile, newPath)
		if err != nil {
			return err
		}
		return nil
	}
	return ErrorUploaderDoesNotSupportUrl
}

func (uploader *localUploader) Url(sourceAsset *SourceAsset, template *Template, page int32) string {
	if template.Id == DocumentConversionTemplateId {
		return fmt.Sprintf("local:///%s/pdf", sourceAsset.Id)
	}
	placeholderSize := template.GetAttribute(TemplateAttributePlaceholderSize)[0]
	return fmt.Sprintf("local:///%s/%s/%d", sourceAsset.Id, placeholderSize, page)
}
