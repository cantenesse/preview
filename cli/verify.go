package cli

import (
	"fmt"
	"github.com/ngerakines/preview/util"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

var inputFiles = []string{
	"test-data/Multipage.pdf",
	"test-data/Multipage.docx",
	"test-data/Animated.gif",
	"test-data/COW.png",
	"test-data/ChefConf2014schedule.docx",
	"test-data/ChefConf2014schedule.pdf",
	"test-data/wallpaper-641916.jpg",
}

type VerifyCommand struct {
	host     string
	filepath string
	verbose  int
}

type verifyJob struct {
	location   string
	id         string
	isComplete bool
	startTime  time.Time
	endTime    time.Time
}

func newVerifyJob(location string) *verifyJob {
	job := new(verifyJob)
	job.location = location
	job.isComplete = false
	job.id, _ = util.NewUuid()
	return job
}

func NewVerifyCommand(arguments map[string]interface{}) *VerifyCommand {
	command := new(VerifyCommand)
	command.host = getConfigString(arguments, "<host>")
	if len(command.host) == 0 {
		command.host = "localhost:8080"
	}
	command.filepath = getConfigString(arguments, "<filepath>")
	command.verbose = getConfigInt(arguments, "--verbose")

	return command
}

func (command *VerifyCommand) String() string {
	return fmt.Sprintf("VerifyCommand<host=%s filepath=%s verbose=%d>", command.host, command.filepath, command.verbose)
}

func (command *VerifyCommand) Execute() {
	jobs := make([]*verifyJob, 0, len(inputFiles))

	for _, loc := range inputFiles {
		jobs = append(jobs, newVerifyJob(loc))
	}

	for _, job := range jobs {
		args := make(map[string]interface{})
		args["<host>"] = command.host
		args["<file>"] = []string{job.location}
		args["--verbose"] = command.verbose
		renderCommand := NewRenderCommand(args)
		job.startTime = time.Now()

		renderCommand.(*RenderCommand).ExecuteWithId(job.id)
	}

	for i := 0; i < 30; i++ {
		workDone := true
		for _, job := range jobs {
			if job.isComplete {
				continue
			}
			response, _ := command.submitPreviewInfoRequest(job.id)
			job.isComplete = command.isComplete(response)

			if job.isComplete {
				job.endTime = time.Now()
				fmt.Println(job.location, "complete")
				fmt.Println("duration", job.endTime.Sub(job.startTime))
			} else {
				workDone = false
			}
		}

		if workDone {
			break
		}

		time.Sleep(0.5 * time.Second)
	}

	for _, job := range jobs {
		if !job.isComplete {
			fmt.Println(job.location, "failed or timed out")
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
		fmt.Println(err.Error())
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err.Error())
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
