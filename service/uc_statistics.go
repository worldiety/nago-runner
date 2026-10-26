// Copyright (c) 2025 worldiety GmbH
//
// This file is part of the NAGO Low-Code Platform.
// Licensed under the terms specified in the LICENSE file.
//
// SPDX-License-Identifier: Custom-License

package service

import (
	"github.com/worldiety/nago-runner/pkg/linux"
	"github.com/worldiety/nago-runner/service/event"
	"log/slog"
	"os"
	"strconv"
	"sync"
	"time"
)

// aptCheckInterval limits the expensive apt simulation.
const aptCheckInterval = 30 * time.Minute

func NewStatistics() Statistics {
	var apt struct {
		mutex     sync.Mutex
		checkedAt time.Time
		count     int
		reboot    bool
	}

	hostStats := func() event.HostStats {
		var h event.HostStats
		h.CPUPercent, _ = linux.CPUBusy(200 * time.Millisecond)
		h.Load1, h.Load5, h.Load15, _ = linux.LoadAvg()
		h.MemAvailable, _ = linux.MemoryAvailable()
		h.DiskTotal, h.DiskFree, _ = linux.DiskUsage(dataPrefix)
		h.UptimeSeconds, _ = linux.Uptime()
		h.Kernel = linux.Kernel()

		apt.mutex.Lock()
		defer apt.mutex.Unlock()
		if time.Since(apt.checkedAt) > aptCheckInterval {
			count, err := linux.AptUpgradable()
			if err != nil {
				slog.Warn("cannot check apt upgrades", "err", err.Error())
			} else {
				apt.count = count
			}
			apt.reboot = linux.RebootRequired()
			apt.checkedAt = time.Now()
		}

		h.AptUpgradable = apt.count
		h.AptCheckedAt = apt.checkedAt
		h.RebootRequired = apt.reboot
		return h
	}

	return func() event.StatisticsUpdated {
		var res event.StatisticsUpdated

		memTotal, _ := linux.MemoryTotal()
		res.MemTotal = memTotal
		res.Host = hostStats()

		entries, err := os.ReadDir("/proc")
		if err != nil {
			slog.Error("Error reading /proc", "err", err.Error())
			return res
		}

		for _, entry := range entries {
			if !entry.IsDir() || !isNumericDir(entry.Name()) {
				continue
			}

			pid, _ := strconv.Atoi(entry.Name())

			uid, err := linux.UID(pid)
			if err != nil {
				continue
			}

			cpuP, _ := linux.SampleCPUTime(pid, time.Millisecond*200)
			memUsage, _ := linux.MemoryUsage(pid)

			res.Processes = append(res.Processes, event.Process{
				PID:        pid,
				UID:        uid,
				User:       linux.Username(uid),
				BinaryPath: linux.BinaryPath(pid),
				BinaryName: linux.BinaryName(pid),
				CPU:        cpuP,
				RSS:        memUsage,
			})

		}

		return res
	}
}

func isNumericDir(name string) bool {
	_, err := strconv.Atoi(name)
	return err == nil
}
