package command

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/mitchellh/cli"
	nerdaws "github.com/nerdalize/nerd/nerd/aws"
	"github.com/nerdalize/nerd/nerd/service/working/v1"
	"github.com/pkg/errors"
)

//WorkerWork command
type WorkerWork struct {
	*command
}

//WorkerWorkFactory returns a factory method for the join command
func WorkerWorkFactory() (cli.Command, error) {
	comm, err := newCommand("nerd worker work <queue-id> <command-tmpl> [arg-tmpl...]", "start working tasks of a queue locally", "", nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create command")
	}
	cmd := &WorkerWork{
		command: comm,
	}
	cmd.runFunc = cmd.DoRun

	return cmd, nil
}

//DoRun is called by run and allows an error to be returned
func (cmd *WorkerWork) DoRun(args []string) (err error) {
	if len(args) < 2 {
		return fmt.Errorf("not enough arguments, see --help")
	}

	bclient, err := NewClient(cmd.ui, cmd.config, cmd.session, cmd.outputter)
	if err != nil {
		HandleError(err)
	}

	ss, err := cmd.session.Read()
	if err != nil {
		HandleError(err)
	}
	creds := nerdaws.NewNerdalizeCredentials(bclient, ss.Project.Name)
	qops, err := nerdaws.NewQueueClient(creds, ss.Project.AWSRegion)
	if err != nil {
		HandleError(err)
	}

	logger := log.New(os.Stderr, "worker/", log.Lshortfile)
	conf := v1working.DefaultConf()

	worker := v1working.NewWorker(logger, bclient, qops, ss.Project.Name, args[0], args[1], args[2:], conf)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go worker.Start(ctx)

	exitCh := make(chan os.Signal, 1)
	signal.Notify(exitCh, os.Interrupt, syscall.SIGTERM)
	<-exitCh

	return nil
}
