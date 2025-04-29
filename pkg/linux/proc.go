// Copyright (c) 2025 worldiety GmbH
//
// This file is part of the NAGO Low-Code Platform.
// Licensed under the terms specified in the LICENSE file.
//
// SPDX-License-Identifier: Custom-License

package linux

import (
	"fmt"
	"os"
	"os/user"
	"strconv"
	"strings"
	"time"
)

func CPUTime(pid int) (utime, stime float64, err error) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return 0, 0, err
	}

	fields := strings.Fields(string(data))
	utime, _ = strconv.ParseFloat(fields[13], 64) // user mode jiffies
	stime, _ = strconv.ParseFloat(fields[14], 64) // kernel mode jiffies
	return utime, stime, nil
}

func SampleCPUTime(pid int, duration time.Duration) (percent int, err error) {
	clkTck := float64(100) // Standard: 100 Jiffies per Sekunde
	t1, s1, _ := CPUTime(pid)
	time.Sleep(duration)
	t2, s2, _ := CPUTime(pid)
	cpuUsage := ((t2 + s2) - (t1 + s1)) / clkTck * 100
	return int(cpuUsage), nil
}

// MemoryUsage returns the RSS memory of the given process.
func MemoryUsage(pid int) (int64, error) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", pid))
	if err != nil {
		return 0, err
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "VmRSS:") {
			fields := strings.Fields(line)
			kb, _ := strconv.ParseInt(fields[1], 10, 64)
			return kb * 1024, nil
		}
	}
	return 0, nil
}

func Username(uid int) string {
	u, err := user.LookupId(strconv.Itoa(uid))
	if err != nil {
		return fmt.Sprintf("UID %d", uid)
	}
	return u.Username
}

func BinaryPath(pid int) string {
	path, err := os.Readlink(fmt.Sprintf("/proc/%d/exe", pid))
	if err != nil {
		return "-"
	}
	return path
}

func UID(pid int) (int, error) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", pid))
	if err != nil {
		return 0, err
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "Uid:") {
			fields := strings.Fields(line)
			return strconv.Atoi(fields[1]) // real UID
		}
	}
	return 0, fmt.Errorf("UID not found")
}

func BinaryName(pid int) string {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/comm", pid))
	if err != nil {
		return "-"
	}
	return strings.TrimSpace(string(data))
}

func MemoryTotal() (int64, error) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, err
	}

	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "MemTotal:") {
			fields := strings.Fields(line)
			kb, _ := strconv.ParseInt(fields[1], 10, 64)
			return kb * 1024, nil
		}
	}
	return 0, fmt.Errorf("MemTotal not found")
}
