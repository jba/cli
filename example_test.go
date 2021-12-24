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
	Verbose bool      `cli:"flag=v,more detail"`
	Bun     Bool      `cli:"flag=, demo underlying bool"`
	Limit   int       `cli:"flag=limit , max to show"`
	Nums    []int     `cli:"flag=nums, some numbers"`
	Envs    []string  `cli:"flag=,oneof=d|s|p, environments"`
	ID      string    `cli:"identifier of value to show"`
	Floats  []float64 `cli:"name=flo, float values"`
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

func Example() {
	err := cli.Run(context.Background(), []string{
		"show", "-v", "-bun", "-limit", "8", "-nums", "1,2,3", "-envs", "d,s,p", "abc", "3.2", "-4"})
	if err != nil {
		fmt.Printf("Error: %v", err)
	}

	// Output:
	// showing abc, [3.2 -4]
	// verbosely
	// bun is true
	// limit = 8
	// nums = [1 2 3]
	// envs = [d s p]
}
