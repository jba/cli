// Copyright 2021 Jonathan Amsterdam.

package cli

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// Parsers for arguments and flag values.

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
	for i := range choices {
		choices[i] = strings.TrimSpace(choices[i])
	}
	return func(s string) (interface{}, error) {
		for _, c := range choices {
			if s == c {
				return c, nil
			}
		}
		return nil, fmt.Errorf("must be one of: %s", strings.Join(choices, ", "))
	}
}
