// Copyright 2021 Jonathan Amsterdam.

package cli

import (
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

func TestParseTagArgs(t *testing.T) {
	type s struct {
		F string
	}
	sf := reflect.TypeOf(s{}).Field(0)
	f := reflect.ValueOf(s{}).Field(0)
	for _, test := range []struct {
		tag      string
		wantName string
		wantDoc  string
		wantErr  bool
	}{
		{
			tag:      "just doc",
			wantName: "F",
			wantDoc:  "just doc",
		},
		{
			tag:      "name=foo, \tand then doc\t ",
			wantName: "foo",
			wantDoc:  "and then doc",
		},
		{
			tag:      "oneof=al|be|ga, some doc",
			wantName: "F",
			wantDoc:  "some doc; one of al, be, ga",
		},
		{
			tag:      "\toneof=ga | la,\tname=bar\t,doc\t",
			wantName: "bar",
			wantDoc:  "doc; one of ga, la",
		},
	} {
		c := initFlags(&Command{})
		err := c.parseTag(test.tag, sf, f)
		if err != nil {
			if !test.wantErr {
				t.Errorf("%q: unwanted error: <%v>", test.tag, err)
			}
		} else {
			got := c.formals[0]
			if got.name != test.wantName {
				t.Errorf("%q, name: got %q, want %q", test.tag, got.name, test.wantName)
			}
			if got.usage != test.wantDoc {
				t.Errorf("%q, doc: got %q, want %q", test.tag, got.usage, test.wantDoc)
			}
			gotp, err := got.parser("ga")
			if err != nil {
				t.Errorf("%q: unexpected parsing error: <%v>", test.tag, err)
			} else if wantp := "ga"; gotp != wantp {
				t.Errorf("%q: got %v, want %v", test.tag, gotp, wantp)
			}
		}
	}
}

func TestParseTagFlags(t *testing.T) {
	type s struct {
		F string
	}
	sf := reflect.TypeOf(s{}).Field(0)
	f := reflect.ValueOf(s{}).Field(0)
	for _, test := range []struct {
		tag      string
		wantName string
		wantDoc  string
		wantErr  bool
	}{
		{
			tag:      "flag=f, doc",
			wantName: "f",
			wantDoc:  "doc",
		},
		{
			tag:      "flag=Goo, more `doc`",
			wantName: "Goo",
			wantDoc:  "more `doc`",
		},
		{
			tag:      "flag=, oneof=a|b|c, something",
			wantName: "f",
			wantDoc:  "something; one of a, b, c",
		},
	} {
		c := initFlags(&Command{})
		err := c.parseTag(test.tag, sf, f)
		if err != nil {
			if !test.wantErr {
				t.Errorf("%q: unwanted error: <%v>", test.tag, err)
			}
			continue
		}
		got := c.flags.Lookup(test.wantName)
		if got == nil {
			t.Errorf("%q: no flag named %q", test.tag, test.wantName)
			continue
		}
		if got.Name != test.wantName {
			t.Errorf("%q, name: got %q, want %q", test.tag, got.Name, test.wantName)
		}
		if got.Usage != test.wantDoc {
			t.Errorf("%q, doc: got %q, want %q", test.tag, got.Usage, test.wantDoc)
		}
	}
}

func TestProcessFieldsErrors(t *testing.T) {
	check := func(s interface{}, want string) {
		t.Helper()
		got := (&Command{Struct: s}).processFields()
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
	t4cmd := &Command{Struct: &t4{}}
	if err := t4cmd.processFields(); err != nil {
		t.Fatal(err)
	}
	got := t4cmd.register(&Command{Name: "sub", Struct: nil})
	want := "is not runnable"
	if got == nil || !strings.Contains(got.Error(), want) {
		t.Errorf("got %v, want error containing %q", got, want)
	}

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
			c := initFlags(&Command{})
			got := c.bindFormals(test.formals, test.args)
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
func TestFlagUsage(t *testing.T) {

	type s struct {
		File string "flag=in, input `filename` for input"
	}
	cmd := Top(&Command{Struct: &s{}})
	f := cmd.flags.Lookup("in")
	if f == nil {
		t.Fatal("flag is nil")
	}
	got := f.Usage
	want := "input `filename` for input"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
