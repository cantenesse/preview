package main

import(
	"fmt"
	"net/http"
	"os"
)

var usage = "Usage: ./notify GENERATED_ASSET_ID"

func main() {
	if len(os.Args) < 2 {
		fmt.Println(usage)
		return
	}
	id := os.Args[1]
	url := "http://localhost:8080/zencoder/" + id
	fmt.Println("POSTing to", url)
	http.Post(url, "", nil)
}
