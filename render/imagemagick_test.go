package render

import (
	"github.com/ngerakines/preview/common"
	"github.com/ngerakines/preview/util"
	"github.com/ngerakines/testutils"
	"github.com/rcrowley/go-metrics"
	"log"
	"path/filepath"
	"testing"
	"time"
)

func init() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
}

// TODO: Write tests for source assets. (i.e. missing, missing/invalid attributes)
// TODO: Write tests for generated assets. (i.e. missing, missing/invalid attributes)
// TODO: Write test for PDF with 0 pages.

func basicTest(t *testing.T, testname, file, filetype string, expectedOutputSize int) {
	if !testutils.Integration() || testing.Short() {
		t.Skip("Skipping integration test", testname)
		return
	}

	common.DumpErrors()

	dm := testutils.NewDirectoryManager()
	defer dm.Close()

	rm, sasm, gasm, tm, uploader := setupTest(dm.Path)
	defer rm.Stop()

	testListener := make(RenderStatusChannel, 25)
	rm.AddListener(testListener)

	sourceAssetId, err := util.NewUuid()
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
	sourceAsset.AddAttribute(common.SourceAssetAttributeSource, []string{fileUrl("test-data", file)})
	sourceAsset.AddAttribute(common.SourceAssetAttributeType, []string{filetype})

	err = sasm.Store(sourceAsset)
	if err != nil {
		t.Errorf("Unexpected error returned: %s", err)
		return
	}

	templates, err := tm.FindByIds(common.LegacyDefaultTemplates)
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

	if assertGeneratedAssetCount(sourceAssetId, gasm, common.GeneratedAssetStatusComplete, expectedOutputSize) {
		t.Errorf("Could not verify that %d generated assets had status '%s' for source asset '%s'", expectedOutputSize, common.GeneratedAssetStatusComplete, sourceAssetId)
		return
	}
}

func TestRenderJpegPreview(t *testing.T) {
	basicTest(t, "TestRenderJpegPreview", "wallpaper-641916.jpg", "jpg", 4)
}

func TestRenderPdfPreview(t *testing.T) {
	basicTest(t, "TestRenderPdfPreview", "ChefConf2014schedule.pdf", "pdf", 4)
}

func TestRenderMultipagePdfPreview(t *testing.T) {
	basicTest(t, "TestRenderMultipagePdfPreview", "Multipage.pdf", "pdf", 12)
}

func TestRenderGifPreview(t *testing.T) {
	basicTest(t, "TestRenderGifPreview", "Animated.gif", "gif", 4)
}

func TestRenderPngPreview(t *testing.T) {
	basicTest(t, "TestRenderPngPreview", "COW.png", "png", 4)
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

func setupTest(path string) (*RenderAgentManager, common.SourceAssetStorageManager, common.GeneratedAssetStorageManager, common.TemplateManager, common.Uploader) {
	tm := common.NewTemplateManager()
	sourceAssetStorageManager := common.NewSourceAssetStorageManager()
	generatedAssetStorageManager := common.NewGeneratedAssetStorageManager(tm)

	tfm := common.NewTemporaryFileManager()
	downloader := common.NewDownloader(path, path, tfm, false, []string{}, nil)
	uploader := common.NewLocalUploader(path)
	registry := metrics.NewRegistry()
	rm := NewRenderAgentManager(registry, sourceAssetStorageManager, generatedAssetStorageManager, tm, tfm, uploader, true, nil, "", "", []string{"docx"}, []string{"jpeg", "jpg", "png", "pdf", "gif"}, nil)

	rm.AddImageMagickRenderAgent(downloader, uploader, 5)
	rm.AddDocumentRenderAgent(downloader, uploader, filepath.Join(path, "doc-cache"), 5)

	return rm, sourceAssetStorageManager, generatedAssetStorageManager, tm, uploader
}

func fileUrl(dir, file string) string {
	return "file://" + filepath.Join(util.Cwd(), "../", dir, file)
}
