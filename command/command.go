package command

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/Sirupsen/logrus"
	"github.com/jessevdk/go-flags"
	"github.com/mitchellh/cli"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/nerdalize/nerd/command/format"
	"github.com/nerdalize/nerd/nerd/conf"
)

var errShowHelp = errors.New("show error")

func newCommand(title, synopsis, help string, opts interface{}) (*command, error) {
	cmd := &command{
		help:     help,
		synopsis: synopsis,
		parser:   flags.NewNamedParser(title, flags.None),
		ui: &cli.BasicUi{
			Reader: os.Stdin,
			Writer: os.Stderr,
		},
		outputter: format.NewOutputter(),
	}
	if opts != nil {
		_, err := cmd.parser.AddGroup("options", "options", opts)
		if err != nil {
			return nil, err
		}
	}
	confOpts := &ConfOpts{
		ConfigFile:  cmd.setConfig,
		SessionFile: cmd.setSession,
		OutputOpts: OutputOpts{
			Output:        cmd.setOutput,
			VerboseOutput: cmd.setVerbose,
			JSONOutput:    cmd.setJSON,
		},
	}
	_, err := cmd.parser.AddGroup("output options", "output options", confOpts)
	if err != nil {
		return nil, err
	}
	return cmd, nil
}

//command is an abstract implementation for embedding in concrete commands and allows basic command functionality to be reused.
type command struct {
	help       string        //extended help message, show when --help a command
	synopsis   string        //short help message, shown on the command overview
	parser     *flags.Parser //option parser that will be used when parsing args
	ui         cli.Ui
	config     *conf.Config
	outputter  *format.Outputter
	jsonOutput bool
	session    *conf.Session
	runFunc    func(args []string) error
}

//Will write help text for when a user uses --help, it automatically renders all option groups of the flags.Parser (augmented with default values). It will show an extended help message if it is not empty, else it shows the synopsis.
func (c *command) Help() string {
	buf := bytes.NewBuffer(nil)
	c.parser.WriteHelp(buf)

	txt := c.help
	if txt == "" {
		txt = c.Synopsis()
	}

	return fmt.Sprintf(`
%s

%s`, txt, buf.String())
}

//Short explanation of the command as passed in the struction initialization
func (c *command) Synopsis() string {
	return c.synopsis
}

//Run wraps a signature that allows returning an error type and parses the arguments for the flags package. If flag parsing fails it sets the exit code to 127, if the command implementation returns a non-nil error the exit code is 1
func (c *command) Run(args []string) int {
	if c.parser != nil {
		var err error
		args, err = c.parser.ParseArgs(args)
		if err != nil {
			return 127
		}
	}

	if err := c.runFunc(args); err != nil {
		if err == errShowHelp {
			return cli.RunResultHelp
		}
		c.outputter.WriteError(err)
		return 1
	}

	return 0
}

//setConfig sets the cmd.config field according to the config file location
func (c *command) setConfig(loc string) {
	if loc == "" {
		var err error
		loc, err = conf.GetDefaultConfigLocation()
		if err != nil {
			c.outputter.WriteError(errors.Wrap(err, "failed to find config location"))
			os.Exit(-1)
		}
		err = createFile(loc, "{}")
		if err != nil {
			c.outputter.WriteError(errors.Wrapf(err, "failed to create config file %v", loc))
			os.Exit(-1)
		}
	}
	conf, err := conf.Read(loc)
	if err != nil {
		c.outputter.WriteError(errors.Wrap(err, "failed to read config file"))
		os.Exit(-1)
	}
	c.config = conf
	if conf.Logging.Enabled {
		logPath, err := homedir.Expand(conf.Logging.FileLocation)
		if err != nil {
			c.outputter.WriteError(errors.Wrap(err, "failed to find home directory"))
			os.Exit(-1)
		}
		err = createFile(logPath, "")
		if err != nil {
			c.outputter.WriteError(errors.Wrapf(err, "failed to create log file %v", logPath))
			os.Exit(-1)
		}
		err = c.outputter.SetLogToDisk(logPath)
		if err != nil {
			c.outputter.WriteError(errors.Wrap(err, "failed to set logging"))
			os.Exit(-1)
		}
	}
}

//setSession sets the cmd.session field according to the session file location
func (c *command) setSession(loc string) {
	if loc == "" {
		var err error
		loc, err = conf.GetDefaultSessionLocation()
		if err != nil {
			c.outputter.WriteError(errors.Wrap(err, "failed to find session location"))
			os.Exit(-1)
		}
		err = createFile(loc, "{}")
		if err != nil {
			c.outputter.WriteError(errors.Wrapf(err, "failed to create session file %v", loc))
			os.Exit(-1)
		}
	}
	c.session = conf.NewSession(loc)
}

//setVerbose sets verbose output formatting
func (c *command) setVerbose(verbose bool) {
	c.outputter.SetVerbose(verbose)
	if verbose {
		logrus.SetFormatter(new(logrus.TextFormatter))
		logrus.SetLevel(logrus.DebugLevel)
	}
}

//setJSON sets json output formatting
func (c *command) setOutput(output string) {
	switch output {
	case "json":
		c.outputter.SetOutputType(format.OutputTypeJSON)
	case "raw":
		c.outputter.SetOutputType(format.OutputTypeRaw)
	case "pretty":
		fallthrough
	default:
		c.outputter.SetOutputType(format.OutputTypePretty)
	}
}

//setJSON sets json output formatting
func (c *command) setJSON(json bool) {
	c.jsonOutput = json
	if json {
		logrus.SetFormatter(new(logrus.JSONFormatter))
	}
}

func createFile(path, content string) error {
	os.MkdirAll(filepath.Dir(path), 0755)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil && !os.IsExist(err) {
		return err
	}
	if err == nil {
		f.Write([]byte(content))
	}
	f.Close()
	return nil
}
