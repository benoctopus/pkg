package sh

import (
	"bytes"
	"context"
	"fmt"
)

type execResult struct{}

// CmdComponent represents a component that can be part of a command.
// Components can be options, arguments, or subcommands.
type CmdComponent interface {
	// Items returns the string representation of this component.
	Items() []string
	// Parent returns the parent component, if any.
	Parent() CmdComponent
}

// ---------------------------------------- cmd builder ------------------------------------

// Builder provides a fluent interface for constructing commands.
// It supports adding options, arguments, and subcommands.
type Builder struct {
	Cmd        string
	components []CmdComponent
}

// Items returns all command components as a slice of strings.
// The first item is always the command name, followed by all components.
func (b *Builder) Items() []string {
	if b == nil {
		return []string{}
	}

	items := make([]string, 0, len(b.components)*2+1)
	items = append(items, b.Cmd)

	for _, component := range b.components {
		items = append(items, component.Items()...)
	}
	return items
}

// Parent returns the parent builder, which is always nil for root builders.
func (b *Builder) Parent() *Builder {
	return nil
}

// SubCommand creates a subcommand builder with the specified name.
// The subcommand will be executed as part of the parent command.
// For example: git.SubCommand("status") creates "git status".
func (b *Builder) SubCommand(name string) *SubCmd {
	subCmd := &SubCmd{
		Builder: &Builder{Cmd: name},
		parent:  b,
	}

	return subCmd
}

// OptB adds a boolean flag to the command.
// Boolean flags are options that don't take a value, like "-v" or "--verbose".
func (s *Builder) OptB(flag string) *Builder {
	// Skip empty flags
	if flag == "" {
		return s
	}

	opt := &Opt{
		Key: flag,
	}

	s.components = append(s.components, opt)
	return s
}

// OptV adds a flag with a value to the command.
// For example: OptV("--output", "json") adds "--output json" to the command.
func (s *Builder) OptV(flag string, value any) *Builder {
	// Skip empty flags
	if flag == "" {
		return s
	}

	opt := &Opt{
		Key:   flag,
		Value: value,
	}

	s.components = append(s.components, opt)
	return s
}

// Arg adds a positional argument to the command.
// Arguments are added in the order they are specified.
func (s *Builder) Arg(value string) *Builder {
	arg := &Arg{
		Value: value,
	}

	s.components = append(s.components, arg)
	return s
}

// ------------------------------------------ sub commands --------------------------------------

// SubCmd represents a subcommand that is part of a larger command structure.
// For example, in "git status --short", "status" would be a SubCmd of "git".
type SubCmd struct {
	*Builder
	parent *Builder
}

// Items returns the complete command including the parent command and this subcommand.
func (s *SubCmd) Items() []string {
	if s == nil {
		return []string{}
	}
	// Get parent items and append only the subcommand components (not the subcommand name again)
	parentItems := s.parent.Items()
	subItems := make([]string, 0, len(s.components)*2+1)
	subItems = append(subItems, s.Cmd) // Add subcommand name

	for _, component := range s.components {
		subItems = append(subItems, component.Items()...)
	}

	result := make([]string, 0, len(parentItems)+len(subItems))
	result = append(result, parentItems...)
	result = append(result, subItems...)

	return result
}

// OptB adds a boolean flag to the subcommand and returns the SubCmd.
func (s *SubCmd) OptB(flag string) *SubCmd {
	s.Builder.OptB(flag)
	return s
}

// OptV adds a flag with a value to the subcommand and returns the SubCmd.
func (s *SubCmd) OptV(flag string, value any) *SubCmd {
	s.Builder.OptV(flag, value)
	return s
}

// Arg adds a positional argument to the subcommand and returns the SubCmd.
func (s *SubCmd) Arg(value string) *SubCmd {
	s.Builder.Arg(value)
	return s
}

// Parent returns the parent builder that this subcommand belongs to.
func (s *SubCmd) Parent() *Builder {
	return s.parent
}

// ----------------------------------------------- opt ------------------------------------------

// Opt represents a command-line option or flag.
// It can be a boolean flag (just a key) or a key-value pair.
type Opt struct {
	Key   string
	Value any
}

// Items returns the string representation of this option.
// For boolean flags, returns just the key. For key-value pairs,
// returns both the key and formatted value.
func (o *Opt) Items() []string {
	if o.Value != nil {
		return []string{o.Key, fmt.Sprintf("%v", o.Value)}
	}
	if o.Key == "" {
		return []string{}
	}

	return []string{o.Key}
}

// Parent returns the parent component, which is always nil for options.
func (o *Opt) Parent() CmdComponent {
	return nil
}

// ----------------------------------------------- arg ------------------------------------------

// Arg represents a positional argument to a command.
type Arg struct {
	Value string
}

// Items returns the argument value as a single-item slice.
func (o *Arg) Items() []string {
	return []string{o.Value}
}

// Parent returns the parent component, which is always nil for arguments.
func (o *Arg) Parent() CmdComponent {
	return nil
}

// ------------------------------------------ Build method ----------------------------------

// Build constructs a Cmd from the builder configuration.
// The returned Cmd can be started asynchronously and supports cancellation
// through the provided context.
func (b *Builder) Build(ctx context.Context) Cmd {
	args := b.Items()
	if len(args) == 0 {
		panic("no command specified")
	}

	cmd := args[0]
	cmdArgs := args[1:]

	childCtx, cancel := context.WithCancel(ctx)

	stdoutBuffer := bytes.NewBuffer(nil)
	stderrBuffer := bytes.NewBuffer(nil)

	return &cmdImpl{
		cmd:          cmd,
		ctx:          childCtx,
		args:         cmdArgs,
		env:          make(map[string]string),
		dir:          "",
		stdoutBuffer: stdoutBuffer,
		stderrBuffer: stderrBuffer,
		stdout:       stdoutBuffer,
		stderr:       stderrBuffer,
		stdin:        nil,
		done:         make(chan any),
		cancel:       cancel,
	}
}

// New creates a new command builder with the specified command name.
// The command name should be the executable to run, such as "ls", "git", or "echo".
func New(cmd string) *Builder {
	return &Builder{
		Cmd: cmd,
	}
}
