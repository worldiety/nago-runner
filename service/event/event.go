// Copyright (c) 2025 worldiety GmbH
//
// This file is part of the NAGO Low-Code Platform.
// Licensed under the terms specified in the LICENSE file.
//
// SPDX-License-Identifier: Custom-License

package event

type RunnerLaunched struct {
	Hostname string `json:"hostname"`
}

func (e RunnerLaunched) isEvent() {}

type Event interface {
	isEvent()
}

type ConnectionCreated struct {
}

func (e ConnectionCreated) isEvent() {}

type Process struct {
	PID        int    `json:"pid"`
	User       string `json:"user"`
	UID        int    `json:"uid"`
	BinaryPath string `json:"binaryPath"`
	BinaryName string `json:"binaryName"`
	CPU        int    `json:"cpu"`
	RSS        int64  `json:"rss"`
}
type StatisticsUpdated struct {
	CPUCount  int       `json:"cpu-count,omitempty"`
	MemTotal  int64     `json:"memTotal,omitempty"`
	Processes []Process `json:"processes,omitempty"`
}

func (StatisticsUpdated) isEvent() {}
