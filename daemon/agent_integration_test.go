package daemon

import (
	. "github.com/franela/goblin"
	"github.com/ngerakines/preview/common"
	"github.com/ngerakines/preview/daemon/agent"
	"github.com/ngerakines/preview/daemon/storage"
	"github.com/ngerakines/testutils"
	. "github.com/onsi/gomega"
	"github.com/rcrowley/go-metrics"
	"log"
	"path/filepath"
	"testing"
	"time"
)

type testInfo struct {
	file          string
	fileType      string
	templateIds   []string
	expectedCount int
}

func TestAgents(t *testing.T) {
	g := Goblin(t, "-goblin.timeout=60s")
	RegisterFailHandler(func(m string, _ ...int) {
		g.Fail(m)
	})

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

	rm.AddRenderAgent("documentRenderAgent", map[string]string{"baseFileTmpPath": filepath.Join(dm.Path, "doc-cache")}, downloader, uploader, 5)
	rm.AddRenderAgent("imageMagickRenderAgent", map[string]string{"maxPages": "10"}, downloader, uploader, 5)

	defer rm.Stop()

	testListener := make(agent.RenderStatusChannel, 25)
	rm.AddListener(testListener)

	tests := make([]testInfo, 0)
	tests = append(tests, testInfo{"Multipage.docx", "docx", []string{common.DocumentConversionTemplateId}, 13})
	tests = append(tests, testInfo{"Multipage.pdf", "pdf", common.LegacyDefaultTemplates, 12})

	g.Describe("render agent", func() {
		for _, info := range tests {
			sourceAssetId, err := common.NewUuid()
			if err != nil {
				t.Errorf("Unexpected error returned: %s", err)
				return
			}

			sourceAsset, err := common.NewSourceAsset(sourceAssetId, common.SourceAssetTypeOrigin)
			g.It("creates source asset", func() {
				Expect(err).To(BeNil())
			})

			sourceAsset.AddAttribute(common.SourceAssetAttributeSize, []string{"12345"})
			sourceAsset.AddAttribute(common.SourceAssetAttributeSource, []string{fileUrl("test-data", info.file)})
			sourceAsset.AddAttribute(common.SourceAssetAttributeType, []string{info.fileType})

			err = sasm.Store(sourceAsset)
			g.It("stores source asset", func() {
				Expect(err).To(BeNil())
			})

			templates, err := tm.FindByIds(info.templateIds)
			g.It("gets templates", func() {
				Expect(err).To(BeNil())
			})
			for _, template := range templates {
				ga, err := common.NewGeneratedAssetFromSourceAsset(sourceAsset, template.Id, uploader.Url(sourceAsset, template, 0))
				g.It("creates generated asset", func() {
					Expect(err).To(BeNil())
				})
				err = gasm.Store(ga)
				g.It("stores generated asset", func() {
					Expect(err).To(BeNil())
				})
				log.Println(ga)
			}

			g.It("creates correct amount of generated assets", func() {
				Expect(assertGeneratedAssetCount(sourceAssetId, gasm, common.GeneratedAssetStatusComplete, info.expectedCount)).To(BeTrue())
			})

		}
	})
}

func assertGeneratedAssetCount(id string, generatedAssetStorageManager common.GeneratedAssetStorageManager, status string, expectedCount int) bool {
	// This will get killed by the testing framework if it exceeds the timeout
	for {
		generatedAssets, err := generatedAssetStorageManager.FindBySourceAssetId(id)
		if err == nil {
			count := 0
			for _, generatedAsset := range generatedAssets {
				if generatedAsset.Status == status {
					count += 1
				}
			}
			if count > 0 {
				log.Println("Count is", count, "but wanted", expectedCount)
			}
			if count == expectedCount {
				return true
			}
		} else {
			return false
		}
		time.Sleep(1 * time.Second)
	}
}

func fileUrl(dir, file string) string {
	return "file://" + filepath.Join(common.Cwd(), "../", dir, file)
}
