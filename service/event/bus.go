// Copyright (c) 2025 worldiety GmbH
//
// This file is part of the NAGO Low-Code Platform.
// Licensed under the terms specified in the LICENSE file.
//
// SPDX-License-Identifier: Custom-License

package event

import "github.com/worldiety/enum"

var (
	_ = enum.Variant[Event, ConnectionCreated]()
	_ = enum.Variant[Event, RunnerLaunched]()
	_ = enum.Variant[Event, StatisticsUpdated]()
	_ = enum.Variant[Event, RunnerConfigurationChanged]()
	_ = enum.Variant[Event, JournalCtlLogRequest]()
	_ = enum.Variant[Event, JournalCtlLogResponse]()
	_ = enum.Variant[Event, DeleteInstanceDataRequested]()
	_ = enum.Variant[Event, WriteFileRequested]()
	_ = enum.Variant[Event, DeleteFileRequested]()
	_ = enum.Variant[Event, ReadFileRequested]()
	_ = enum.Variant[Event, ReadFileResponse]()
	_ = enum.Variant[Event, ReadDirRequested]()
	_ = enum.Variant[Event, ReadDirResponse]()
	_ = enum.Variant[Event, ExecRequest]()
	_ = enum.Variant[Event, ExecResponse]()
	_ = enum.Variant[Event, Response]()
	_ = enum.Variant[Event, BackupRequest]()
	_ = enum.Variant[Event, RestoreRequest]()
	_ = enum.Variant[Event, ProgressUpdated]()
)

type Bus interface {
	Publish(Event)
	Subscribe(fn func(evt Event)) (close func())
}
