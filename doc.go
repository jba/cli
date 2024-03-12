// Copyright 2021 Jonathan Amsterdam.

/*
Package cli helps to build command-line programs with multiple commands.
A command is created by defining a struct with a Run method. The exported
fields of the struct are populated with the flags and command-line arguments
passed on the command line, as determined by struct tags. For example,
here is a struct that could be used for a "compare" command that takes
two files and a "-v" flag:

	type compare struct {
	  Verbose bool         `cli:"flag=v, verbose output"`
	  File1, File2 string
	}

The command's logic is provided in a Run method:

	func (c *compare) Run(ctx context.Context) error {
	  return diff(c.Verbose, c.File1, c.File2)
	}

Before the Run method is called, the command line flags and arguments are parsed
and assigned to the fields of the receiver struct.

# Registration

All commands must be registered, usually at program startup. Begin with a
top-level command representing the program itself. Typically no behavior is
associated with the top-level command, so the line

	var top = cli.Top(nil)

suffices. The top command uses the default flag set of the standard library's
flag package, so global flags can be defined as usual. The value of top is a
[*cli.Command].

Sub-commands are configured by registering a [Command] struct with an existing
Command. Given the compare struct and the top variable shown above, we can
register a compare command with

	top.RegisterCommand(&cli.Command{
	  Name: "compare",
	  Struct: &compare{},
	  Usage: "compare two files",
	})

or more succinctly,

	top.Command("compare", &compare{}, "compare two files")

That code can be put in an init method or at the start of main.

The Top function takes a Command just like the RegisterCommand function, so you
can provide behavior for the top-level command by defining a struct with a Run
method, constructing a Command with it, and passing it to Top.

# Struct Tags

The struct associated with a command completely describes the command's flags
and positional arguments. Each exported field can have a struct tag with a "cli"
key that provides the usage documentation for the argument or flag as well as
some options. An exported field without a tag is treated as a positional
argument with no documentation. Unexported fields are ignored.

A field's type can be any string, bool, integer, floating point or duration
type, or a slice of one of those types. If the slice is used for a flag, the
flag's value is split on commas to populate the slice. Otherwise, the slice
field must represent the last positional argument, and its value is taken from
the remaining command-line arguments.

The tag syntax is a comma-separated lists of key=value pairs. The keys are:

  - flag:  The field is a flag. The value is the flag's name; if empty, the lower-cased
    field name is used.
  - name:  The value is the name of the positional argument, used in documentation.
    If empty, the upper-cased field name is used.
  - doc:   The value is the usage string. This key can be omitted when the usage string
    is last.
  - opt:   This and the following positional arguments are optional.
  - oneof: The value is a "|"-separated list of strings that the provided value
    must match. A field with "oneof" must be of type string.
  - min:   For positional slice fields, the minimum number of arguments.

For example, the field and struct tag

	Environ string `cli:"name=env, oneof=dev|prod, development environment"`

will display the argument as ENV in documentation along with the string
"development environment", and will check that the value on the command line is
either "dev" or "prod".

See the package examples for more.

The Go flag package provides control over the word printed as the flag's value in documentation,
by looking for backticks in the usage string. To use this feature, enclose the struct tag
in double quotes instead of backticks. As a convenience, this package can interpret
bare struct tags that don't have the usual 'key:"value"' format, which makes this cleaner:

	type cmd struct {
	  InFile string "flag=in, the input `filename`"
	}

# Execution

Once the top-level command has been created and all sub-commands have been
registered, call the Main method to invoke the appropriate command and get back
an exit code. If the other work has been done with a global "top" variable and
init functions, then the entire main function can be

	func main() {
	  os.Exit(top.Main(context.Background()))
	}

For more control, you can call Command.Run with a context and a slice of arguments,
and handle the error yourself.

# Completion

Shell completion for common shells is supported with the
github.com/posener/complete/v2 package. Completion logic is automatically
invoked if your program calls Command.Main. To install completion for a program,
run it with the COMP_INSTALL environment variable set to 1.
*/
package cli
