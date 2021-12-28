// Copyright 2021 Jonathan Amsterdam.

package cli

// Methods for github.com/posener/complete.Completer.

func (c *Command) SubCmdList() []string {
	var names []string
	for _, s := range c.subs {
		names = append(names, s.Name)
	}
	return names
}
