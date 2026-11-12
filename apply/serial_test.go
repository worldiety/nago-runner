package apply

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestSerialMergesRequestsAndNeverOverlaps(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var runs, active, overlaps atomic.Int32
	release := make(chan struct{})
	started := make(chan struct{}, 10)

	s := NewSerial(ctx, func() error {
		if active.Add(1) > 1 {
			overlaps.Add(1)
		}
		defer active.Add(-1)

		started <- struct{}{}
		if runs.Add(1) == 1 {
			<-release
		}

		return nil
	})

	s.Request()
	<-started

	// these arrive while the first run blocks and must collapse into one more run
	for range 5 {
		s.Request()
	}

	close(release)
	<-started

	time.Sleep(50 * time.Millisecond)
	if got := runs.Load(); got != 2 {
		t.Fatalf("expected 2 runs, got %d", got)
	}

	if overlaps.Load() != 0 {
		t.Fatal("runs overlapped")
	}
}

func TestSerialRetriesFailedRunsUntilSuccess(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var runs atomic.Int32
	done := make(chan struct{})

	s := newSerial(ctx, func() error {
		if runs.Add(1) < 3 {
			return errors.New("transient failure")
		}

		close(done)
		return nil
	}, 10*time.Millisecond, 20*time.Millisecond)

	// a single request, e.g. the last change announced by the hub after an installation
	s.Request()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("failed run was not retried, runs=%d", runs.Load())
	}

	time.Sleep(100 * time.Millisecond)
	if got := runs.Load(); got != 3 {
		t.Fatalf("a successful run must end the retries, got %d runs", got)
	}
}

func TestSerialStopsRetryingWhenDone(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	var runs atomic.Int32
	s := newSerial(ctx, func() error {
		runs.Add(1)
		return errors.New("permanent failure")
	}, 10*time.Millisecond, 10*time.Millisecond)

	s.Request()
	time.Sleep(50 * time.Millisecond)
	cancel()
	time.Sleep(30 * time.Millisecond)

	n := runs.Load()
	time.Sleep(100 * time.Millisecond)
	if runs.Load() != n {
		t.Fatal("retries must stop with the context")
	}
}

func TestNextDelay(t *testing.T) {
	retryMin, retryMax := 30*time.Second, 5*time.Minute

	var got []time.Duration
	var d time.Duration
	for range 6 {
		d = nextDelay(d, retryMin, retryMax)
		got = append(got, d)
	}

	want := []time.Duration{30 * time.Second, time.Minute, 2 * time.Minute, 4 * time.Minute, 5 * time.Minute, 5 * time.Minute}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("step %d: got %v, want %v", i, got[i], want[i])
		}
	}
}
