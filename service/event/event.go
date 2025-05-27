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

type RunnerConfigurationChanged struct {
	RunnerID string `json:"runnerID"`
}

func (e RunnerConfigurationChanged) isEvent() {}

type JournalCtlEntry struct {
	RealtimeTimestamp   string `json:"__REALTIME_TIMESTAMP,omitempty"`
	StreamID            string `json:"_STREAM_ID,omitempty"`
	UID                 string `json:"_UID,omitempty"`
	SyslogFacility      string `json:"SYSLOG_FACILITY,omitempty"`
	Transport           string `json:"_TRANSPORT,omitempty"`
	Priority            string `json:"PRIORITY,omitempty"`
	PID                 string `json:"_PID,omitempty"`
	SystemdCgroup       string `json:"_SYSTEMD_CGROUP,omitempty"`
	SyslogIdentifier    string `json:"SYSLOG_IDENTIFIER,omitempty"`
	MonotonicTimestamp  string `json:"__MONOTONIC_TIMESTAMP,omitempty"`
	Cursor              string `json:"__CURSOR,omitempty"`
	SystemdInvocationID string `json:"_SYSTEMD_INVOCATION_ID,omitempty"`
	Exe                 string `json:"_EXE,omitempty"`
	Cmdline             string `json:"_CMDLINE,omitempty"`
	SystemdUnit         string `json:"_SYSTEMD_UNIT,omitempty"`
	BootID              string `json:"_BOOT_ID,omitempty"`
	SystemdSlice        string `json:"_SYSTEMD_SLICE,omitempty"`
	Comm                string `json:"_COMM,omitempty"`
	MachineID           string `json:"_MACHINE_ID,omitempty"`
	GID                 string `json:"_GID,omitempty"`
	CapEffective        string `json:"_CAP_EFFECTIVE,omitempty"`
	RuntimeScope        string `json:"_RUNTIME_SCOPE,omitempty"`
	SeqnumID            string `json:"__SEQNUM_ID,omitempty"`
	SELinuxContext      string `json:"_SELINUX_CONTEXT,omitempty"`
	Host                string `json:"_HOSTNAME,omitempty"`
	Seqnum              string `json:"__SEQNUM,omitempty"`
	Message             string `json:"MESSAGE,omitempty"`
}

type JournalCtlLogResponse struct {
	RequestID int64             `json:"rid"`
	Entries   []JournalCtlEntry `json:"entries"`
}

func (e JournalCtlLogResponse) isEvent() {}

type JournalCtlLogRequest struct {
	RequestID int64  `json:"rid"`
	Unit      string `json:"unit"`
	LastN     int    `json:"lastN"`
	Since     string `json:"since"`
	Until     string `json:"until"`
}

func (e JournalCtlLogRequest) isEvent() {}
