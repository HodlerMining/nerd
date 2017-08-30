package command

import (
	"net/url"

	"github.com/mitchellh/cli"
	"github.com/nerdalize/nerd/nerd/client/auth/v1"
	"github.com/nerdalize/nerd/nerd/conf"
	"github.com/nerdalize/nerd/nerd/oauth"
	"github.com/pkg/errors"
)

//ProjectSet command
type ProjectSet struct {
	*command
}

//ProjectSetFactory returns a factory method for the join command
func ProjectSetFactory() (cli.Command, error) {
	comm, err := newCommand("nerd project set", "Set current working project.", "", nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create command")
	}
	cmd := &ProjectSet{
		command: comm,
	}
	cmd.runFunc = cmd.DoRun

	return cmd, nil
}

//DoRun is called by run and allows an error to be returned
func (cmd *ProjectSet) DoRun(args []string) (err error) {
	if len(args) < 1 {
		return errShowHelp("Not enough arguments, see below for usage.")
	}
	projectSlug := args[0]

	authbase, err := url.Parse(cmd.config.Auth.APIEndpoint)
	if err != nil {
		return errors.Wrapf(err, "auth endpoint '%v' is not a valid URL", cmd.config.Auth.APIEndpoint)
	}
	authOpsClient := v1auth.NewOpsClient(v1auth.OpsClientConfig{
		Base:   authbase,
		Logger: cmd.outputter.Logger,
	})
	client := v1auth.NewClient(v1auth.ClientConfig{
		Base:               authbase,
		Logger:             cmd.outputter.Logger,
		OAuthTokenProvider: oauth.NewConfigProvider(authOpsClient, cmd.config.Auth.ClientID, cmd.session),
	})

	projects, err := client.ListProjects()
	if err != nil {
		return HandleError(err)
	}

	ok := false
	for _, project := range projects.Projects {
		if project.Slug == args[0] {
			ok = true
		}
	}

	if !ok {
		return HandleError(errors.New("Project not found, please check the project name. You can get a list of your projects by running `nerd project list`."))
	}

	err = cmd.session.WriteProject(projectSlug, conf.DefaultAWSRegion)
	if err != nil {
		return HandleError(err)
	}

	return nil
}
