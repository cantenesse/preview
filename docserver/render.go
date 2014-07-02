package docserver

import (
	"github.com/ngerakines/preview/common"
	"github.com/ngerakines/preview/util"
	"hash/crc32"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"time"
)

type conversionAgent struct {
	manager *ConversionManager
}

func NewConversionAgent(manager *ConversionManager) *conversionAgent {
	agent := new(conversionAgent)
	agent.manager = manager
	agent.start()
	return agent
}

func (agent *conversionAgent) start() {
	go func() {
		for {
			select {
			case job := <-agent.manager.workChan:
				{
					job.Status = "processing"
					agent.manager.UpdateJob(job)
					j := agent.processJob(job)
					agent.manager.UpdateJob(j)
				}
			case <-agent.manager.stop:
				{
					return
				}
			}
		}
	}()
}

func (agent *conversionAgent) processJob(job *ConvertDocumentJob) *ConvertDocumentJob {
	log.Println("Processing job", job)
	log.Println("Downloading file")
	log.Println(job.SourceLocation)
	docFile, err := agent.tryDownload([]string{job.SourceLocation}, "")
	if err != nil {
		job.Status = "failed"
		return job
	}
	defer docFile.Release()
	log.Println(docFile)
	log.Println("Creating tempdir")
	destination, err := agent.createTemporaryDestinationDirectory()
	if err != nil {
		log.Println("Failed to create temporary destination directory")
		job.Status = "failed"
		return job
	}
	log.Println("Creating TempFile")
	destinationTemporaryFile := agent.manager.tfm.Create(destination)
	if err != nil {
		log.Println("Failed to create temporary file")
		job.Status = "failed"
		return job
	}
	defer destinationTemporaryFile.Release()
	log.Println("Creating PDF")
	err = agent.createPdf(docFile.Path(), job.Filetype)
	if err != nil {
		log.Println("Failed to create PDF")
		job.Status = "failed"
		return job
	}

	log.Println("Getting PDF")
	pdfFile, err := agent.getPdfFile(path.Base(docFile.Path()))
	if err != nil {
		log.Println("Failed to get PDF")
		job.Status = "failed"
		return job
	}

	log.Println("Moving PDF")
	err = agent.movePdfFile(pdfFile, destinationTemporaryFile.Path())
	if err != nil {
		log.Println("Failed to move PDF")
		job.Status = "failed"
		return job
	}

	job.Location = destinationTemporaryFile.Path() + "/out.pdf"
	job.Url = agent.manager.host + "/document/" + job.Id + "/data"
	job.Status = "completed"

	return job
}

func (agent *conversionAgent) movePdfFile(source, dest string) error {
	dest = dest + "/out.pdf"

	sourceChecksum, err := computeCRC32(source)
	if err != nil {
		log.Println("Failed to compute checksum")
		return err
	}

	err = os.Rename(source, dest)
	if err != nil {
		log.Println("Failed to move file", source, "to", dest)
		return err
	}
	cmd := exec.Command("sync")
	log.Println(cmd)
	err = cmd.Run()
	if err != nil {
		log.Println("sync failed")
		return err
	}

	for i := 0; i < 10; i++ {
		checksum, err := computeCRC32(dest)
		if err != nil {
			log.Println("Failed to compute checksum")
			return err
		}
		if checksum == sourceChecksum {
			log.Println("CRC Validated for file", dest)
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	log.Println("File timeout")
	return common.ErrorNotImplemented
}

func computeCRC32(path string) (uint32, error) {
	in, err := ioutil.ReadFile(path)
	if err != nil {
		return 0, err
	}
	return crc32.ChecksumIEEE(in), nil
}

func (agent *conversionAgent) createPdf(source, fileType string) error {
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

func (agent *conversionAgent) getPdfFile(id string) (string, error) {
	iterations := 0
	log.Println(id)
	// This is necessary because the applescript command can exit before the PDF printer finishes printing
	for {
		// PDFs get put here from PDFwriter; the UUID in the filename lets us find it easily
		pdfs, err := filepath.Glob("/Users/Shared/PDFwriter/*/job_*" + id + ".pdf")
		if err != nil {
			log.Println("error running command", err)
			return "", err
		}
		if len(pdfs) == 1 {
			return pdfs[0], nil
		}
		time.Sleep(1 * time.Second)
		if iterations > 10 {
			log.Println("Timeout")
			return "", common.ErrorNotImplemented
		}
		iterations++
	}
}

func (agent *conversionAgent) createTemporaryDestinationDirectory() (string, error) {
	uuid, err := util.NewUuid()
	if err != nil {
		return "", err
	}
	tmpPath := filepath.Join(agent.manager.tempFileBasePath, uuid)
	err = os.MkdirAll(tmpPath, 0777)
	if err != nil {
		log.Println("error creating tmp dir", err)
		return "", err
	}
	return tmpPath, nil
}

func (agent *conversionAgent) tryDownload(urls []string, source string) (common.TemporaryFile, error) {
	log.Println(urls)
	for _, url := range urls {
		tempFile, err := agent.manager.downloader.Download(url, source)
		if err == nil {
			return tempFile, nil
		}
	}
	return nil, common.ErrorNoDownloadUrlsWork
}
