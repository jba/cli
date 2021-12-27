// Copyright 2021 Jonathan Amsterdam.

package main

import (
	"errors"
	"flag"
	"io/fs"
	"os"
	"testing"

	"github.com/google/go-cmdtest"
)

var update = flag.Bool("update", false, "update test files with results")

func Test(t *testing.T) {
	_, err := os.Stat("./school")
	if errors.Is(err, fs.ErrNotExist) {
		t.Skip("skipping because ./school does not exist. Run `go build` first.")
	} else if err != nil {
		t.Fatal(err)
	}
	ts, err := cmdtest.Read("testdata")
	if err != nil {
		t.Fatal(err)
	}
	ts.Commands["school"] = cmdtest.Program("./school")

	ts.Run(t, *update)
}
