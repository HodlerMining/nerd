package command

import (
	"os"

	"github.com/mitchellh/cli"
	"github.com/olekukonko/tablewriter"
	"github.com/pkg/errors"
)

//DatasetList command
type DatasetList struct {
	*command
}

//DatasetListFactory returns a factory method for the join command
func DatasetListFactory() (cli.Command, error) {
	comm, err := newCommand("nerd dataset list", "show a list of all datasets", "", nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create command")
	}
	cmd := &DatasetList{
		command: comm,
	}
	cmd.runFunc = cmd.DoRun

	return cmd, nil
}

//DoRun is called by run and allows an error to be returned
func (cmd *DatasetList) DoRun(args []string) (err error) {
	bclient, err := NewClient(cmd.config, cmd.session)
	if err != nil {
		HandleError(err)
	}

	ss, err := cmd.session.Read()
	if err != nil {
		HandleError(err)
	}
	out, err := bclient.ListDatasets(ss.Project.Name)
	if err != nil {
		HandleError(err)
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"ProjectID", "DatasetID"})
	for _, t := range out.Datasets {
		row := []string{}
		row = append(row, t.ProjectID)
		row = append(row, t.DatasetID)
		table.Append(row)
	}

	table.Render()
	return nil
}
