package common

import (
	"bytes"
	"encoding/json"
	"fmt"
)

type GeneratePreviewRequest struct {
	id          string
	requestType string
	url         string
	size        int64
}

type PreviewInfoResponse struct {
	Version string                   `json:"version"`
	Files   []*PreviewInfoCollection `json:"files"`
}

type PreviewInfoCollection struct {
	FileId string     `json:"file_id"`
	Jumbo  *ImageInfo `json:"jumbo"`
	Large  *ImageInfo `json:"large"`
	Medium *ImageInfo `json:"medium"`
	Small  *ImageInfo `json:"small"`
}

type ImageInfo struct {
	Url           string `json:"url"`
	Width         int32  `json:"width"`
	Height        int32  `json:"height"`
	Expires       int64  `json:"expires"`
	IsFinal       bool   `json:"isFinal"`
	IsPlaceholder bool   `json:"isPlaceholder"`
}

type MultipagePreviewView struct {
	PageCount int32                `json:"pageCount"`
	Pages     map[string]*PageView `json:"pages"`
}

type PageView struct {
	Jumbo  *PageInfoView `json:"jumbo"`
	Large  *PageInfoView `json:"large"`
	Medium *PageInfoView `json:"medium"`
	Small  *PageInfoView `json:"small"`
}

type PageInfoView struct {
	Url     string `json:"url"`
	Width   int32  `json:"width"`
	Height  int32  `json:"height"`
	Expires int64  `json:"expires"`
	Status  string `json:"status"`
}

func NewGeneratePreviewRequest(id, requestType, url string, size int64) *GeneratePreviewRequest {
	return &GeneratePreviewRequest{
		id:          id,
		requestType: requestType,
		url:         url,
		size:        size,
	}
}

func NewPreviewInfoResponse(version string) *PreviewInfoResponse {
	response := new(PreviewInfoResponse)
	response.Version = version
	response.Files = make([]*PreviewInfoCollection, 0, 0)
	return response
}

func NewPreviewInfoCollection() *PreviewInfoCollection {
	return &PreviewInfoCollection{}
}

func NewImageInfo(url string, width, height int32, expires int64, isFinal, isPlaceholder bool) *ImageInfo {
	return &ImageInfo{
		Url:           url,
		Width:         width,
		Height:        height,
		Expires:       expires,
		IsFinal:       isFinal,
		IsPlaceholder: isPlaceholder,
	}
}

func NewPreviewInfoResponseFromJson(payload []byte) (*PreviewInfoResponse, error) {
	var response PreviewInfoResponse
	err := json.Unmarshal(payload, &response)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

func NewMultipageView() map[string]*MultipagePreviewView {
	return make(map[string]*MultipagePreviewView)
}

func NewMultipagePreviewView() *MultipagePreviewView {
	view := new(MultipagePreviewView)
	view.PageCount = 0
	view.Pages = make(map[string]*PageView)
	return view
}

func NewPageView() *PageView {
	view := new(PageView)
	return view
}

func NewPageInfoView(url string, height, width int32, expires int64, status string) *PageInfoView {
	return &PageInfoView{
		Url:     url,
		Width:   width,
		Height:  height,
		Expires: expires,
		Status:  status,
	}
}

func (request *GeneratePreviewRequest) Id() string {
	return request.id
}

func (request *GeneratePreviewRequest) Url() string {
	return request.url
}

func (request *GeneratePreviewRequest) RequestType() string {
	return request.requestType
}

func (request *GeneratePreviewRequest) Size() int64 {
	return request.size
}

func (request *GeneratePreviewRequest) String() string {
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("type: %s\n", request.requestType))
	buffer.WriteString(fmt.Sprintf("url: %s\n", request.url))
	buffer.WriteString(fmt.Sprintf("size: %d\n", request.size))
	return buffer.String()
}

func (response *PreviewInfoResponse) AddCollection(collection *PreviewInfoCollection) {
	response.Files = append(response.Files, collection)
}

func (view *MultipagePreviewView) SetPage(id int32, page *PageView) {
	view.Pages[fmt.Sprintf("%d", id)] = page
}
