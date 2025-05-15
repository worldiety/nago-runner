// Copyright (c) 2025 worldiety GmbH
//
// This file is part of the NAGO Low-Code Platform.
// Licensed under the terms specified in the LICENSE file.
//
// SPDX-License-Identifier: Custom-License

package linux

import (
	"errors"
	"github.com/worldiety/nago-runner/pkg/run"
	"os/exec"
)

// Which returns the empty string, if which was successful but the program was not found.
func Which(name string) (string, error) {
	val, err := run.CommandString("which", name)
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			if exitErr.ExitCode() == 1 {
				return "", nil
			}
		}
	}

	return val, err
}
