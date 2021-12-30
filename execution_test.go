// Copyright 2021 Jonathan Amsterdam.

package cli

import (
	"context"
	"fmt"
	"os"
	"testing"
)

type runnable struct {
	run func(context.Context) error
}

func (r runnable) Run(ctx context.Context) error {
	return r.run(ctx)
}

type suberr struct{ runnable }

func TestExitCode(t *testing.T) {
	defer func(f *os.File) { os.Stderr = f }(os.Stderr)
	os.Stderr = nil

	type c struct {
		F int `cli:"flag="`
	}
	top := Top(&Command{Struct: &c{}})
	top.Command("com", &c{}, "com usage").
		Command("sub", &suberr{runnable{func(context.Context) error {
			return context.Canceled
		}}}, "")

	for _, test := range []struct {
		args []string
		want int
	}{
		{args: nil, want: 2},
		{args: []string{"-h"}, want: 0},
		{args: []string{"-f", "x"}, want: 2}, // should be an int
		{args: []string{"com"}, want: 2},
		{args: []string{"com", "-h"}, want: 0},
		{args: []string{"com", "sub"}, want: 1},
		{args: []string{"com", "sub", "-h"}, want: 0},
		{args: []string{"com", "sub", "foo"}, want: 2}, // too many args

	} {
		got := top.mainWithArgs(context.Background(), test.args)
		if got != test.want {
			t.Errorf("%v: got %d, want %d", test.args, got, test.want)
		}
	}
}

type (
	c1 struct{ A int }
	c2 struct{ B bool }
)

func (c *c1) Run(context.Context) error {
	return fmt.Errorf("A=%d", c.A)
}

func (c *c2) Run(context.Context) error {
	return fmt.Errorf("B=%t", c.B)
}

func TestRun(t *testing.T) {
	top := Top(nil)
	top.Command("c1", &c1{}, "").Command("c2", &c2{}, "")

	ctx := context.Background()
	for _, test := range []struct {
		args []string
		want string
	}{
		{nil, "missing sub-command"},
		{[]string{"foo"}, `unknown command "foo"`},
		{[]string{"c1"}, "too few arguments"},
		{[]string{"c1", "3"}, "A=3"},
		{[]string{"c1", "c2", "true"}, "B=true"},
		{[]string{"c1", "c2"}, "too few arguments"},
	} {
		err := top.Run(ctx, test.args)
		var got string
		if err != nil {
			got, _, _ = stringsCut(err.Error(), "\n")
		}
		if _, after, found := stringsCut(got, ": "); found {
			got = after
		}
		if got != test.want {
			t.Errorf("%v:\ngot %q\nwant %q", test.args, got, test.want)
		}
	}
}
