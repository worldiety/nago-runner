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
)

type Bus interface {
	Publish(Event)
	Subscribe(fn func(evt Event)) (close func())
}
