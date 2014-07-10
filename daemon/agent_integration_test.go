package daemon

import (
	"github.com/ngerakines/preview/common"
	"github.com/ngerakines/preview/daemon/agent"
	"github.com/ngerakines/preview/daemon/storage"
	"github.com/ngerakines/testutils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/rcrowley/go-metrics"
	"log"
	"path/filepath"
	"time"
)

type testInfo struct {
	file          string
	fileType      string
	templateIds   []string
	expectedCount int
}

// TODO[JSH]: Find the proper way to do this. Integrating this with ginkgo right now is a bit of a hack
var _ = It("Agent integration test", func() {
	dm := testutils.NewDirectoryManager()
	defer dm.Close()

	tm := common.NewTemplateManager()
	sasm := storage.NewSourceAssetStorageManager()
	gasm := storage.NewGeneratedAssetStorageManager(tm)

	tfm := common.NewTemporaryFileManager()
	downloader := newDownloader(dm.Path, dm.Path, tfm, false, []string{}, nil)
	uploader := newLocalUploader(dm.Path)
	registry := metrics.NewRegistry()

	agentFileTypes := make(map[string]map[string]int)
	agentFileTypes["documentRenderAgent"] = map[string]int{
		"doc":  60,
		"docx": 60,
		"ppt":  60,
		"pptx": 60,
	}
	agentFileTypes["imageMagickRenderAgent"] = map[string]int{
		"jpg":  60,
		"jpeg": 60,
		"png":  60,
		"pdf":  60,
		"gif":  60,
	}

	rm := agent.NewRenderAgentManager(registry, sasm, gasm, tm, tfm, uploader, true, nil, agentFileTypes)

	for i := 0; i < 4; i++ {
		rm.AddRenderAgent("documentRenderAgent", map[string]string{"tempFileBasePath": filepath.Join(dm.Path, "doc-cache")}, downloader, uploader, 5)
	}

	for i := 0; i < 8; i++ {
		rm.AddRenderAgent("imageMagickRenderAgent", map[string]string{"maxPages": "10"}, downloader, uploader, 5)
	}

	defer rm.Stop()

	tests := make([]testInfo, 0)
	tests = append(tests, testInfo{"Multipage.docx", "docx", []string{common.DocumentConversionTemplateId}, 13})
	tests = append(tests, testInfo{"ChefConf2014schedule.docx", "docx", []string{common.DocumentConversionTemplateId}, 5})
	tests = append(tests, testInfo{"ChefConf2014schedule.docx", "docx", []string{common.DocumentConversionTemplateId}, 5})
	tests = append(tests, testInfo{"ChefConf2014schedule.docx", "docx", []string{common.DocumentConversionTemplateId}, 5})
	tests = append(tests, testInfo{"Multipage.pdf", "pdf", common.LegacyDefaultTemplates, 12})
	tests = append(tests, testInfo{"wallpaper-641916.jpg", "jpg", common.LegacyDefaultTemplates, 4})
	tests = append(tests, testInfo{"Animated.gif", "gif", common.LegacyDefaultTemplates, 4})
	tests = append(tests, testInfo{"COW.png", "png", common.LegacyDefaultTemplates, 4})
	tests = append(tests, testInfo{"ChefConf2014schedule.pdf", "pdf", common.LegacyDefaultTemplates, 4})
	tests = append(tests, testInfo{"ChefConf2014schedule.docx", "docx", []string{common.DocumentConversionTemplateId}, 5})

	for _, info := range tests {
		sourceAssetId, err := common.NewUuid()
		Expect(err).To(BeNil())

		sourceAsset, err := common.NewSourceAsset(sourceAssetId, common.SourceAssetTypeOrigin)
		Expect(err).To(BeNil())

		sourceAsset.AddAttribute(common.SourceAssetAttributeSize, []string{"12345"})
		sourceAsset.AddAttribute(common.SourceAssetAttributeSource, []string{fileUrl("test-data", info.file)})
		sourceAsset.AddAttribute(common.SourceAssetAttributeType, []string{info.fileType})

		err = sasm.Store(sourceAsset)
		Expect(err).To(BeNil())

		templates, err := tm.FindByIds(info.templateIds)
		Expect(err).To(BeNil())

		for _, template := range templates {
			ga, err := common.NewGeneratedAssetFromSourceAsset(sourceAsset, template.Id, uploader.Url(sourceAsset, template, 0))
			Expect(err).To(BeNil())

			err = gasm.Store(ga)
			Expect(err).To(BeNil())

			log.Println(ga)
		}

		// 1 minute timeout, 1 second update interval
		Eventually(func() bool {
			return isComplete(sourceAssetId, gasm, info.expectedCount)
		}, 1*time.Minute, 1*time.Second).Should(BeTrue())
	}
})

func isComplete(id string, generatedAssetStorageManager common.GeneratedAssetStorageManager, expectedCount int) bool {
	generatedAssets, err := generatedAssetStorageManager.FindBySourceAssetId(id)
	if err == nil {
		count := 0
		for _, generatedAsset := range generatedAssets {
			if generatedAsset.Status == common.GeneratedAssetStatusComplete {
				count += 1
			}
		}
		if count == expectedCount {
			return true
		}
		if count > 0 {
			log.Println("Count is", count, "but wanted", expectedCount)
			return false
		}
	}
	return false
}

func fileUrl(dir, file string) string {
	return "file://" + filepath.Join(common.Cwd(), "../", dir, file)
}
