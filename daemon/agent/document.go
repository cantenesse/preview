package agent

import (
	"bytes"
	"encoding/json"
	"github.com/ngerakines/preview/common"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func init() {
	Renderers["documentRenderAgent"] = newDocumentRenderer
}

type documentRenderer struct {
	renderAgent      *genericRenderAgent
	tempFileBasePath string
	engine           string
	officeRenderUrl  string
}

func newDocumentRenderer(renderAgent *genericRenderAgent, params map[string]string) Renderer {
	renderer := new(documentRenderer)
	renderer.renderAgent = renderAgent
	renderer.tempFileBasePath = params["tempFileBasePath"]
	renderer.engine = params["engine"]
	renderer.officeRenderUrl = params["officeRenderUrl"]

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
func (renderer *documentRenderer) renderGeneratedAsset(id string) {
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

	// 3. Get the template... not needed yet

	// 4. Fetch the source asset file
	urls := sourceAsset.GetAttribute(common.SourceAssetAttributeSource)
	var sourceFile common.TemporaryFile
	if renderer.engine != "msoffice" {
		sourceFile, err = renderer.renderAgent.tryDownload(urls, common.SourceAssetSource(sourceAsset))
		if err != nil {
			statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorNoDownloadUrlsWork), nil}
			return
		}
		defer sourceFile.Release()
	}

	var destination string
	// 5. Create a temporary destination directory.
	if renderer.engine == "libreoffice" {
		destination, err := renderer.createTemporaryDestinationDirectory()
		if err != nil {
			statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorNotImplemented), nil}
			return
		}
		destinationTemporaryFile := renderer.renderAgent.temporaryFileManager.Create(destination)
		defer destinationTemporaryFile.Release()
	}

	var pdfGAid string
	renderer.renderAgent.metrics.convertTime.Time(func() {
		switch renderer.engine {
		case "libreoffice":
			err = renderer.createPdf(sourceFile.Path(), destination)
		case "msoffice":
			if len(urls) == 0 {
				err = common.ErrorNoDownloadUrlsWork
				return // Returns from this anon function
			}
			pdfGAid, err = renderer.createPdfWithOffice(urls[0], fileType)
		}
	})
	if err != nil {
		log.Println("Conversion error:", err)
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorCouldNotResizeImage), nil}
		return
	}

	var pdfFile string
	switch renderer.engine {
	case "libreoffice":
		files, err := renderer.getRenderedFiles(destination)
		if err != nil {
			statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorNotImplemented), nil}
			return
		}
		if len(files) != 1 {
			statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorNotImplemented), nil}
			return
		}
		pdfFile = files[0]
	case "msoffice":
		tf, err := renderer.getPdfFile(pdfGAid)
		defer tf.Release()
		if err != nil {
			log.Println("Could not retrieve rendered PDF file from MS Office render agent")
			statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorNotImplemented), nil}
			return
		}
		pdfFile = tf.Path()
	}

	pages, err := common.GetPdfPageCount(pdfFile)
	if err != nil {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorNotImplemented), nil}
		return
	}

	err = renderer.renderAgent.uploader.Upload(generatedAsset.Location, pdfFile)
	if err != nil {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorCouldNotUploadAsset), nil}
		return
	}

	pdfFileSize, err := common.FileSize(pdfFile)
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
	renderer.renderAgent.sasm.Store(pdfSourceAsset)
	legacyDefaultTemplates, err := renderer.renderAgent.templateManager.FindByIds(common.LegacyDefaultTemplates)
	if err != nil {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorNotImplemented), nil}
		return
	}
	// Only process first page because imageMagickRenderAgent will automatically create derived work for the other pages
	renderer.renderAgent.agentManager.CreateDerivedWork(pdfSourceAsset, legacyDefaultTemplates, 0, 1)

	statusCallback <- generatedAssetUpdate{common.GeneratedAssetStatusComplete, nil}
}

func (renderer *documentRenderer) createPdf(source, destination string) error {
	_, err := exec.LookPath("soffice")
	if err != nil {
		log.Println("soffice command not found")
		return err
	}

	// TODO: Make this path configurable.
	cmd := exec.Command("soffice", "--headless", "--nologo", "--nofirststartwizard", "--convert-to", "pdf", source, "--outdir", destination)
	log.Println(cmd)

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	err = cmd.Run()
	log.Println(buf.String())
	if err != nil {
		log.Println("error running command", err)
		return err
	}

	return nil
}

func (renderer *documentRenderer) getRenderedFiles(path string) ([]string, error) {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		log.Println("Error reading files in placeholder base directory:", err)
		return nil, err
	}
	paths := make([]string, 0, 0)
	for _, file := range files {
		if !file.IsDir() {
			// NKG: The convert command will create files of the same name but with the ".pdf" extension.
			if strings.HasSuffix(file.Name(), ".pdf") {
				paths = append(paths, filepath.Join(path, file.Name()))
			}
		}
	}
	return paths, nil
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

func (renderer *documentRenderer) createPdfWithOffice(source, fileType string) (string, error) {
	id, _ := common.NewUuid()
	jsonRequest := `{
"sourceAssets": [
		{
			"fileId": "` + id + `",
			"url": "` + source + `",
			"attributes": {
				"type": [
					"` + fileType + `"
				]
			}
		}
	],
	"templateIds": [
		"0507D650-0731-4D86-8394-8082BB520A21"
	]
}`

	req, err := http.NewRequest("PUT", renderer.officeRenderUrl, strings.NewReader(jsonRequest))
	if err != nil {
		return "", err
	}
	client := common.NewHttpClient(true, 10*time.Second)
	httpResp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	resp, err := ioutil.ReadAll(httpResp.Body)
	if err != nil {
		return "", err
	}
	log.Println("Received response", string(resp))
	// TODO: Get rid of these anonymous structures when API V2 structs are made public in common
	var apiResp struct {
		SourceAssets []struct {
			SourceAsset     *common.SourceAsset      `json:"sourceAsset"`
			GeneratedAssets []*common.GeneratedAsset `json:"generatedAssets"`
		} `json:"sourceAssets"`
	}
	err = json.Unmarshal(resp, &apiResp)
	if err != nil {
		log.Println("Error unmarshaling api response")
		return "", err
	}

	for i := 0; i < 60; i++ {
		status, err := renderer.submitStatusRequest(id)
		if err != nil {
			log.Println("Error getting status")
			return "", err
		}
		ga, err := common.NewGeneratedAssetFromJson(status)
		if err != nil {
			log.Println("Error unmarshaling GA")
			return "", err
		}
		switch ga.Status {
		case common.GeneratedAssetStatusFailed:
			log.Println("MS Office render agent failed to convert document")
			return "", common.ErrorNotImplemented
		case common.GeneratedAssetStatusComplete:
			return id, nil
		}
		time.Sleep(1 * time.Second)
	}
	log.Println("Timeout waiting for MS Office render agent")
	return "", common.ErrorNotImplemented
}

func (renderer *documentRenderer) submitStatusRequest(id string) ([]byte, error) {
	url := renderer.officeRenderUrl + id + "/" + common.MsOfficeTemplateId + "/0"
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func (renderer *documentRenderer) getPdfFile(id string) (common.TemporaryFile, error) {
	url := renderer.officeRenderUrl + id + "/" + common.MsOfficeTemplateId + "/0/data"
	// TODO: Tram support
	return renderer.renderAgent.downloader.Download(url, "")
}
