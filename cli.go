// Copyright 2021 Jonathan Amsterdam.

//TODO:
// distinguish -h,-help and exit 0
// sub-commands
// improve rendering of default flag values for slices, strings, (etc.?)
// split command doc on lines, do uniform indentation

package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Command interface {
	Run(ctx context.Context) error
}

type Cmd struct {
	name    string
	c       Command
	doc     string
	flags   *flag.FlagSet
	nFlags  int
	formals []*formal
	subs    []*Cmd
}

var topCmd *Cmd = &Cmd{
	name:  filepath.Base(os.Args[0]),
	flags: flag.CommandLine,
}

func usage(w io.Writer) {
	fmt.Fprintf(w, "Usage of %s:\n", topCmd.name)
	for _, c := range topCmd.subs {
		c.usage(w, false)
	}
	fmt.Fprintln(w, "\nGlobal flags (specify before command name):")
	topCmd.flags.SetOutput(w)
	topCmd.flags.PrintDefaults()
}

func (c *Cmd) usage(w io.Writer, single bool) {
	h := c.usageHeader()
	if single && len(h)+len(c.doc) <= 76 {
		fmt.Fprintf(w, "%s    %s\n", h, c.doc)
	} else {
		fmt.Fprintf(w, "%s\n  %s\n", h, c.doc)
	}
	for _, f := range c.formals {
		if f.doc != "" {
			fmt.Fprintf(w, "  %-10s %s\n", f.name, f.doc)
		}
	}
	c.flags.SetOutput(w)
	c.flags.PrintDefaults()
	fmt.Fprintln(w)
}

func (c *Cmd) usageHeader() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s", c.name)
	if c.nFlags > 0 {
		fmt.Fprint(&b, " [flags]")
	}
	for _, f := range c.formals {
		fmt.Fprintf(&b, " %s", f.name)
		if f.min >= 0 {
			fmt.Fprint(&b, "...")
		}
	}
	return b.String()
}

// UsageError is an error in how the command is invoked.
// If returned from Command.Run, then the usage message for
// the command will be printed in addition to the underlying error,
// and the process will exit with code 2.
type UsageError struct {
	err error
}

func (u *UsageError) Error() string {
	return u.err.Error()
}

func (u *UsageError) Unwrap() error {
	return u.err
}

// Cut cuts s around the first instance of sep,
// returning the text before and after sep.
// The found result reports whether sep appears in s.
// If sep does not appear in s, cut returns s, "", false.
//
// TODO: remove when go1.18 is out.
func stringsCut(s, sep string) (before, after string, found bool) {
	if i := strings.Index(s, sep); i >= 0 {
		return s[:i], s[i+len(sep):], true
	}
	return s, "", false
}
