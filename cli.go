// Copyright 2021 Jonathan Amsterdam.

//TODO:
// improve rendering of default flag values for slices, strings, (etc.?)
// split command doc on lines, do uniform indentation

package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"reflect"
	"strings"
)

// A Command represents a single command, or a group of commands.
type Command struct {
	// The name of the command as users will type it on the command line.
	Name string

	// A short string describing the command.
	Usage string

	// If not nil, then a pointer to a struct with some exported fields.
	// Each exported field is either a flag or an argument for the command,
	// as determined by the struct tag for the field.
	// See the package documentation for the syntax of the struct tags.
	// If the struct pointer implements Runnable, then it can be run
	// as a command. Otherwise, it represents a group of sub-commands.
	Struct interface{}

	flags   *flag.FlagSet
	formals []*formal
	super   *Command
	subs    []*Command
}

// A formal describes a positional argument.
type formal struct {
	name   string        // display name
	field  reflect.Value // "pointer" to corresponding field
	usage  string
	min    int       // for last slice, minimum args needed
	opt    bool      // if true, this and all following formals are optional
	parser parseFunc // convert and/or validate
}

// A Runnable is a command that can be run.
type Runnable interface {
	Run(ctx context.Context) error
}

func (c *Command) validate() error {
	// Check that c.c is either a Command, or has sub-commands.
	if _, ok := c.Struct.(Runnable); !ok && len(c.subs) == 0 {
		return fmt.Errorf("%s is not runnable and has no sub-commands", c.Name)
	}
	return nil
}

func (c *Command) validateAll() error {
	if err := c.validate(); err != nil {
		return err
	}
	for _, s := range c.subs {
		if err := s.validateAll(); err != nil {
			return err
		}
	}
	return nil
}

func (c *Command) usage(w io.Writer, single bool) {
	if single {
		fmt.Fprintln(w, "Usage:")
	}
	// If this is a group and we're only printing this and there are no flags, don't print a header.
	if !(single && len(c.subs) > 0 && c.numFlags() == 0) {
		h := c.usageHeader()
		if single && len(h)+len(c.Usage) <= 76 {
			fmt.Fprintf(w, "%s    %s\n", h, c.Usage)
		} else {
			fmt.Fprintf(w, "%s\n  %s\n", h, c.Usage)
		}
		for _, f := range c.formals {
			if f.usage != "" {
				fmt.Fprintf(w, "  %-10s %s\n", f.name, f.usage)
			}
		}
	}
	c.flags.SetOutput(w)
	c.flags.PrintDefaults()
	if single {
		for i, s := range c.subs {
			if i > 0 {
				fmt.Fprintln(w)
			}
			s.usage(w, false)
		}
	}
}

func (c *Command) fullName() string {
	name := c.Name
	if c.numFlags() > 0 {
		name += " [flags]"
	}
	if c.super == nil {
		return name
	}
	return c.super.fullName() + " " + name
}

func (c *Command) usageHeader() string {
	var b strings.Builder
	fmt.Fprint(&b, c.fullName())
	for _, f := range c.formals {
		fmt.Fprintf(&b, " %s", f.name)
		if f.min >= 0 {
			fmt.Fprint(&b, "...")
		}
	}
	return b.String()
}

func (c *Command) numFlags() int {
	n := 0
	c.flags.VisitAll(func(*flag.Flag) { n++ })
	return n
}

// UsageError is an error in how a command is invoked.
type UsageError struct {
	cmd *Command
	Err error
}

// NewUsageError constructs a UsageError from an error.
func NewUsageError(err error) *UsageError {
	return &UsageError{Err: err}
}

// Error implements the error interface.
func (u *UsageError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s: %v\n", u.cmd.Name, u.Err.Error())
	u.cmd.usage(&b, true)
	s := b.String()
	return s[:len(s)-1] // trim final newline
}

// Unwrap supports errors.Is and errors.As.
func (u *UsageError) Unwrap() error {
	return u.Err
}

// Cut cuts s around the first instance of sep,
// returning the text before and after sep.
// The found result reports whether sep appears in s.
// If sep does not appear in s, cut returns s, "", false.
//
// TODO: remove when go1.19 is out.
func stringsCut(s, sep string) (before, after string, found bool) {
	if i := strings.Index(s, sep); i >= 0 {
		return s[:i], s[i+len(sep):], true
	}
	return s, "", false
}
