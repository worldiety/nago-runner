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
	"log/slog"
	"os"
	"time"
)

func NewReadDir() ReadDir {
	return func(req event.ReadDirRequested) (event.ReadDirResponse, error) {
		files, err := os.ReadDir(req.Path)
		if err != nil {
			return event.ReadDirResponse{}, fmt.Errorf("read dir err: %w", err)
		}

		res := event.ReadDirResponse{
			RequestID: req.RequestID,
			Path:      req.Path,
		}

		for _, file := range files {
			info, err := file.Info()
			if err != nil {
				slog.Error("failed to read file info", "err", err.Error(), "path", req.Path, "file", file.Name())
			}

			var modTime time.Time
			var size int64
			if info != nil {
				modTime = info.ModTime()
				size = info.Size()
			}

			res.Files = append(res.Files, event.File{
				Name:    file.Name(),
				Mode:    file.Type(),
				ModTime: modTime,
				Size:    size,
			})
		}

		return res, nil
	}
}
