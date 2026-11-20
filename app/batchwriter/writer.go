// Package batchwriter drains existing bounded channels without discarding a failed batch.
package batchwriter

import (
	"context"
	"fmt"
	"time"
)

// Run retries the same batch before accepting newer items. Queue capacity remains
// the caller's responsibility; it does not create an unbounded overflow queue.
// Cancellation returns the unwritten batch for the caller to handle on shutdown.
func Run[T any](ctx context.Context, input <-chan T, size int, interval time.Duration, save func([]T) error, failed func(error, int)) []T {
	return run(ctx, input, size, interval, save, failed, true)
}

// RunOnce logs a failed output batch and discards it. This policy is for ES
// reports only; financial operations must retain their own confirmation rules.
func RunOnce[T any](ctx context.Context, input <-chan T, size int, interval time.Duration, save func([]T) error, failed func(error, int)) []T {
	return run(ctx, input, size, interval, save, failed, false)
}

func run[T any](ctx context.Context, input <-chan T, size int, interval time.Duration, save func([]T) error, failed func(error, int), retry bool) []T {
	if size < 1 || interval <= 0 {
		panic("invalid batch writer configuration")
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	batch := make([]T, 0, size)
	blocked := false
	flush := func() {
		if len(batch) == 0 {
			return
		}
		err := guardedSave(batch, save)
		blocked = err != nil && retry
		if err != nil {
			if failed != nil {
				failed(err, len(batch))
			}
			if retry {
				return
			}
		}
		clear(batch)
		batch = batch[:0]
	}
	for {
		active := input
		if blocked {
			active = nil
		}
		select {
		case <-ctx.Done():
			return batch
		case <-ticker.C:
			flush()
			if input == nil && len(batch) == 0 {
				return nil
			}
		case item, ok := <-active:
			if !ok {
				input = nil
				flush()
				if len(batch) == 0 {
					return nil
				}
				continue
			}
			batch = append(batch, item)
			if len(batch) >= size {
				flush()
			}
		}
	}
}

func guardedSave[T any](batch []T, save func([]T) error) (err error) {
	defer func() {
		if recover() != nil {
			err = fmt.Errorf("batch writer panicked; retaining batch")
		}
	}()
	return save(batch)
}
