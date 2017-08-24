package command

import (
	"github.com/mitchellh/cli"
	"github.com/pkg/errors"
)

//WorkloadStop command
type WorkloadStop struct {
	*command
}

//WorkloadStopFactory returns a factory method for the join command
func WorkloadStopFactory() (cli.Command, error) {
	comm, err := newCommand("nerd workload stop <workload-id>", "Stop a workload from providing compute capacity.", "", nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create command")
	}
	cmd := &WorkloadStop{
		command: comm,
	}
	cmd.runFunc = cmd.DoRun

	return cmd, nil
}

//DoRun is called by run and allows an error to be returned
func (cmd *WorkloadStop) DoRun(args []string) (err error) {
	if len(args) < 1 {
		return errors.Wrap(errShowHelp("show help"), "Not enough arguments, see below for usage.")
	}

	bclient, err := NewClient(cmd.config, cmd.session, cmd.outputter)
	if err != nil {
		return HandleError(err)
	}

	ss, err := cmd.session.Read()
	if err != nil {
		return HandleError(err)
	}

	projectID, err := ss.RequireProjectID()
	if err != nil {
		return HandleError(err)
	}

	_, err = bclient.StopWorkload(projectID, args[0])
	if err != nil {
		return HandleError(err)
	}

	cmd.outputter.Logger.Printf("Workload '%s' was stopped", args[0])
	return nil
}
