package command

import (
	"fmt"
	"strconv"

	"github.com/Sirupsen/logrus"
	"github.com/mitchellh/cli"
	"github.com/pkg/errors"
)

//TaskStop command
type TaskStop struct {
	*command
}

//TaskStopFactory returns a factory method for the join command
func TaskStopFactory() (cli.Command, error) {
	comm, err := newCommand("nerd task stop <workload-id> <task-id>", "abort any run(s) of the specified task on a queue", "", nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create command")
	}
	cmd := &TaskStop{
		command: comm,
	}
	cmd.runFunc = cmd.DoRun

	return cmd, nil
}

//DoRun is called by run and allows an error to be returned
func (cmd *TaskStop) DoRun(args []string) (err error) {
	if len(args) < 2 {
		return fmt.Errorf("not enough arguments, see --help")
	}

	bclient, err := NewClient(cmd.ui, cmd.config, cmd.session)
	if err != nil {
		HandleError(err)
	}

	taskID, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		HandleError(errors.Wrap(err, "invalid task ID, must be a number"))
	}

	ss, err := cmd.session.Read()
	if err != nil {
		HandleError(err)
	}
	out, err := bclient.StopTask(ss.Project.Name, args[0], taskID)
	if err != nil {
		HandleError(err)
	}

	logrus.Infof("Task Stop: %v", out)
	return nil
}
