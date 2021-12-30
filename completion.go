// Copyright 2021 Jonathan Amsterdam.

package cli

// Methods for github.com/posener/complete/v2.Completer.

import (
	"github.com/posener/complete/v2"
)

func (c *Command) SubCmdList() []string {
	var names []string
	for _, s := range c.subs {
		names = append(names, s.Name)
	}
	return names
}

func (c *Command) SubCmdGet(name string) complete.Completer {
	return c.findSub(name)
}

func (c *Command) FlagList() []string {
	return complete.FlagSet(c.flags).FlagList()
}

func (c *Command) FlagGet(flag string) complete.Predictor {
	return complete.FlagSet(c.flags).FlagGet(flag)
}

func (c *Command) ArgsGet() complete.Predictor {
	return nil
}
