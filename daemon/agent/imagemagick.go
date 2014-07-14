package agent

import (
	"github.com/ngerakines/codederror"
	"github.com/ngerakines/preview/common"
	"log"
	"strconv"
)

func init() {
	rendererConstructors["imageMagickRenderAgent"] = newImageMagickRenderer
	localTemplates := []*common.Template{
		&common.Template{
			Id:          "04a2c710-8872-4c88-9c75-a67175d3a8e7",
			RenderAgent: "imageMagickRenderAgent",
			Group:       "4C96",
			Attributes: []common.Attribute{
				common.Attribute{common.TemplateAttributeWidth, []string{"1040"}},
				common.Attribute{common.TemplateAttributeHeight, []string{"780"}},
				common.Attribute{common.TemplateAttributeDensity, []string{"144"}},
				common.Attribute{common.TemplateAttributeOutput, []string{"jpg"}},
				common.Attribute{common.TemplateAttributePlaceholderSize, []string{common.PlaceholderSizeJumbo}},
			},
		},
		&common.Template{
			Id:          "2eee7c27-75e2-4682-9920-9a4e14caa433",
			RenderAgent: "imageMagickRenderAgent",
			Group:       "4C96",
			Attributes: []common.Attribute{
				common.Attribute{common.TemplateAttributeWidth, []string{"520"}},
				common.Attribute{common.TemplateAttributeHeight, []string{"390"}},
				common.Attribute{common.TemplateAttributeDensity, []string{"144"}},
				common.Attribute{common.TemplateAttributeOutput, []string{"jpg"}},
				common.Attribute{common.TemplateAttributePlaceholderSize, []string{common.PlaceholderSizeLarge}},
			},
		},
		&common.Template{
			Id:          "a89a6a0d-51d9-4d99-b278-0c5dfc538984",
			RenderAgent: "imageMagickRenderAgent",
			Group:       "4C96",
			Attributes: []common.Attribute{
				common.Attribute{common.TemplateAttributeWidth, []string{"500"}},
				common.Attribute{common.TemplateAttributeHeight, []string{"376"}},
				common.Attribute{common.TemplateAttributeDensity, []string{"144"}},
				common.Attribute{common.TemplateAttributeOutput, []string{"jpg"}},
				common.Attribute{common.TemplateAttributePlaceholderSize, []string{common.PlaceholderSizeMedium}},
			},
		},
		&common.Template{
			Id:          "eaa7be0e-354f-482c-ac75-75cbdafecb6e",
			RenderAgent: "imageMagickRenderAgent",
			Group:       "4C96",
			Attributes: []common.Attribute{
				common.Attribute{common.TemplateAttributeWidth, []string{"250"}},
				common.Attribute{common.TemplateAttributeHeight, []string{"188"}},
				common.Attribute{common.TemplateAttributeDensity, []string{"144"}},
				common.Attribute{common.TemplateAttributeOutput, []string{"jpg"}},
				common.Attribute{common.TemplateAttributePlaceholderSize, []string{common.PlaceholderSizeSmall}},
			},
		},
	}
	templates = append(templates, localTemplates...)
}

type imageMagickRenderer struct {
	maxPages int
}

func newImageMagickRenderer(params map[string]string) Renderer {
	renderer := new(imageMagickRenderer)
	maxPagesString, ok := params["maxPages"]
	if !ok {
		log.Fatal("Missing maxPages parameter from imageMagickRenderAgent")
	}
	renderer.maxPages, _ = strconv.Atoi(maxPagesString)

	return renderer
}

func (renderer *imageMagickRenderer) renderGeneratedAsset(id string, renderAgent *genericRenderAgent) {
	renderAgent.metrics.workProcessed.Mark(1)

	generatedAsset, err := renderAgent.gasm.FindById(id)
	if err != nil {
		log.Println("No Generated Asset with that ID can be retreived from storage: ", id)
		return
	}

	statusCallback := renderAgent.commitStatus(generatedAsset.Id, generatedAsset.Attributes)
	defer func() { close(statusCallback) }()

	generatedAsset.Status = common.GeneratedAssetStatusProcessing
	renderAgent.gasm.Update(generatedAsset)

	sourceAsset, err := renderAgent.getSourceAsset(generatedAsset)
	if err != nil {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorUnableToFindSourceAssetsById), nil}
		return
	}

	fileType, err := getSourceAssetFileType(sourceAsset)
	if err != nil {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorCouldNotDetermineFileType), nil}
		return
	}
	renderAgent.metrics.fileTypeCount[fileType].Inc(1)

	templates, err := renderAgent.templateManager.FindByIds([]string{generatedAsset.TemplateId})
	if err != nil {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorUnableToFindTemplatesById), nil}
		return
	}
	if len(templates) == 0 {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorNoTemplatesFoundForId), nil}
		return
	}
	template := templates[0]

	urls := sourceAsset.GetAttribute(common.SourceAssetAttributeSource)
	sourceFile, err := renderAgent.tryDownload(urls, common.SourceAssetSource(sourceAsset))
	if err != nil {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorNoDownloadUrlsWork), nil}
		return
	}
	defer sourceFile.Release()

	destination := sourceFile.Path() + "-" + template.Id + ".jpg"
	destinationTemporaryFile := renderAgent.temporaryFileManager.Create(destination)
	defer destinationTemporaryFile.Release()

	size, err := getSize(template)
	if err != nil {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorCouldNotDetermineRenderSize), nil}
		return
	}

	density, err := getDensity(template)
	if err != nil {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorCouldNotDetermineRenderDensity), nil}
		return
	}

	var ce codederror.CodedError
	renderAgent.metrics.convertTime.Time(func() {
		if fileType == "pdf" {
			page, _ := getGeneratedAssetPage(generatedAsset)
			if page == 0 {
				pages, err := getPdfPageCount(sourceFile.Path())
				if err != nil {
					statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorNotImplemented), nil}
					return
				}
				// Create derived work for all pages but first one
				renderAgent.agentManager.CreateDerivedWork(sourceAsset, templates, 1, common.Min(pages, renderer.maxPages))
			}
			ce = imageFromPdf(sourceFile.Path(), destination, size, page, density, renderAgent.fileTypes[fileType])
		} else if fileType == "gif" {
			ce = firstGifFrame(sourceFile.Path(), destination, size, renderAgent.fileTypes[fileType])
		} else {
			ce = resize(sourceFile.Path(), destination, size, renderAgent.fileTypes[fileType])
		}
		if ce != nil {
			statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(ce), nil}
		}
	})

	if ce != nil {
		return
	}

	log.Println("---- generated asset is at", destination, "can load file?", common.CanLoadFile(destination))

	err = renderAgent.uploader.Upload(generatedAsset.Location, destination)
	if err != nil {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorCouldNotUploadAsset), nil}
		return
	}

	bounds, err := getBounds(destination)
	if err != nil {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorCouldNotDetermineRenderSize), nil}
		return
	}

	generatedAssetFileSize, err := common.FileSize(destination)
	if err != nil {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorCouldNotDetermineFileSize), nil}
		return
	}

	newAttributes := []common.Attribute{
		generatedAsset.AddAttribute("imageHeight", []string{strconv.Itoa(bounds.Max.X)}),
		generatedAsset.AddAttribute("imageWidth", []string{strconv.Itoa(bounds.Max.Y)}),
		// NKG: I'm sure this is going to break something.
		generatedAsset.AddAttribute("fileSize", []string{strconv.FormatInt(generatedAssetFileSize, 10)}),
	}

	statusCallback <- generatedAssetUpdate{common.GeneratedAssetStatusComplete, newAttributes}
}
