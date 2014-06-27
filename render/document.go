package render

import (
	"bytes"
	"fmt"
	"github.com/ngerakines/preview/common"
	"github.com/ngerakines/preview/util"
	"github.com/rcrowley/go-metrics"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type documentRenderAgent struct {
	metrics              *documentRenderAgentMetrics
	sasm                 common.SourceAssetStorageManager
	gasm                 common.GeneratedAssetStorageManager
	templateManager      common.TemplateManager
	downloader           common.Downloader
	uploader             common.Uploader
	workChannel          RenderAgentWorkChannel
	statusListeners      []RenderStatusChannel
	temporaryFileManager common.TemporaryFileManager
	agentManager         *RenderAgentManager
	tempFileBasePath     string
	stop                 chan (chan bool)
}

type documentRenderAgentMetrics struct {
	workProcessed metrics.Meter
	convertTime   metrics.Timer
	fileTypeCount map[string]metrics.Counter
}

func newDocumentRenderAgent(
	metrics *documentRenderAgentMetrics,
	agentManager *RenderAgentManager,
	sasm common.SourceAssetStorageManager,
	gasm common.GeneratedAssetStorageManager,
	templateManager common.TemplateManager,
	temporaryFileManager common.TemporaryFileManager,
	downloader common.Downloader,
	uploader common.Uploader,
	tempFileBasePath string,
	workChannel RenderAgentWorkChannel) RenderAgent {

	renderAgent := new(documentRenderAgent)
	renderAgent.metrics = metrics
	renderAgent.agentManager = agentManager
	renderAgent.sasm = sasm
	renderAgent.gasm = gasm
	renderAgent.templateManager = templateManager
	renderAgent.temporaryFileManager = temporaryFileManager
	renderAgent.downloader = downloader
	renderAgent.uploader = uploader
	renderAgent.workChannel = workChannel
	renderAgent.tempFileBasePath = tempFileBasePath
	renderAgent.statusListeners = make([]RenderStatusChannel, 0, 0)
	renderAgent.stop = make(chan (chan bool))

	go renderAgent.start()

	return renderAgent
}

func newDocumentRenderAgentMetrics(registry metrics.Registry, supportedFileTypes []string) *documentRenderAgentMetrics {
	documentMetrics := new(documentRenderAgentMetrics)
	documentMetrics.workProcessed = metrics.NewMeter()
	documentMetrics.convertTime = metrics.NewTimer()

	documentMetrics.fileTypeCount = make(map[string]metrics.Counter)

	for _, filetype := range supportedFileTypes {
		documentMetrics.fileTypeCount[filetype] = metrics.NewCounter()
		registry.Register(fmt.Sprintf("documentRenderAgent.%sCount", filetype), documentMetrics.fileTypeCount[filetype])
	}

	registry.Register("documentRenderAgent.workProcessed", documentMetrics.workProcessed)
	registry.Register("documentRenderAgent.convertTime", documentMetrics.convertTime)

	return documentMetrics
}

func (renderAgent *documentRenderAgent) start() {
	for {
		select {
		case ch, ok := <-renderAgent.stop:
			{
				log.Println("Stopping")
				if !ok {
					return
				}
				ch <- true
				return
			}
		case id, ok := <-renderAgent.workChannel:
			{
				if !ok {
					return
				}
				log.Println("Received dispatch message", id)
				renderAgent.renderGeneratedAsset(id)
			}
		}
	}
}

func (renderAgent *documentRenderAgent) Stop() {
	callback := make(chan bool)
	renderAgent.stop <- callback
	select {
	case <-callback:
	case <-time.After(5 * time.Second):
	}
	close(renderAgent.stop)
}

func (renderAgent *documentRenderAgent) AddStatusListener(listener RenderStatusChannel) {
	renderAgent.statusListeners = append(renderAgent.statusListeners, listener)
}

func (renderAgent *documentRenderAgent) Dispatch() RenderAgentWorkChannel {
	return renderAgent.workChannel
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
func (renderAgent *documentRenderAgent) renderGeneratedAsset(id string) {
	renderAgent.metrics.workProcessed.Mark(1)

	// 1. Get the generated asset
	generatedAsset, err := renderAgent.gasm.FindById(id)
	if err != nil {
		log.Fatal("No Generated Asset with that ID can be retreived from storage: ", id)
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

	fileType, err := common.GetFirstAttribute(sourceAsset, common.SourceAssetAttributeType)
	if err == nil {
		renderAgent.metrics.fileTypeCount[fileType].Inc(1)
	}

	// 3. Get the template... not needed yet

	// 4. Fetch the source asset file
	urls := sourceAsset.GetAttribute(common.SourceAssetAttributeSource)
	sourceFile, err := renderAgent.tryDownload(urls, common.SourceAssetSource(sourceAsset))
	if err != nil {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorNoDownloadUrlsWork), nil}
		return
	}
	defer sourceFile.Release()

	//      // 5. Create a temporary destination directory.
	destination, err := renderAgent.createTemporaryDestinationDirectory()
	if err != nil {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorNotImplemented), nil}
		return
	}
	destinationTemporaryFile := renderAgent.temporaryFileManager.Create(destination)
	defer destinationTemporaryFile.Release()

	renderAgent.metrics.convertTime.Time(func() {
		err = renderAgent.createPdfWithMSOffice(sourceFile.Path(), destination, fileType)
		if err != nil {
			statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorCouldNotResizeImage), nil}
			return
		}
	})

	files, err := renderAgent.getRenderedFiles(destination)

	if err != nil {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorNotImplemented), nil}
		return
	}
	if len(files) != 1 {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorNotImplemented), nil}
		return
	}

	pages, err := util.GetPdfPageCount(files[0])
	if err != nil {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorNotImplemented), nil}
		return
	}

	err = renderAgent.uploader.Upload(generatedAsset.Location, files[0])
	if err != nil {
		statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorCouldNotUploadAsset), nil}
		return
	}

	pdfFileSize, err := util.FileSize(destination)
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

	/*
	   // TODO: Have the new source asset and generated assets be created in batch in the storage managers.
	   for page := 0; page < pages; page++ {
	           for _, legacyTemplate := range legacyDefaultTemplates {
	                   // TODO: This can be put into a small lookup table create/set at the time of structure init.
	                   placeholderSize, err := common.GetFirstAttribute(legacyTemplate, common.TemplateAttributePlaceholderSize)
	                   if err != nil {
	                           statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorNotImplemented), nil}
	                           return
	                   }
	                   // TODO: Update simple blueprint and image magick render agent to use this url structure.
	                   location := renderAgent.uploader.Url(sourceAsset.Id, legacyTemplate.Id, placeholderSize, int32(page))
	                   pdfGeneratedAsset, err := common.NewGeneratedAssetFromSourceAsset(pdfSourceAsset, legacyTemplate, location)
	                   if err != nil {
	                           statusCallback <- generatedAssetUpdate{common.NewGeneratedAssetError(common.ErrorNotImplemented), nil}
	                           return
	                   }
	                   pdfGeneratedAsset.AddAttribute(common.GeneratedAssetAttributePage, []string{strconv.Itoa(page)})
	                   log.Println("pdfGeneratedAsset", pdfGeneratedAsset)
	                   renderAgent.gasm.Store(pdfGeneratedAsset)
	           }
	   }
	*/

	statusCallback <- generatedAssetUpdate{common.GeneratedAssetStatusComplete, nil}
}

func (renderAgent *documentRenderAgent) getSourceAsset(generatedAsset *common.GeneratedAsset) (*common.SourceAsset, error) {
	sourceAssets, err := renderAgent.sasm.FindBySourceAssetId(generatedAsset.SourceAssetId)
	if err != nil {
		return nil, err
	}
	for _, sourceAsset := range sourceAssets {
		if sourceAsset.IdType == generatedAsset.SourceAssetType {
			return sourceAsset, nil
		}
	}
	return nil, common.ErrorNoSourceAssetsFoundForId
}

func (renderAgent *documentRenderAgent) createPdf(source, destination string) error {
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

// Currently this is a total hack that is only a proof of concept
func (renderAgent *documentRenderAgent) createPdfWithMSOffice(source, destination, fileType string) error {
	_, err := exec.LookPath("osascript")
	if err != nil {
		log.Println("osascript command not found")
		return err
	}
	id := path.Base(source)

	// For MS Office to open it, it has to have a filetype
	fileWithExtension := source + "." + fileType
	cmd := exec.Command("mv", source, fileWithExtension)

	log.Println(cmd)
	err = cmd.Run()
	if err != nil {
		log.Println("error running command", err)
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
	cmd = exec.Command("osascript", script, fileWithExtension, path.Base(fileWithExtension))
	log.Println(cmd)
	err = cmd.Run()
	if err != nil {
		log.Println("error running command", err)
		return err
	}
	var pdfFile string
	iterations := 0
	// This is necessary because the applescript command can exit before the PDF printer finishes printing
	for {
		// PDFs get put here from PDFwriter; the UUID in the filename lets us find it easily
		pdfs, err := filepath.Glob("/Users/Shared/PDFwriter/*/job_*" + id + ".pdf")
		if err != nil {
			log.Println("error running command", err)
			return err
		}
		if len(pdfs) == 1 {
			pdfFile = pdfs[0]
			break
		}
		time.Sleep(1 * time.Second)
		if iterations > 10 {
			log.Println("Timeout")
			return common.ErrorNotImplemented
		}
		iterations++;
	}

	// Put the file where Preview expects it to be
	cmd = exec.Command("mv", pdfFile, destination+"/"+id+".pdf")
	log.Println(cmd)
	err = cmd.Run()
	if err != nil {
		log.Println("error running command", err)
		return err
	}

	// Undo the move we did earlier
	// Again, this is a total hack and only a proof-of-concept
	cmd = exec.Command("mv", fileWithExtension, source)
	log.Println(cmd)
	err = cmd.Run()
	if err != nil {
		log.Println("error running command", err)
		return err
	}
	// For some reason on Mac, the call to pdfinfo will crash if we don't wait here
	// Maybe the move command exits prematurely before the file actually moves?
	// I'm guessing and don't really know
	time.Sleep(1 * time.Second)
	return nil
}

func (renderAgent *documentRenderAgent) tryDownload(urls []string, source string) (common.TemporaryFile, error) {
	for _, url := range urls {
		tempFile, err := renderAgent.downloader.Download(url, source)
		if err == nil {
			return tempFile, nil
		}
	}
	return nil, common.ErrorNoDownloadUrlsWork
}

func (renderAgent *documentRenderAgent) commitStatus(id string, existingAttributes []common.Attribute) chan generatedAssetUpdate {
	commitChannel := make(chan generatedAssetUpdate, 10)

	go func() {
		status := common.NewGeneratedAssetError(common.ErrorUnknownError)
		attributes := make([]common.Attribute, 0, 0)
		for _, attribute := range existingAttributes {
			attributes = append(attributes, attribute)
		}
		for {
			select {
			case message, ok := <-commitChannel:
				{
					if !ok {
						for _, listener := range renderAgent.statusListeners {
							listener <- RenderStatus{id, status, common.RenderAgentDocument}
						}
						generatedAsset, err := renderAgent.gasm.FindById(id)
						if err != nil {
							panic(err)
							return
						}
						generatedAsset.Status = status
						generatedAsset.Attributes = attributes
						log.Println("Updating", generatedAsset)
						renderAgent.gasm.Update(generatedAsset)
						return
					}
					status = message.status
					if message.attributes != nil {
						for _, attribute := range message.attributes {
							attributes = append(attributes, attribute)
						}
					}
				}
			}
		}
	}()
	return commitChannel
}

func (renderAgent *documentRenderAgent) createTemporaryDestinationDirectory() (string, error) {
	uuid, err := util.NewUuid()
	if err != nil {
		return "", err
	}
	tmpPath := filepath.Join(renderAgent.tempFileBasePath, uuid)
	err = os.MkdirAll(tmpPath, 0777)
	if err != nil {
		log.Println("error creating tmp dir", err)
		return "", err
	}
	return tmpPath, nil
}

func (renderAgent *documentRenderAgent) getRenderedFiles(path string) ([]string, error) {
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
