package apply

import (
	"context"
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

	s := NewSerial(ctx, func() {
		if active.Add(1) > 1 {
			overlaps.Add(1)
		}
		defer active.Add(-1)

		started <- struct{}{}
		if runs.Add(1) == 1 {
			<-release
		}
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
