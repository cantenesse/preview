package agent

import (
	"github.com/ngerakines/preview/common"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"time"
)

func init() {
	Renderers["msOfficeRenderAgent"] = newMsOfficeRenderer
}

/*
This render agent requires the following template in the config file:
       {
	    "id":"0507D650-0731-4D86-8394-8082BB520A21",
	    "renderAgent":"msOfficeRenderAgent",
	    "group":"B4CD",
	    "attributes":{
		"output":["pdf"]
	    }
	}
*/

type msOfficeRenderer struct {
	renderAgent        *genericRenderAgent
	tempFileBasePath   string
	pdfOutputDirectory string
}

func newMsOfficeRenderer(renderAgent *genericRenderAgent, params map[string]string) Renderer {
	renderer := new(msOfficeRenderer)
	renderer.renderAgent = renderAgent
	renderer.tempFileBasePath = params["tempFileBasePath"]
	renderer.pdfOutputDirectory = params["pdfOutputDirectory"]

	return renderer
}

func (renderer *msOfficeRenderer) renderGeneratedAsset(id string) {
	renderer.renderAgent.metrics.workProcessed.Mark(1)

	// 1. Get the generated asset
	generatedAsset, err := renderer.renderAgent.gasm.FindById(id)
	if err != nil {
		log.Fatal("No Generated Asset with that ID can be retreived from storage: ", id)
		return
	}

	statusCallback := renderer.renderAgent.commitStatus(generatedAsset.Id, generatedAsset.Attributes)
	defer func() { close(statusCallback) }()

	generatedAsset.Status = common.GeneratedAssetStatusProcessing
	renderer.renderAgent.gasm.Update(generatedAsset)

	// 2. Get the source asset
	sourceAsset, err := renderer.renderAgent.getSourceAsset(generatedAsset)
	if err != nil {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorUnableToFindSourceAssetsById), nil}
		return
	}

	fileType, err := common.GetFirstAttribute(sourceAsset, common.SourceAssetAttributeType)
	if err == nil {
		renderer.renderAgent.metrics.fileTypeCount[fileType].Inc(1)
	}

	urls := sourceAsset.GetAttribute(common.SourceAssetAttributeSource)
	sourceFile, err := renderer.renderAgent.tryDownload(urls, common.SourceAssetSource(sourceAsset))
	if err != nil {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorNoDownloadUrlsWork), nil}
		return
	}
	defer sourceFile.Release()

	destination, err := renderer.createTemporaryDestinationDirectory()
	if err != nil {
		log.Println("Failed to create temporary destination directory")
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorNotImplemented), nil}
		return
	}

	destinationTemporaryFile := renderer.renderAgent.temporaryFileManager.Create(destination)
	defer destinationTemporaryFile.Release()

	log.Println("Creating PDF")
	renderer.renderAgent.metrics.convertTime.Time(func() {
		err = createPdf(sourceFile.Path(), fileType)
	})
	if err != nil {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorCouldNotResizeImage), nil}
		return
	}

	log.Println("Getting PDF Location")
	pdfFile, err := renderer.getPdfFileLocation(path.Base(sourceFile.Path()))
	defer func() {
		os.Remove(pdfFile)
	}()
	if err != nil {
		log.Println("Failed to locate PDF")
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorNotImplemented), nil}
		return
	}

	err = renderer.renderAgent.uploader.Upload(generatedAsset.Location, pdfFile)
	if err != nil {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorCouldNotUploadAsset), nil}
		return
	}
	statusCallback <- generatedAssetUpdate{common.GeneratedAssetStatusComplete, nil}
}

func createPdf(source, fileType string) error {
	_, err := exec.LookPath("osascript")
	if err != nil {
		log.Println("osascript command not found")
		return err
	}

	var script string
	switch fileType {
	case "docx", "doc":
		script = "applescripts/WordToPdf.scpt"
	case "pptx", "ppt":
		script = "applescripts/PowerpointToPdf.scpt"
	case "xlsx", "xls":
		script = "applescripts/ExcelToPdf.scpt"
	}

	// This is what actually converts the file by using applescript to print it to the PDFwriter printer
	cmd := exec.Command("osascript", script, source, path.Base(source))
	log.Println(cmd)
	err = cmd.Run()
	if err != nil {
		log.Println("error running command", err)
		return err
	}
	return nil
}

func (renderer *msOfficeRenderer) getPdfFileLocation(id string) (string, error) {
	iterations := 0
	log.Println(id)
	// This is necessary because the applescript command can exit before the PDF printer finishes printing
	for {
		// PDFs get put here from PDFwriter; the UUID in the filename lets us find it easily
		pdfs, err := filepath.Glob(filepath.Join(renderer.pdfOutputDirectory, "*"+id+".pdf"))
		if err != nil {
			log.Println("error finding PDF", err)
			return "", err
		}
		if len(pdfs) == 1 {
			return pdfs[0], nil
		}
		if len(pdfs) > 1 {
			log.Println("error: multiple PDFs with same UUID found")
			return "", common.ErrorNotImplemented
		}
		time.Sleep(1 * time.Second)
		if iterations > 10 {
			log.Println("Timeout")
			return "", common.ErrorNotImplemented
		}
		iterations++
	}
}

func (renderer *msOfficeRenderer) createTemporaryDestinationDirectory() (string, error) {
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
