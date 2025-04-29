// Copyright (c) 2025 worldiety GmbH
//
// This file is part of the NAGO Low-Code Platform.
// Licensed under the terms specified in the LICENSE file.
//
// SPDX-License-Identifier: Custom-License

package service

import (
	"context"
	"github.com/worldiety/nago-runner/service/event"
	"time"
)

func NewSchedulerStatistics(bus event.Bus, statistics Statistics) SchedulerStatistics {
	return func(ctx context.Context) {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case <-time.After(30 * time.Second):
					bus.Publish(statistics())
				}
			}
		}()
	}
}
