// Copyright 2021 Jonathan Amsterdam.

package cli

// Methods for github.com/posener/complete.Completer.

func (c *Cmd) SubCmdList() []string {
	var names []string
	for _, s := range c.subs {
		names = append(names, s.name)
	}
	return names
}
