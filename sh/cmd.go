package sh

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"sync"

	"github.com/benoctopus/pkg/future"
)

var (
	stdout io.Writer = os.Stdout
	stderr io.Writer = os.Stderr
	stdin  io.Reader = os.Stdin
)

// SetDefaultStdout sets the default stdout writer for all commands.
// If w is nil, the default stdout is not changed.
func SetDefaultStdout(w io.Writer) {
	if w != nil {
		stdout = w
	}
}

// SetDefaultStderr sets the default stderr writer for all commands.
// If w is nil, the default stderr is not changed.
func SetDefaultStderr(w io.Writer) {
	if w != nil {
		stderr = w
	}
}

// SetDefaultStdin sets the default stdin reader for all commands.
// If r is nil, the default stdin is not changed.
func SetDefaultStdin(r io.Reader) {
	if r != nil {
		stdin = r
	}
}

// Cmd represents a command that can be executed asynchronously.
// It implements the Future pattern for non-blocking command execution
// and provides methods for I/O redirection and command piping.
type Cmd interface {
	future.Future[Result]
	// WithStderr adds an additional stderr writer to the command.
	// The writer will receive stderr output in addition to any existing writers.
	WithStderr(stderr io.Writer) Cmd
	// WithStdout adds an additional stdout writer to the command.
	// The writer will receive stdout output in addition to any existing writers.
	WithStdout(stdout io.Writer) Cmd
	// WithStdin sets the stdin reader for the command.
	WithStdin(stdin io.Reader) Cmd
	// WithEnv sets an environment variable for the command.
	WithEnv(key, value string) Cmd
	// WithDir sets the working directory for the command.
	WithDir(dir string) Cmd
	// WithInteractive configures the command for interactive use with default I/O.
	WithInteractive() Cmd
	// Pipe creates a pipe builder that will pipe this command's stdout
	// to the stdin of the specified command.
	Pipe(cmd string) *PipeBuilder
	// Start begins the command execution asynchronously.
	// Returns a Future that can be used to wait for completion.
	Run() (Result, error)
}

// Context is an alias for context.Context for convenience.
type Context = context.Context

type cmdImpl struct {
	parent       Cmd
	cmd          string
	ctx          Context
	args         []string
	env          map[string]string
	stdoutBuffer *bytes.Buffer
	stderrBuffer *bytes.Buffer
	stdout       io.Writer
	stderr       io.Writer
	stdin        io.Reader
	dir          string

	// Future implementation fields
	result Result
	err    error
	done   chan any
	cancel context.CancelFunc
	once   sync.Once
	mu     sync.RWMutex
}

// PipeBuilder is used to construct command pipes where the output
// of one command becomes the input of another.
type PipeBuilder struct {
	from *cmdImpl
	*Builder
}

// Build constructs the piped command with the source command's stdout
// connected to this command's stdin.
func (pb *PipeBuilder) Build() Cmd {
	cm := pb.Builder.Build(pb.from.ctx).(*cmdImpl)
	cm.parent = pb.from

	return cm
}

// OptB adds a boolean flag to the pipe command and returns the PipeBuilder.
func (pb *PipeBuilder) OptB(flag string) *PipeBuilder {
	pb.Builder.OptB(flag)
	return pb
}

// OptV adds a flag with a value to the pipe command and returns the PipeBuilder.
func (pb *PipeBuilder) OptV(flag string, value any) *PipeBuilder {
	pb.Builder.OptV(flag, value)
	return pb
}

// Arg adds a positional argument to the pipe command and returns the PipeBuilder.
func (pb *PipeBuilder) Arg(value string) *PipeBuilder {
	pb.Builder.Arg(value)
	return pb
}

func (cm *cmdImpl) Pipe(cmd string) *PipeBuilder {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	b := New(cmd)

	return &PipeBuilder{
		from:    cm,
		Builder: b,
	}
}

func (cm *cmdImpl) WithStderr(stderr io.Writer) Cmd {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.stderr = io.MultiWriter(cm.stderr, stderr)
	return cm
}

func (cm *cmdImpl) WithStdout(stdout io.Writer) Cmd {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.stdout = io.MultiWriter(cm.stdout, stdout)
	return cm
}

func (cm *cmdImpl) WithStdin(stdin io.Reader) Cmd {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.stdin = stdin
	return cm
}

func (cm *cmdImpl) WithInteractive() Cmd {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.stdin = stdin
	// Use MultiWriter to write to both the buffer and the default stdout/stderr
	cm.stdout = io.MultiWriter(cm.stdoutBuffer, stdout)
	cm.stderr = io.MultiWriter(cm.stderrBuffer, stderr)
	return cm
}

func (cm *cmdImpl) WithDir(dir string) Cmd {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.dir = dir
	return cm
}

func (cm *cmdImpl) WithEnv(key, value string) Cmd {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	if cm.env == nil {
		cm.env = make(map[string]string)
	}
	cm.env[key] = value
	return cm
}

// Result represents the result of a command execution.
// It provides access to the exit code and captured output.
type Result interface {
	// ExitCode returns the exit code of the command.
	// Returns 0 for successful execution, non-zero for errors.
	ExitCode() int
	// Stdout returns the captured stdout output as bytes.
	Stdout() []byte
	// Stderr returns the captured stderr output as bytes.
	Stderr() []byte
}

type resultImpl struct {
	exitCode int
	stdout   []byte
	stderr   []byte
}

func (r *resultImpl) ExitCode() int {
	return r.exitCode
}

func (r *resultImpl) Stdout() []byte {
	return r.stdout
}

func (r *resultImpl) Stderr() []byte {
	return r.stderr
}

// ------------------------------------------- Future impl --------------------------------------

func (cm *cmdImpl) Start() future.Future[Result] {
	if cm.parent != nil {
		cm.parent.Start()
	}

	cm.once.Do(func() {
		go cm.execute()
	})

	return cm
}

func (cm *cmdImpl) Cancel() {
	if cm.parent != nil {
		cm.parent.Cancel()
	}

	if cm.cancel != nil {
		cm.cancel()
	}
}

func (cm *cmdImpl) Run() (Result, error) {
	cm.execute()
	return cm.result, cm.err
}

func (cm *cmdImpl) Wait() (Result, error) {
	cm.Start()
	if cm.parent != nil {
		r, err := cm.parent.Wait()
		if err != nil {
			return r, err
		}
	}

	<-cm.done
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.result, cm.err
}

func (cm *cmdImpl) Done() chan any {
	return cm.done
}

func (cm *cmdImpl) IsDone() bool {
	select {
	case <-cm.done:
		return true
	default:
		return false
	}
}

func (cm *cmdImpl) execute() {
	defer close(cm.done)

	// If this is a piped command, wait for parent to complete first
	if cm.parent != nil {
		_, err := cm.parent.Wait()
		if err != nil {
			cm.mu.Lock()
			cm.result = &resultImpl{exitCode: -1, stdout: []byte{}, stderr: []byte{}}
			cm.err = err
			cm.mu.Unlock()
			return
		}
		// Now set stdin to parent's stdout buffer
		cm.stdin = cm.parent.(*cmdImpl).stdoutBuffer
	}

	cmd := exec.CommandContext(cm.ctx, cm.cmd, cm.args...)

	if cm.dir != "" {
		cmd.Dir = cm.dir
	}

	if len(cm.env) > 0 {
		env := make([]string, 0, len(cm.env))
		for k, v := range cm.env {
			env = append(env, k+"="+v)
		}
		cmd.Env = env
	}

	if cm.stdin != nil {
		cmd.Stdin = cm.stdin
	}

	// Set up output capture
	cmd.Stdout = cm.stdout
	cmd.Stderr = cm.stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			exitCode = -1
		}
	}

	result := &resultImpl{
		exitCode: exitCode,
		stdout:   cm.stdoutBuffer.Bytes(),
		stderr:   cm.stderrBuffer.Bytes(),
	}

	cm.mu.Lock()
	cm.result = result
	cm.err = err
	cm.mu.Unlock()
}
