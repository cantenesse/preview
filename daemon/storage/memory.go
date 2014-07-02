package storage

import (
	"github.com/ngerakines/preview/common"
	"log"
	"time"
)

type inMemorySourceAssetStorageManager struct {
	sourceAssets []*common.SourceAsset
}

type inMemoryGeneratedAssetStorageManager struct {
	generatedAssets []*common.GeneratedAsset
	templateManager common.TemplateManager
}

func NewSourceAssetStorageManager() common.SourceAssetStorageManager {
	return &inMemorySourceAssetStorageManager{make([]*common.SourceAsset, 0, 0)}
}

func NewGeneratedAssetStorageManager(templateManager common.TemplateManager) common.GeneratedAssetStorageManager {
	return &inMemoryGeneratedAssetStorageManager{make([]*common.GeneratedAsset, 0, 0), templateManager}
}

func (sasm *inMemorySourceAssetStorageManager) Store(sourceAsset *common.SourceAsset) error {
	sasm.sourceAssets = append(sasm.sourceAssets, sourceAsset)
	return nil
}

func (sasm *inMemorySourceAssetStorageManager) FindBySourceAssetId(id string) ([]*common.SourceAsset, error) {
	results := make([]*common.SourceAsset, 0, 0)
	for _, sourceAsset := range sasm.sourceAssets {
		if sourceAsset.Id == id {
			results = append(results, sourceAsset)
		}
	}
	return results, nil
}

func (gasm *inMemoryGeneratedAssetStorageManager) Store(generatedAsset *common.GeneratedAsset) error {
	gasm.generatedAssets = append(gasm.generatedAssets, generatedAsset)
	return nil
}

func (gasm *inMemoryGeneratedAssetStorageManager) FindById(id string) (*common.GeneratedAsset, error) {
	for _, generatedAsset := range gasm.generatedAssets {
		if generatedAsset.Id == id {
			return generatedAsset, nil
		}
	}
	return nil, common.ErrorNoGeneratedAssetsFoundForId
}

func (gasm *inMemoryGeneratedAssetStorageManager) FindByIds(ids []string) ([]*common.GeneratedAsset, error) {
	results := make([]*common.GeneratedAsset, 0, 0)
	for _, generatedAsset := range gasm.generatedAssets {
		for _, id := range ids {
			if generatedAsset.Id == id {
				results = append(results, generatedAsset)
			}
		}
	}
	return results, nil
}

func (gasm *inMemoryGeneratedAssetStorageManager) FindBySourceAssetId(id string) ([]*common.GeneratedAsset, error) {
	results := make([]*common.GeneratedAsset, 0, 0)
	for _, generatedAsset := range gasm.generatedAssets {
		if generatedAsset.SourceAssetId == id {
			results = append(results, generatedAsset)
		}
	}
	return results, nil
}

func (gasm *inMemoryGeneratedAssetStorageManager) FindWorkForService(serviceName string, workCount int) ([]*common.GeneratedAsset, error) {
	templates, _ := gasm.templateManager.FindByRenderService(serviceName)
	log.Println("templates for", serviceName, ":", templates)
	results := make([]*common.GeneratedAsset, 0, 0)
	for _, generatedAsset := range gasm.generatedAssets {
		for _, template := range templates {
			if generatedAsset.TemplateId == template.Id {
				if generatedAsset.Status == common.GeneratedAssetStatusWaiting {
					generatedAsset.Status = common.GeneratedAssetStatusScheduled
					generatedAsset.UpdatedAt = time.Now().UnixNano()
					results = append(results, generatedAsset)
				}
				if len(results) >= workCount {
					return results, nil
				}
			}
		}
	}
	log.Println("generated assets for service", serviceName, ":", buildGeneratedAssetIds(results))
	return results, nil
}

func buildGeneratedAssetIds(generatedAssets []*common.GeneratedAsset) []string {
	results := make([]string, len(generatedAssets))
	for index, generatedAsset := range generatedAssets {
		results[index] = generatedAsset.Id
	}
	return results
}

func (gasm *inMemoryGeneratedAssetStorageManager) Update(givenGeneratedAsset *common.GeneratedAsset) error {
	for _, generatedAsset := range gasm.generatedAssets {
		if generatedAsset.Id == givenGeneratedAsset.Id {
			generatedAsset.Status = givenGeneratedAsset.Status
			generatedAsset.Attributes = givenGeneratedAsset.Attributes
			generatedAsset.UpdatedAt = time.Now().UnixNano()
			return nil
		}

	}
	return common.ErrorGeneratedAssetCouldNotBeUpdated
}
