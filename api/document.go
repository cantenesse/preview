package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/bmizerany/pat"
	"github.com/ngerakines/preview/common"
	"github.com/ngerakines/preview/docserver"
	"github.com/ngerakines/preview/util"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

type documentBlueprint struct {
	base              string
	conversionManager *docserver.ConversionManager
}

type ConvertDocumentRequest struct {
	Location string `json:"location"`
	Filetype string `json:"filetype"`
}

func NewDocumentBlueprint(conversionManager *docserver.ConversionManager) *documentBlueprint {
	blueprint := new(documentBlueprint)
	blueprint.base = "/document"
	blueprint.conversionManager = conversionManager
	return blueprint
}

func (blueprint *documentBlueprint) AddRoutes(p *pat.PatternServeMux) {
	p.Put(blueprint.base+"/", http.HandlerFunc(blueprint.convertDocumentHandler))
	p.Get(blueprint.base+"/:id/data", http.HandlerFunc(blueprint.serveDocumentHandler))
	p.Get(blueprint.base+"/:id", http.HandlerFunc(blueprint.documentInfoHandler))
}

func (blueprint *documentBlueprint) convertDocumentHandler(res http.ResponseWriter, req *http.Request) {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		http.Error(res, "", 400)
		return
	}
	defer req.Body.Close()

	job, err := createJobFromRequest(body)
	if err != nil {
		http.Error(res, "", 400)
		return
	}

	blueprint.conversionManager.EnqueueWork(job)
	http.Redirect(res, req, fmt.Sprintf("/document/%s", job.Id), 303)
}

func (blueprint *documentBlueprint) serveDocumentHandler(res http.ResponseWriter, req *http.Request) {
	id := req.URL.Query().Get(":id")
	job, err := blueprint.conversionManager.GetJob(id)
	if err != nil {
		http.Error(res, "", 400)
		return
	}
	if job.Status != "completed" {
		http.Error(res, "", 400)
		return
	}

	_, err = os.Stat(job.Location)
	if err == nil {
		http.ServeFile(res, req, job.Location)
		return
	}
	log.Println("File does not exist", job.Location)
	http.Error(res, "", 500)
}

func (blueprint *documentBlueprint) documentInfoHandler(res http.ResponseWriter, req *http.Request) {
	id := req.URL.Query().Get(":id")
	job, err := blueprint.conversionManager.GetJob(id)
	if err != nil {
		log.Println(err)
		http.Error(res, "", 500)
		return
	}
	blueprint.conversionManager.JobMutex.Lock()
	jsonData, err := marshalJob(job)
	blueprint.conversionManager.JobMutex.Unlock()
	if err != nil {
		log.Println(err)
		http.Error(res, "", 500)
		return
	}

	http.ServeContent(res, req, "", time.Now(), bytes.NewReader(jsonData))

}

func marshalJob(job *docserver.ConvertDocumentJob) ([]byte, error) {
	data, err := json.Marshal(job)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func createJobFromRequest(req []byte) (*docserver.ConvertDocumentJob, error) {
	var request ConvertDocumentRequest
	err := json.Unmarshal(req, &request)
	if err != nil {
		log.Println("Error creating job")
		return nil, common.ErrorNotImplemented
	}
	uuid, err := util.NewUuid()
	if err != nil {
		return nil, err
	}
	job := docserver.ConvertDocumentJob{
		SourceLocation: request.Location,
		Id:             uuid,
		Filetype:       request.Filetype,
	}
	return &job, nil
}
