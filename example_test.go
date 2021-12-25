// Copyright 2021 Jonathan Amsterdam.

package cli_test

import (
	"context"
	"errors"
	"fmt"

	"github.com/jba/cli"
)

type (
	Bool bool
	Int  int
)

type show struct {
	Verbose bool     `cli:"flag=v,more detail"`
	Bun     Bool     `cli:"flag=, demo underlying bool"`
	Limit   int      `cli:"flag=limit , max to show"`
	Nums    []int    `cli:"flag=nums, some numbers"`
	Envs    []string `cli:"flag=,oneof=d|s|p, environments"`

	ID     string    `cli:"identifier of value to show"`
	Floats []float64 `cli:"name=numbers, float values"`
}

func init() {
	cli.Register("show", &show{}, "show a thing")
}

func (c *show) Run(ctx context.Context) error {
	fmt.Printf("showing %s, %v\n", c.ID, c.Floats)
	if c.Verbose {
		fmt.Println("verbosely")
	}
	if c.Bun {
		fmt.Println("bun is true")
	}
	fmt.Printf("limit = %d\n", c.Limit)
	fmt.Printf("nums = %v\n", c.Nums)
	fmt.Printf("envs = %v\n", c.Envs)
	return nil
}

func Example_show() {
	cli.RunTest("show", "-v", "-bun", "-limit", "8", "-nums", "1,2,3", "-envs", "d,s,p", "abc", "3.2", "-4")

	// Output:
	// showing abc, [3.2 -4]
	// verbosely
	// bun is true
	// limit = 8
	// nums = [1 2 3]
	// envs = [d s p]
}

type opts struct {
	Req  string `required arg`
	Opt1 string `opt=, optional 1`
	Opt2 string `optional 2`
}

func (x *opts) Run(ctx context.Context) error {
	fmt.Printf("%+v\n", x)
	return nil
}

func Example_opts() {
	c := &opts{}
	cli.Register("opts", c, "optional args")
	cli.RunTest("opts", "req", "o1", "o2")
	*c = opts{}
	cli.RunTest("opts", "req")

	// Output:
	// &{Req:req Opt1:o1 Opt2:o2}
	// &{Req:req Opt1: Opt2:}
}

type subs struct {
	F int `flag=, a flag`
}

type subs_a struct {
	A int
}

type subs_b struct {
	B int
}

var subsCmd *cli.Cmd

func init() {
	subsCmd = cli.Register("subs", &subs{}, "subs")
	subsCmd.Register("a", &subs_a{}, "do a to subs")
	subsCmd.Register("b", &subs_b{}, "do b to subs")
}

func (s *subs) Run(ctx context.Context) error {
	return errors.New("this is a group")
}
func (s *subs_a) Run(ctx context.Context) error {
	fmt.Println("a", s)
	return nil
}

func (s *subs_b) Run(ctx context.Context) error {
	fmt.Println("b", s)
	return nil
}

func Example_subs() {
	cli.RunTest("subs", "a", "3")
	cli.RunTest("subs", "b", "2")

	// Output:
	// a &{3}
	// b &{2}
}
