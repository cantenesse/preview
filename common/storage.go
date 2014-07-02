package common

import (
	"strings"
)

type SourceAssetStorageManager interface {
	Store(sourceAsset *SourceAsset) error
	FindBySourceAssetId(id string) ([]*SourceAsset, error)
}

type GeneratedAssetStorageManager interface {
	Store(generatedAsset *GeneratedAsset) error
	Update(generatedAsset *GeneratedAsset) error
	FindById(id string) (*GeneratedAsset, error)
	FindByIds(ids []string) ([]*GeneratedAsset, error)
	FindBySourceAssetId(id string) ([]*GeneratedAsset, error)
	FindWorkForService(serviceName string, workCount int) ([]*GeneratedAsset, error)
}

type TemplateManager interface {
	Store(template *Template) error
	FindByIds(id []string) ([]*Template, error)
	FindByRenderService(renderService string) ([]*Template, error)
}

type inMemoryTemplateManager struct {
	templates []*Template
}

func NewTemplateManager() TemplateManager {
	tm := new(inMemoryTemplateManager)
	tm.templates = make([]*Template, 0, 0)
	return tm
}

// NKG: This is lazy, I know.
func BuildIn(count int) string {
	results := make([]string, 0, 0)
	for i := 0; i < count; i++ {
		results = append(results, "?")
	}
	return strings.Join(results, ",")
}

func (tm *inMemoryTemplateManager) Store(template *Template) error {
	tm.templates = append(tm.templates, template)
	return nil

}

func (tm *inMemoryTemplateManager) FindByIds(ids []string) ([]*Template, error) {
	results := make([]*Template, 0, 0)
	for _, template := range tm.templates {
		for _, id := range ids {
			if template.Id == id {
				results = append(results, template)
			}
		}
	}
	return results, nil
}

func (tm *inMemoryTemplateManager) FindByRenderService(renderService string) ([]*Template, error) {
	results := make([]*Template, 0, 0)
	for _, template := range tm.templates {
		if template.RenderAgent == renderService {
			results = append(results, template)
		}
	}
	return results, nil
}
