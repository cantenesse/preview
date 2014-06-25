package api

import (
	"encoding/json"
	"fmt"
	"github.com/ngerakines/preview/common"
	"github.com/ngerakines/preview/render"
	"github.com/ngerakines/preview/util"
	"github.com/ngerakines/testutils"
	"github.com/rcrowley/go-metrics"
	"testing"
)

func TestApiGPR(t *testing.T) {

	ida, err := util.NewUuid()
	if err != nil {
		t.Errorf("Error generating UUIDs", err)
	}

	idb, err := util.NewUuid()
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
	if err != nil {
		t.Errorf("Error creating GPR", err)
	}
	if len(gprs) != 2 {
		t.Errorf("Should have created 2 GPRs")
	}
	emptyData := `
{
	"sourceAssets": [ ],
	"templateIds": [ ]
}
`
	gprs, err = newApiGeneratePreviewRequest(emptyData)
	if err != nil {
		t.Errorf("Error creating GPR", err)
	}
	if len(gprs) != 0 {
		t.Errorf("Should have created 0 GPRs")
	}
	invalidData := `
{
	"sourceAssets": [ "bla bla bla bla"],
	"templateIds": [ { "this":"is a completely invalid request"}]
}
`
	gprs, err = newApiGeneratePreviewRequest(invalidData)
	if err == nil {
		t.Errorf("Invalid data should have given an error")
	}
}

// From the render/ tests
func setupTest(path string) (*render.RenderAgentManager, common.SourceAssetStorageManager, common.GeneratedAssetStorageManager, common.TemplateManager, *apiBlueprint) {
	tm := common.NewTemplateManager()
	sourceAssetStorageManager := common.NewSourceAssetStorageManager()
	generatedAssetStorageManager := common.NewGeneratedAssetStorageManager(tm)

	tfm := common.NewTemporaryFileManager()
	//downloader := common.NewDownloader(path, path, tfm, false, []string{}, nil)
	uploader := common.NewLocalUploader(path)
	registry := metrics.NewRegistry()
	rm := render.NewRenderAgentManager(registry, sourceAssetStorageManager, generatedAssetStorageManager, tm, tfm, uploader, true, nil, nil)

	blueprint := NewApiBlueprint("/api/v2", rm, generatedAssetStorageManager, sourceAssetStorageManager, registry, nil)
	return rm, sourceAssetStorageManager, generatedAssetStorageManager, tm, blueprint
}

func TestGAMarshalling(t *testing.T) {
	common.DumpErrors()

	dm := testutils.NewDirectoryManager()
	defer dm.Close()

	rm, sasm, gasm, _, blueprint := setupTest(dm.Path)
	defer rm.Stop()

	testListener := make(render.RenderStatusChannel, 25)
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

	err = sasm.Store(sourceAsset)
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
	gasm.Store(ga)

	jsonData, err := blueprint.marshalGeneratedAssets(sourceAssetId, templateId, "")
	if err != nil {
		t.Errorf("Unexpected error returned: %s", err)
		return
	}
	var arr []*common.GeneratedAsset
	err = json.Unmarshal(jsonData, &arr)
	if err != nil {
		t.Errorf("Unexpected error returned: %s", err)
		return
	}
	if len(arr) != 1 {
		t.Errorf("Invalid GA array length")
		return
	}
	if arr[0].Id != ga.Id {
		t.Errorf("Invalid GA ID")
		return
	}
}
