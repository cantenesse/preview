package agent

import (
	"encoding/json"
	"github.com/ngerakines/preview/common"
	"github.com/ngerakines/preview/util"
	"github.com/ngerakines/testutils"
	"github.com/rcrowley/go-metrics"
	"log"
	"path/filepath"
	"testing"
	"time"
)

type Template struct {
	Id          string              `json:"id"`
	RenderAgent string              `json:"renderAgent"`
	Group       string              `json:"group"`
	Attributes  map[string][]string `json:"attributes"`
}

var templatejson = `[
        {
            "id":"04a2c710-8872-4c88-9c75-a67175d3a8e7",
            "renderAgent":"imageMagickRenderAgent",
            "group":"4C96",
            "attributes":{
                "width":["1040"],
                "height":["780"],
                "output":["jpg"],
                "placeholderSize":["jumbo"]
            }
        },
	{
            "id":"2eee7c27-75e2-4682-9920-9a4e14caa433",
            "renderAgent":"imageMagickRenderAgent",
            "group":"4C96",
            "attributes":{
                "width":["520"],
                "height":["390"],
                "output":["jpg"],
                "placeholderSize":["large"]
            }
        },
	{
            "id":"a89a6a0d-51d9-4d99-b278-0c5dfc538984",
            "renderAgent":"imageMagickRenderAgent",
            "group":"4C96",
            "attributes":{
                "width":["500"],
                "height":["376"],
                "output":["jpg"],
                "placeholderSize":["medium"]
            }
        },
	{
            "id":"eaa7be0e-354f-482c-ac75-75cbdafecb6e",
            "renderAgent":"imageMagickRenderAgent",
            "group":"4C96",
            "attributes":{
                "width":["250"],
                "height":["188"],
                "output":["jpg"],
                "placeholderSize":["small"]
            }
        },
	{
            "id":"9B17C6CE-7B09-4FD5-92AD-D85DD218D6D7",
            "renderAgent":"documentRenderAgent",
            "group":"A907",
            "attributes":{
                "output":["pdf"]
            }
        },
	{
            "id":"4128966B-9F69-4E56-AD5C-1FDB3C24F910",
            "renderAgent":"videoRenderAgent",
            "group":"7A96",
            "attributes":{
                "output":["m3u8"],
		"forceS3Location":["true"],
                "zencoderNotificationUrl":["http://example.com/zencoderhandler"]
            }
        }
    ]`

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
		log.Println("ga", ga)
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
	templates := make([]*Template, 0)
	json.Unmarshal([]byte(templatejson), &templates)
	for _, template := range templates {
		temp := new(common.Template)
		temp.Id = template.Id
		temp.RenderAgent = template.RenderAgent
		temp.Group = template.Group
		for k, v := range template.Attributes {
			temp.AddAttribute(k, v)
		}
		tm.Store(temp)
	}

	sourceAssetStorageManager := common.NewSourceAssetStorageManager()
	generatedAssetStorageManager := common.NewGeneratedAssetStorageManager(tm)

	tfm := common.NewTemporaryFileManager()
	downloader := common.NewDownloader(path, path, tfm, false, []string{}, nil)
	uploader := common.NewLocalUploader(path)
	registry := metrics.NewRegistry()
	supportedFileTypes := make(map[string][]string)
	supportedFileTypes["imageMagickRenderAgent"] = []string{"jpeg", "jpg", "png", "pdf", "gif"}
	supportedFileTypes["documentRenderAgent"] = []string{"docx"}
	rm := NewRenderAgentManager(registry, sourceAssetStorageManager, generatedAssetStorageManager, tm, tfm, uploader, true, nil, supportedFileTypes)

	rendererParams := make(map[string]string)
	rm.AddRenderAgent("imageMagickRenderAgent", rendererParams, downloader, uploader, 5)
	rendererParams["tempFileBasePath"] = filepath.Join(path, "doc-cache")
	rm.AddRenderAgent("documentRenderAgent", rendererParams, downloader, uploader, 5)

	return rm, sourceAssetStorageManager, generatedAssetStorageManager, tm, uploader
}

func fileUrl(dir, file string) string {
	return "file://" + filepath.Join(util.Cwd(), "../", dir, file)
}
