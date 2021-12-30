// Copyright 2021 Jonathan Amsterdam.

package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"reflect"

	"github.com/posener/complete/v2"
)

// Code for running commands.

// Main invokes a command using the program's command-line arguments, passing it
// the given context. It returns the exit code for the process.
// Main returns 0 for success, 1 for an error in command execution, and 2
// for a usage error (wrong number of arguments, unknown flag, etc.).
//
// Typically, Main is called on the top Command with the background context, and
// its return value is passed to os.Exit, like so:
//
//     var top = cli.Top(nil)
//     os.Exit(top.Main(context.Background()))
func (c *Command) Main(ctx context.Context) int {
	return c.mainWithArgs(ctx, os.Args[1:])
}

// Separated for testing.
func (c *Command) mainWithArgs(ctx context.Context, args []string) int {
	complete.Complete(os.Args[0], c)
	if err := c.validateAll(); err != nil {
		panic(err)
	}
	if c.flags == flag.CommandLine {
		c.flags.Init(flag.CommandLine.Name(), flag.ContinueOnError)
	}
	if err := c.Run(ctx, args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		fmt.Fprintln(flag.CommandLine.Output(), err)
		var uerr *UsageError
		if errors.As(err, &uerr) {
			return 2
		}
		return 1
	}
	return 0
}

// Run invokes the command on the arguments.
//
// If a command has both sub-commands and positional arguments, sub-commands
// take precedence. For example, if command C has sub-command S, then the command
// line
//   C S A
// will invoke S with argument A, while
//   C T A
// will invoke C with arguments T and A.
func (c *Command) Run(ctx context.Context, args []string) (err error) {
	defer func() {
		var uerr *UsageError
		if errors.As(err, &uerr) && uerr.cmd == nil {
			uerr.cmd = c
		}
	}()

	if err := c.validate(); err != nil {
		return err
	}
	if err := c.flags.Parse(args); err != nil {
		return &UsageError{c, err}
	}
	if b, ok := c.Struct.(interface{ Before(context.Context) error }); ok {
		if err := b.Before(ctx); err != nil {
			return err
		}
	}
	if c.flags.NArg() > 0 {
		// There are command-line arguments. Prefer a sub-command if there is one.
		if subc := c.findSub(c.flags.Arg(0)); subc != nil {
			return subc.Run(ctx, c.flags.Args()[1:])
		}
		// If there are sub-commands but no formals, then the error should be
		// that the sub-command is unknown, not that there are too many args.
		if len(c.subs) > 0 && len(c.formals) == 0 {
			return &UsageError{c, fmt.Errorf("unknown command %q", c.flags.Arg(0))}
		}
	}
	if err := c.bindFormals(c.formals, c.flags.Args()); err != nil {
		return err
	}
	if r, ok := c.Struct.(Runnable); ok {
		return r.Run(ctx)
	}
	// c is a group, but it is not a command.
	return &UsageError{c, errors.New("missing sub-command")}
}

func (c *Command) bindFormals(formals []*formal, args []string) error {
	a := 0 // index into args
	for i, f := range formals {
		if f.min >= 0 {
			// "Rest" arg. We've already checked that this is the last formal.
			nArgsLeft := len(args) - i
			if nArgsLeft < f.min {
				arg := "argument"
				if f.min != 1 {
					arg += "s"
				}
				return &UsageError{
					cmd: c,
					Err: fmt.Errorf("%s: need at least %d %s, got %d", f.name, f.min, arg, nArgsLeft),
				}
			}
			slice := reflect.MakeSlice(f.field.Type(), 0, nArgsLeft)
			for j := i; j < len(args); j++ {
				v, err := f.parser(args[j])
				if err != nil {
					return fmt.Errorf("%s: %v", f.name, err)
				}
				slice = reflect.Append(slice, reflect.ValueOf(v))
			}
			f.field.Set(slice)
			return nil
		} else if i >= len(args) {
			if f.opt {
				// This and all following args are optional, so we can skip.
				return nil
			}
			return &UsageError{cmd: c, Err: errors.New("too few arguments")}
		} else {
			v, err := f.parser(args[a])
			if err != nil {
				return fmt.Errorf("%s: %v", f.name, err)
			}
			f.field.Set(reflect.ValueOf(v))
			a++
		}
	}
	if a < len(args) {
		return &UsageError{cmd: c, Err: errors.New("too many arguments")}
	}
	return nil
}
