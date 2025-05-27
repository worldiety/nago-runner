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
	"log/slog"
)

type Hello func(evt event.ConnectionCreated) event.RunnerLaunched

type Statistics func() event.StatisticsUpdated
type SchedulerStatistics func(ctx context.Context)

type Deployment struct {
	AppID           string `json:"appID"`
	BinaryID        string `json:"binaryID"`
	OrgSlug         string `json:"orgSlug"`
	AppSlug         string `json:"appSlug"`
	BinarySha256    string `json:"binarySha256"`
	MaxMemoryMiB    int    `json:"maxMemory"`   // e.g. 512 for 512 MiB
	MaxCPUQuota     int    `json:"maxCPUQuota"` // range 1-100 percent
	TimeoutStartSec int    `json:"timeoutStartSec"`
	Port            int    `json:"port"`
}

type CollectLogs func(request event.JournalCtlLogRequest) ([]event.JournalCtlEntry, error)

type UseCases struct {
	Hello              Hello
	Statistics         Statistics
	ScheduleStatistics SchedulerStatistics
	CollectLogs        CollectLogs
}

func NewUseCases(bus event.Bus) UseCases {

	statisticsFn := NewStatistics()

	uc := UseCases{
		Hello:              NewHello(),
		Statistics:         statisticsFn,
		ScheduleStatistics: NewSchedulerStatistics(bus, statisticsFn),
		CollectLogs:        NewCollectLogs(),
	}

	bus.Subscribe(func(evt event.Event) {
		switch evt := evt.(type) {
		case event.ConnectionCreated:
			bus.Publish(uc.Hello(evt))
		case event.JournalCtlLogRequest:
			slog.Info("requested log", "id", evt.RequestID, "unit", evt.Unit)
			entries, err := uc.CollectLogs(evt)
			if err != nil {
				slog.Error("Error collecting logs", "err", err.Error())
			}

			bus.Publish(event.JournalCtlLogResponse{
				RequestID: evt.RequestID,
				Entries:   entries,
			})
		}
	})

	return uc
}
