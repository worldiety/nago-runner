// Copyright (c) 2025 worldiety GmbH
//
// This file is part of the NAGO Low-Code Platform.
// Licensed under the terms specified in the LICENSE file.
//
// SPDX-License-Identifier: Custom-License

package linux

import (
	"fmt"
	"os/exec"
	"strings"
)

func AptInstall(names ...string) error {
	var args []string
	args = append(args, "install", "-y")
	args = append(args, names...)

	installCmd := exec.Command("apt", args...)
	if output, err := installCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to install package %s: %w: %s", strings.Join(args, " "), err, string(output))
	}

	return nil
}

func AptUpdate() error {
	updateCmd := exec.Command("apt", "update")
	updateCmd.Stderr = nil
	if output, err := updateCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to update package list: %v, output: %s", err, output)
	}

	return nil
}
