// Copyright (c) 2025 worldiety GmbH
//
// This file is part of the NAGO Low-Code Platform.
// Licensed under the terms specified in the LICENSE file.
//
// SPDX-License-Identifier: Custom-License

package service

import (
	"github.com/worldiety/nago-runner/service/event"
	"os"
	"path/filepath"
)

func NewWriteFile() WriteFile {
	return func(req event.WriteFileRequested) error {
		parentDir := filepath.Dir(req.Path)
		if _, err := os.Stat(parentDir); os.IsNotExist(err) {
			_ = os.MkdirAll(parentDir, req.Mode)
		}

		return os.WriteFile(req.Path, req.Content, req.Mode)
	}
}
