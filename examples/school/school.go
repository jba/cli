// Copyright 2021 Jonathan Amsterdam.

// School is an example command-line tool using github.com/jba/cli.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/jba/cli"
)

var top = cli.Top(nil)

func main() {
	os.Exit(top.Main(context.Background()))
}

type Student struct {
	Name string
	GPA  float64
}

type Course struct {
	Name string
}

var students = []*Student{
	{"Pat", 3.2},
	{"Al", 4.0},
	{"Cam", 2.8},
}

var courses = []*Course{
	{"Math"},
	{"Science"},
	{"History"},
}

func init() {
	cmd := top.Register("students", nil, "commands for students")
	cmd.Register("list", &studentsList{}, "list students")
	cmd.Register("show", &studentsShow{}, "show a single student")

	cmd = top.Register("courses", &coursesGroup{}, "commands for courses")
	cmd.Register("list", &coursesList{}, "list courses")
	cmd.Register("show", &coursesShow{}, "show some courses")
}

// The "students" command group
type studentsGroup struct{}

type studentsList struct {
	MinGPA float64 `flag=min, list only students above this GPA`
}

func (c *studentsList) Run(ctx context.Context) error {
	if c.MinGPA < 0 || c.MinGPA > 4.0 {
		return cli.NewUsageError(errors.New("min GPA out of range [0, 4]"))
	}
	for _, s := range students {
		if c.MinGPA == 0 || s.GPA >= c.MinGPA {
			fmt.Printf("%-8s  %g\n", s.Name, s.GPA)
		}
	}
	return nil
}

type studentsShow struct {
	Verbose bool `flag=v, show more detail`
	Name    string
}

func (c *studentsShow) Run(ctx context.Context) error {
	for _, s := range students {
		if s.Name == c.Name {
			fmt.Println(s.Name)
			if c.Verbose {
				fmt.Println("GPA:", s.GPA)
			}
			return nil
		}
	}
	return fmt.Errorf("no student named %q", c.Name)

}

type coursesGroup struct{}

type coursesList struct{}

func (c *coursesList) Run(ctx context.Context) error {
	for _, r := range courses {
		fmt.Println(r.Name)
	}
	return nil
}

type coursesShow struct {
	Names []string
}

func (c *coursesShow) Run(ctx context.Context) error {
	for _, n := range c.Names {
		found := false
		for _, r := range courses {
			if r.Name == n {
				fmt.Println(r.Name)
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("no course named %q", n)
		}
	}
	return nil
}
