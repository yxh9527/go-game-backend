package batchwriter

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestRetryRetainsOrderAndSurvivesPanic(t *testing.T) {
	in := make(chan int, 4)
	for _, n := range []int{1, 2, 3, 4} {
		in <- n
	}
	close(in)
	var attempts [][]int
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	left := Run(ctx, in, 2, time.Millisecond, func(batch []int) error {
		attempts = append(attempts, append([]int(nil), batch...))
		if len(attempts) == 1 {
			return errors.New("ES unavailable")
		}
		if len(attempts) == 2 {
			panic("temporary writer panic")
		}
		return nil
	}, nil)
	if len(left) != 0 || !reflect.DeepEqual(attempts, [][]int{{1, 2}, {1, 2}, {1, 2}, {3, 4}}) {
		t.Fatalf("left=%v attempts=%v", left, attempts)
	}
}

func TestOutageKeepsBatchBoundedAndReturnsPendingOnCancel(t *testing.T) {
	in := make(chan int, 10)
	for i := 0; i < 10; i++ {
		in <- i
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	left := Run(ctx, in, 2, time.Millisecond, func([]int) error { return errors.New("outage") }, func(error, int) { cancel() })
	if !reflect.DeepEqual(left, []int{0, 1}) || len(in) != 8 {
		t.Fatalf("left=%v queued=%d", left, len(in))
	}
}
