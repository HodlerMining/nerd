package cmd

import (
	"context"
	"time"

	"github.com/mitchellh/cli"
	"github.com/nerdalize/nerd/svc"
	"github.com/pkg/errors"
)

//JobRun command
type JobRun struct {
	KubeOpts

	*command
}

//JobRunFactory creates the command
func JobRunFactory() cli.CommandFactory {
	cmd := &JobRun{}
	cmd.command = createCommand(cmd.Execute, cmd.Description, cmd.Usage, cmd)
	return func() (cli.Command, error) {
		return cmd, nil
	}
}

//Execute runs the command
func (cmd *JobRun) Execute(args []string) (err error) {
	if len(args) < 1 {
		return errors.New(MessageNotEnoughArguments)
	}

	kopts := cmd.KubeOpts
	deps, err := NewDeps(kopts)
	if err != nil {
		return errors.Wrap(err, "failed to configure")
	}

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	in := &svc.RunJobInput{
		Image: args[0],
		//@TODO add more options and args
	}

	kube := svc.NewKube(deps, kopts.Namespace)
	out, err := kube.RunJob(ctx, in)
	if err != nil {
		return errors.Wrap(err, "failed to run job")
	}

	//@TODO find a way of formatting the output

	cmd.logs.Printf("%#v", out)
	return nil
}

// Description returns long-form help text
func (cmd *JobRun) Description() string { return PlaceholderHelp }

// Synopsis returns a one-line
func (cmd *JobRun) Synopsis() string { return PlaceholderSynopsis }

// Usage shows usage
func (cmd *JobRun) Usage() string { return PlaceholderUsage }
