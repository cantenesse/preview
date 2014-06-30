package cli

import (
	"fmt"
	"github.com/bmizerany/pat"
	"github.com/codegangsta/negroni"
	"github.com/etix/stoppableListener"
	"github.com/ngerakines/preview/api"
	"github.com/ngerakines/preview/common"
	"github.com/ngerakines/preview/config"
	"github.com/ngerakines/preview/docserver"
	"log"
	"net"
	"net/http"
	"os"
	"time"
)

type DocumentDaemonCommand struct {
	config string
}

type DocumentDaemonContext struct {
	appConfig            *config.DocumentConfig
	downloader           common.Downloader
	temporaryFileManager common.TemporaryFileManager
	negroni              *negroni.Negroni
	documentBlueprint    api.Blueprint
	conversionManager    *docserver.ConversionManager
	listener             *stoppableListener.StoppableListener
}

func NewDocumentDaemonCommand(arguments map[string]interface{}) PreviewCliCommand {
	command := new(DocumentDaemonCommand)
	command.config = getConfigString(arguments, "--config")
	return command
}

func (command *DocumentDaemonCommand) String() string {
	return fmt.Sprintf("DocumentDaemonCommand<config=%s>", command.config)
}

func (command *DocumentDaemonCommand) Execute() {
	docConfig, err := config.LoadDocumentConfig(command.config)
	if err != nil {
		log.Fatal(err.Error())
		return
	}

	app := new(DocumentDaemonContext)
	app.appConfig = docConfig
	app.temporaryFileManager = common.NewTemporaryFileManager()
	app.downloader = common.NewDownloader(app.appConfig.Downloader.BasePath, app.appConfig.Common.LocalAssetStoragePath, app.temporaryFileManager, false, []string{}, nil)

	app.conversionManager = docserver.NewConversionManager(app.downloader, app.temporaryFileManager, app.appConfig.Conversion.BasePath)

	for i := 0; i < app.appConfig.Conversion.MaxWork; i++ {
		app.conversionManager.AddConversionAgent()
	}

	p := pat.New()
	app.documentBlueprint = api.NewDocumentBlueprint(app.conversionManager)
	app.documentBlueprint.AddRoutes(p)

	app.negroni = negroni.Classic()
	app.negroni.UseHandler(p)

	httpListener, err := net.Listen("tcp", app.appConfig.Http.Listen)
	if err != nil {
		panic(err)
	}
	app.listener = stoppableListener.Handle(httpListener)
	http.Serve(app.listener, app.negroni)

	if app.listener.Stopped {
		var alive int

		/* Wait at most 5 seconds for the clients to disconnect */
		for i := 0; i < 5; i++ {
			/* Get the number of clients still connected */
			alive = app.listener.ConnCount.Get()
			if alive == 0 {
				break
			}
			log.Printf("%d client(s) still connected…\n", alive)
			time.Sleep(1 * time.Second)
		}

		alive = app.listener.ConnCount.Get()
		if alive > 0 {
			log.Fatalf("Server stopped after 5 seconds with %d client(s) still connected.", alive)
		} else {
			log.Println("Server stopped gracefully.")
			os.Exit(0)
		}
	} else if err != nil {
		log.Fatal(err)
	}
}
