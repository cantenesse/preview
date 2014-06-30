package cli

import (
	"encoding/json"
	"fmt"
	"github.com/ngerakines/preview/common"
	"github.com/ngerakines/preview/util"
	"io/ioutil"
	"log"
	"net/http"
	"path"
	"path/filepath"
	"strings"
	"time"
)

type DocumentRenderCommand struct {
	host    string
	file    string
	verbose int
}

type DocumentRenderRequest struct {
	Location string `json:"location"`
	Filetype string `json:"filetype"`
}

func NewDocumentRenderCommand(arguments map[string]interface{}) *DocumentRenderCommand {
	command := new(DocumentRenderCommand)
	command.host = getConfigString(arguments, "<host>")
	if len(command.host) == 0 {
		command.host = "localhost:8080"
	}

	//TODO: Find out why this has to happen
	command.file = getConfigStringArray(arguments, "<file>")[0]
	command.verbose = getConfigInt(arguments, "--verbose")
	return command
}

func (command *DocumentRenderCommand) Execute() {
	request := DocumentRenderRequest{
		Location: command.getFullLocation(),
		Filetype: command.getFiletype(),
	}

	requrl := fmt.Sprintf("http://%s/document/", command.host)
	bytes, _ := json.Marshal(request)
	if command.verbose > 1 {
		prettyjson, _ := json.MarshalIndent(request, "", "	")
		log.Println("Request URL:", requrl)
		log.Println("Request:")
		log.Println(string(prettyjson))
	}
	hr, err := http.NewRequest("PUT", requrl, strings.NewReader(string(bytes)))

	if err != nil {
		fmt.Println(err.Error())
		return
	}
	client := common.NewHttpClient(true, 10*time.Second)
	resp, err := client.Do(hr)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	if command.verbose > 1 {
		log.Println("Response body:")
		b, _ := ioutil.ReadAll(resp.Body)
		fmt.Println(string(b))
	}
}

func (command *DocumentRenderCommand) getFiletype() string {
	return path.Ext(command.file)[1:]
}

func (command *DocumentRenderCommand) getFullLocation() string {
	shouldTry, path := command.absFilePath(command.file)
	log.Println(shouldTry, path)
	if shouldTry {
		return "file://" + path
	} else {
		if util.IsHttpUrl(path) {
			return path
		}
		if util.IsS3Url(path) {
			return path
		}
	}
	log.Fatal("Unrecognized url")
	return ""
}

func (command *DocumentRenderCommand) absFilePath(file string) (bool, string) {
	if util.IsHttpUrl(file) {
		return false, file
	}
	if util.IsS3Url(file) {
		return false, file
	}
	if util.IsFileUrl(file) {
		return true, file[7:]
	}
	if strings.HasPrefix(file, "/") {
		return true, file
	}
	return true, filepath.Join(util.Cwd(), file)
}
