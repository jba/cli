// Copyright 2021 Jonathan Amsterdam.

package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Command interface {
	Run(ctx context.Context) error
}

type Cmd struct {
	name    string
	c       Command
	doc     string
	flags   *flag.FlagSet
	formals []*formal
	subs    []*Cmd
}

func (c *Cmd) Register(name string, co Command) {
	panic("unimp")
}

var (
	mu   sync.Mutex
	cmds []*Cmd
)

func findCmd(name string) *Cmd {
	for _, c := range cmds {
		if c.name == name {
			return c
		}
	}
	return nil
}

func Register(name string, c Command, doc string) {
	mu.Lock()
	defer mu.Unlock()
	if findCmd(name) != nil {
		panic(fmt.Sprintf("duplicate command: %q", name))
	}
	cmd := &Cmd{
		name:  name,
		c:     c,
		doc:   doc,
		flags: flag.NewFlagSet(name, flag.ExitOnError),
	}
	if err := cmd.processFields(c); err != nil {
		panic(err)
	}
	cmds = append(cmds, cmd)
}

type formal struct {
	name   string        // display name
	field  reflect.Value // "pointer" to field to set
	doc    string
	min    int                               // for last slice, minimum args needed
	parser func(string) (interface{}, error) // convert and/or validate
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
		if err := c.parseTag(tag, f.Name, v.Field(i)); err != nil {
			return fmt.Errorf("command %s, field %s: %v", c.name, f.Name, err)
		}
	}
	return nil
}

// A tag representing an argument is most simply
// just the doc for that arg.
// It can also start with some options:
// - name=xyz, which will use xyz for the name in the usage doc.
// - flag=f, which makes a flag named f
// - oneof=a|b|c, which which validate that the arg is one of those strings.
// A full example:
//   Env `cli:"name=env oneof=dev|prod  development environment"`
func (c *Cmd) parseTag(tag, fieldName string, field reflect.Value) error {
	m := tagToMap(tag)
	for k := range m {
		if k == "" {
			return errors.New("empty key")
		}
	}
	if m["flag"] != "" && m["name"] != "" {
		return errors.New("either 'flag' or 'name', but not both")
	}
	parser, err := parserForType(field.Type())
	if err != nil {
		return err
	}
	if of, ok := m["oneof"]; ok {
		if of == "" {
			return errors.New("empty value for oneof")
		}
		op := parserForOneof(strings.Split(of, "|"))
		parser = func(s string) (interface{}, error) {
			x, err := op(s)
			if err != nil {
				return nil, err
			}
			return parser(x.(string))
		}
	}
	if fname, ok := m["flag"]; ok {
		if fname == "" {
			fname = strings.ToLower(fieldName)
		}
		if field.Kind() == reflect.Bool {
			ptr := field.Addr().Convert(reflect.PtrTo(reflect.TypeOf(true))).Interface().(*bool)
			c.flags.BoolVar(ptr, fname, *ptr, m["doc"])
		} else {
			c.flags.Func(fname, m["doc"], func(s string) error {
				val, err := parser(s)
				if err != nil {
					return err
				}
				field.Set(reflect.ValueOf(val))
				return nil
			})
		}
	} else {
		name := m["name"]
		if name == "" {
			name = strings.ToLower(fieldName)
		}
		f := &formal{
			name:   name,
			field:  field,
			doc:    m["doc"],
			min:    -1,
			parser: parser,
		}
		c.formals = append(c.formals, f)
	}
	return nil
}

func tagToMap(tag string) map[string]string {
	m := map[string]string{}
	tag = strings.TrimSpace(tag)
	for len(tag) > 0 {
		before, after, found := stringsCut(tag, ",")
		if !found {
			m["doc"] = tag
			break
		}
		bef, aft, found := stringsCut(before, "=")
		if !found {
			m["doc"] = tag
			break
		}
		m[strings.TrimSpace(bef)] = strings.TrimSpace(aft)
		tag = strings.TrimSpace(after)
	}
	return m
}

type parseFunc func(string) (interface{}, error)

func parserForOneof(choices []string) parseFunc {
	return func(s string) (interface{}, error) {
		for _, c := range choices {
			if s == c {
				return c, nil
			}
		}
		return nil, fmt.Errorf("must be one of: %s", strings.Join(choices, ", "))
	}
}

var durationType = reflect.TypeOf(time.Duration(0))

func parserForType(t reflect.Type) (parseFunc, error) {
	if t == durationType {
		return func(s string) (interface{}, error) {
			return time.ParseDuration(s)
		}, nil
	}

	convert := func(v interface{}) interface{} {
		return reflect.ValueOf(v).Convert(t).Interface()
	}

	switch t.Kind() {
	case reflect.String:
		return func(s string) (interface{}, error) {
			return convert(s), nil
		}, nil
	case reflect.Bool:
		return func(s string) (interface{}, error) {
			b, err := strconv.ParseBool(s)
			if err != nil {
				return nil, err
			}
			return convert(b), nil
		}, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return func(s string) (interface{}, error) {
			i, err := strconv.ParseInt(s, 10, 64)
			if err != nil {
				return nil, err
			}
			return convert(i), nil
		}, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return func(s string) (interface{}, error) {
			u, err := strconv.ParseUint(s, 10, 64)
			if err != nil {
				return nil, err
			}
			return convert(u), nil
		}, nil
	case reflect.Float32, reflect.Float64:
		return func(s string) (interface{}, error) {
			f, err := strconv.ParseFloat(s, 64)
			if err != nil {
				return nil, err
			}
			return convert(f), nil
		}, nil
	default:
		return nil, fmt.Errorf("cannot parse string into %s", t)
	}
}

type usageError struct {
	err error
}

func (u *usageError) Error() string {
	return u.err.Error()
}

func (u *usageError) Unwrap() error {
	return u.err
}

func (c *Cmd) run(ctx context.Context, args []string) error {
	if err := c.bindArgs(args); err != nil {
		return &usageError{err}
	}
	return c.c.Run(ctx)
}

func (c *Cmd) bindArgs(args []string) error {
	_ = c.flags.Parse(args)
	// Check number of args.
	nargs := c.flags.NArg()
	if len(c.formals) == 0 {
		if nargs > 0 {
			return &usageError{errors.New("too many args")}
		}
	} else {
		min := c.formals[len(c.formals)-1].min
		if min == -1 {
			// no rest arg
			if nargs != len(c.formals) {
				return &usageError{errors.New("wrong number of args")}
			}
		} else if nargs < len(c.formals)-1+min {
			return &usageError{errors.New("too few args")}
		}
	}
	for i, f := range c.formals {
		v, err := f.parser(c.flags.Arg(i))
		if err != nil {
			return fmt.Errorf("%s: %v", f.name, err)
		}
		f.field.Set(reflect.ValueOf(v))
	}

	return nil
}

// Cut cuts s around the first instance of sep,
// returning the text before and after sep.
// The found result reports whether sep appears in s.
// If sep does not appear in s, cut returns s, "", false.
//
// https://golang.org/issue/46336 is an accepted proposal to add this to the
// standard library. It will presumably land in Go 1.18, so this can be removed
// when pkgsite moves to that version.
func stringsCut(s, sep string) (before, after string, found bool) {
	if i := strings.Index(s, sep); i >= 0 {
		return s[:i], s[i+len(sep):], true
	}
	return s, "", false
}

func Main() {
	if err := Run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func Run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return errors.New("no args")
	}
	c := findCmd(args[0])
	if c == nil {
		return fmt.Errorf("unknown command: %q", args[0])
	}
	return c.run(ctx, args[1:])
}
