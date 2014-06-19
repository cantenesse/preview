package cli

import (
	"encoding/json"
	"fmt"
	"github.com/ngerakines/preview/util"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"strings"
)

type RenderV2Command struct {
	host        string
	files       []string
	templateIds []string
	verbose     int
}

type sourceAssetRequest struct {
	Id         string              `json:"fileId"`
	Url        string              `json:"url"`
	Attributes map[string][]string `json:"attributes"`
}

type previewGenerateRequest struct {
	SourceAssets []sourceAssetRequest `json:"sourceAssets"`
	TemplateIds  []string             `json:"templateIds"`
}

func NewRenderV2Command(arguments map[string]interface{}) PreviewCliCommand {
	command := new(RenderV2Command)
	command.host = getConfigString(arguments, "<host>")
	if len(command.host) == 0 {
		command.host = "localhost:8080"
	}
	command.files = getConfigStringArray(arguments, "<file>")
	command.templateIds = getConfigStringArray(arguments, "<templateId>")
	command.verbose = getConfigInt(arguments, "--verbose")

	return command
}

func (command *RenderV2Command) Execute() {
	var arr []sourceAssetRequest
	saids := make([]string, 0, 0)
	for _, file := range command.filesToSubmit() {
		if command.verbose > 0 {
			log.Println("Adding file to request:", file)
		}
		attrs := make(map[string][]string)
		attrs["type"] = []string{filepath.Ext(file[5:])[1:]}
		uuid, _ := util.NewUuid()
		saids = append(saids, uuid)
		req := sourceAssetRequest{
			Id:         uuid,
			Url:        file,
			Attributes: attrs,
		}
		arr = append(arr, req)
	}
	req := previewGenerateRequest{
		SourceAssets: arr,
		TemplateIds:  command.templateIds,
	}
	url := fmt.Sprintf("http://%s/api/v2/preview/", command.host)
	bytes, _ := json.Marshal(req)
	if command.verbose > 1 {
		prettyjson, _ := json.MarshalIndent(req, "", "	")
		log.Println("Request:")
		log.Println(string(prettyjson))
	}
	hr, err := http.NewRequest("PUT", url, strings.NewReader(string(bytes)))

	if err != nil {
		fmt.Println(err.Error())
		return
	}
	client := &http.Client{}
	resp, err := client.Do(hr)

	if err != nil {
		fmt.Println(err.Error())
		return
	}
	target := url
	for idx, i := range saids {
		if idx != 0 {
			target += "&"
		} else {
			target += "?"
		}
		target += "id=" + i
	}
	if command.verbose > 0 {
		log.Println("Response found at:", target)
	}
	if command.verbose > 1 {
		log.Println("Response body:")
		b, _ := ioutil.ReadAll(resp.Body)
		fmt.Println(string(b))
	}
}

func (command *RenderV2Command) filesToSubmit() []string {
	files := make([]string, 0, 0)
	for _, file := range command.files {
		shouldTry, path := command.absFilePath(file)
		log.Println(shouldTry, path)
		if shouldTry {
			isDir, err := util.IsDirectory(path)
			if err == nil {
				if isDir {
					subdirFiles, err := ioutil.ReadDir(path)
					if err == nil {
						for _, subdirFile := range subdirFiles {
							if !subdirFile.IsDir() {
								files = append(files, "file://"+filepath.Join(path, subdirFile.Name()))
							}
						}
					}
				} else {
					files = append(files, "file://"+path)
				}
			}
		} else {
			if strings.HasPrefix(file, "http://") || strings.HasPrefix(file, "https://") {
				files = append(files, path)
			}
			if strings.HasPrefix(file, "s3://") {
				files = append(files, path)
			}
		}
	}
	return files
}

func (command *RenderV2Command) absFilePath(file string) (bool, string) {
	if strings.HasPrefix(file, "http://") || strings.HasPrefix(file, "https://") {
		return false, file
	}
	if strings.HasPrefix(file, "s3://") {
		return false, file
	}
	if strings.HasPrefix(file, "file://") {
		return true, file[7:]
	}
	if strings.HasPrefix(file, "/") {
		return true, file
	}
	return true, filepath.Join(util.Cwd(), file)
}
