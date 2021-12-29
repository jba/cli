// Copyright 2021 Jonathan Amsterdam.

package cli

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

// Registering and preparing commands.

// Top prepares its argument to be the top-level command of a program,
// then returns it.
func Top(c *Command) *Command {
	if c == nil {
		c = &Command{}
	}
	if c.Name == "" {
		c.Name = filepath.Base(os.Args[0])
	}
	c.flags = flag.CommandLine
	flag.CommandLine.Init(flag.CommandLine.Name(), flag.ContinueOnError)
	flag.Usage = func() {
		c.usage(c.flags.Output(), true)
	}
	c.processFields()
	return c
}

// Command constructs a Command with the Name, Struct and Usage fields populated,
// then calls Register.
func (c *Command) Command(name string, str interface{}, usage string) *Command {
	return c.Register(&Command{
		Name:   name,
		Struct: str,
		Usage:  usage,
	})
}

// Register registers a sub-command of the receiver Command.
func (c *Command) Register(sub *Command) *Command {
	if err := c.register(sub); err != nil {
		panic(err)
	}
	return sub
}

func initFlags(c *Command) *Command {
	c.flags = flag.NewFlagSet(c.Name, flag.ContinueOnError)
	c.flags.Usage = func() {
		c.usage(c.flags.Output(), true)
	}
	return c
}

func (c *Command) register(sub *Command) error {
	if sub.Name == "" {
		return fmt.Errorf("sub-command of %s has no name", c.Name)
	}
	initFlags(sub)
	if len(c.formals) > 0 {
		return fmt.Errorf("%s: a command cannot have both arguments and sub-commands", c.Name)
	}
	if c.findSub(sub.Name) != nil {
		return fmt.Errorf("duplicate sub-command: %q", sub.Name)
	}
	if err := sub.processFields(); err != nil {
		return err
	}
	c.subs = append(c.subs, sub)
	sub.super = c
	return nil
}

func (c *Command) findSub(name string) *Command {
	for _, c := range c.subs {
		if c.Name == name {
			return c
		}
	}
	return nil
}

func (c *Command) processFields() error {
	if c.Struct == nil {
		return nil
	}
	v := reflect.ValueOf(c.Struct)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("%s.Struct: %T is not a pointer to a struct", c.Name, c.Struct)
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
			return fmt.Errorf("command %q, field %q: %v", c.Name, f.Name, err)
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
func (c *Command) parseTag(tag string, sf reflect.StructField, field reflect.Value) error {
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
	_, isFlag := m["flag"]
	if isFlag && m["name"] != "" {
		return errors.New("either 'flag' or 'name', but not both")
	}
	if _, isOpt := m["opt"]; isOpt && isFlag {
		return errors.New("either 'flag' or 'opt', but not both")
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
			usage:  m["doc"],
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
