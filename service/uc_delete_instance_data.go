// Copyright (c) 2025 worldiety GmbH
//
// This file is part of the NAGO Low-Code Platform.
// Licensed under the terms specified in the LICENSE file.
//
// SPDX-License-Identifier: Custom-License

package service

import (
	"github.com/worldiety/nago-runner/pkg/run"
	"github.com/worldiety/nago-runner/service/event"
	"log/slog"
	"os"
	"path/filepath"
)

func NewDeleteInstanceData() DeleteInstanceData {
	return func(req event.DeleteInstanceDataRequested) error {

		if err := run.Command("systemctl", "stop", req.Unit); err != nil {
			slog.Warn("failed to stop service, ignoring", "service", req.Unit)
		}

		// TODO we don't have access to the actual systemd configuration here, we blindly delete by convention
		path := filepath.Join("/var/lib/ngr", req.Unit)
		slog.Warn("trying to delete service data dir by convention", "path", path)
		if err := os.RemoveAll(path); err != nil {
			slog.Error("failed to delete service data dir", "path", path)
		} else {
			slog.Info("service data dir deleted", "path", path)
		}

		if err := run.Command("systemctl", "start", req.Unit); err != nil {
			slog.Warn("failed to start service, ignoring", "service", req.Unit)
		}

		return nil
	}
}
