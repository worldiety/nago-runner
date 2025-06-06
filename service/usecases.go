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
	"github.com/worldiety/nago-runner/setup"
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

type DeleteInstanceData func(req event.DeleteInstanceDataRequested) error

type WriteFile func(req event.WriteFileRequested) error
type DeleteFile func(req event.DeleteFileRequested) error
type ReadFile func(req event.ReadFileRequested) (event.ReadFileResponse, error)
type ReadDir func(req event.ReadDirRequested) (event.ReadDirResponse, error)
type Exec func(req event.ExecRequest) (event.ExecResponse, error)

type DoBackup func(req event.BackupRequest) error
type DoRestore func(req event.RestoreRequest) error

type UseCases struct {
	Hello              Hello
	Statistics         Statistics
	ScheduleStatistics SchedulerStatistics
	CollectLogs        CollectLogs
	DeleteInstanceData DeleteInstanceData
	WriteFile          WriteFile
	DeleteFile         DeleteFile
	ReadFile           ReadFile
	ReadDir            ReadDir
	Exec               Exec
	DoBackup           DoBackup
	DoRestore          DoRestore
}

func NewUseCases(bus event.Bus, settings setup.Settings) UseCases {
	statisticsFn := NewStatistics()

	uc := UseCases{
		Hello:              NewHello(),
		Statistics:         statisticsFn,
		ScheduleStatistics: NewSchedulerStatistics(bus, statisticsFn),
		CollectLogs:        NewCollectLogs(),
		DeleteInstanceData: NewDeleteInstanceData(),
		DeleteFile:         NewDeleteFile(),
		ReadFile:           NewReadFile(),
		ReadDir:            NewReadDir(),
		WriteFile:          NewWriteFile(),
		Exec:               NewExec(),
		DoBackup:           NewDoBackup(settings, bus),
		DoRestore:          NewDoRestore(settings, bus),
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

		case event.DeleteInstanceDataRequested:
			if err := uc.DeleteInstanceData(evt); err != nil {
				slog.Error("Error deleting instance data", "err", err.Error())
			}

		case event.WriteFileRequested:
			if err := uc.WriteFile(evt); err != nil {
				slog.Error("Error writing file", "err", err.Error())
				bus.Publish(event.Response{
					RequestID: evt.RequestID,
					Error:     err.Error(),
				})
				return
			}

			bus.Publish(event.Response{
				RequestID: evt.RequestID,
			})
		case event.DeleteFileRequested:
			if err := uc.DeleteFile(evt); err != nil {
				slog.Error("Error deleting file", "err", err.Error())
				bus.Publish(event.Response{
					RequestID: evt.RequestID,
					Error:     err.Error(),
				})
				return
			}

			bus.Publish(event.Response{
				RequestID: evt.RequestID,
			})
		case event.ReadFileRequested:
			resp, err := uc.ReadFile(evt)
			if err != nil {
				slog.Error("Error reading file", "err", err.Error())
				return
			}
			bus.Publish(resp)

		case event.ReadDirRequested:
			resp, err := uc.ReadDir(evt)
			if err != nil {
				slog.Error("Error reading dir", "err", err.Error())
			}

			bus.Publish(resp)
		case event.ExecRequest:
			resp, err := uc.Exec(evt)
			if err != nil {
				slog.Error("Error exec", "err", err.Error())
			}

			// always respond
			bus.Publish(resp)

		case event.BackupRequest:
			go func() {
				err := uc.DoBackup(evt)

				if err != nil {
					slog.Error("Error performing async backup", "err", err.Error())
				}
			}()

			bus.Publish(event.Response{RequestID: evt.RequestID})

		case event.RestoreRequest:
			go func() {
				err := uc.DoRestore(evt)

				if err != nil {
					slog.Error("Error performing async restore", "err", err.Error())
				}
			}()

			bus.Publish(event.Response{RequestID: evt.RequestID})
		}

	})

	return uc
}
