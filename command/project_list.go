package command

import (
	"fmt"
	"net/url"

	"github.com/Sirupsen/logrus"
	"github.com/mitchellh/cli"
	v1auth "github.com/nerdalize/nerd/nerd/client/auth/v1"
	"github.com/nerdalize/nerd/nerd/oauth"
	"github.com/pkg/errors"
)

//ProjectList command
type ProjectList struct {
	*command
}

//ProjectListFactory returns a factory method for the join command
func ProjectListFactory() (cli.Command, error) {
	comm, err := newCommand("nerd project list", "list all your projects", "", nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create command")
	}
	cmd := &ProjectList{
		command: comm,
	}
	cmd.runFunc = cmd.DoRun

	return cmd, nil
}

//DoRun is called by run and allows an error to be returned
func (cmd *ProjectList) DoRun(args []string) (err error) {
	authbase, err := url.Parse(cmd.config.Auth.APIEndpoint)
	if err != nil {
		HandleError(errors.Wrapf(err, "auth endpoint '%v' is not a valid URL", cmd.config.Auth.APIEndpoint))
	}
	authOpsClient := v1auth.NewOpsClient(v1auth.OpsClientConfig{
		Base:   authbase,
		Logger: logrus.StandardLogger(),
	})
	client := v1auth.NewClient(v1auth.ClientConfig{
		Base:               authbase,
		Logger:             logrus.StandardLogger(),
		OAuthTokenProvider: oauth.NewConfigProvider(authOpsClient, cmd.config.Auth.ClientID, cmd.session),
	})

	projects, err := client.ListProjects()
	if err != nil {
		HandleError(err)
	}
	for _, project := range projects.Projects {
		fmt.Printf("%v\n", project.Code)
	}

	return nil
}
