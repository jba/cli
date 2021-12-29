// Copyright 2021 Jonathan Amsterdam.

package cli_test

import (
	"context"
	"fmt"
	"log"

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
	top := cli.Top(&cli.Command{})
	top.Command("show", &show{}, "show a thing")

	must(top.Run(context.Background(), []string{"show", "-v", "-bun", "-limit", "8", "-nums", "1,2,3", "-envs", "d,s,p", "abc", "3.2", "-4"}))

	// Output:
	// showing abc, [3.2 -4]
	// verbosely
	// bun is true
	// limit = 8
	// nums = [1 2 3]
	// envs = [d s p]
}

type opts struct {
	Req  string `cli:"required arg"`
	Opt1 string `cli:"opt=, optional 1"`
	Opt2 string `cli:"optional 2"`
}

func (x *opts) Run(ctx context.Context) error {
	fmt.Printf("%+v\n", x)
	return nil
}

func Example_opts() {
	ctx := context.Background()
	c := &opts{}
	top := cli.Top(&cli.Command{})
	top.Command("opts", c, "optional args")
	must(top.Run(ctx, []string{"opts", "req", "o1", "o2"}))
	*c = opts{}
	must(top.Run(ctx, []string{"opts", "req"}))

	// Output:
	// &{Req:req Opt1:o1 Opt2:o2}
	// &{Req:req Opt1: Opt2:}
}

type subs struct {
	F int `cli:"flag=, a flag"`
}

type subs_a struct {
	A int
}

type subs_b struct {
	B int
}

var top, subsCmd *cli.Command

func init() {
	top = cli.Top(nil)
	subsCmd = top.Command("subs", &subs{}, "doc for subs")
	subsCmd.Command("a", &subs_a{}, "doc for a")
	subsCmd.Command("b", &subs_b{}, "doc for b")
}

func (s *subs_a) Run(ctx context.Context) error {
	fmt.Println("a", s)
	return nil
}

func (s *subs_b) Run(ctx context.Context) error {
	fmt.Println("b", s)
	return nil
}

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func Example_subs() {
	ctx := context.Background()
	must(top.Run(ctx, []string{"subs", "a", "3"}))
	must(top.Run(ctx, []string{"subs", "b", "2"}))
	fmt.Println(top.Run(ctx, []string{"subs"}))

	// Output:
	// a &{3}
	// b &{2}
	// subs: missing sub-command
	// Usage:
	// cli.test [flags] subs [flags]    doc for subs
	//   -f value
	//     	a flag
	// cli.test [flags] subs [flags] a A
	//   doc for a
	//
	// cli.test [flags] subs [flags] b B
	//   doc for b
}
