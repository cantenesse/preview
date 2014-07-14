package agent

import (
	"github.com/ngerakines/codederror"
	"github.com/ngerakines/preview/common"
	"log"
	"os"
	"path/filepath"
	"strconv"
)

func init() {
	rendererConstructors["documentRenderAgent"] = newDocumentRenderer
}

type documentRenderer struct {
	tempFileBasePath string
}

func (renderer *documentRenderer) getTemplates() []*common.Template {
	templates := []*common.Template{
		&common.Template{
			Id:          common.DocumentConversionTemplateId,
			RenderAgent: "documentRenderAgent",
			Group:       "A907",
			Attributes: []common.Attribute{
				common.Attribute{common.TemplateAttributeOutput, []string{"pdf"}},
			},
		},
	}
	return templates
}

func newDocumentRenderer(params map[string]string) Renderer {
	renderer := new(documentRenderer)
	var ok bool
	renderer.tempFileBasePath, ok = params["tempFileBasePath"]
	if !ok {
		log.Fatal("Missing tempFileBasePath parameter from documentRenderAgent")
	}

	return renderer
}

/*
1. Get the generated asset
2. Get the source asset
3. Get the template
4. Fetch the source asset file
5. Create a temporary destination directory.
6. Convert the source asset file into a pdf using the temporary destination directory.
7. Given a file in that directory exists, determine how many pages it contains.
8. Create a new source asset record for the pdf.
9. Upload the new source asset pdf file.
10. For each page in the pdf, create a generated asset record for each of the default templates.
11. Update the status of the generated asset as complete.
*/
func (renderer *documentRenderer) renderGeneratedAsset(id string, renderAgent *genericRenderAgent) {
	renderAgent.metrics.workProcessed.Mark(1)

	// 1. Get the generated asset
	generatedAsset, err := renderAgent.gasm.FindById(id)
	if err != nil {
		log.Println("No Generated Asset with that ID can be retreived from storage: ", id)
		return
	}

	statusCallback := renderAgent.commitStatus(generatedAsset.Id, generatedAsset.Attributes)
	defer func() { close(statusCallback) }()

	generatedAsset.Status = common.GeneratedAssetStatusProcessing
	renderAgent.gasm.Update(generatedAsset)

	// 2. Get the source asset
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

	// 3. Get the template... not needed yet

	// 4. Fetch the source asset file
	urls := sourceAsset.GetAttribute(common.SourceAssetAttributeSource)
	sourceFile, err := renderAgent.tryDownload(urls, common.SourceAssetSource(sourceAsset))
	if err != nil {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorNoDownloadUrlsWork), nil}
		return
	}
	defer sourceFile.Release()

	// 5. Create a temporary destination directory.
	destination, err := renderer.createTemporaryDestinationDirectory()
	if err != nil {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorNotImplemented), nil}
		return
	}
	destinationTemporaryFile := renderAgent.temporaryFileManager.Create(destination)
	defer destinationTemporaryFile.Release()

	var ce codederror.CodedError
	renderAgent.metrics.convertTime.Time(func() {
		ce = createPdf(sourceFile.Path(), destination, renderAgent.fileTypes[fileType])
		if ce != nil {
			statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(ce), nil}
		}
	})

	if ce != nil {
		return
	}

	files, err := getRenderedFiles(destination)
	if err != nil {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorNotImplemented), nil}
		return
	}
	if len(files) != 1 {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorNotImplemented), nil}
		return
	}

	pages, err := getPdfPageCount(files[0])
	if err != nil {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorNotImplemented), nil}
		return
	}

	err = renderAgent.uploader.Upload(generatedAsset.Location, files[0])
	if err != nil {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorCouldNotUploadAsset), nil}
		return
	}

	pdfFileSize, err := common.FileSize(destination)
	if err != nil {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorCouldNotDetermineFileSize), nil}
		return
	}

	pdfSourceAsset, err := common.NewSourceAsset(sourceAsset.Id, common.SourceAssetTypePdf)
	if err != nil {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorNotImplemented), nil}
		return
	}

	pdfSourceAsset.AddAttribute(common.SourceAssetAttributeSize, []string{strconv.FormatInt(pdfFileSize, 10)})
	pdfSourceAsset.AddAttribute(common.SourceAssetAttributePages, []string{strconv.Itoa(pages)})
	pdfSourceAsset.AddAttribute(common.SourceAssetAttributeSource, []string{generatedAsset.Location})
	pdfSourceAsset.AddAttribute(common.SourceAssetAttributeType, []string{"pdf"})
	// TODO: Add support for the expiration attribute.

	log.Println("pdfSourceAsset", pdfSourceAsset)
	renderAgent.sasm.Store(pdfSourceAsset)
	legacyDefaultTemplates, err := renderAgent.templateManager.FindByIds(common.LegacyDefaultTemplates)
	if err != nil {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorNotImplemented), nil}
		return
	}
	// Only process first page because imageMagickRenderAgent will automatically create derived work for the other pages
	renderAgent.agentManager.CreateDerivedWork(pdfSourceAsset, legacyDefaultTemplates, 0, 1)

	statusCallback <- generatedAssetUpdate{common.GeneratedAssetStatusComplete, nil}
}

func (renderer *documentRenderer) createTemporaryDestinationDirectory() (string, error) {
	uuid, err := common.NewUuid()
	if err != nil {
		return "", err
	}
	tmpPath := filepath.Join(renderer.tempFileBasePath, uuid)
	err = os.MkdirAll(tmpPath, 0777)
	if err != nil {
		log.Println("error creating tmp dir", err)
		return "", err
	}
	return tmpPath, nil
}
