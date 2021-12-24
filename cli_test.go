// Copyright 2021 Jonathan Amsterdam.

package cli

import (
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
