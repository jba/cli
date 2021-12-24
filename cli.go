// Copyright 2021 Jonathan Amsterdam.

//TODO:
// global flag.Usage
// print global flags in Usage
// trim whitespace from command doc
// distinguish -h,-help and exit 0
// sub-commands

package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
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
	nFlags  int
	formals []*formal
	mu      sync.Mutex
	subs    []*Cmd
}

var topCmd *Cmd = &Cmd{
	name:  filepath.Base(os.Args[0]),
	flags: flag.CommandLine,
}

func (c *Cmd) Register(name string, co Command, doc string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.find(name) != nil {
		panic(fmt.Sprintf("duplicate command: %q", name))
	}
	cmd := newCmd(name, co, doc)
	if err := cmd.processFields(co); err != nil {
		panic(err)
	}
	c.subs = append(c.subs, cmd)
}

func Register(name string, c Command, doc string) {
	topCmd.Register(name, c, doc)
}

// c.mu should be held.
func (c *Cmd) find(name string) *Cmd {
	for _, c := range c.subs {
		if c.name == name {
			return c
		}
	}
	return nil
}

func Main() {
	flag.Usage = func() {
		Usage(flag.CommandLine.Output())
	}
	flag.Parse()
	Run(context.Background(), os.Args[1:])
}

func Run(ctx context.Context, args []string) {
	if len(args) == 0 {
		Usage(os.Stderr)
		os.Exit(2)
	}
	topCmd.mu.Lock()
	c := topCmd.find(args[0])
	topCmd.mu.Unlock()
	if c == nil {
		fmt.Fprintf(os.Stderr, "unknown command: %q\n", args[0])
		Usage(os.Stderr)
		os.Exit(2)
	}
	if err := c.run(ctx, args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		var uerr *UsageError
		if errors.As(err, &uerr) {
			c.flags.Usage()
			os.Exit(2)
		}
		os.Exit(1)
	}
}

func Usage(w io.Writer) {
	fmt.Fprintf(w, "Usage of %s:\n", topCmd.name)
	topCmd.mu.Lock()
	cs := topCmd.subs
	topCmd.mu.Unlock()
	for _, c := range cs {
		c.usage(w, false)
	}
}

func (c *Cmd) usage(w io.Writer, single bool) {
	h := c.usageHeader()
	if single && len(h)+len(c.doc) <= 76 {
		fmt.Fprintf(w, "%s    %s\n", h, c.doc)
	} else {
		fmt.Fprintf(w, "%s\n  %s\n", h, c.doc)
	}
	for _, f := range c.formals {
		fmt.Fprintf(w, "  %-10s %s\n", f.name, f.doc)
	}
	c.flags.SetOutput(w)
	c.flags.PrintDefaults()
	fmt.Fprintln(w)
}

func (c *Cmd) usageHeader() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s", c.name)
	if c.nFlags > 0 {
		fmt.Fprint(&b, " [flags]")
	}
	for _, f := range c.formals {
		fmt.Fprintf(&b, " %s", f.name)
		if f.min >= 0 {
			fmt.Fprint(&b, "...")
		}
	}
	return b.String()
}

func newCmd(name string, c Command, doc string) *Cmd {
	cmd := &Cmd{
		name:  name,
		c:     c,
		doc:   doc,
		flags: flag.NewFlagSet(name, flag.ExitOnError),
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
		if tag == "" {
			continue
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

// A tag representing an argument is most simply
// just the doc for that arg.
// It can also start with some options:
// - name=xyz, which will use xyz for the name in the usage doc.
// - flag=f, which makes a flag named f
// - oneof=a|b|c, which which validate that the arg is one of those strings.
// A full example:
//   Env `cli:"name=env oneof=dev|prod  development environment"`
func (c *Cmd) parseTag(tag string, sf reflect.StructField, field reflect.Value) error {
	if !sf.IsExported() {
		return errors.New("cli tag on unexported field")
	}
	m := tagToMap(tag)
	for k := range m {
		if k == "" {
			return errors.New("empty key")
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
		c.nFlags++
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
		// positional arg
		name := m["name"]
		if name == "" {
			name = strings.ToUpper(sf.Name)
		}
		f := &formal{
			name:   name,
			field:  field,
			doc:    m["doc"],
			min:    -1,
			parser: parser,
		}
		if sf.Type.Kind() == reflect.Slice {
			f.min = 0
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

type parseFunc func(string) (interface{}, error)

func buildParser(t reflect.Type, tagMap map[string]string) (parseFunc, error) {
	if t.Kind() == reflect.Slice {
		if _, isFlag := tagMap["flag"]; isFlag {
			return parserForSlice(t, tagMap, ",")
		}
		return parserForType(t.Elem(), tagMap)
	}
	return parserForType(t, tagMap)
}

func parserForSlice(t reflect.Type, tagMap map[string]string, sep string) (parseFunc, error) {
	elp, err := parserForType(t.Elem(), tagMap)
	if err != nil {
		return nil, err
	}
	return func(s string) (interface{}, error) {
		parts := strings.Split(s, sep)
		slice := reflect.MakeSlice(t, len(parts), len(parts))
		for i, p := range parts {
			el, err := elp(p)
			if err != nil {
				return nil, fmt.Errorf("%q: %v", p, err)
			}
			slice.Index(i).Set(reflect.ValueOf(el))
		}
		return slice.Interface(), nil
	}, nil
}

var durationType = reflect.TypeOf(time.Duration(0))

// scalar types only
func parserForType(t reflect.Type, tagMap map[string]string) (parseFunc, error) {
	if oneof, ok := tagMap["oneof"]; ok {
		fmt.Println("oneof")
		if oneof == "" {
			return nil, errors.New("empty oneof")
		}
		if t.Kind() != reflect.String {
			return nil, fmt.Errorf("oneof must be string type, not %s", t)
		}
		return parserForOneof(strings.Split(oneof, "|")), nil
	}
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

// UsageError is an error in how the command is invoked.
// If returned from Command.Run, then the usage message for
// the command will be printed in addition to the underlying error,
// and the process will exit with code 2.
type UsageError struct {
	err error
}

func (u *UsageError) Error() string {
	return u.err.Error()
}

func (u *UsageError) Unwrap() error {
	return u.err
}

func (c *Cmd) run(ctx context.Context, args []string) error {
	if err := c.bindArgs(args); err != nil {
		return &UsageError{err}
	}
	return c.c.Run(ctx)
}

func (c *Cmd) bindArgs(args []string) error {
	_ = c.flags.Parse(args)
	// Check number of args.
	nargs := c.flags.NArg()
	if len(c.formals) == 0 {
		if nargs > 0 {
			return &UsageError{errors.New("too many args")}
		}
	} else {
		min := c.formals[len(c.formals)-1].min
		if min == -1 {
			// no rest arg
			if nargs < len(c.formals) {
				return &UsageError{errors.New("too few args")}
			}
			if nargs > len(c.formals) {
				return &UsageError{errors.New("too many args")}
			}
		} else if nargs < len(c.formals)-1+min {
			return &UsageError{errors.New("too few args")}
		}
	}
	for i, f := range c.formals {
		if f.min >= 0 {
			// We've already checked that this is the last formal.
			nArgsLeft := c.flags.NArg() - i
			if nArgsLeft < f.min {
				return fmt.Errorf("%s: need at least %d args, got %d", f.name, f.min, nArgsLeft)
			}
			slice := reflect.MakeSlice(f.field.Type(), 0, nArgsLeft)
			for j := i; j < c.flags.NArg(); j++ {
				v, err := f.parser(c.flags.Arg(j))
				if err != nil {
					return fmt.Errorf("%s: %v", f.name, err)
				}
				slice = reflect.Append(slice, reflect.ValueOf(v))
			}
			f.field.Set(slice)
		} else {
			v, err := f.parser(c.flags.Arg(i))
			if err != nil {
				return fmt.Errorf("%s: %v", f.name, err)
			}
			f.field.Set(reflect.ValueOf(v))
		}
	}

	return nil
}

// Cut cuts s around the first instance of sep,
// returning the text before and after sep.
// The found result reports whether sep appears in s.
// If sep does not appear in s, cut returns s, "", false.
//
// TODO: remove when go1.18 is out.
func stringsCut(s, sep string) (before, after string, found bool) {
	if i := strings.Index(s, sep); i >= 0 {
		return s[:i], s[i+len(sep):], true
	}
	return s, "", false
}
