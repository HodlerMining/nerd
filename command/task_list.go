package command

import (
	"fmt"
	"os"

	"github.com/mitchellh/cli"
	"github.com/olekukonko/tablewriter"
	"github.com/pkg/errors"
)

//TaskList command
type TaskList struct {
	*command
}

//TaskListFactory returns a factory method for the join command
func TaskListFactory() (cli.Command, error) {
	comm, err := newCommand("nerd task list <workload-id>", "show a list of all task currently in a queue", "", nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create command")
	}
	cmd := &TaskList{
		command: comm,
	}
	cmd.runFunc = cmd.DoRun

	return cmd, nil
}

//DoRun is called by run and allows an error to be returned
func (cmd *TaskList) DoRun(args []string) (err error) {
	if len(args) < 1 {
		return fmt.Errorf("not enough arguments, see --help")
	}

	bclient, err := NewClient(cmd.config, cmd.session, cmd.outputter)
	if err != nil {
		return HandleError(err)
	}

	ss, err := cmd.session.Read()
	if err != nil {
		return HandleError(err)
	}
	out, err := bclient.ListTasks(ss.Project.Name, args[0], false)
	if err != nil {
		return HandleError(err)
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"WorkloadID", "TaskID", "Status", "OutputDataset"})
	for _, t := range out.Tasks {
		row := []string{}
		row = append(row, t.WorkloadID)
		row = append(row, fmt.Sprintf("%d", t.TaskID))
		row = append(row, t.Status)
		row = append(row, t.OutputDatasetID)
		table.Append(row)
	}

	table.Render()
	return nil
}
