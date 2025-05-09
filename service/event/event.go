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
	CPUCount    int          `json:"cpu-count,omitempty"`
	MemTotal    int64        `json:"memTotal,omitempty"`
	Processes   []Process    `json:"processes,omitempty"`
	Deployments []Deployment `json:"deployments,omitempty"`
}

func (StatisticsUpdated) isEvent() {}

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
type DeploymentRequired struct {
	Deployment
}

func (e DeploymentRequired) isEvent() {}
