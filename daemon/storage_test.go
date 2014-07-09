package daemon

import (
	. "github.com/franela/goblin"
	"github.com/ngerakines/preview/common"
	"github.com/ngerakines/preview/daemon/storage"
	. "github.com/onsi/gomega"
	"testing"
)

func TestStorage(t *testing.T) {
	g := Goblin(t)
	RegisterFailHandler(func(m string, _ ...int) {
		g.Fail(m)
	})

	sasm := storage.NewSourceAssetStorageManager()
	g.Describe("inMemorySourceAssetStorageManager", func() {
		sourceAsset, err := common.NewSourceAsset("4AE594A7-A48E-45E4-A5E1-4533E50BBDA3", common.SourceAssetTypeOrigin)
		g.It("creates a source asset", func() {
			Expect(err).To(BeNil())
		})

		err = sasm.Store(sourceAsset)
		g.It("stores the source asset", func() {
			Expect(err).To(BeNil())
		})

		results, err := sasm.FindBySourceAssetId("4AE594A7-A48E-45E4-A5E1-4533E50BBDA3")
		g.It("finds the source asset", func() {
			Expect(err).To(BeNil())
			Expect(results).To(HaveLen(1))
			Expect(results[0].Id).To(Equal("4AE594A7-A48E-45E4-A5E1-4533E50BBDA3"))
		})
	})
}
