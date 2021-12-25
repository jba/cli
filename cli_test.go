// Copyright 2021 Jonathan Amsterdam.

package cli

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestTagToMap(t *testing.T) {
	for _, test := range []struct {
		tag  string
		want map[string]string
	}{
		{"", map[string]string{}},
		{
			" flag=fl,\t name=n, some doc   ",
			map[string]string{
				"flag": "fl",
				"name": "n",
				"doc":  "some doc",
			},
		},
		{
			"oneof=a|b",
			map[string]string{"oneof": "a|b"},
		},
	} {
		got := tagToMap(test.tag)
		if !cmp.Equal(got, test.want) {
			t.Errorf("%q:\ngot  %+v\nwant %+v", test.tag, got, test.want)
		}
	}
}

func TestParseTag(t *testing.T) {
	// type s struct{
	// 	F int
	// }
	// f := reflect.ValueOf(s{}).Field(0)
	// for _, test := range []struct {
	// 	tag      string
	// 	wantName string
	// 	wantDoc  string
	// 	wantErr  bool
	// }{
	// 	{
	// 		tag:     "just doc",
	// 		wantDoc: "just doc",
	// 	},
	// 	{
	// 		tag:      "name:foo \tand then doc\t ",
	// 		wantName: "foo",
	// 		wantDoc:  "and then doc",
	// 	},
	// 	{
	// 		tag:     "oneof:1|2|4 some doc",
	// 		wantDoc: "some doc",
	// 	},
	// 	{
	// 		tag:     "oneof:1|2|4 some doc",
	// 		wantDoc: "some doc",
	// 	},
	// 	{
	// 		tag:      "\toneof:2\tname:bar\tdoc\t",
	// 		wantName: "bar",
	// 		wantDoc:  "doc",
	// 	},
	// } {
	// 	c := &cm
	// 	got, err := parseTag(test.tag, f)
	// 	if err != nil {
	// 		if !test.wantErr {
	// 			t.Errorf("%q: unwanted error: <%v>", test.tag, err)
	// 		}
	// 	} else {
	// 		if got.name != test.wantName {
	// 			t.Errorf("%q, name: got %q, want %q", test.tag, got.name, test.wantName)
	// 		}
	// 		if got.doc != test.wantDoc {
	// 			t.Errorf("%q, doc: got %q, want %q", test.tag, got.doc, test.wantDoc)
	// 		}
	// 		gotp, err := got.parser("2")
	// 		if err != nil {
	// 			t.Errorf("%q: unexpected parsing error: <%v>", test.tag, err)
	// 		} else if wantp := int64(2); gotp != wantp {
	// 			t.Errorf("%q: got %v, want %v", test.tag, gotp, wantp)
	// 		}
	// 	}
	// }

}

func TestProcessFieldsErrors(t *testing.T) {
	check := func(s interface{}, want string) {
		t.Helper()
		got := newCmd("", nil, "").processFields(s)
		if got == nil || !strings.Contains(got.Error(), want) {
			t.Errorf("got %v, want error containing %q", got, want)
		}
	}

	// oneof for non-string field
	type t1 struct {
		F int `cli:"oneof=a|b"`
	}
	check(&t1{}, "must be string")

	// tag on unexported field
	type t2 struct {
		f int `cli:"foo"`
	}
	check(&t2{}, "tag on unexported")

	// slice is not last
	type t3 struct {
		A []int `cli:"doc"`
		B bool  `cli:"doc"`
	}
	check(&t3{}, "last")

	// both args and sub-commands
	type t4 struct {
		A int
	}
	cmd := newCmd("", nil, "")
	if err := cmd.processFields(&t4{}); err != nil {
		t.Fatal(err)
	}
	_, got := cmd.register("sub", &sub{}, "")
	want := "cannot have both"
	if got == nil || !strings.Contains(got.Error(), want) {
		t.Errorf("got %v, want error containing %q", got, want)
	}

}

type sub struct {
}

func (sub) Run(context.Context) error {
	return nil
}
func TestBindFormals(t *testing.T) {
	var f1, f2, f3 string
	var r []string
	parser := func(s string) (interface{}, error) { return s, nil }

	form := func(p *string, opt bool) *formal {
		return &formal{min: -1, opt: opt, parser: parser, field: reflect.ValueOf(p).Elem()}
	}
	req := func(p *string) *formal { return form(p, false) }
	opt := func(p *string) *formal { return form(p, true) }
	rest := func(min int, p *[]string) *formal {
		return &formal{name: "r", min: min, parser: parser, field: reflect.ValueOf(p).Elem()}
	}

	for _, test := range []struct {
		name    string
		formals []*formal
		args    []string
		want    func() bool
		wantErr string
	}{
		{
			name: "empty",
		},
		{
			name:    "required",
			formals: []*formal{req(&f1), req(&f2)},
			args:    []string{"a", "b"},
			want:    func() bool { return f1 == "a" && f2 == "b" },
		},
		{
			name:    "required too few",
			formals: []*formal{req(&f1), req(&f2)},
			args:    []string{"a"},
			wantErr: "too few",
		},
		{
			name:    "required too many",
			formals: []*formal{req(&f1)},
			args:    []string{"a", "b"},
			wantErr: "too many",
		},
		{
			name:    "min 0 none",
			formals: []*formal{req(&f1), rest(0, &r)},
			args:    []string{"a"},
			want:    func() bool { return f1 == "a" && len(r) == 0 },
		},
		{
			name:    "min 0 two",
			formals: []*formal{req(&f1), rest(0, &r)},
			args:    []string{"a", "b", "c"},
			want: func() bool {
				return f1 == "a" && reflect.DeepEqual(r, []string{"b", "c"})
			},
		},
		{
			name:    "min 1 none",
			formals: []*formal{req(&f1), rest(1, &r)},
			args:    []string{"a"},
			wantErr: "at least 1",
		},
		{
			name:    "min 1 one",
			formals: []*formal{req(&f1), rest(1, &r)},
			args:    []string{"a", "b"},
			want: func() bool {
				return f1 == "a" && reflect.DeepEqual(r, []string{"b"})
			},
		},
		{
			name:    "opt absent",
			formals: []*formal{req(&f1), opt(&f2), req(&f3)},
			args:    []string{"a"},
			want:    func() bool { return f1 == "a" && f2 == "" && f3 == "" },
		},
		{
			name:    "opt present",
			formals: []*formal{req(&f1), opt(&f2), req(&f3)},
			args:    []string{"a", "b", "c"},
			want:    func() bool { return f1 == "a" && f2 == "b" && f3 == "c" },
		},
		{
			name:    "opt some",
			formals: []*formal{req(&f1), opt(&f2), req(&f3)},
			args:    []string{"a", "b"},
			wantErr: "too few",
		},
		{
			name:    "opt rest", // e.g. tsranks SPEC [TERM PACKAGE1 PACKAGE2 ...]
			formals: []*formal{req(&f1), opt(&f2), rest(1, &r)},
			args:    []string{"spec"},
			want:    func() bool { return f1 == "spec" && f2 == "" && r == nil },
		},
		{
			name:    "opt rest 2",
			formals: []*formal{req(&f1), opt(&f2), rest(1, &r)},
			args:    []string{"spec", "term"},
			wantErr: "at least 1",
		},
		{
			name:    "opt rest 3",
			formals: []*formal{req(&f1), opt(&f2), rest(1, &r)},
			args:    []string{"spec", "term", "p1"},
			want: func() bool {
				return f1 == "spec" && f2 == "term" && reflect.DeepEqual(r, []string{"p1"})
			},
		},
		{
			name:    "opt rest 4",
			formals: []*formal{req(&f1), opt(&f2), rest(1, &r)},
			args:    []string{"spec", "term", "p1", "p2"},
			want: func() bool {
				return f1 == "spec" && f2 == "term" &&
					reflect.DeepEqual(r, []string{"p1", "p2"})
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			f1 = ""
			f2 = ""
			f3 = ""
			r = nil
			got := bindFormals(test.formals, test.args)
			if got == nil && test.wantErr != "" {
				t.Error("got no error, wanted one")
			} else if got != nil && test.wantErr == "" {
				t.Errorf("got %q, wanted no error", got)
			} else if got != nil && !strings.Contains(got.Error(), test.wantErr) {
				t.Errorf("got %q, wanted error containing %q", got, test.wantErr)
			} else if test.want != nil && !test.want() {
				t.Error("want function returned false")
			}
		})
	}
}
