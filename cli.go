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
	c       interface{} // either Command, or a pure group
	doc     string
	flags   *flag.FlagSet
	nFlags  int
	formals []*formal
	super   *Cmd
	subs    []*Cmd
}

var topCmd = &Cmd{
	name:  filepath.Base(os.Args[0]),
	flags: flag.CommandLine,
}

func (c *Cmd) validate() error {
	// Check that c.c is either a Command, or has sub-commands.
	if _, ok := c.c.(Command); !ok && len(c.subs) == 0 {
		return fmt.Errorf("%s is not a Command and has no sub-commands", c.name)
	}
	return nil
}

func (c *Cmd) usage(w io.Writer, single bool) {
	if single {
		fmt.Fprintln(w, "Usage:")
	}
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
	if single {
		for _, s := range c.subs {
			s.usage(w, false)
		}
	}
	fmt.Fprintln(w)
}

func (c *Cmd) fullName() string {
	if c.super == nil {
		return c.name
	}
	return c.super.fullName() + " " + c.name
}

func (c *Cmd) usageHeader() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s", c.fullName())
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
type UsageError struct {
	Cmd *Cmd
	Err error
}

func (u *UsageError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s: %v\n", u.Cmd.name, u.Err.Error())
	u.Cmd.usage(&b, true)
	return b.String()
}

func (u *UsageError) Unwrap() error {
	return u.Err
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
