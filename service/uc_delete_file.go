// Copyright (c) 2025 worldiety GmbH
//
// This file is part of the NAGO Low-Code Platform.
// Licensed under the terms specified in the LICENSE file.
//
// SPDX-License-Identifier: Custom-License

package service

import (
	"fmt"
	"github.com/worldiety/nago-runner/service/event"
	"os"
)

func NewDeleteFile() DeleteFile {
	return func(req event.DeleteFileRequested) error {
		if req.Path == "" || req.Path == "/" {
			return fmt.Errorf("invalid path")
		}

		return os.RemoveAll(req.Path)
	}
}
