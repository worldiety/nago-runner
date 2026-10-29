// Copyright (c) 2025 worldiety GmbH
//
// This file is part of the NAGO Low-Code Platform.
// Licensed under the terms specified in the LICENSE file.
//
// SPDX-License-Identifier: Custom-License

package apply

import "context"

// Serial runs a function on a single goroutine, thus two runs never overlap. Requests which arrive while a run is
// in progress are merged into exactly one further run. This fits configuration changes, because every run fetches
// the complete and current configuration anyway. Overlapping runs would download the same executables into the
// same files and abort each other.
type Serial struct {
	pending chan struct{}
}

// NewSerial starts the worker, which stops when the context is done.
func NewSerial(ctx context.Context, fn func()) *Serial {
	s := &Serial{pending: make(chan struct{}, 1)}
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-s.pending:
				fn()
			}
		}
	}()

	return s
}

// Request schedules a run and never blocks.
func (s *Serial) Request() {
	select {
	case s.pending <- struct{}{}:
	default:
		// a run is already pending and will see the newest configuration
	}
}
