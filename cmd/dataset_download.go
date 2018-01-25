package cmd

import (
	"context"

	flags "github.com/jessevdk/go-flags"
	"github.com/mitchellh/cli"
	"github.com/nerdalize/nerd/pkg/transfer"
	"github.com/nerdalize/nerd/svc"
	"github.com/pkg/errors"
)

const (
	//OutputDirPermissions are the output directory's permissions.
	OutputDirPermissions = 0755
)

//DatasetDownload command
type DatasetDownload struct {
	KubeOpts
	TransferOpts
	JobOutput string `long:"job-output" description:"output of the precised job"`
	JobInput  string `long:"job-input" description:"input of the precised job"`

	*command
}

//DatasetDownloadFactory creates the command
func DatasetDownloadFactory(ui cli.Ui) cli.CommandFactory {
	cmd := &DatasetDownload{}
	cmd.command = createCommand(ui, cmd.Execute, cmd.Description, cmd.Usage, cmd, flags.None)
	return func() (cli.Command, error) {
		return cmd, nil
	}
}

//Execute runs the command
func (cmd *DatasetDownload) Execute(args []string) (err error) {
	if len(args) < 2 {
		return errShowUsage(MessageNotEnoughArguments)
	}

	deps, err := NewDeps(cmd.Logger(), cmd.KubeOpts)
	if err != nil {
		return renderConfigError(err, "failed to configure")
	}

	trans, err := cmd.TransferOpts.Transfer()
	if err != nil {
		return errors.Wrap(err, "failed configure transfer")
	}

	ref := &transfer.Ref{
		Bucket: cmd.TransferOpts.AWSS3Bucket,
		Key:    args[0],
	}

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, cmd.Timeout)
	defer cancel()

	err = trans.Download(ctx, ref, args[1])
	if err != nil {
		return errors.Wrap(err, "failed to download")
	}

	in := &svc.DownloadDatasetInput{
		JobInput:  cmd.JobInput,
		JobOutput: cmd.JobOutput,
		Name:      args[0],
		// Dest:      outputDir,
	}
	kube := svc.NewKube(deps)
	out, err := kube.DownloadDataset(ctx, in)
	if err != nil {
		return renderServiceError(err, "failed to download dataset")
	}

	cmd.out.Infof("Downloaded dataset: '%s'", out.Name)
	cmd.out.Infof("To delete the dataset from the cloud, use: `nerd dataset delete %s`", out.Name)
	return nil
}

// Description returns long-form help text
func (cmd *DatasetDownload) Description() string { return cmd.Synopsis() }

// Synopsis returns a one-line
func (cmd *DatasetDownload) Synopsis() string {
	return "Download results from a running job"
}

// Usage shows usage
func (cmd *DatasetDownload) Usage() string {
	return "nerd dataset download <DATASET-NAME> [--job-output=JOB-NAME] [--job-input=JOB-NAME] ~/my-projects/my-output-1"
}
