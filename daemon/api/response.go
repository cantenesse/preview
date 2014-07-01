package api

type previewInfoCollection struct {
	FileId string     `json:"file_id"`
	Page   int32      `json:"-"`
	Jumbo  *imageInfo `json:"jumbo"`
	Large  *imageInfo `json:"large"`
	Medium *imageInfo `json:"medium"`
	Small  *imageInfo `json:"small"`
}

type imageInfo struct {
	Url           string `json:"url"`
	Width         int32  `json:"width"`
	Height        int32  `json:"height"`
	Expires       int64  `json:"expires"`
	IsFinal       bool   `json:"isFinal"`
	IsPlaceholder bool   `json:"isPlaceholder"`
	Page          int32  `json:"-"`
}

type previewInfoResponse struct {
	Version string                   `json:"version"`
	Files   []*previewInfoCollection `json:"files"`
}

type simpleView struct {
	Previews map[string]multipagePreviewView
}

type multipagePreviewView struct {
	PageCount int32                `json:"pageCount"`
	Pages     map[string]*pageView `json:"pages"`
}

type pageView struct {
	Jumbo  *pageInfoView `json:"jumbo"`
	Large  *pageInfoView `json:"large"`
	Medium *pageInfoView `json:"medium"`
	Small  *pageInfoView `json:"small"`
}

type pageInfoView struct {
	Url     string `json:"url"`
	Width   int32  `json:"width"`
	Height  int32  `json:"height"`
	Expires int64  `json:"expires"`
	Status  string `json:"status"`
}
