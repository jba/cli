// Copyright 2021 Jonathan Amsterdam.

package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"reflect"
)

// Code for running commands.

// Main invokes a command using the program's command-line arguments, passing it
// the given context. It returns the exit code for the process. Typical use:
//
//     var top = cli.Top(&cli.Command{...})
//     os.Exit(top.Main(context.Background()))
func (c *Command) Main(ctx context.Context) int {
	return c.mainWithArgs(ctx, os.Args[1:])
}

// Separated for testing.
func (c *Command) mainWithArgs(ctx context.Context, args []string) int {
	if err := c.validateAll(); err != nil {
		panic(err)
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
func (c *Command) Run(ctx context.Context, args []string) error {
	if err := c.validate(); err != nil {
		return err
	}
	if err := c.flags.Parse(args); err != nil {
		return &UsageError{c, err}
	}
	if c.flags.NArg() > 0 && len(c.subs) > 0 {
		// There are more args and there are sub-commands, so run a sub-command.
		subc := c.findSub(c.flags.Arg(0))
		if subc == nil {
			return &UsageError{c, fmt.Errorf("unknown command: %q", c.flags.Arg(0))}
		}
		return subc.Run(ctx, c.flags.Args()[1:])
	}
	if err := c.bindFormals(c.formals, c.flags.Args()); err != nil {
		return err
	}
	if r, ok := c.Struct.(Runnable); ok {
		err := r.Run(ctx)
		var uerr *UsageError
		if errors.As(err, &uerr) {
			uerr.cmd = c
		}
		return err
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
				return fmt.Errorf("%s: need at least %d args, got %d", f.name, f.min, nArgsLeft)
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
			return &UsageError{cmd: c, Err: errors.New("too few args")}
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
		return &UsageError{cmd: c, Err: errors.New("too many args")}
	}
	return nil
}
