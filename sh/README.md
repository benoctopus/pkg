# Command Builder Package

A flexible command builder and runner abstraction that implements the Future pattern for asynchronous command execution.

## Features

- **Builder Pattern**: Fluent API for constructing commands with options and subcommands
- **Future Interface**: Asynchronous execution with cancellation support
- **Flexible Configuration**: Support for environment variables, working directory, and I/O redirection
- **Subcommands**: Support for complex command structures like `git status --short`

## Usage

### Basic Command Execution

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/benoctopus/pkg/cmdbuilder"
)

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    // Build and execute a simple command
    cmd := cmdbuilder.New("echo").
        WithEnv("MY_VAR", "hello").
        Build(ctx)

    // Start execution
    cmd.Start()

    // Wait for completion
    result, err := cmd.Wait()
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Exit Code: %d\n", result.ExitCode())
    fmt.Printf("Output: %s\n", string(result.Stdout()))
}
```

### Command with Options and Arguments

```go
// Build a command with flags and values
cmd := cmdbuilder.New("ls").
    OptB("-l").                    // Boolean flag
    OptV("--color", "always").     // Flag with value
    WithDir("/tmp").               // Set working directory
    Build(ctx)

cmd.Start()
result, err := cmd.Wait()
```

### Subcommands

```go
// Build git-like subcommands
builder := cmdbuilder.New("git")
subCmd := builder.SubCommand("status").
    OptB("--short").
    OptB("--porcelain")

cmd := subCmd.Build(ctx)
cmd.Start()
result, err := cmd.Wait()
```

### Asynchronous Execution with Cancellation

```go
ctx := context.Background()

// Start a long-running command
cmd := cmdbuilder.New("sleep").Build(ctx)
cmdImpl := cmd.(*cmdbuilder.cmdImpl)
cmdImpl.args = []string{"30"}  // Sleep for 30 seconds

cmd.Start()

// Cancel after 5 seconds
go func() {
    time.Sleep(5 * time.Second)
    cmd.Cancel()
}()

// This will return quickly due to cancellation
result, err := cmd.Wait()
if err != nil {
    fmt.Println("Command was cancelled:", err)
}
```

### I/O Redirection

```go
var stdout, stderr bytes.Buffer

cmd := cmdbuilder.New("some-command").
    WithStdout(&stdout).
    WithStderr(&stderr).
    WithStdin(strings.NewReader("input data")).
    Build(ctx)

cmd.Start()
result, err := cmd.Wait()

// stdout and stderr buffers now contain the output
```

## API Reference

### Builder Methods

- `New(cmd string) *Builder` - Create a new command builder
- `OptB(flag string) *Builder` - Add a boolean flag
- `OptV(flag, value string) *Builder` - Add a flag with value
- `SubCommand(name string) *SubCmd` - Create a subcommand
- `WithEnv(key, value string) *Builder` - Set environment variable
- `WithDir(dir string) *Builder` - Set working directory
- `WithStdout(w io.Writer) *Builder` - Set stdout writer
- `WithStderr(w io.Writer) *Builder` - Set stderr writer
- `WithStdin(r io.Reader) *Builder` - Set stdin reader
- `Build(ctx context.Context) Cmd` - Build the final command

### Cmd Interface (Future[Result])

- `Start()` - Start command execution
- `Cancel()` - Cancel the running command
- `Wait() (Result, error)` - Wait for completion and get result
- `Done() chan any` - Get completion channel
- `IsDone() bool` - Check if command is complete

### Result Interface

- `ExitCode() int` - Get command exit code
- `Stdout() []byte` - Get stdout output
- `Stderr() []byte` - Get stderr output

## Implementation Details

The package uses Go's `os/exec` package internally and implements the Future pattern for asynchronous execution. Commands are executed in separate goroutines with proper synchronization and cancellation support through Go's context package.
