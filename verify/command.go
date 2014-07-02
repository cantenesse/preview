package verify

import (
	"encoding/json"
	"fmt"
	"github.com/ngerakines/preview/common"
	"github.com/ngerakines/preview/render"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"
)

var inputFiles = []string{
	"Multipage.pdf",
	"Multipage.docx",
	"Animated.gif",
	"COW.png",
	"ChefConf2014schedule.docx",
	"ChefConf2014schedule.pdf",
	"wallpaper-641916.jpg",
}

type VerifyCommand struct {
	host     string
	filepath string
	verbose  int
	timeout  int
}

type verifyJob struct {
	location   string
	id         string
	isComplete bool
	startTime  time.Time
	endTime    time.Time
}

type imageInfo struct {
	Url           string  `json:"url"`
	Width         float64 `json:"width"`
	Height        float64 `json:"height"`
	Expires       float64 `json:"expires"`
	IsFinal       bool    `json:"isFinal"`
	IsPlaceholder bool    `json:"isPlaceholder"`
}

type previewInfoResponse struct {
	Version string `json:"version"`
	Files   []struct {
		FileId string    `json:"file_id"`
		Jumbo  imageInfo `json:"jumbo"`
		Large  imageInfo `json:"large"`
		Medium imageInfo `json:"medium"`
		Small  imageInfo `json:"small"`
	} `json:"files"`
}

func newVerifyJob(location string) *verifyJob {
	job := new(verifyJob)
	job.location = location
	job.isComplete = false
	job.id, _ = common.NewUuid()
	return job
}

func NewVerifyCommand(arguments map[string]interface{}) common.Command {
	command := new(VerifyCommand)
	command.host = common.GetConfigString(arguments, "<host>")
	if len(command.host) == 0 {
		command.host = "localhost:8080"
	}
	command.filepath = common.GetConfigString(arguments, "<filepath>")
	command.verbose = common.GetConfigInt(arguments, "--verbose")
	timeout := common.GetConfigString(arguments, "--timeout")

	if len(timeout) > 0 {
		var err error
		command.timeout, err = strconv.Atoi(timeout)
		if err != nil {
			log.Println("Invalid timeout; ignoring")
		}
	}

	if command.timeout == 0 {
		command.timeout = 30
	}

	return command
}

func (command *VerifyCommand) String() string {
	return fmt.Sprintf("VerifyCommand<host=%s filepath=%s verbose=%d>", command.host, command.filepath, command.verbose)
}

func (command *VerifyCommand) Execute() {
	jobs := make([]*verifyJob, 0, len(inputFiles))

	for _, loc := range inputFiles {
		jobs = append(jobs, newVerifyJob(common.JoinUrl(command.filepath, loc)))
	}

	for _, job := range jobs {
		args := make(map[string]interface{})
		args["<host>"] = command.host
		args["<file>"] = []string{job.location}
		args["--verbose"] = command.verbose
		// TODO[NKG]: Improve this.
		renderCommand := render.NewRenderCommand(args)
		job.startTime = time.Now()

		renderCommand.ExecuteWithId(job.id)
	}
	// Each loop waits for 0.5 seconds, so we must loop <2 * timeout> times in order to take <timeout> seconds
	iterations := command.timeout * 2
	for i := 0; i < iterations; i++ {
		workDone := true
		for _, job := range jobs {
			if job.isComplete {
				continue
			}
			response, err := command.submitPreviewInfoRequest(job.id)
			if err != nil {
				log.Println("Error getting preview response:", err)
				workDone = false
				continue
			}

			job.isComplete = command.isComplete(response)

			if job.isComplete {
				job.endTime = time.Now()
				log.Println(job.location, "complete")
				log.Println("duration", job.endTime.Sub(job.startTime))
			} else {
				workDone = false
			}
		}

		if workDone {
			break
		}

		time.Sleep(500 * time.Millisecond)
	}

	for _, job := range jobs {
		if !job.isComplete {
			log.Println(job.location, "failed or timed out")
		}
	}
}

func (command *VerifyCommand) buildSubmitPreviewInfoRequest(id string) string {
	return fmt.Sprintf("http://%s/api/v1/preview/%s", command.host, id)
}

func (command *VerifyCommand) submitPreviewInfoRequest(id string) (*previewInfoResponse, error) {
	url := command.buildSubmitPreviewInfoRequest(id)
	if command.verbose > 0 {
		log.Println("Submitting request to", url)
	}
	resp, err := http.Get(url)
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}

	return newPreviewInfoResponse(body)
}

func (command *VerifyCommand) isComplete(response *previewInfoResponse) bool {
	complete := true
	for _, file := range response.Files {
		if file.Jumbo.IsFinal == false {
			if command.verbose > 1 {
				log.Println("File", file.FileId, "incomplete:", file.Jumbo.Url)
			}
			complete = false
		}
		if file.Large.IsFinal == false {
			if command.verbose > 1 {
				log.Println("File", file.FileId, "incomplete:", file.Large.Url)
			}
			complete = false
		}
		if file.Medium.IsFinal == false {
			if command.verbose > 1 {
				log.Println("File", file.FileId, "incomplete:", file.Medium.Url)
			}
			complete = false
		}
		if file.Small.IsFinal == false {
			if command.verbose > 1 {
				log.Println("File", file.FileId, "incomplete:", file.Small.Url)
			}
			complete = false
		}
	}
	if complete && command.verbose > 0 {
		for _, file := range response.Files {
			log.Println("File", file.FileId, "complete:", file.Jumbo.Url)
			log.Println("File", file.FileId, "complete:", file.Large.Url)
			log.Println("File", file.FileId, "complete:", file.Medium.Url)
			log.Println("File", file.FileId, "complete:", file.Small.Url)
		}
	}
	return complete
}

func newPreviewInfoResponse(body []byte) (*previewInfoResponse, error) {
	var response previewInfoResponse
	e := json.Unmarshal(body, &response)
	if e != nil {
		return nil, e
	}
	return &response, nil
}
