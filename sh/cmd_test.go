package sh_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/benoctopus/pkg/sh"
)

func TestCmdBuilder(t *testing.T) {
	// Test basic command building and execution
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Build a simple echo command
	cmd := sh.New("echo").
		OptV("-n", ""). // no newline
		Build(ctx)

	// Start the command
	cmd.Start()

	// Wait for result
	result, err := cmd.Wait()
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	if result.ExitCode() != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode())
	}
}

func TestCmdBuilderWithArgs(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Build echo command with arguments using the proper API
	cmd := sh.New("echo").
		Arg("hello world").
		Build(ctx)

	cmd.Start()

	result, err := cmd.Wait()
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	if result.ExitCode() != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode())
	}

	output := strings.TrimSpace(string(result.Stdout()))
	if output != "hello world" {
		t.Errorf("Expected 'hello world', got '%s'", output)
	}
}

func TestCmdBuilderCancel(t *testing.T) {
	ctx := context.Background()

	// Build a long-running command (sleep) with proper API
	cmd := sh.New("sleep").
		Arg("10").
		Build(ctx)

	// Start the command
	cmd.Start()

	// Cancel immediately
	cmd.Cancel()

	// Wait should return quickly with context cancellation
	_, err := cmd.Wait()
	if err == nil {
		t.Error("Expected error due to cancellation")
	}
}

func TestSubCommand(t *testing.T) {
	// Test git-like subcommand structure
	builder := sh.New("git")
	subCmd := builder.SubCommand("status")
	subCmd.OptB("--short")

	items := subCmd.Items()
	expected := []string{"git", "status", "--short"}

	if len(items) != len(expected) {
		t.Fatalf("Expected %d items, got %d", len(expected), len(items))
	}

	for i, item := range items {
		if item != expected[i] {
			t.Errorf("Expected item %d to be '%s', got '%s'", i, expected[i], item)
		}
	}
}

func TestArgBuilder(t *testing.T) {
	// Test the Arg() method
	builder := sh.New("echo")
	builder.Arg("hello").Arg("world")

	items := builder.Items()
	expected := []string{"echo", "hello", "world"}

	if len(items) != len(expected) {
		t.Fatalf("Expected %d items, got %d", len(expected), len(items))
	}

	for i, item := range items {
		if item != expected[i] {
			t.Errorf("Expected item %d to be '%s', got '%s'", i, expected[i], item)
		}
	}
}

func TestCmdWithEnv(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test environment variable setting on Cmd interface
	cmd := sh.New("env").
		Build(ctx).
		WithEnv("CMD_TEST_VAR", "cmd_value")

	cmd.Start()
	result, err := cmd.Wait()
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	if result.ExitCode() != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode())
	}

	output := string(result.Stdout())
	if !strings.Contains(output, "CMD_TEST_VAR=cmd_value") {
		t.Errorf("Expected output to contain 'CMD_TEST_VAR=cmd_value', got: %s", output)
	}
}

func TestCmdWithDir(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test working directory setting
	cmd := sh.New("pwd").
		Build(ctx).
		WithDir("/tmp")

	cmd.Start()
	result, err := cmd.Wait()
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	if result.ExitCode() != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode())
	}

	output := strings.TrimSpace(string(result.Stdout()))
	if output != "/tmp" {
		t.Errorf("Expected '/tmp', got '%s'", output)
	}
}

func TestCmdWithStdout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var buf strings.Builder
	cmd := sh.New("echo").
		Arg("test output").
		Build(ctx).
		WithStdout(&buf)

	cmd.Start()
	result, err := cmd.Wait()
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	if result.ExitCode() != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode())
	}

	// Check that output was written to our buffer
	bufOutput := strings.TrimSpace(buf.String())
	if bufOutput != "test output" {
		t.Errorf("Expected 'test output' in buffer, got '%s'", bufOutput)
	}

	// Check that output is also available in result
	resultOutput := strings.TrimSpace(string(result.Stdout()))
	if resultOutput != "test output" {
		t.Errorf("Expected 'test output' in result, got '%s'", resultOutput)
	}
}

func TestCmdWithStderr(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var buf strings.Builder
	// Use a command that writes to stderr
	cmd := sh.New("sh").
		OptV("-c", "echo 'error message' >&2").
		Build(ctx).
		WithStderr(&buf)

	cmd.Start()
	result, err := cmd.Wait()
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	if result.ExitCode() != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode())
	}

	// Check that stderr was written to our buffer
	bufOutput := strings.TrimSpace(buf.String())
	if bufOutput != "error message" {
		t.Errorf("Expected 'error message' in buffer, got '%s'", bufOutput)
	}

	// Check that stderr is also available in result
	resultOutput := strings.TrimSpace(string(result.Stderr()))
	if resultOutput != "error message" {
		t.Errorf("Expected 'error message' in result, got '%s'", resultOutput)
	}
}

func TestCmdWithStdin(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	input := strings.NewReader("hello from stdin")
	cmd := sh.New("cat").
		Build(ctx).
		WithStdin(input)

	cmd.Start()
	result, err := cmd.Wait()
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	if result.ExitCode() != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode())
	}

	output := strings.TrimSpace(string(result.Stdout()))
	if output != "hello from stdin" {
		t.Errorf("Expected 'hello from stdin', got '%s'", output)
	}
}

func TestPipeBuilder(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create a pipe: echo "hello world" | wc -w
	echoCmd := sh.New("echo").
		Arg("hello world").
		Build(ctx)

	pipeBuilder := echoCmd.Pipe("wc")
	pipeCmd := pipeBuilder.OptB("-w").Build()

	pipeCmd.Start()
	result, err := pipeCmd.Wait()
	if err != nil {
		t.Fatalf("Pipe command failed: %v", err)
	}

	if result.ExitCode() != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode())
	}

	// wc -w should return "2" for "hello world"
	output := strings.TrimSpace(string(result.Stdout()))
	if output != "2" {
		t.Errorf("Expected '2', got '%s'", output)
	}
}

func TestCmdWithInteractive(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test WithInteractive method - this sets up the command to use default I/O
	cmd := sh.New("echo").
		Arg("interactive test").
		Build(ctx).
		WithInteractive()

	cmd.Start()
	result, err := cmd.Wait()
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	if result.ExitCode() != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode())
	}

	// The output should still be captured in the result
	output := strings.TrimSpace(string(result.Stdout()))
	if output != "interactive test" {
		t.Errorf("Expected 'interactive test', got '%s'", output)
	}
}

func TestDefaultIOSetters(t *testing.T) {
	// Test setting default I/O
	var stdoutBuf, stderrBuf strings.Builder
	stdinReader := strings.NewReader("test input")

	// Save original defaults
	originalStdout := sh.SetDefaultStdout
	originalStderr := sh.SetDefaultStderr
	originalStdin := sh.SetDefaultStdin

	// Set new defaults
	sh.SetDefaultStdout(&stdoutBuf)
	sh.SetDefaultStderr(&stderrBuf)
	sh.SetDefaultStdin(stdinReader)

	// Test with nil (should not change)
	sh.SetDefaultStdout(nil)
	sh.SetDefaultStderr(nil)
	sh.SetDefaultStdin(nil)

	// Restore original functions (not actual restoration, just for completeness)
	_ = originalStdout
	_ = originalStderr
	_ = originalStdin
}

func TestOptVWithDifferentTypes(t *testing.T) {
	// Test OptV with different value types
	builder := sh.New("test")
	builder.OptV("--string", "value")
	builder.OptV("--int", 42)
	builder.OptV("--float", 3.14)
	builder.OptV("--bool", true)

	items := builder.Items()
	expected := []string{
		"test",
		"--string",
		"value",
		"--int",
		"42",
		"--float",
		"3.14",
		"--bool",
		"true",
	}

	if len(items) != len(expected) {
		t.Fatalf("Expected %d items, got %d", len(expected), len(items))
	}

	for i, item := range items {
		if item != expected[i] {
			t.Errorf("Expected item %d to be '%s', got '%s'", i, expected[i], item)
		}
	}
}

func TestComplexSubCommand(t *testing.T) {
	// Test complex subcommand with multiple options
	builder := sh.New("git")
	subCmd := builder.SubCommand("log").
		OptB("--oneline").
		OptV("--max-count", 5).
		Arg("HEAD")

	items := subCmd.Items()
	expected := []string{"git", "log", "--oneline", "--max-count", "5", "HEAD"}

	if len(items) != len(expected) {
		t.Fatalf("Expected %d items, got %d", len(expected), len(items))
	}

	for i, item := range items {
		if item != expected[i] {
			t.Errorf("Expected item %d to be '%s', got '%s'", i, expected[i], item)
		}
	}
}

func TestEmptyOptHandling(t *testing.T) {
	// Test handling of empty options
	builder := sh.New("test")
	builder.OptV("", "should_be_ignored")
	builder.OptB("")

	items := builder.Items()
	// Should only contain the command name since empty options are ignored
	expected := []string{"test"}

	if len(items) != len(expected) {
		t.Fatalf("Expected %d items, got %d", len(expected), len(items))
	}

	for i, item := range items {
		if item != expected[i] {
			t.Errorf("Expected item %d to be '%s', got '%s'", i, expected[i], item)
		}
	}
}

// TestCmdRun tests the synchronous Run() method
func TestCmdRun(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test basic Run() execution
	cmd := sh.New("echo").
		Arg("hello run").
		Build(ctx)

	result, err := cmd.Run()
	if err != nil {
		t.Fatalf("Run() failed: %v", err)
	}

	if result.ExitCode() != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode())
	}

	output := strings.TrimSpace(string(result.Stdout()))
	if output != "hello run" {
		t.Errorf("Expected 'hello run', got '%s'", output)
	}
}

// TestCmdRunWithError tests Run() method with a failing command
func TestCmdRunWithError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Use a command that will fail
	cmd := sh.New("false").Build(ctx)

	result, err := cmd.Run()
	// err should not be nil for a failing command, but the result should still be available
	if err == nil {
		t.Error("Expected error from failing command")
	}

	if result.ExitCode() == 0 {
		t.Errorf("Expected non-zero exit code, got %d", result.ExitCode())
	}
}

// TestCmdRunWithStdout tests Run() method with stdout redirection
func TestCmdRunWithStdout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var buf strings.Builder
	cmd := sh.New("echo").
		Arg("run stdout test").
		Build(ctx).
		WithStdout(&buf)

	result, err := cmd.Run()
	if err != nil {
		t.Fatalf("Run() failed: %v", err)
	}

	if result.ExitCode() != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode())
	}

	// Check that output was written to our buffer
	bufOutput := strings.TrimSpace(buf.String())
	if bufOutput != "run stdout test" {
		t.Errorf("Expected 'run stdout test' in buffer, got '%s'", bufOutput)
	}

	// Check that output is also available in result
	resultOutput := strings.TrimSpace(string(result.Stdout()))
	if resultOutput != "run stdout test" {
		t.Errorf("Expected 'run stdout test' in result, got '%s'", resultOutput)
	}
}

// TestCmdRunWithStderr tests Run() method with stderr redirection
func TestCmdRunWithStderr(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var buf strings.Builder
	cmd := sh.New("sh").
		OptV("-c", "echo 'run stderr test' >&2").
		Build(ctx).
		WithStderr(&buf)

	result, err := cmd.Run()
	if err != nil {
		t.Fatalf("Run() failed: %v", err)
	}

	if result.ExitCode() != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode())
	}

	// Check that stderr was written to our buffer
	bufOutput := strings.TrimSpace(buf.String())
	if bufOutput != "run stderr test" {
		t.Errorf("Expected 'run stderr test' in buffer, got '%s'", bufOutput)
	}

	// Check that stderr is also available in result
	resultOutput := strings.TrimSpace(string(result.Stderr()))
	if resultOutput != "run stderr test" {
		t.Errorf("Expected 'run stderr test' in result, got '%s'", resultOutput)
	}
}

// TestCmdRunWithStdin tests Run() method with stdin input
func TestCmdRunWithStdin(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	input := strings.NewReader("run stdin test")
	cmd := sh.New("cat").
		Build(ctx).
		WithStdin(input)

	result, err := cmd.Run()
	if err != nil {
		t.Fatalf("Run() failed: %v", err)
	}

	if result.ExitCode() != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode())
	}

	output := strings.TrimSpace(string(result.Stdout()))
	if output != "run stdin test" {
		t.Errorf("Expected 'run stdin test', got '%s'", output)
	}
}

// TestCmdRunWithEnv tests Run() method with environment variables
func TestCmdRunWithEnv(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := sh.New("env").
		Build(ctx).
		WithEnv("RUN_TEST_VAR", "run_value")

	result, err := cmd.Run()
	if err != nil {
		t.Fatalf("Run() failed: %v", err)
	}

	if result.ExitCode() != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode())
	}

	output := string(result.Stdout())
	if !strings.Contains(output, "RUN_TEST_VAR=run_value") {
		t.Errorf("Expected output to contain 'RUN_TEST_VAR=run_value', got: %s", output)
	}
}

// TestCmdRunWithDir tests Run() method with working directory
func TestCmdRunWithDir(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := sh.New("pwd").
		Build(ctx).
		WithDir("/tmp")

	result, err := cmd.Run()
	if err != nil {
		t.Fatalf("Run() failed: %v", err)
	}

	if result.ExitCode() != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode())
	}

	output := strings.TrimSpace(string(result.Stdout()))
	if output != "/tmp" {
		t.Errorf("Expected '/tmp', got '%s'", output)
	}
}

// TestCmdRunWithInteractive tests Run() method with interactive mode
func TestCmdRunWithInteractive(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := sh.New("echo").
		Arg("run interactive test").
		Build(ctx).
		WithInteractive()

	result, err := cmd.Run()
	if err != nil {
		t.Fatalf("Run() failed: %v", err)
	}

	if result.ExitCode() != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode())
	}

	// The output should still be captured in the result
	output := strings.TrimSpace(string(result.Stdout()))
	if output != "run interactive test" {
		t.Errorf("Expected 'run interactive test', got '%s'", output)
	}
}

// TestCmdRunVsStartWait tests that Run() produces the same result as Start()/Wait()
func TestCmdRunVsStartWait(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test with Run()
	cmd1 := sh.New("echo").
		Arg("comparison test").
		Build(ctx)

	result1, err1 := cmd1.Run()
	if err1 != nil {
		t.Fatalf("Run() failed: %v", err1)
	}

	// Test with Start()/Wait()
	cmd2 := sh.New("echo").
		Arg("comparison test").
		Build(ctx)

	cmd2.Start()
	result2, err2 := cmd2.Wait()
	if err2 != nil {
		t.Fatalf("Start()/Wait() failed: %v", err2)
	}

	// Results should be identical
	if result1.ExitCode() != result2.ExitCode() {
		t.Errorf(
			"Exit codes differ: Run()=%d, Start()/Wait()=%d",
			result1.ExitCode(),
			result2.ExitCode(),
		)
	}

	output1 := string(result1.Stdout())
	output2 := string(result2.Stdout())
	if output1 != output2 {
		t.Errorf("Stdout differs: Run()='%s', Start()/Wait()='%s'", output1, output2)
	}

	stderr1 := string(result1.Stderr())
	stderr2 := string(result2.Stderr())
	if stderr1 != stderr2 {
		t.Errorf("Stderr differs: Run()='%s', Start()/Wait()='%s'", stderr1, stderr2)
	}
}

// TestCmdRunSynchronous tests that Run() is synchronous (blocks until completion)
func TestCmdRunSynchronous(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	start := time.Now()
	cmd := sh.New("sleep").
		Arg("1").
		Build(ctx)

	result, err := cmd.Run()
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Run() failed: %v", err)
	}

	if result.ExitCode() != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode())
	}

	// Should have taken at least 1 second (synchronous execution)
	if elapsed < time.Second {
		t.Errorf("Expected Run() to block for at least 1 second, but it took %v", elapsed)
	}
}

// TestCmdRunWithPipe tests Run() method with piped commands
func TestCmdRunWithPipe(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create a pipe: echo "hello world test" | wc -w
	echoCmd := sh.New("echo").
		Arg("hello world test").
		Build(ctx)

	pipeBuilder := echoCmd.Pipe("wc")
	pipeCmd := pipeBuilder.OptB("-w").Build()

	result, err := pipeCmd.Run()
	if err != nil {
		t.Fatalf("Run() with pipe failed: %v", err)
	}

	if result.ExitCode() != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode())
	}

	// wc -w should return "3" for "hello world test"
	output := strings.TrimSpace(string(result.Stdout()))
	if output != "3" {
		t.Errorf("Expected '3', got '%s'", output)
	}
}
