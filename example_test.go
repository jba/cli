// Copyright 2021 Jonathan Amsterdam.

package cli_test

import (
	"context"
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
	cli.Run(context.Background(), []string{
		"show", "-v", "-bun", "-limit", "8", "-nums", "1,2,3", "-envs", "d,s,p", "abc", "3.2", "-4"})

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
	cli.Run(context.Background(), []string{"opts", "req", "o1", "o2"})
	*c = opts{}
	cli.Run(context.Background(), []string{"opts", "req"})

	// Output:
	// &{Req:req Opt1:o1 Opt2:o2}
	// &{Req:req Opt1: Opt2:}
}
