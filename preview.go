package main

import (
	"github.com/docopt/docopt.go"
	"github.com/ngerakines/preview/common"
	"github.com/ngerakines/preview/daemon"
	"github.com/ngerakines/preview/render"
	"github.com/ngerakines/preview/verify"
	"log"
)

var (
	githash string = ""
)

func main() {

	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	usage := `Preview

Usage: preview [--help --version --config=<file>]
       preview daemon [--help --version --config <file>]
       preview render [--verbose... --verify] <host> <file>...
       preview renderV2 [--verbose...] <host> (--template <templateId>)... <file>...
       preview verify [--verbose... --config=<file> --timeout=<timeout>] <host> <filepath>

Options:
  --help           Show this screen.
  --version        Show version.
  --verbose        Verbose
  --verify         Verify that a generate preview request completes
  --config=<file>  The configuration file to use.`

	arguments, _ := docopt.Parse(usage, nil, true, version(), false)

	var command common.Command
	switch getCommand(arguments) {
	case "render":
		{
			command = render.NewRenderCommand(arguments)
		}
	case "daemon":
		{
			command = daemon.NewDaemonCommand(arguments)
		}
	case "renderV2":
		{
			command = render.NewRenderV2Command(arguments)
		}
	case "verify":
		{
			command = verify.NewVerifyCommand(arguments)
		}
	}
	command.Execute()
}

func version() string {
	previewVersion := "1.2.0"
	if len(githash) > 0 {
		return previewVersion + "+" + githash
	}
	return previewVersion
}

func getCommand(arguments map[string]interface{}) string {
	if common.GetConfigBool(arguments, "render") {
		return "render"
	} else if common.GetConfigBool(arguments, "renderV2") {
		return "renderV2"
	} else if common.GetConfigBool(arguments, "verify") {
		return "verify"
	}
	return "daemon"
}
