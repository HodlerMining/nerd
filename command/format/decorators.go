package format

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"text/tabwriter"

	"github.com/pkg/errors"
)

type jsonDecorator struct {
	v interface{}
}

//JSONDecorator is a decorator that outputs JSON
func JSONDecorator(v interface{}) *jsonDecorator {
	return &jsonDecorator{
		v: v,
	}
}

//Decorate writes JSON to out
func (d *jsonDecorator) Decorate(out io.Writer) error {
	enc := json.NewEncoder(out)
	return enc.Encode(d.v)
}

type tmplDecorator struct {
	v    interface{}
	tmpl string
}

//TmplDecorator is a decorator that uses golang's templating
func TmplDecorator(v interface{}, tmpl string) *tmplDecorator {
	return &tmplDecorator{
		v:    v,
		tmpl: tmpl,
	}
}

//Decorate writes templated output to out
func (d *tmplDecorator) Decorate(out io.Writer) error {
	tmpl, err := template.New("tmpl").Parse(d.tmpl)
	if err != nil {
		return errors.Wrapf(err, "failed to create new output template for template %v", d.tmpl)
	}
	err = tmpl.Execute(out, d.v)
	if err != nil {
		return errors.Wrap(err, "failed to parse output into template")
	}
	return nil
}

type tableDecorator struct {
	v      interface{}
	header string
	tmpl   string
}

//TableDecorator is a decorator that writes a table using golang's templating
func TableDecorator(v interface{}, header, tmpl string) *tableDecorator {
	return &tableDecorator{
		v:      v,
		header: header,
		tmpl:   tmpl,
	}
}

//Decorate writes the table to out
func (d *tableDecorator) Decorate(out io.Writer) error {
	w := tabwriter.NewWriter(out, 0, 0, 3, ' ', tabwriter.TabIndent)
	fmt.Fprintln(w, d.header)
	tmpl, err := template.New("tmpl").Parse(d.tmpl)
	if err != nil {
		return errors.Wrapf(err, "failed to create new output template for template %v", d.tmpl)
	}
	err = tmpl.Execute(w, d.v)
	if err != nil {
		return errors.Wrap(err, "failed to parse output into template")
	}
	w.Flush()
	return nil
}

type notImplDecorator struct {
	outputType OutputType
}

//NotImplDecorator can be used when a decorator is not implemented
func NotImplDecorator(ot OutputType) *notImplDecorator {
	return &notImplDecorator{
		outputType: ot,
	}
}

//Decorate shows a message to indicate that this decorator type is not implemented
func (d *notImplDecorator) Decorate(out io.Writer) error {
	fmt.Fprintf(out, "the selected output format (%v) is not supported", d.outputType)
	return nil
}
