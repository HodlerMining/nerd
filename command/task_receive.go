package command

import (
	"fmt"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/mitchellh/cli"
	nerdaws "github.com/nerdalize/nerd/nerd/aws"
	"github.com/pkg/errors"
)

//TaskReceive command
type TaskReceive struct {
	*command
}

//TaskReceiveFactory returns a factory method for the join command
func TaskReceiveFactory() (cli.Command, error) {
	comm, err := newCommand("nerd task receive <queue-id>", "wait for a new task run to be available on a queue", "", nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create command")
	}
	cmd := &TaskReceive{
		command: comm,
	}
	cmd.runFunc = cmd.DoRun

	return cmd, nil
}

//DoRun is called by run and allows an error to be returned
func (cmd *TaskReceive) DoRun(args []string) (err error) {
	if len(args) < 1 {
		return fmt.Errorf("not enough arguments, see --help")
	}

	bclient, err := NewClient(cmd.ui, cmd.config, cmd.session)
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

	out, err := bclient.ReceiveTaskRuns(ss.Project.Name, args[0], time.Minute*3, qops)
	if err != nil {
		HandleError(err)
	}

	logrus.Infof("Task Receiving: %v", out)
	return nil
}
