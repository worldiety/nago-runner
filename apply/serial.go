// Copyright (c) 2025 worldiety GmbH
//
// This file is part of the NAGO Low-Code Platform.
// Licensed under the terms specified in the LICENSE file.
//
// SPDX-License-Identifier: Custom-License

package apply

import (
	"context"
	"log/slog"
	"time"
)

const (
	// defaultRetryMin is the delay before the first retry of a failed run.
	defaultRetryMin = 30 * time.Second
	// defaultRetryMax limits the delay between retries, which doubles with every failed run.
	defaultRetryMax = 5 * time.Minute
)

// Serial runs a function on a single goroutine, thus two runs never overlap. Requests which arrive while a run is
// in progress are merged into exactly one further run. This fits configuration changes, because every run fetches
// the complete and current configuration anyway. Overlapping runs would download the same executables into the
// same files and abort each other.
//
// A failed run is retried with a growing delay until a run succeeds. Otherwise, a transient failure (e.g. an
// aborted download or a failed restart) would only be repaired by the next change announced by the hub, which may
// never come, e.g. right after an installation.
type Serial struct {
	pending chan struct{}
}

// NewSerial starts the worker, which stops when the context is done.
func NewSerial(ctx context.Context, fn func() error) *Serial {
	return newSerial(ctx, fn, defaultRetryMin, defaultRetryMax)
}

func newSerial(ctx context.Context, fn func() error, retryMin, retryMax time.Duration) *Serial {
	s := &Serial{pending: make(chan struct{}, 1)}
	go func() {
		var delay time.Duration
		var retry *time.Timer
		stopRetry := func() {
			if retry != nil {
				retry.Stop()
				retry = nil
			}
		}

		defer stopRetry()

		for {
			select {
			case <-ctx.Done():
				return
			case <-s.pending:
				// a request in between, e.g. a change announced by the hub, runs right away and replaces the retry
				stopRetry()

				if err := fn(); err != nil {
					delay = nextDelay(delay, retryMin, retryMax)
					slog.Warn("run failed, retrying", "in", delay.String(), "err", err.Error())
					retry = time.AfterFunc(delay, s.Request)
					continue
				}

				delay = 0
			}
		}
	}()

	return s
}

// nextDelay doubles the delay within the given bounds.
func nextDelay(delay, retryMin, retryMax time.Duration) time.Duration {
	if delay < retryMin {
		return retryMin
	}

	return min(delay*2, retryMax)
}

// Request schedules a run and never blocks.
func (s *Serial) Request() {
	select {
	case s.pending <- struct{}{}:
	default:
		// a run is already pending and will see the newest configuration
	}
}
