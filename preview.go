package main

import (
	"github.com/docopt/docopt.go"
	"github.com/ngerakines/preview/cli"
)

var (
	githash string = ""
)

func main() {
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

	var command cli.PreviewCliCommand
	switch cli.GetCommand(arguments) {
	case "render":
		{
			command = cli.NewRenderCommand(arguments)
		}
	case "daemon":
		{
			command = cli.NewDaemonCommand(arguments)
		}
	case "renderV2":
		{
			command = cli.NewRenderV2Command(arguments)
		}
	case "verify":
		{
			command = cli.NewVerifyCommand(arguments)
		}
	}
	command.Execute()
}

func version() string {
	previewVersion := "1.1.0"
	if len(githash) > 0 {
		return previewVersion + "+" + githash
	}
	return previewVersion
}
