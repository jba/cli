// Copyright 2021 Jonathan Amsterdam.

package cli

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// Parsers for arguments and flag values.

// parseFunc is the type of functions that parse argument or flag strings into values.
type parseFunc func(string) (interface{}, error)

// buildParser constructs a parser for type t, or for the list of choices.
func buildParser(t reflect.Type, choices []string, isFlag bool) (parseFunc, error) {
	if t.Kind() != reflect.Slice {
		return parserForType(t, choices)
	} else if isFlag {
		return parserForSlice(t, choices, ",")
	} else {
		return parserForType(t.Elem(), choices)
	}
}

// parserForSlice returns a parser for a string representing a slice of values.
// t is the slice type.
// sep separates elements in the string.
func parserForSlice(t reflect.Type, choices []string, sep string) (parseFunc, error) {
	elp, err := parserForType(t.Elem(), choices)
	if err != nil {
		return nil, err
	}
	return func(s string) (interface{}, error) {
		parts := strings.Split(s, sep)
		slice := reflect.MakeSlice(t, len(parts), len(parts))
		for i, p := range parts {
			p = strings.TrimSpace(p)
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

// parserForType returns a parser for scalar types.
func parserForType(t reflect.Type, choices []string) (parseFunc, error) {
	if choices != nil {
		if t.Kind() != reflect.String {
			return nil, fmt.Errorf("oneof must be string type, not %s", t)
		}
		return parserForOneof(choices), nil
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
		if err := checkOneof(s, choices); err != nil {
			return nil, err
		}
		return s, nil
	}
}
