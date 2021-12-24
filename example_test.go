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
	Verbose bool    `cli:"flag=v,more detail"`
	Bun     Bool    `cli:"flag=, demo underlying bool"`
	Limit   int     `cli:"flag=limit , max to show"`
	Nums    []int   `cli:"flag=nums, some numbers"`
	ID      string  `cli:"identifier of value to show"`
	F       float64 `cli:"name=flo, a float value"`
}

func init() {
	cli.Register("show", &show{}, "show a thing")
}

func (c *show) Run(ctx context.Context) error {
	fmt.Printf("showing %s, %g\n", c.ID, c.F)
	if c.Verbose {
		fmt.Println("verbosely")
	}
	if c.Bun {
		fmt.Println("bun is true")
	}
	fmt.Printf("limit = %d\n", c.Limit)
	fmt.Printf("nums = %v\n", c.Nums)
	return nil
}

func Example() {
	err := cli.Run(context.Background(), []string{
		"show", "-v", "-bun", "-limit", "8", "-nums", "1,2,3", "abc", "3.2"})
	if err != nil {
		fmt.Printf("Error: %v", err)
	}

	// Output:
	// showing abc, 3.2
	// verbosely
	// bun is true
	// limit = 8
	// nums = [1 2 3]
}
