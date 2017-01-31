package main

import (
	"fmt"
	"os"

	"github.com/nerdalize/nerd/command"

	"github.com/mitchellh/cli"
)

var (
	name    = "nerd"
	version = "build.from.src"
)

func main() {
	c := cli.NewCLI(name, version)
	c.Args = os.Args[1:]
	c.Commands = map[string]cli.CommandFactory{
		"login":    command.LoginFactory(),
		"upload":   command.UploadFactory(),
		"run":      command.RunFactory(),
		"logs":     command.LogsFactory(),
		"work":     command.WorkFactory(),
		"download": command.DownloadFactory(),
	}

	status, err := c.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s", name, err)
	}

	os.Exit(status)
}
