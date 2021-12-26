// Copyright 2021 Jonathan Amsterdam.

package cli

import (
	"errors"
	"flag"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

// Code to register and prepare commands.

// The second arg should can be a Command, or if not then it's a group.
func Register(name string, x interface{}, doc string) *Cmd {
	return topCmd.Register(name, x, doc)
}

// Register a sub-command or sub-group of c.
func (c *Cmd) Register(name string, x interface{}, doc string) *Cmd {
	cmd, err := c.register(name, x, doc)
	if err != nil {
		panic(err)
	}
	return cmd
}

func (c *Cmd) register(name string, x interface{}, doc string) (*Cmd, error) {
	if len(c.formals) > 0 {
		return nil, fmt.Errorf("%s: a command cannot have both arguments and sub-commands", c.name)
	}
	if c.findSub(name) != nil {
		return nil, fmt.Errorf("duplicate sub-command: %q", name)
	}
	cmd := newCmd(name, x, strings.TrimSpace(doc))
	if err := cmd.processFields(x); err != nil {
		return nil, err
	}
	c.subs = append(c.subs, cmd)
	return cmd, nil
}

func (c *Cmd) findSub(name string) *Cmd {
	for _, c := range c.subs {
		if c.name == name {
			return c
		}
	}
	return nil
}

func newCmd(name string, x interface{}, doc string) *Cmd {
	cmd := &Cmd{
		name:  name,
		c:     x,
		doc:   doc,
		flags: flag.NewFlagSet(name, flag.ContinueOnError),
	}
	cmd.flags.Usage = func() {
		fmt.Fprintln(cmd.flags.Output(), "Usage:")
		cmd.usage(cmd.flags.Output(), true)
	}
	return cmd
}

type formal struct {
	name   string        // display name
	field  reflect.Value // "pointer" to field to set
	doc    string
	min    int       // for last slice, minimum args needed
	opt    bool      // if true, this and all following formals are optional
	parser parseFunc // convert and/or validate
}

func (c *Cmd) processFields(x interface{}) error {
	v := reflect.ValueOf(x)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("%T is not a pointer to a struct", x)
	}
	v = v.Elem()
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		tag := f.Tag.Get("cli")
		if tag == "" {
			// If the "cli" key is missing, assume the entire tag is a cli spec,
			// for convenience.
			tag = string(f.Tag)
		}
		if err := c.parseTag(tag, f, v.Field(i)); err != nil {
			return fmt.Errorf("command %q, field %q: %v", c.name, f.Name, err)
		}
	}
	for i, f := range c.formals {
		if f.min >= 0 && i != len(c.formals)-1 {
			return fmt.Errorf("%q is a slice but not the last arg", f.name)
		}
	}
	return nil
}

var validKeys = map[string]bool{
	"flag":  true,
	"name":  true,
	"min":   true,
	"oneof": true,
	"doc":   true,
	"opt":   true,
}

// A tag representing an argument is most simply
// just the doc for that arg.
// It can also start with some options:
// - name=xyz, which will use xyz for the name in the usage doc.
// - flag=f, which makes a flag named f
// - oneof=a|b|c, which which validate that the arg is one of those strings.
// A full example:
//   Env `cli:"name=env, oneof=dev|prod, development environment"`
func (c *Cmd) parseTag(tag string, sf reflect.StructField, field reflect.Value) error {
	if tag != "" && !sf.IsExported() {
		return errors.New("cli tag on unexported field")
	}
	m := tagToMap(tag)
	for k := range m {
		if k == "" {
			return errors.New("empty key")
		}
		if !validKeys[k] {
			return fmt.Errorf("invalid key: %q", k)
		}
	}
	if m["flag"] != "" && m["name"] != "" {
		return errors.New("either 'flag' or 'name', but not both")
	}

	parser, err := buildParser(field.Type(), m)
	if err != nil {
		return err
	}
	if fname, ok := m["flag"]; ok {
		// flag
		if fname == "" {
			fname = strings.ToLower(sf.Name)
		}
		if fname[0] == '-' {
			fname = fname[1:]
		}
		c.nFlags++
		if field.Kind() == reflect.Bool {
			ptr := field.Addr().Convert(reflect.PtrTo(reflect.TypeOf(true))).Interface().(*bool)
			c.flags.BoolVar(ptr, fname, *ptr, m["doc"])
		} else {
			usage := m["doc"]
			if !field.IsZero() {
				usage += fmt.Sprintf(" (default %v)", field)
			}
			c.flags.Func(fname, usage, func(s string) error {
				val, err := parser(s)
				if err != nil {
					return err
				}
				field.Set(reflect.ValueOf(val))
				return nil
			})
		}
	} else {
		// positional arg
		name := m["name"]
		if name == "" {
			name = strings.ToUpper(sf.Name)
		}
		optVal, opt := m["opt"]
		if optVal != "" {
			return errors.New(`"opt" should not have a value`)
		}
		f := &formal{
			name:   name,
			field:  field,
			doc:    m["doc"],
			min:    -1,
			opt:    opt,
			parser: parser,
		}
		minTag, hasMinTag := m["min"]
		if sf.Type.Kind() == reflect.Slice {
			f.min = 0
			if hasMinTag {
				min, err := strconv.Atoi(minTag)
				if err != nil {
					return fmt.Errorf("min: %w", err)
				}
				if min < 0 {
					return errors.New("min cannot be negative")
				}
				f.min = min
			}
		} else if hasMinTag {
			return errors.New("min is only for slice args")
		}
		c.formals = append(c.formals, f)
	}
	return nil
}

var keyRegexp = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9]+=`)

func tagToMap(tag string) map[string]string {
	m := map[string]string{}
	tag = strings.TrimSpace(tag)
	for len(tag) > 0 {
		loc := keyRegexp.FindStringIndex(tag)
		if loc == nil {
			m["doc"] = tag
			break
		}
		key := tag[:loc[1]-1]
		tag = tag[loc[1]:]
		before, after, found := stringsCut(tag, ",")
		var value string
		if !found {
			value = tag
			tag = ""
		} else {
			value = before
			tag = strings.TrimSpace(after)
		}
		m[key] = strings.TrimSpace(value)
	}
	return m
}
