// Copyright (c) 2025 worldiety GmbH
//
// This file is part of the NAGO Low-Code Platform.
// Licensed under the terms specified in the LICENSE file.
//
// SPDX-License-Identifier: Custom-License

package linux

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// LoadAvg reads the 1, 5 and 15 minute load averages.
func LoadAvg() (l1, l5, l15 float64, err error) {
	buf, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return 0, 0, 0, err
	}

	fields := strings.Fields(string(buf))
	if len(fields) < 3 {
		return 0, 0, 0, fmt.Errorf("unexpected loadavg: %q", string(buf))
	}

	l1, _ = strconv.ParseFloat(fields[0], 64)
	l5, _ = strconv.ParseFloat(fields[1], 64)
	l15, _ = strconv.ParseFloat(fields[2], 64)
	return l1, l5, l15, nil
}

// MemoryAvailable returns the MemAvailable value of /proc/meminfo in bytes.
func MemoryAvailable() (int64, error) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, err
	}

	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "MemAvailable:") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			break
		}

		kb, err := strconv.ParseInt(fields[1], 10, 64)
		if err != nil {
			return 0, err
		}

		return kb * 1024, nil
	}

	return 0, fmt.Errorf("MemAvailable not found")
}

// DiskUsage returns the total and free bytes of the file system containing path.
func DiskUsage(path string) (total, free int64, err error) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return 0, 0, err
	}

	bsize := int64(st.Bsize)
	return int64(st.Blocks) * bsize, int64(st.Bavail) * bsize, nil
}

// Uptime returns the seconds since boot.
func Uptime() (int64, error) {
	buf, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0, err
	}

	fields := strings.Fields(string(buf))
	if len(fields) < 1 {
		return 0, fmt.Errorf("unexpected uptime: %q", string(buf))
	}

	secs, err := strconv.ParseFloat(fields[0], 64)
	return int64(secs), err
}

// Kernel returns the running kernel release.
func Kernel() string {
	buf, err := os.ReadFile("/proc/sys/kernel/osrelease")
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(buf))
}

// RebootRequired reports whether the system asks for a reboot after package updates.
func RebootRequired() bool {
	_, err := os.Stat("/var/run/reboot-required")
	return err == nil
}

var aptUpgradedRegex = regexp.MustCompile(`(\d+) upgraded, (\d+) newly installed`)

// AptUpgradable simulates an upgrade and returns the number of packages which would be upgraded. It does not
// refresh the package lists, so the result reflects the last apt update.
func AptUpgradable() (int, error) {
	cmd := exec.Command("apt-get", "-s", "-o", "Debug::NoLocking=1", "upgrade")
	cmd.Env = append(os.Environ(), "LC_ALL=C", "DEBIAN_FRONTEND=noninteractive")
	out, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("apt-get simulation failed: %w", err)
	}

	m := aptUpgradedRegex.FindSubmatch(out)
	if m == nil {
		return 0, fmt.Errorf("cannot parse apt-get output")
	}

	return strconv.Atoi(string(m[1]))
}

// CPUBusy samples /proc/stat twice and returns the busy percentage of all cpus in between.
func CPUBusy(duration time.Duration) (int, error) {
	idle1, total1, err := cpuTicks()
	if err != nil {
		return 0, err
	}

	time.Sleep(duration)

	idle2, total2, err := cpuTicks()
	if err != nil {
		return 0, err
	}

	dTotal := total2 - total1
	if dTotal <= 0 {
		return 0, nil
	}

	return int(100 * (dTotal - (idle2 - idle1)) / dTotal), nil
}

func cpuTicks() (idle, total int64, err error) {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return 0, 0, err
	}

	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 5 || fields[0] != "cpu" {
			continue
		}

		for i, field := range fields[1:] {
			v, _ := strconv.ParseInt(field, 10, 64)
			total += v
			// idle and iowait
			if i == 3 || i == 4 {
				idle += v
			}
		}

		return idle, total, nil
	}

	return 0, 0, fmt.Errorf("cpu line not found")
}
