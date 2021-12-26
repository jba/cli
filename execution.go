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

func Main() {
	MainContext(context.Background())
}

func MainContext(ctx context.Context) {
	flag.Usage = func() {
		topCmd.usage(flag.CommandLine.Output(), true)
	}
	flag.Parse()
	if err := topCmd.Run(ctx, os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		var uerr *UsageError
		if errors.As(err, &uerr) {
			os.Exit(2)
		}
		os.Exit(1)
	}
}

func (c *Cmd) Run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return &UsageError{c, errors.New("no arguments")}
	}
	return c.findAndRun(ctx, args)
}

func (c *Cmd) findAndRun(ctx context.Context, args []string) error {
	if err := c.validate(); err != nil {
		return err
	}
	subc := c.findSub(args[0])
	if subc == nil {
		return &UsageError{c, fmt.Errorf("unknown command: %q", args[0])}
	}
	return subc.run(ctx, args[1:])
}

func (c *Cmd) run(ctx context.Context, args []string) error {
	if err := c.flags.Parse(args); err != nil {
		return err
	}
	if c.flags.NArg() > 0 && len(c.subs) > 0 {
		// There are more args and there are sub-commands, so run a sub-command.
		return c.findAndRun(ctx, c.flags.Args())
	}
	if err := bindFormals(c.formals, c.flags.Args()); err != nil {
		return err
	}
	if com, ok := c.c.(Command); ok {
		return com.Run(ctx)
	}
	// c is a group, but it is not a command.
	return &UsageError{c, errors.New("missing sub-command")}
}

func RunTest(args ...string) error {
	return topCmd.Run(context.Background(), args)
}

func bindFormals(formals []*formal, args []string) error {
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
			return errors.New("too few args")
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
		return errors.New("too many args")
	}
	return nil
}
