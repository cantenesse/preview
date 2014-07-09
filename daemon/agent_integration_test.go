package daemon

import (
	. "github.com/franela/goblin"
	"github.com/ngerakines/preview/common"
	"github.com/ngerakines/preview/daemon/agent"
	"github.com/ngerakines/preview/daemon/storage"
	"github.com/ngerakines/testutils"
	"github.com/rcrowley/go-metrics"
	"log"
	"testing"
	//	. "github.com/onsi/gomega"
	"path/filepath"
	"time"
)

type testInfo struct {
	file          string
	fileType      string
	templateIds   []string
	expectedCount int
}

func Test(t *testing.T) {
	g := Goblin(t)
	_ = g

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

	for _, info := range tests {
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

		sourceAsset.AddAttribute(common.SourceAssetAttributeSize, []string{"12345"})
		sourceAsset.AddAttribute(common.SourceAssetAttributeSource, []string{fileUrl("test-data", info.file)})
		sourceAsset.AddAttribute(common.SourceAssetAttributeType, []string{info.fileType})

		err = sasm.Store(sourceAsset)
		if err != nil {
			t.Errorf("Unexpected error returned: %s", err)
			return
		}

		templates, err := tm.FindByIds(info.templateIds)
		if err != nil {
			t.Errorf("Unexpected error returned: %s", err)
			return
		}
		for _, template := range templates {
			ga, err := common.NewGeneratedAssetFromSourceAsset(sourceAsset, template.Id, uploader.Url(sourceAsset, template, 0))
			if err != nil {
				t.Errorf("Unexpected error returned: %s", err)
				return
			}
			gasm.Store(ga)
			log.Println(ga)
		}

		if assertGeneratedAssetCount(sourceAssetId, gasm, common.GeneratedAssetStatusComplete, info.expectedCount) {
			t.Errorf("Could not verify that %d generated assets had status '%s' for source asset '%s'", info.expectedCount, common.GeneratedAssetStatusComplete, sourceAssetId)
			return
		}
	}
}

func assertGeneratedAssetCount(id string, generatedAssetStorageManager common.GeneratedAssetStorageManager, status string, expectedCount int) bool {
	callback := make(chan bool)
	go func() {
		for {
			generatedAssets, err := generatedAssetStorageManager.FindBySourceAssetId(id)
			if err == nil {
				count := 0
				for _, generatedAsset := range generatedAssets {
					if generatedAsset.Status == status {
						count = count + 1
					}
				}
				if count > 0 {
					log.Println("Count is", count, "but wanted", expectedCount)
				}
				if count == expectedCount {
					callback <- false
				}
			} else {
				callback <- true
			}
			time.Sleep(1 * time.Second)
		}
	}()

	for {
		select {
		case result := <-callback:
			return result
		case <-time.After(20 * time.Second):
			generatedAssets, err := generatedAssetStorageManager.FindBySourceAssetId(id)
			log.Println("Timed out. generatedAssets", generatedAssets, "err", err)
			return true
		}
	}
}

func fileUrl(dir, file string) string {
	return "file://" + filepath.Join(common.Cwd(), "../", dir, file)
}
