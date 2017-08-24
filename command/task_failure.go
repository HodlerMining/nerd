package command

import (
	"strconv"

	"github.com/mitchellh/cli"
	"github.com/pkg/errors"
)

//TaskFailure command
type TaskFailure struct {
	*command
}

//TaskFailureFactory returns a factory method for the join command
func TaskFailureFactory() (cli.Command, error) {
	comm, err := newCommand("nerd task failure <workload-id> <task-id> <run-token> <error-code> <err-message>", "Mark a task run as being failed.", "", nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create command")
	}
	cmd := &TaskFailure{
		command: comm,
	}
	cmd.runFunc = cmd.DoRun

	return cmd, nil
}

//DoRun is called by run and allows an error to be returned
func (cmd *TaskFailure) DoRun(args []string) (err error) {
	if len(args) < 5 {
		return errors.Wrap(errShowHelp("show help"), "Not enough arguments, see below for usage.")
	}

	bclient, err := NewClient(cmd.config, cmd.session, cmd.outputter)
	if err != nil {
		return HandleError(err)
	}

	taskID, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		return HandleError(errors.Wrap(err, "invalid task ID, must be a number"))
	}

	ss, err := cmd.session.Read()
	if err != nil {
		return HandleError(err)
	}

	projectID, err := ss.RequireProjectID()
	if err != nil {
		return HandleError(err)
	}

	out, err := bclient.SendRunFailure(projectID, args[0], taskID, args[2], args[3], args[4])
	if err != nil {
		return HandleError(err)
	}

	cmd.outputter.Logger.Printf("Task Failure: %v", out)
	return nil
}
