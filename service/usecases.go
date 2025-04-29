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
)

type Hello func(evt event.ConnectionCreated) event.RunnerLaunched

type Statistics func() event.StatisticsUpdated
type SchedulerStatistics func(ctx context.Context)

type UseCases struct {
	Hello              Hello
	Statistics         Statistics
	ScheduleStatistics SchedulerStatistics
}

func NewUseCases(bus event.Bus) UseCases {

	statisticsFn := NewStatistics()

	uc := UseCases{
		Hello:              NewHello(),
		Statistics:         statisticsFn,
		ScheduleStatistics: NewSchedulerStatistics(bus, statisticsFn),
	}

	bus.Subscribe(func(evt event.Event) {
		switch evt := evt.(type) {
		case event.ConnectionCreated:
			bus.Publish(uc.Hello(evt))
		}
	})

	return uc
}
