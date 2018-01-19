package cmd

import (
	"context"
	"sort"
	"strings"

	humanize "github.com/dustin/go-humanize"
	flags "github.com/jessevdk/go-flags"
	"github.com/mitchellh/cli"
	"github.com/nerdalize/nerd/svc"
)

//DatasetList command
type DatasetList struct {
	KubeOpts

	*command
}

//DatasetListFactory creates the command
func DatasetListFactory(ui cli.Ui) cli.CommandFactory {
	cmd := &DatasetList{}
	cmd.command = createCommand(ui, cmd.Execute, cmd.Description, cmd.Usage, cmd, flags.None)
	return func() (cli.Command, error) {
		return cmd, nil
	}
}

//Execute runs the command
func (cmd *DatasetList) Execute(args []string) (err error) {
	kopts := cmd.KubeOpts
	deps, err := NewDeps(cmd.Logger(), kopts)
	if err != nil {
		return renderConfigError(err, "failed to configure")
	}

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, cmd.Timeout)
	defer cancel()

	in := &svc.ListDatasetsInput{}
	kube := svc.NewKube(deps)
	out, err := kube.ListDatasets(ctx, in)
	if err != nil {
		return renderServiceError(err, "failed to list datasets")
	}

	sort.Slice(out.Items, func(i int, j int) bool {
		return out.Items[i].Details.CreatedAt.After(out.Items[j].Details.CreatedAt)
	})

	hdr := []string{"DATASET", "CREATED AT", "SIZE", "OUTPUT OF", "INPUT FOR"}
	rows := [][]string{}
	for _, item := range out.Items {
		rows = append(rows, []string{
			item.Name,
			humanize.Time(item.Details.CreatedAt),
			humanize.Bytes(item.Details.Size),
			strings.Join(item.Details.OutputOf, ","),
			strings.Join(item.Details.InputFor, ","),
		})
	}

	return cmd.out.Table(hdr, rows)
}

// Description returns long-form help text
func (cmd *DatasetList) Description() string { return cmd.Synopsis() }

// Synopsis returns a one-line
func (cmd *DatasetList) Synopsis() string { return "Return datasets that are managed by the cluster" }

// Usage shows usage
func (cmd *DatasetList) Usage() string { return "nerd dataset list" }
