package docserver

import (
	"github.com/ngerakines/preview/common"
	"log"
)

type workChannel chan *ConvertDocumentJob

type ConvertDocumentJob struct {
	Id             string `json:"id"`
	SourceLocation string `json:"sourceLocation"`
	Filetype       string `json:"filetype"`
	Location       string `json:"location"`
	Status         string `json:"status"`
	Url            string `json:"url"`
}

type ConversionManager struct {
	downloader       common.Downloader
	tfm              common.TemporaryFileManager
	workChan         workChannel
	stop             chan bool
	activeJobs       map[string]*ConvertDocumentJob
	tempFileBasePath string
	host             string
}

func NewConversionManager(downloader common.Downloader, tfm common.TemporaryFileManager, tempFileBasePath, host string) *ConversionManager {
	manager := new(ConversionManager)
	manager.downloader = downloader
	manager.tfm = tfm
	manager.tempFileBasePath = tempFileBasePath
	manager.host = host
	manager.workChan = make(workChannel, 200)
	manager.stop = make(chan bool)
	manager.activeJobs = make(map[string]*ConvertDocumentJob)
	return manager
}

func (manager *ConversionManager) EnqueueWork(job *ConvertDocumentJob) {
	// TODO: Manage active jobs; delete ones that have finished/failed and are older than
	// certain time
	log.Println("Enqueueing job", job)
	manager.activeJobs[job.Id] = job
	manager.workChan <- job
}

func (manager *ConversionManager) AddConversionAgent() {
	NewConversionAgent(manager)
}

func (manager *ConversionManager) GetJob(id string) (*ConvertDocumentJob, error) {
	job, hasJob := manager.activeJobs[id]
	if !hasJob {
		log.Println("Could not find job with id", id)
		return nil, common.ErrorNotImplemented
	}
	return job, nil
}
