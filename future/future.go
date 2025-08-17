// Package future provides a Future type that allows for asynchronous computation.
package future

import (
	"context"
	"sync"
	"time"
)

type Future[T any] interface {
	Start() Future[T]
	Cancel()
	Wait() (T, error)
	Done() chan any
	IsDone() bool
}

type futureImpl[T any] struct {
	res    T
	err    error
	done   chan any
	fn     func(ctx context.Context) (T, error)
	ctx    context.Context
	cancel context.CancelFunc

	// Synchronization primitives
	once sync.Once
	mu   sync.RWMutex
}

func (fu *futureImpl[T]) Start() Future[T] {
	fu.once.Do(func() {
		go fu.execute()
	})
	return fu
}

func (fu *futureImpl[T]) Cancel() {
	fu.cancel()
}

func (fu *futureImpl[T]) Wait() (T, error) {
	<-fu.done
	fu.mu.RLock()
	defer fu.mu.RUnlock()
	return fu.res, fu.err
}

func (fu *futureImpl[T]) Done() chan any {
	return fu.done
}

func (fu *futureImpl[T]) IsDone() bool {
	select {
	case <-fu.done:
		return true
	default:
		return false
	}
}

// execute runs the function in a goroutine and handles the result
func (fu *futureImpl[T]) execute() {
	defer close(fu.done)

	result, err := fu.fn(fu.ctx)

	fu.mu.Lock()
	fu.res = result
	fu.err = err
	fu.mu.Unlock()
}

// New creates a new Future that executes the given function with the provided context.
// The Future is not started automatically - call Start() to begin execution.
func New[T any](ctx context.Context, fn func(ctx context.Context) (T, error)) Future[T] {
	childCtx, cancel := context.WithCancel(ctx)

	return &futureImpl[T]{
		done:   make(chan any),
		fn:     fn,
		ctx:    childCtx,
		cancel: cancel,
	}
}

// Start creates a new Future and immediately starts its execution.
func Start[T any](ctx context.Context, fn func(ctx context.Context) (T, error)) Future[T] {
	future := New(ctx, fn)
	future.Start()
	return future
}

// WaitAll waits for all provided futures to complete and returns their results.
// immediately cancels all futures if any of them fails or if the context is done.
// If any future returns an error, it will return the first error encountered.
func WaitAll[T any](ctx context.Context, fus ...Future[T]) ([]T, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	res := make([]T, 0, len(fus))

	for _, fu := range fus {
		select {
		case <-fu.Done():
			r, err := fu.Wait()
			if err != nil {
				return nil, err
			}

			res = append(res, r)
		case <-ctx.Done():
			for _, fu := range fus {
				fu.Cancel()
			}
			return nil, ctx.Err()
		}
	}

	return res, nil
}

func WaitTimeout[T any](d time.Duration, fu Future[T]) (r T, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()

	select {
	case <-fu.Done():
		return fu.Wait()
	case <-ctx.Done():
		fu.Cancel()
		return r, ctx.Err()
	}
}
