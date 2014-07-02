package api

import (
	"encoding/json"
	"github.com/ngerakines/preview/common"
	"strconv"
)

func newGeneratePreviewRequestFromText(id, body string) ([]*common.GeneratePreviewRequest, error) {
	if len(id) == 0 {
		return nil, common.ErrorInvalidFileId
	}
	vals := splitText(body)

	requestType, hasRequestType := vals["type"]
	if !hasRequestType {
		return nil, common.ErrorMissingFieldType
	}

	url, hasUrl := vals["url"]
	if !hasUrl {
		return nil, common.ErrorMissingFieldUrl
	}

	size, hasSize := vals["size"]
	if !hasSize {
		return nil, common.ErrorMissingFieldSize
	}
	sizeValue, err := strconv.ParseInt(size, 10, 64)
	if err != nil {
		// TODO: This should return a different error.
		return nil, common.ErrorMissingFieldSize
	}

	gpr := common.NewGeneratePreviewRequest(id, requestType, url, sizeValue)

	gprs := make([]*common.GeneratePreviewRequest, 0, 0)
	gprs = append(gprs, gpr)
	return gprs, nil
}

func newGeneratePreviewRequestFromJson(body string) ([]*common.GeneratePreviewRequest, error) {
	var data struct {
		Version int `json:"version"`
		Files   []struct {
			Id          string `json:"file_id"`
			RequestType string `json:"type"`
			Url         string `json:"url"`
			Size        string `json:"size"`
		} `json:"files"`
	}
	err := json.Unmarshal([]byte(body), &data)
	if err != nil {
		return nil, err
	}

	gprs := make([]*common.GeneratePreviewRequest, 0, 0)
	for _, file := range data.Files {
		sizeValue, err := strconv.ParseInt(file.Size, 10, 64)
		if err != nil {
			return nil, common.ErrorMissingFieldSize
		}
		gpr := common.NewGeneratePreviewRequest(file.Id, file.RequestType, file.Url, sizeValue)
		gprs = append(gprs, gpr)
	}
	return gprs, nil
}
