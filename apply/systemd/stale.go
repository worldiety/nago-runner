// Copyright (c) 2025 worldiety GmbH
//
// This file is part of the NAGO Low-Code Platform.
// Licensed under the terms specified in the LICENSE file.
//
// SPDX-License-Identifier: Custom-License

package systemd

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// runsReplacedExecutable tells whether the main process of the service still executes an executable which has
// been replaced on disk in the meantime. This happens, if a former apply swapped the executable but failed before
// the restart. The hash of the file on disk then matches the configuration, thus nothing else would ever restart
// the service and the new version never runs. A stopped service is never reported, e.g. during a restore.
func runsReplacedExecutable(service Service) (bool, error) {
	out, err := exec.Command("systemctl", "show", "--property=MainPID", "--value", service.Name()).Output()
	if err != nil {
		return false, fmt.Errorf("cannot query main pid: %w", err)
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		return false, fmt.Errorf("unexpected main pid: %q", string(out))
	}

	if pid <= 0 {
		return false, nil
	}

	link, err := os.Readlink(fmt.Sprintf("/proc/%d/exe", pid))
	if err != nil {
		return false, fmt.Errorf("cannot read executable of pid %d: %w", pid, err)
	}

	return isReplacedExecutable(link, service.Paths().ExecFilename), nil
}

// isReplacedExecutable interprets the /proc/<pid>/exe link. The kernel appends " (deleted)", if the file of the
// running executable has been unlinked, which is the case after renaming a new executable over it.
func isReplacedExecutable(link string, execFilename string) bool {
	return link == execFilename+" (deleted)"
}
