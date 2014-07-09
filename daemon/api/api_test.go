package api

import (
	"encoding/json"
	"fmt"
	. "github.com/franela/goblin"
	"github.com/ngerakines/preview/common"
	"github.com/ngerakines/preview/daemon/storage"
	"github.com/ngerakines/testutils"
	. "github.com/onsi/gomega"
	"github.com/rcrowley/go-metrics"
	"testing"
)

func TestApi(t *testing.T) {
	g := Goblin(t)
	RegisterFailHandler(func(m string, _ ...int) {
		g.Fail(m)
	})
	g.Describe("valid generate preview request", func() {
		ida, err := common.NewUuid()
		if err != nil {
			t.Errorf("Error generating UUIDs", err)
		}

		idb, err := common.NewUuid()
		if err != nil {
			t.Errorf("Error generating UUIDs", err)
		}

		data := `
{
        "sourceAssets": [
                {
                        "fileId": "` + ida + `",
                        "url": "s3://bla/foo.jpg",
                        "attributes": {
                                "type": [
                                        "jpg"
                                ]
                        }
                },
                {
                        "fileId": "` + idb + `",
                        "url": "file:///path/to/some/interesting/file/bla.pdf",
                        "attributes": {
                                "type": [
                                        "pdf"
                                ]
                        }
                }
        ],
        "templateIds": [
                "` + common.LegacyDefaultTemplates[0] + `",
                "` + common.DocumentConversionTemplateId + `"
        ]
}
`
		gprs, err := newApiGeneratePreviewRequest(data)
		g.It("successfully creates a GPR", func() {
			Expect(err).To(BeNil())
		})

		g.It("produces the right amount of GPRs", func() {
			Expect(gprs).To(HaveLen(2))
		})
	})

	g.Describe("empty generate preview request", func() {
		emptyData := `
{
	"sourceAssets": [ ],
	"templateIds": [ ]
}
`
		gprs, err := newApiGeneratePreviewRequest(emptyData)
		g.It("successfully creates a GPR", func() {
			Expect(err).To(BeNil())
		})

		g.It("produces the right amount of GPRs", func() {
			Expect(gprs).To(HaveLen(0))
		})
	})

	g.Describe("invalid generate preview request", func() {
		invalidData := `
{
	"sourceAssets": [ "bla bla bla bla"],
	"templateIds": [ { "this":"is a completely invalid request"}]
}
`
		_, err := newApiGeneratePreviewRequest(invalidData)
		g.It("produces an error", func() {
			Expect(err).ToNot(BeNil())
		})
	})

	dm := testutils.NewDirectoryManager()
	defer dm.Close()

	tm := common.NewTemplateManager()
	sourceAssetStorageManager := storage.NewSourceAssetStorageManager()
	generatedAssetStorageManager := storage.NewGeneratedAssetStorageManager(tm)

	registry := metrics.NewRegistry()

	blueprint := NewApiBlueprint("/api/preview", nil, generatedAssetStorageManager, sourceAssetStorageManager, registry, nil)

	sourceAssetId, err := common.NewUuid()
	if err != nil {
		t.Errorf("Unexpected error returned: %s", err)
		return
	}
	sourceAsset, err := common.NewSourceAsset(sourceAssetId, common.SourceAssetTypeOrigin)
	if err != nil {
		t.Errorf("Unexpected error returned: %s", err)
		return
	}

	err = sourceAssetStorageManager.Store(sourceAsset)
	if err != nil {
		t.Errorf("Unexpected error returned: %s", err)
		return
	}

	templateId := "04a2c710-8872-4c88-9c75-a67175d3a8e7"
	ga, err := common.NewGeneratedAssetFromSourceAsset(sourceAsset, templateId, fmt.Sprintf("local:///%s/pdf", sourceAssetId))
	if err != nil {
		t.Errorf("Unexpected error returned: %s", err)
		return
	}
	generatedAssetStorageManager.Store(ga)

	g.Describe("generated asset serializing", func() {
		jsonData, err := blueprint.marshalGeneratedAssets(sourceAssetId, templateId, "")
		g.It("successfully serializes the assets", func() {
			Expect(err).To(BeNil())
		})
		var arr []*common.GeneratedAsset
		err = json.Unmarshal(jsonData, &arr)
		g.It("produces a valid serialization", func() {
			Expect(err).To(BeNil())
			Expect(arr).To(HaveLen(1))
			Expect(arr[0]).To(Equal(ga))
		})
	})

	g.Describe("source asset serializing", func() {
		jsonData, err := blueprint.marshalSourceAssetsFromIds([]string{sourceAssetId})
		g.It("successfully serializes the assets", func() {
			Expect(err).To(BeNil())
		})
		var resp sourceAssetView
		err = json.Unmarshal(jsonData, &resp)
		g.It("produces a valid serialization", func() {
			Expect(err).To(BeNil())
			Expect(resp.SourceAssets).To(HaveLen(1))
			Expect(resp.SourceAssets[0].SourceAsset.Id).To(Equal(sourceAssetId))
			Expect(resp.SourceAssets[0].GeneratedAssets).To(HaveLen(1))
			Expect(resp.SourceAssets[0].GeneratedAssets[0]).To(Equal(ga))
		})
	})
}
