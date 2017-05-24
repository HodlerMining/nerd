package command

import (
	"github.com/mitchellh/cli"
	"github.com/pkg/errors"
)

//Task command
type Task struct {
	*command
}

//TaskFactory returns a factory method for the join command
func TaskFactory() (cli.Command, error) {
	comm, err := newCommand("nerd task <subcommand>", "manage the lifecycle of compute tasks", "", nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create command")
	}
	cmd := &Task{
		command: comm,
	}
	cmd.runFunc = cmd.DoRun

	return cmd, nil
}

//DoRun is called by run and allows an error to be returned
func (cmd *Task) DoRun(args []string) (err error) {
	return errShowHelp
}
