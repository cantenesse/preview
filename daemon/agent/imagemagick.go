package agent

import (
	"github.com/ngerakines/preview/common"
	"log"
	"strconv"
)

func init() {
	renderers["imageMagickRenderAgent"] = newImageMagickRenderer
}

type imageMagickRenderer struct {
	renderAgent *genericRenderAgent
	maxPages    int
}

func newImageMagickRenderer(renderAgent *genericRenderAgent, params map[string]string) Renderer {
	renderer := new(imageMagickRenderer)
	renderer.renderAgent = renderAgent
	renderer.maxPages, _ = strconv.Atoi(params["maxPages"])

	return renderer
}

func (renderer *imageMagickRenderer) renderGeneratedAsset(id string) {
	renderer.renderAgent.metrics.workProcessed.Mark(1)

	generatedAsset, err := renderer.renderAgent.gasm.FindById(id)
	if err != nil {
		log.Println("No Generated Asset with that ID can be retreived from storage: ", id)
		return
	}

	statusCallback := renderer.renderAgent.commitStatus(generatedAsset.Id, generatedAsset.Attributes)
	defer func() { close(statusCallback) }()

	generatedAsset.Status = common.GeneratedAssetStatusProcessing
	renderer.renderAgent.gasm.Update(generatedAsset)

	sourceAsset, err := renderer.renderAgent.getSourceAsset(generatedAsset)
	if err != nil {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorUnableToFindSourceAssetsById), nil}
		return
	}

	fileType, err := getSourceAssetFileType(sourceAsset)
	if err != nil {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorCouldNotDetermineFileType), nil}
		return
	}
	renderer.renderAgent.metrics.fileTypeCount[fileType].Inc(1)

	templates, err := renderer.renderAgent.templateManager.FindByIds([]string{generatedAsset.TemplateId})
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
	sourceFile, err := renderer.renderAgent.tryDownload(urls, common.SourceAssetSource(sourceAsset))
	if err != nil {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorNoDownloadUrlsWork), nil}
		return
	}
	defer sourceFile.Release()

	destination := sourceFile.Path() + "-" + template.Id + ".jpg"
	destinationTemporaryFile := renderer.renderAgent.temporaryFileManager.Create(destination)
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
	renderer.renderAgent.metrics.convertTime.Time(func() {
		if fileType == "pdf" {
			page, _ := getGeneratedAssetPage(generatedAsset)
			if page == 0 {
				pages, err := getPdfPageCount(sourceFile.Path())
				if err != nil {
					statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorNotImplemented), nil}
					return
				}
				// Create derived work for all pages but first one
				renderer.renderAgent.agentManager.CreateDerivedWork(sourceAsset, templates, 1, common.Min(pages, renderer.maxPages))
			}
			err = imageFromPdf(sourceFile.Path(), destination, size, page, density, renderer.renderAgent.fileTypes[fileType])
		} else if fileType == "gif" {
			err = firstGifFrame(sourceFile.Path(), destination, size, renderer.renderAgent.fileTypes[fileType])
		} else {
			err = resize(sourceFile.Path(), destination, size, renderer.renderAgent.fileTypes[fileType])
		}
		if err != nil {
			statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorCouldNotResizeImage), nil}
			return
		}
	})

	log.Println("---- generated asset is at", destination, "can load file?", common.CanLoadFile(destination))

	err = renderer.renderAgent.uploader.Upload(generatedAsset.Location, destination)
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
