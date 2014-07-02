package daemon

import (
	"fmt"
	"github.com/ngerakines/ketama"
	"github.com/ngerakines/preview/common"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type s3Uploader struct {
	bucketRing ketama.HashRing
	s3Client   common.S3Client
}

type localUploader struct {
	basePath string
}

func newUploader(buckets []string, s3Client common.S3Client) common.Uploader {
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

func newLocalUploader(basePath string) common.Uploader {
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
	return common.ErrorUploaderDoesNotSupportUrl
}

func (uploader *s3Uploader) Url(sourceAsset *common.SourceAsset, template *common.Template, page int32) string {
	bucket := uploader.bucketRing.Hash(sourceAsset.Id)
	if template.Id == common.DocumentConversionTemplateId {
		return fmt.Sprintf("s3://%s/%s-pdf", bucket, sourceAsset.Id)
	}
	placeholderSize := template.GetAttribute(common.TemplateAttributePlaceholderSize)[0]
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
		err := common.CopyFile(existingFile, newPath)
		if err != nil {
			log.Println(err)
			return err
		}
		return nil
	}
	return common.ErrorUploaderDoesNotSupportUrl
}

func (uploader *localUploader) Url(sourceAsset *common.SourceAsset, template *common.Template, page int32) string {
	if template.Id == common.DocumentConversionTemplateId {
		return fmt.Sprintf("local:///%s/pdf", sourceAsset.Id)
	} else if template.Id == common.MsOfficeTemplateId {
		return fmt.Sprintf("local:///%s/pdf", sourceAsset.Id)
	}
	placeholderSize := template.GetAttribute(common.TemplateAttributePlaceholderSize)[0]
	return fmt.Sprintf("local:///%s/%s/%d", sourceAsset.Id, placeholderSize, page)
}
