// Copyright 2021 Jonathan Amsterdam.

package cli

import (
	"context"
	"os"
	"testing"
)

type suberr struct{}

func (suberr) Run(context.Context) error { return context.Canceled }

func TestExitCode(t *testing.T) {
	defer func(f *os.File) { os.Stderr = f }(os.Stderr)
	os.Stderr = nil

	type c struct {
		F int `cli:"flag="`
	}
	top := Top(&Command{Struct: &c{}})
	top.Register("com", &c{}, "com usage").
		Register("sub", &suberr{}, "")

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
