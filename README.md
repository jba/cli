# cli: A Command-Line Package

Why another package for simplifying command-line programs?

My typical command-line program is a development tool that doesn't require
much beyond some simple, in-program documentation. So I wanted something easy to
use and understand. I find many of the existing packages dauntingly complex.

I also had the idea of using struct tags to describe flags and arguments. This
virtually eliminates the painful boilerplate of checking the number of arguments
and then parsing the strings. Instead, one defines a struct with fields of the
desired types, and the struct's Run method has those fields at hand, assigned
and validated. I've only seen this idea in one other package,
[github.com/mkideal/cli](https://pkg.go.dev/github.com/mkideal/cli), but there
it is only applied to flags, not positional arguments.

## Features

- Define flags and arguments with struct field tags.

- Nested commands.

- Shell completion.

- Multi-valued flags: parsing comma-separated flag values into a slice.

- Oneofs: specify a list of valid strings for a flag or argument.
