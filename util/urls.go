package util

import (
	"strings"
)

func IsLocalUrl(url string) bool {
	return strings.HasPrefix(url, "local://")
}

func IsS3Url(url string) bool {
	return strings.HasPrefix(url, "s3://")
}

func IsHttpUrl(url string) bool {
	return strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")
}

func IsFileUrl(url string) bool {
	return strings.HasPrefix(url, "file://")
}
