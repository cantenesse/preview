package daemon

import (
	"github.com/ngerakines/preview/common"
	"github.com/ngerakines/preview/daemon/storage"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("inMemorySourceAssetStorageManager", func() {
	sasm := storage.NewSourceAssetStorageManager()
	sourceAsset, err := common.NewSourceAsset("4AE594A7-A48E-45E4-A5E1-4533E50BBDA3", common.SourceAssetTypeOrigin)
	It("creates a source asset", func() {
		Expect(err).To(BeNil())
	})

	err = sasm.Store(sourceAsset)
	It("stores the source asset", func() {
		Expect(err).To(BeNil())
	})

	results, err := sasm.FindBySourceAssetId("4AE594A7-A48E-45E4-A5E1-4533E50BBDA3")
	It("finds the source asset", func() {
		Expect(err).To(BeNil())
		Expect(results).To(HaveLen(1))
		Expect(results[0].Id).To(Equal("4AE594A7-A48E-45E4-A5E1-4533E50BBDA3"))
	})
})

var _ = Describe("inMemoryGeneratedAssetStorageManager", func() {
	tm := common.NewTemplateManager()
	gasm := storage.NewGeneratedAssetStorageManager(tm)

	sourceAsset, err := common.NewSourceAsset("4AE594A7-A48E-45E4-A5E1-4533E50BBDA3", common.SourceAssetTypeOrigin)
	It("creates a source asset", func() {
		Expect(err).To(BeNil())
	})

	generatedAsset, err := common.NewGeneratedAssetFromSourceAsset(sourceAsset, "12345678", "http://somewhere.interesting/foo.bar")
	It("creates a generated asset", func() {
		Expect(err).To(BeNil())
	})

	err = gasm.Store(generatedAsset)
	It("stores the generated asset", func() {
		Expect(err).To(BeNil())
	})

	results, err := gasm.FindBySourceAssetId("4AE594A7-A48E-45E4-A5E1-4533E50BBDA3")
	It("finds the generated asset", func() {
		Expect(err).To(BeNil())
		Expect(results).To(HaveLen(1))
		Expect(results[0].Id).To(Equal(generatedAsset.Id))
	})
})
