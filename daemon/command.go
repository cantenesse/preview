package daemon

import (
	"fmt"
	"github.com/ngerakines/preview/common"
	"log"
	"os"
	"os/signal"
)

type DaemonCommand struct {
	config string
}

func NewDaemonCommand(arguments map[string]interface{}) common.Command {
	command := new(DaemonCommand)
	command.config = common.GetConfigString(arguments, "--config")
	return command
}

func (command *DaemonCommand) String() string {
	return fmt.Sprintf("DaemonCommand<config=%s>", command.config)
}

func (command *DaemonCommand) Execute() {
	config, err := loadDaemonConfig(command.config)
	if err != nil {
		log.Fatal(err.Error())
		return
	}
	context, err := NewDaemonContext(config)
	if err != nil {
		log.Fatal(err.Error())
		return
	}

	k := make(chan os.Signal, 1)
	signal.Notify(k, os.Interrupt, os.Kill)
	go func() {
		<-k
		context.Stop()
	}()

	context.Start()
}
