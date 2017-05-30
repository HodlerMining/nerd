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
	v1datatransfer "github.com/nerdalize/nerd/nerd/service/datatransfer/v1"
	"github.com/nerdalize/nerd/nerd/service/working/v1"
	"github.com/pkg/errors"
)

//WorkloadWorkOpts describes command options
type WorkloadWorkOpts struct {
	OutputDir string `long:"output-dir" default:"" default-mask:"" description:"when set, data in --output-dir will be uploaded after each task run"`
}

//WorkloadWork command
type WorkloadWork struct {
	*command
	opts *WorkloadWorkOpts
}

//WorkloadWorkFactory returns a factory method for the join command
func WorkloadWorkFactory() (cli.Command, error) {
	opts := &WorkloadWorkOpts{}
	comm, err := newCommand("nerd workload work <workload-id> <command> [args...]", "start working tasks of a queue locally", "", opts)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create command")
	}
	cmd := &WorkloadWork{
		command: comm,
		opts:    opts,
	}
	cmd.runFunc = cmd.DoRun

	return cmd, nil
}

//DoRun is called by run and allows an error to be returned
func (cmd *WorkloadWork) DoRun(args []string) (err error) {
	if len(args) < 2 {
		return fmt.Errorf("not enough arguments, see --help")
	}

	bclient, err := NewClient(cmd.config, cmd.session)
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

	var worker *v1working.Worker
	if cmd.opts.OutputDir != "" {
		dataOps, err := nerdaws.NewDataClient(
			nerdaws.NewNerdalizeCredentials(bclient, ss.Project.Name),
			ss.Project.AWSRegion,
		)
		if err != nil {
			HandleError(errors.Wrap(err, "could not create aws dataops client"))
		}
		uploadConf := &v1datatransfer.UploadConfig{
			BatchClient: bclient,
			DataOps:     dataOps,
			LocalDir:    cmd.opts.OutputDir,
			ProjectID:   ss.Project.Name,
			Concurrency: 64,
		}
		worker = v1working.NewWorker(logger, bclient, qops, ss.Project.Name, args[0], args[1], args[2:], uploadConf, conf)
	} else {
		worker = v1working.NewWorker(logger, bclient, qops, ss.Project.Name, args[0], args[1], args[2:], nil, conf)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go worker.Start(ctx)

	exitCh := make(chan os.Signal, 1)
	signal.Notify(exitCh, os.Interrupt, syscall.SIGTERM)
	<-exitCh

	return nil
}
