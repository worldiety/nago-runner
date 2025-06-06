// Copyright (c) 2025 worldiety GmbH
//
// This file is part of the NAGO Low-Code Platform.
// Licensed under the terms specified in the LICENSE file.
//
// SPDX-License-Identifier: Custom-License

package event

import (
	"os"
	"time"
)

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

type DeleteInstanceDataRequested struct {
	RequestID int64  `json:"rid"`
	Unit      string `json:"unit"`
}

func (e DeleteInstanceDataRequested) isEvent() {}

type WriteFileRequested struct {
	RequestID int64       `json:"rid"`
	Path      string      `json:"path"`
	Mode      os.FileMode `json:"mode"`
	Content   []byte      `json:"content"`
}

func (e WriteFileRequested) isEvent() {}

type DeleteFileRequested struct {
	RequestID int64  `json:"rid"`
	Path      string `json:"path"`
}

func (e DeleteFileRequested) isEvent() {}

type Response struct {
	RequestID int64  `json:"rid"`
	Error     string `json:"err"`
}

func (e Response) isEvent() {}

type ReadFileRequested struct {
	RequestID int64  `json:"rid"`
	Path      string `json:"path"`
	MaxSize   int64  `json:"maxSize"` // defaults to 1MiB
}

func (e ReadFileRequested) isEvent() {}

type ReadFileResponse struct {
	RequestID int64  `json:"rid"`
	Path      string `json:"path"`
	File      File   `json:"file"`
	Content   []byte `json:"content"`
}

func (e ReadFileResponse) isEvent() {}

type ReadDirRequested struct {
	RequestID int64  `json:"rid"`
	Path      string `json:"path"`
}

func (e ReadDirRequested) isEvent() {}

type File struct {
	Name    string      `json:"name"`
	Mode    os.FileMode `json:"mode"`
	ModTime time.Time   `json:"modTime"`
	Size    int64       `json:"size"`
	// optional
	Sha3v512 string `json:"sha512"`
}
type ReadDirResponse struct {
	RequestID int64  `json:"rid"`
	Path      string `json:"path"`
	Files     []File
}

func (e ReadDirResponse) isEvent() {}

type ExecRequest struct {
	RequestID     int64    `json:"rid"`
	Cmd           string   `json:"cmd"`
	Args          []string `json:"args"`
	CollectStdOut bool     `json:"collectStdOut"`
	CollectErrOut bool     `json:"collectErrOut"`
}

func (e ExecRequest) isEvent() {}

type ExecResponse struct {
	RequestID int64    `json:"rid"`
	Cmd       string   `json:"cmd"`
	Args      []string `json:"args"`
	StdOut    []byte   `json:"stdOut"`
	ErrOut    []byte   `json:"errOut"`
	ExitCode  int      `json:"exitCode"`
	Error     string   `json:"error"`
}

func (e ExecResponse) isEvent() {}

type BackupRequest struct {
	RequestID  int64  `json:"rid"`
	ProgressID string `json:"progressId"`
	InstanceID string `json:"instanceID"`
}

func (e BackupRequest) isEvent() {}

func (e BackupRequest) ReqID() int64 {
	return e.RequestID
}

type RestoreRequest struct {
	RequestID  int64  `json:"rid"`
	InstanceID string `json:"instanceID"`
	ProgressID string `json:"progressId"`
	Exec       File   `json:"exec"`
	Data       []File `json:"data"`
}

func (e RestoreRequest) isEvent() {}

func (e RestoreRequest) ReqID() int64 {
	return e.RequestID
}

type ProgressUpdated struct {
	ProgressID string `json:"progressId"`
	Percent    int    `json:"percent"`
	Finished   bool   `json:"done,omitempty"`
	Error      string `json:"error,omitempty"`
}

func (e ProgressUpdated) isEvent() {}

func (e ProgressUpdated) ReqID() int64 {
	return 0
}
