package api

import (
	"encoding/json"
	"fmt"
	"github.com/ngerakines/preview/common"
	"github.com/ngerakines/preview/daemon/storage"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/rcrowley/go-metrics"
)

var _ = Describe("generate preview requests", func() {
	Context("valid", func() {
		ida, err := common.NewUuid()
		Expect(err).To(BeNil())
		idb, err := common.NewUuid()
		Expect(err).To(BeNil())

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
		It("successfully creates a GPR", func() {
			Expect(err).To(BeNil())
		})

		It("produces the right amount of GPRs", func() {
			Expect(gprs).To(HaveLen(2))
		})
	})
	Context("empty", func() {
		emptyData := `
{
	"sourceAssets": [ ],
	"templateIds": [ ]
}
`
		gprs, err := newApiGeneratePreviewRequest(emptyData)
		It("successfully creates a GPR", func() {
			Expect(err).To(BeNil())
		})

		It("produces the right amount of GPRs", func() {
			Expect(gprs).To(HaveLen(0))
		})
	})

	Context("invalid", func() {
		invalidData := `
{
	"sourceAssets": [ "bla bla bla bla"],
	"templateIds": [ { "this":"is a completely invalid request"}]
}
`
		_, err := newApiGeneratePreviewRequest(invalidData)
		It("produces an error", func() {
			Expect(err).ToNot(BeNil())
		})
	})
})

var _ = Describe("serializing", func() {
	tm := common.NewTemplateManager()
	sourceAssetStorageManager := storage.NewSourceAssetStorageManager()
	generatedAssetStorageManager := storage.NewGeneratedAssetStorageManager(tm)

	registry := metrics.NewRegistry()

	blueprint := NewApiBlueprint("/api/preview", nil, generatedAssetStorageManager, sourceAssetStorageManager, registry, nil)

	sourceAssetId, err := common.NewUuid()
	Expect(err).To(BeNil())
	sourceAsset, err := common.NewSourceAsset(sourceAssetId, common.SourceAssetTypeOrigin)
	Expect(err).To(BeNil())

	err = sourceAssetStorageManager.Store(sourceAsset)
	Expect(err).To(BeNil())

	templateId := "04a2c710-8872-4c88-9c75-a67175d3a8e7"
	ga, err := common.NewGeneratedAssetFromSourceAsset(sourceAsset, templateId, fmt.Sprintf("local:///%s/pdf", sourceAssetId))
	Expect(err).To(BeNil())

	generatedAssetStorageManager.Store(ga)

	Context("generated asset", func() {
		jsonData, err := blueprint.marshalGeneratedAssets(sourceAssetId, templateId, "")
		It("successfully serializes the assets", func() {
			Expect(err).To(BeNil())
		})
		var arr []*common.GeneratedAsset
		err = json.Unmarshal(jsonData, &arr)
		It("produces a valid serialization", func() {
			Expect(err).To(BeNil())
			Expect(arr).To(HaveLen(1))
			Expect(arr[0]).To(Equal(ga))
		})
	})

	Context("source asset", func() {
		jsonData, err := blueprint.marshalSourceAssetsFromIds([]string{sourceAssetId})
		It("successfully serializes the assets", func() {
			Expect(err).To(BeNil())
		})
		var resp sourceAssetView
		err = json.Unmarshal(jsonData, &resp)
		It("produces a valid serialization", func() {
			Expect(err).To(BeNil())
			Expect(resp.SourceAssets).To(HaveLen(1))
			Expect(resp.SourceAssets[0].SourceAsset.Id).To(Equal(sourceAssetId))
			Expect(resp.SourceAssets[0].GeneratedAssets).To(HaveLen(1))
			Expect(resp.SourceAssets[0].GeneratedAssets[0]).To(Equal(ga))
		})
	})
})
