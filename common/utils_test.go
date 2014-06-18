package common

type mockUploader struct {
}

func (uploader *mockUploader) Upload(destination string, path string) error {
	return nil
}

func (uploader *mockUploader) Url(sa *SourceAsset, temp *Template, page int32) string {
	return "mock://" + sa.Id
}

func newMockUploader() Uploader {
	return new(mockUploader)
}
