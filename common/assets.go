package common

import (
	"encoding/json"
	"fmt"
	"github.com/ngerakines/preview/util"
	"log"
	"time"
)

// Attributed structures have internal attributes that can be manipulated.
type Attributed interface {
	// AddAttribute adds key and array of values to the structure.
	AddAttribute(name string, value []string) Attribute
	// HasAttribute returns true if the structure has a given attribute, false if otherwise.
	HasAttribute(name string) bool
	// GetAttribute retuirns the values for the given attribute.
	GetAttribute(key string) []string
}

// SourceAsset describes an asset that is used as a source of data for generated assets.
type SourceAsset struct {
	Id         string
	IdType     string
	CreatedAt  int64
	CreatedBy  string
	UpdatedAt  int64
	UpdatedBy  string
	Attributes []Attribute
}

// GeneratedAsset describes an asset that is generated by the system from a source asset.
type GeneratedAsset struct {
	Id              string
	SourceAssetId   string
	SourceAssetType string
	TemplateId      string
	Location        string
	Status          string
	CreatedAt       int64
	CreatedBy       string
	UpdatedAt       int64
	UpdatedBy       string
	Attributes      []Attribute
}

// Attribute is simply a key/value pair container used by source assets, generated assets and templates.
type Attribute struct {
	Key   string
	Value []string
}

var (
	// GeneratedAssetStatusWaiting is the initial, unprocessed state of a generated asset.
	GeneratedAssetStatusWaiting = "waiting"
	// GeneratedAssetStatusScheduled is the state of a generated asset that indicates that it has been scheduled for processing but processing has not yet begun.
	GeneratedAssetStatusScheduled = "scheduled"
	// GeneratedAssetStatusProcessing is the state of a generated asset that indicates that processing has begun.
	GeneratedAssetStatusProcessing = "processing"
	// GeneratedAssetStatusComplete is the state of a generated asset that indicates that processing has completed.
	GeneratedAssetStatusComplete = "complete"
	// GeneratedAssetStatusFailed is the state of a generated asset that indicates that processing has completed, but failed. This value is just a prefix and is accompanied by a coded error.
	GeneratedAssetStatusFailed = "failed"
	// GeneratedAssetStatusDelegated means that the asset is processing in a 3rd party tool (Zencoder)
	GeneratedAssetStatusDelegated = "delegated"
	// DefaultGeneratedAssetStatus is the default state of a generated asset when it is created.
	DefaultGeneratedAssetStatus = GeneratedAssetStatusWaiting

	// SourceAssetAttributeSource is a constant for the source attribute that can be set for source assets.
	SourceAssetAttributeSource = "source"
	// SourceAssetAttributeType is a constant for the type attribute that can be set for source assets.
	SourceAssetAttributeType = "type"
	// SourceAssetAttributeSize is a constant for the size attribute that can be set for source assets.
	SourceAssetAttributeSize = "size"
	// SourceAssetAttributePages is a constant for the pages attribute that can be set for source assets.
	SourceAssetAttributePages = "pages"

	// GeneratedAssetAttributePage is a constant for the page attribute that can be set for generated assets.
	GeneratedAssetAttributePage = "page"

	// SourceAssetTypeOrigin is a constant that represents origin types for source assets.
	SourceAssetTypeOrigin = "origin"
	// SourceAssetTypePdf is a constant that represents a generated PDF type for source assets.
	SourceAssetTypePdf = "pdf"
)

// NewSourceAsset creates a new source asset, filling in default values for everything but the id, type and location.
func NewSourceAsset(id, idType string) (*SourceAsset, error) {
	now := time.Now().UnixNano()
	sa := new(SourceAsset)
	sa.Id = id
	sa.IdType = idType
	sa.CreatedAt = now
	sa.CreatedBy = ""
	sa.UpdatedAt = now
	sa.UpdatedBy = ""
	sa.Attributes = make([]Attribute, 0, 0)
	return sa, nil
}

func NewSourceAssetFromJson(payload []byte) (*SourceAsset, error) {
	var sa SourceAsset

	err := json.Unmarshal(payload, &sa)
	if err != nil {
		return nil, err
	}
	return &sa, nil
}

// NewGeneratedAssetFromSourceAsset creates a new generated asset from a given source asset and template, filling in everything but location.
func NewGeneratedAssetFromSourceAsset(sourceAsset *SourceAsset, templateId, location string) (*GeneratedAsset, error) {
	uuid, err := util.NewUuid()
	if err != nil {
		return nil, err
	}
	now := time.Now().UnixNano()
	ga := new(GeneratedAsset)
	ga.Id = uuid
	ga.SourceAssetId = sourceAsset.Id
	ga.SourceAssetType = sourceAsset.IdType
	ga.TemplateId = templateId
	ga.Location = location
	ga.Status = DefaultGeneratedAssetStatus
	ga.CreatedAt = now
	ga.CreatedBy = ""
	ga.UpdatedAt = now
	ga.UpdatedBy = ""
	ga.Attributes = make([]Attribute, 0, 0)
	return ga, nil
}

func NewGeneratedAssetFromJson(payload []byte) (*GeneratedAsset, error) {
	var ga GeneratedAsset

	err := json.Unmarshal(payload, &ga)
	if err != nil {
		return nil, err
	}
	return &ga, nil
}

func (sa *SourceAsset) Serialize() ([]byte, error) {
	bytes, err := json.Marshal(sa)
	log.Println("Serialized source asset is", string(bytes))
	if err != nil {
		return nil, err
	}
	return bytes, nil
}

func (sa *SourceAsset) AddAttribute(name string, value []string) Attribute {
	attribute := Attribute{name, value}
	sa.Attributes = append(sa.Attributes, attribute)
	return attribute
}

func (sa *SourceAsset) HasAttribute(name string) bool {
	for _, attribute := range sa.Attributes {
		if attribute.Key == name {
			return true
		}
	}
	return false
}

func (sa *SourceAsset) GetAttribute(key string) []string {
	for _, attribute := range sa.Attributes {
		if attribute.Key == key {
			return attribute.Value
		}
	}
	return []string{}
}

func (ga *GeneratedAsset) Serialize() ([]byte, error) {
	bytes, err := json.Marshal(ga)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}

func (ga *GeneratedAsset) String() string {
	bytes, err := ga.Serialize()
	if err != nil {
		return ga.Id
	}
	return string(bytes)
}

func (ga *GeneratedAsset) AddAttribute(name string, value []string) Attribute {
	attribute := Attribute{name, value}
	ga.Attributes = append(ga.Attributes, attribute)
	return attribute
}

func (ga *GeneratedAsset) HasAttribute(name string) bool {
	for _, attribute := range ga.Attributes {
		if attribute.Key == name {
			return true
		}
	}
	return false
}

func (ga *GeneratedAsset) GetAttribute(key string) []string {
	for _, attribute := range ga.Attributes {
		if attribute.Key == key {
			return attribute.Value
		}
	}
	return []string{}
}

func GetFirstAttribute(attributed Attributed, key string) (string, error) {
	values := attributed.GetAttribute(key)
	if len(values) > 0 {
		return values[0], nil
	}
	// TODO: write this code
	return "", ErrorNotImplemented
}

func SourceAssetSource(sourceAsset *SourceAsset) string {
	return fmt.Sprintf("%s:%s", sourceAsset.Id, sourceAsset.IdType)
}
