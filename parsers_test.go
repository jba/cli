// Copyright 2021 Jonathan Amsterdam.

package cli

import (
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
)

type Int int

func TestParsers(t *testing.T) {
	for _, test := range []struct {
		name    string
		tval    interface{}
		choices []string
		isFlag  bool
		input   string
		want    interface{}
	}{
		{
			name:  "string",
			tval:  "",
			input: "foo",
			want:  "foo",
		},
		{
			name:  "bool",
			tval:  false,
			input: "TRUE",
			want:  true,
		},
		{
			name:  "int",
			tval:  0,
			input: "-5",
			want:  -5,
		},
		{
			name:  "Int",
			tval:  Int(0),
			input: "1",
			want:  Int(1),
		},
		{
			name:  "uint16",
			tval:  uint16(0),
			input: "32767",
			want:  uint16(32767),
		},
		{
			name:  "[]int arg",
			tval:  []int(nil),
			input: "3",
			want:  3,
		},
		{
			name:   "[]int flag",
			tval:   []int(nil),
			isFlag: true,
			input:  "1 , -2,3",
			want:   []int{1, -2, 3},
		},
		{
			name:    "oneof",
			tval:    "",
			choices: []string{"a", "b"},
			input:   "b",
			want:    "b",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			parser, err := buildParser(reflect.TypeOf(test.tval), test.choices, test.isFlag)
			if err != nil {
				t.Fatal(err)
			}
			got, err := parser(test.input)
			if err != nil {
				t.Fatal(err)
			}
			if !cmp.Equal(got, test.want) {
				t.Errorf("got %v, want %v", got, test.want)
			}
		})
	}
}
