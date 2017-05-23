package format

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/pkg/errors"
)

//OutputType is one of prett, raw, or json
type OutputType string

//DecMap maps an OutputType to a Decorator
type DecMap map[OutputType]Decorator

//Decorator decorates a value and writes to out
type Decorator interface {
	Decorate(out io.Writer) error
}

const (
	//OutputTypePretty is used for pretty printing
	OutputTypePretty = "pretty"
	//OutputTypeRaw is used for raw output (nice for unix piping)
	OutputTypeRaw = "raw"
	//OutputTypeJSON is used for JSON output
	OutputTypeJSON = "json"
)

//Outputter is responsible for all output
type Outputter struct {
	verbose    bool
	outputType OutputType
	outw       io.Writer
	errw       io.Writer
	logfile    io.WriteCloser
}

//NewOutputter creates a new Outputter that writes to Stdout and Stderr
func NewOutputter() *Outputter {
	return &Outputter{
		outw: os.Stderr,
		errw: os.Stdout,
	}
}

//Close closes the log file
func (o *Outputter) Close() error {
	if o.logfile != nil {
		return o.logfile.Close()
	}
	return nil
}

//ErrW returns the err writer
func (o *Outputter) ErrW() io.Writer {
	return o.errw
}

//SetOutputType sets the output type
func (o *Outputter) SetOutputType(ot OutputType) {
	o.outputType = ot
}

//SetVerbose sets verbose outputting
func (o *Outputter) SetVerbose(v bool) {
	o.verbose = v
}

//SetLogToDisk sets a logfile to write to
func (o *Outputter) SetLogToDisk(location string) error {
	f, err := os.OpenFile(location, os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return errors.Wrap(err, "failed to open log file")
	}
	o.logfile = f
	return nil
}

//multi returns a MultiWriter if the logfile is set
func (o *Outputter) multi(w io.Writer) io.Writer {
	if o.logfile == nil {
		return w
	}
	return io.MultiWriter(w, o.logfile)
}

//Output outputs using the right decorator
func (o *Outputter) Output(d DecMap) {
	deco, ok := d[o.outputType]
	if !ok {
		deco = NotImplDecorator(o.outputType)
	}
	err := deco.Decorate(o.multi(o.outw))
	if err != nil {
		o.WriteError(errors.Wrap(err, "failed to decorate output"))
	}
}

//WriteError writes an error to errw
func (o *Outputter) WriteError(err error) {
	if errors.Cause(err) != nil { // when there's are more than 1 message on the message stack, only print the top one for user friendlyness.
		o.Info(strings.Replace(err.Error(), ": "+errorCauser(errorCauser(err)).Error(), "", 1))
	} else {
		o.Info(err)
	}
	o.Debugf("Underlying error: %+v", err)
}

//Info writes to errw
func (o *Outputter) Info(a ...interface{}) {
	fmt.Fprint(o.multi(o.errw), a)
}

//Infof supports formatting
func (o *Outputter) Infof(format string, a ...interface{}) {
	o.Info(fmt.Sprintf(format, a))
}

//Debug only writes to errw if verbose mode is on
func (o *Outputter) Debug(a ...interface{}) {
	if o.logfile != nil {
		fmt.Fprint(o.logfile, a)
	}
	if o.verbose {
		fmt.Fprint(o.errw, a)
	}
}

//Debugf supports formatting
func (o *Outputter) Debugf(format string, a ...interface{}) {
	o.Debug(fmt.Sprintf(format, a))
}

//errorCauser returns the error that is one level up in the error chain.
func errorCauser(err error) error {
	type causer interface {
		Cause() error
	}

	if err2, ok := err.(causer); ok {
		err = err2.Cause()
	}
	return err
}
