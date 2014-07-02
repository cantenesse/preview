package agent

import (
	"bytes"
	"fmt"
	"github.com/ngerakines/preview/common"
	"image"
	"image/jpeg"
	"log"
	"os"
	"os/exec"
	"strconv"
)

func init() {
	Renderers["imageMagickRenderAgent"] = newImageMagickRenderer
}

type imageMagickRenderer struct {
	renderAgent *genericRenderAgent
}

func newImageMagickRenderer(renderAgent *genericRenderAgent, params map[string]string) Renderer {
	renderer := new(imageMagickRenderer)
	renderer.renderAgent = renderAgent

	return renderer
}

func (renderer *imageMagickRenderer) renderGeneratedAsset(id string) {
	renderer.renderAgent.metrics.workProcessed.Mark(1)

	generatedAsset, err := renderer.renderAgent.gasm.FindById(id)
	if err != nil {
		log.Fatal("No Generated Asset with that ID can be retreived from storage: ", id)
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

	fileType, err := renderer.getSourceAssetFileType(sourceAsset)
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

	size, err := renderer.getSize(template)
	if err != nil {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorCouldNotDetermineRenderSize), nil}
		return
	}

	density, err := renderer.renderAgent.getDensity(template)
	if err != nil {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorCouldNotDetermineRenderDensity), nil}
		return
	}
	renderer.renderAgent.metrics.convertTime.Time(func() {
		if fileType == "pdf" {
			page, _ := renderer.renderAgent.getGeneratedAssetPage(generatedAsset)
			if page == 0 {
				pages, err := common.GetPdfPageCount(sourceFile.Path())
				if err != nil {
					statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorNotImplemented), nil}
					return
				}
				// Create derived work for all pages but first one
				renderer.renderAgent.agentManager.CreateDerivedWork(sourceAsset, templates, 1, pages)
			}
			err = renderer.imageFromPdf(sourceFile.Path(), destination, size, page, density)
		} else if fileType == "gif" {
			err = renderer.firstGifFrame(sourceFile.Path(), destination, size)
		} else {
			err = renderer.resize(sourceFile.Path(), destination, size)
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

	bounds, err := renderer.getBounds(destination)
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

func (renderer *imageMagickRenderer) resize(source, destination string, size int) error {
	_, err := exec.LookPath("convert")
	if err != nil {
		log.Println("convert command not found")
		return err
	}

	cmd := exec.Command("convert", source, "-resize", strconv.Itoa(size), destination)
	log.Println(cmd)

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	err = cmd.Run()
	if err != nil {
		return err
	}
	log.Println(buf.String())

	return nil
}

func (renderer *imageMagickRenderer) imageFromPdf(source, destination string, size, page, density int) error {
	_, err := exec.LookPath("convert")
	if err != nil {
		log.Println("convert command not found")
		return err
	}

	cmd := exec.Command("convert", "-density", strconv.Itoa(density), "-colorspace", "RGB", fmt.Sprintf("%s[%d]", source, page), "-resize", strconv.Itoa(size), "-flatten", "+adjoin", destination)
	log.Println(cmd)

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	err = cmd.Run()
	if err != nil {
		return err
	}
	log.Println(buf.String())

	return nil
}

func (renderer *imageMagickRenderer) firstGifFrame(source, destination string, size int) error {
	_, err := exec.LookPath("convert")
	if err != nil {
		log.Println("convert command not found")
		return err
	}

	cmd := exec.Command("convert", fmt.Sprintf("%s[0]", source), "-resize", strconv.Itoa(size), destination)
	log.Println(cmd)

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	err = cmd.Run()
	if err != nil {
		return err
	}
	log.Println(buf.String())

	return nil
}

func (renderer *imageMagickRenderer) getBounds(path string) (*image.Rectangle, error) {
	reader, err := os.Open(path)
	if err != nil {
		log.Println("os.Open error", err)
		return nil, err
	}
	defer reader.Close()
	image, err := jpeg.Decode(reader)
	if err != nil {
		log.Println("jpeg.Decode error", err)
		return nil, err
	}
	bounds := image.Bounds()
	return &bounds, nil
}

func (renderer *imageMagickRenderer) getSize(template *common.Template) (int, error) {
	rawSize, err := common.GetFirstAttribute(template, common.TemplateAttributeHeight)
	if err == nil {
		sizeValue, err := strconv.Atoi(rawSize)
		if err == nil {
			return sizeValue, nil
		}
		return 0, err
	}
	return 0, err
}

func (renderAgent *genericRenderAgent) getDensity(template *common.Template) (int, error) {
	rawDensity, err := common.GetFirstAttribute(template, common.TemplateAttributeDensity)
	if err == nil {
		density, err := strconv.Atoi(rawDensity)
		if err == nil {
			return density, nil
		}
		return 0, err
	}
	return 0, err
}

func (renderAgent *genericRenderAgent) getGeneratedAssetPage(generatedAsset *common.GeneratedAsset) (int, error) {
	rawPage, err := common.GetFirstAttribute(generatedAsset, common.GeneratedAssetAttributePage)
	if err == nil {
		pageValue, err := strconv.Atoi(rawPage)
		if err == nil {
			return pageValue, nil
		}
		return 0, err
	}
	return 0, err
}

func (renderer *imageMagickRenderer) getSourceAssetFileType(sourceAsset *common.SourceAsset) (string, error) {
	fileType, err := common.GetFirstAttribute(sourceAsset, common.SourceAssetAttributeType)
	if err == nil {
		return fileType, nil
	}
	return "unknown", err
}
