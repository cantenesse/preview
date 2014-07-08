package render

import (
	"encoding/json"
	"fmt"
	"github.com/ngerakines/preview/common"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"
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

var templateAliases = map[string]string{
	"jumbo":     "04a2c710-8872-4c88-9c75-a67175d3a8e7",
	"large":     "2eee7c27-75e2-4682-9920-9a4e14caa433",
	"medium":    "a89a6a0d-51d9-4d99-b278-0c5dfc538984",
	"small":     "eaa7be0e-354f-482c-ac75-75cbdafecb6e",
	"document":  common.DocumentConversionTemplateId,
	"video":     common.VideoConversionTemplateId,
	"optimized": common.OptimizedJumboTemplateId,
}

func NewRenderV2Command(arguments map[string]interface{}) common.Command {
	command := new(RenderV2Command)
	command.host = common.GetConfigString(arguments, "<host>")
	if len(command.host) == 0 {
		command.host = "localhost:8080"
	}
	command.files = common.GetConfigStringArray(arguments, "<file>")
	command.templateIds = common.GetConfigStringArray(arguments, "<templateId>")

	for idx, id := range command.templateIds {
		if alias, hasAlias := templateAliases[id]; hasAlias {
			command.templateIds[idx] = alias
		}
	}

	command.verbose = common.GetConfigInt(arguments, "--verbose")

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
		uuid, _ := common.NewUuid()
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
	requrl := fmt.Sprintf("http://%s/api/preview/", command.host)
	bytes, _ := json.Marshal(req)
	if command.verbose > 1 {
		prettyjson, _ := json.MarshalIndent(req, "", "	")
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
	target := requrl + "?"
	params := url.Values{}
	for _, id := range saids {
		params.Add("id", id)
	}
	target += params.Encode()

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
			files = append(files, urlsFromDirectory(path)...)
		} else {
			if common.IsHttpUrl(file) {
				files = append(files, path)
			}
			if common.IsS3Url(file) {
				files = append(files, path)
			}
		}
	}
	return files
}

func urlsFromDirectory(path string) []string {
	files := make([]string, 0, 0)
	isDir, err := common.IsDirectory(path)
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
	return files
}

func (command *RenderV2Command) absFilePath(file string) (bool, string) {
	if common.IsHttpUrl(file) {
		return false, file
	}
	if common.IsS3Url(file) {
		return false, file
	}
	if common.IsFileUrl(file) {
		return true, file[7:]
	}
	if strings.HasPrefix(file, "/") {
		return true, file
	}
	return true, filepath.Join(common.Cwd(), file)
}
