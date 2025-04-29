// Copyright (c) 2025 worldiety GmbH
//
// This file is part of the NAGO Low-Code Platform.
// Licensed under the terms specified in the LICENSE file.
//
// SPDX-License-Identifier: Custom-License

package linux

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
)

type File struct {
	Filename string
	Data     []byte
	Mode     os.FileMode
	Status   Status
}

func ApplyFile(file File) Result {
	var exists bool
	if stat, err := os.Stat(file.Filename); err == nil {
		exists = true
		existingBuf, err := os.ReadFile(file.Filename)
		if err != nil {
			return Result{Error: fmt.Errorf("could not read %s: %w", file.Filename, err)}
		}

		if bytes.Equal(existingBuf, file.Data) {
			if stat.Mode() != file.Mode {
				if err := os.Chmod(file.Filename, file.Mode); err != nil {
					return Result{Error: fmt.Errorf("could not chmod %s: %w", file.Filename, err)}
				}
				return Result{Action: Updated}
			}

			return Result{Action: Ignored}
		}

	}

	if !exists && file.Status == Absent {
		return Result{Action: Ignored}
	}

	if exists && file.Status == Absent {
		if err := os.RemoveAll(file.Filename); err != nil {
			return Result{Error: fmt.Errorf("could not remove %s: %w", file.Filename, err)}
		}

		return Result{Action: Deleted}
	}

	tmp := file.Filename + ".tmp"

	parentDir := filepath.Dir(tmp)
	if _, err := os.Stat(parentDir); os.IsNotExist(err) {
		if err := os.MkdirAll(parentDir, 0755); err != nil {
			return Result{Error: fmt.Errorf("could not create parent directory %s: %w", parentDir, err)}
		}
	}

	if err := os.WriteFile(tmp, file.Data, file.Mode); err != nil {
		return Result{Error: fmt.Errorf("could not write tmp file: %s: %w", file.Filename, err)}
	}

	if err := os.Rename(file.Filename, tmp); err != nil {
		return Result{Error: fmt.Errorf("could not rename %s: %w", file.Filename, err)}
	}

	if exists {
		return Result{Action: Updated}
	}

	return Result{Action: Created}
}
