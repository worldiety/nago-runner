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
	"time"
)

func NewStatistics(deployments FindDeployments) Statistics {
	return func() event.StatisticsUpdated {
		var res event.StatisticsUpdated

		memTotal, _ := linux.MemoryTotal()
		res.MemTotal = memTotal

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

		dpls, err := deployments()
		if err != nil {
			slog.Error("Error getting deployments", "err", err.Error())
		}

		for _, dpl := range dpls {
			res.Deployments = append(res.Deployments, event.Deployment(dpl))
		}

		return res
	}
}

func isNumericDir(name string) bool {
	_, err := strconv.Atoi(name)
	return err == nil
}
