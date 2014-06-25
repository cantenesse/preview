package util

import (
	"fmt"
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

func JoinUrl(base, file string) string {
	if !strings.HasSuffix(base, "/") {
		base += "/"
	}
	if strings.HasPrefix(file, "/") {
		file = file[:len(file)-1]
	}
	return base + file
}

func S3ToHttps(url string) string {
	arr := strings.Split(url, "/")
	return fmt.Sprintf("https://%s.s3.amazonaws.com/%s", arr[2], strings.Join(arr[3:], "/"))
}
