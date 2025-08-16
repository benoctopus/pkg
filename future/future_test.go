package future

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	ctx := context.Background()
	fn := func(ctx context.Context) (string, error) {
		return "test", nil
	}

	future := New(ctx, fn)

	// Should not be done initially
	if future.IsDone() {
		t.Error("Future should not be done before starting")
	}

	// Should be able to start
	future.Start()

	result, err := future.Wait()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if result != "test" {
		t.Errorf("Expected 'test', got %v", result)
	}

	// Should be done after completion
	if !future.IsDone() {
		t.Error("Future should be done after completion")
	}
}

func TestStart(t *testing.T) {
	ctx := context.Background()
	fn := func(ctx context.Context) (int, error) {
		return 42, nil
	}

	future := Start(ctx, fn)

	result, err := future.Wait()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if result != 42 {
		t.Errorf("Expected 42, got %v", result)
	}
}

func TestIdempotentStart(t *testing.T) {
	ctx := context.Background()
	counter := 0
	fn := func(ctx context.Context) (int, error) {
		counter++
		return counter, nil
	}

	future := New(ctx, fn)

	// Call Start multiple times
	future.Start()
	future.Start()
	future.Start()

	result, err := future.Wait()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if result != 1 {
		t.Errorf("Expected function to be called only once, counter = %v", result)
	}
}

func TestCancel(t *testing.T) {
	ctx := context.Background()
	fn := func(ctx context.Context) (string, error) {
		select {
		case <-time.After(100 * time.Millisecond):
			return "completed", nil
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}

	future := Start(ctx, fn)

	// Cancel immediately
	future.Cancel()

	result, err := future.Wait()
	if err == nil {
		t.Error("Expected cancellation error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
	if result != "" {
		t.Errorf("Expected empty result on cancellation, got %v", result)
	}
}

func TestDoneChannel(t *testing.T) {
	ctx := context.Background()
	fn := func(ctx context.Context) (string, error) {
		time.Sleep(50 * time.Millisecond)
		return "done", nil
	}

	future := Start(ctx, fn)

	// Test Done() channel
	select {
	case <-future.Done():
		// Should complete
	case <-time.After(200 * time.Millisecond):
		t.Error("Future should have completed within timeout")
	}

	if !future.IsDone() {
		t.Error("Future should be done after Done() channel closes")
	}
}

func TestErrorHandling(t *testing.T) {
	ctx := context.Background()
	expectedErr := errors.New("test error")
	fn := func(ctx context.Context) (string, error) {
		return "", expectedErr
	}

	future := Start(ctx, fn)

	result, err := future.Wait()
	if err != expectedErr {
		t.Errorf("Expected %v, got %v", expectedErr, err)
	}
	if result != "" {
		t.Errorf("Expected empty result on error, got %v", result)
	}
}

func TestConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	fn := func(ctx context.Context) (int, error) {
		time.Sleep(10 * time.Millisecond)
		return 100, nil
	}

	future := New(ctx, fn)

	// Start multiple goroutines trying to start and access the future
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			future.Start()
			result, err := future.Wait()
			if err != nil || result != 100 {
				t.Errorf("Concurrent access failed: result=%v, err=%v", result, err)
			}
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		select {
		case <-done:
		case <-time.After(1 * time.Second):
			t.Fatal("Timeout waiting for concurrent access test")
		}
	}
}

func TestContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	fn := func(ctx context.Context) (string, error) {
		select {
		case <-time.After(100 * time.Millisecond):
			return "completed", nil
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}

	future := Start(ctx, fn)

	// Cancel the parent context
	cancel()

	result, err := future.Wait()
	if err == nil {
		t.Error("Expected cancellation error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
	if result != "" {
		t.Errorf("Expected empty result on cancellation, got %v", result)
	}
}
