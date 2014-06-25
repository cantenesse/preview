package common

type Template struct {
	Id          string
	RenderAgent string
	Group       string
	Attributes  []Attribute
}

// Templates are now defined in the configuration file
var (
	LegacyDefaultTemplates = []string{
		"04a2c710-8872-4c88-9c75-a67175d3a8e7",
		"2eee7c27-75e2-4682-9920-9a4e14caa433",
		"a89a6a0d-51d9-4d99-b278-0c5dfc538984",
		"eaa7be0e-354f-482c-ac75-75cbdafecb6e",
	}

	// These IDs are required for the old API to work
	DocumentConversionTemplateId = "9B17C6CE-7B09-4FD5-92AD-D85DD218D6D7"
	VideoConversionTemplateId    = "4128966B-9F69-4E56-AD5C-1FDB3C24F910"

	// TemplateAttributeHeight is a constant for the height attribute that can be set for templates.
	TemplateAttributeHeight = "height"
	// TemplateAttributeWidth is a constant for the width attribute that can be set for templates.
	TemplateAttributeWidth = "width"
	// TemplateAttributeOutput is a constant for the output attribute that can be set for templates.
	TemplateAttributeOutput = "output"
	// TemplateAttributePlaceholderSize is a constant for the placeholderSize attribute that can be set for templates.
	TemplateAttributePlaceholderSize = "placeholderSize"
)

func (template *Template) AddAttribute(name string, value []string) Attribute {
	attribute := Attribute{name, value}
	template.Attributes = append(template.Attributes, attribute)
	return attribute
}

func (template *Template) HasAttribute(name string) bool {
	for _, attribute := range template.Attributes {
		if attribute.Key == name {
			return true
		}
	}
	return false
}

func (template *Template) GetAttribute(key string) []string {
	for _, attribute := range template.Attributes {
		if attribute.Key == key {
			return attribute.Value
		}
	}
	return []string{}
}
