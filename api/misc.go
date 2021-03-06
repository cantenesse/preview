package api

import (
	"strings"
)

func splitText(input string) map[string]string {
	vals := make(map[string]string)
	for _, line := range strings.Split(input, "\n") {
		if len(line) > 0 {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				vals[parts[0]] = strings.TrimSpace(parts[1])
			}
		}
	}
	return vals
}

func splitS3Url(url string) (string, string) {
	usableData := url[5:]
	// NKG: The url will have the following format: `s3://[bucket][path]`
	// where path will begin with a `/` character.
	parts := strings.SplitN(usableData, "/", 2)
	return parts[0], parts[1]
}
