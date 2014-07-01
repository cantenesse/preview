package docserver

import (
	"github.com/ngerakines/preview/common"
	"log"
	"sync"
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
	mu               sync.Mutex
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

// Store a copy, so if the job gets enqueued then changed, the manager's copy
// won't change unless UpdateJob is called
// Is this necessary?
func (manager *ConversionManager) UpdateJob(job *ConvertDocumentJob) {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	copy := new(ConvertDocumentJob)
	*copy = *job
	manager.activeJobs[job.Id] = copy
}

// Return a copy of a job, so that if the job gets updated while someone is using it,
// the job won't change
// Is this necessary?
func (manager *ConversionManager) GetJob(id string) (*ConvertDocumentJob, error) {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	job, hasJob := manager.activeJobs[id]
	if !hasJob {
		log.Println("Could not find job with id", id)
		return nil, common.ErrorNotImplemented
	}
	copy := new(ConvertDocumentJob)
	*copy = *job
	return copy, nil
}
