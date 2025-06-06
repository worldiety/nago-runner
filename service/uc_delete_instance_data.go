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
	"path"
	"path/filepath"
	"time"
)

func NewDeleteInstanceData() DeleteInstanceData {
	return func(req event.DeleteInstanceDataRequested) error {

		if err := run.Command("systemctl", "stop", req.Unit); err != nil {
			slog.Warn("failed to stop service, ignoring", "service", req.Unit)
		}

		slog.Info("awaiting service shutdown")
		time.Sleep(time.Second * 15)

		// TODO we don't have access to the actual systemd configuration here, we blindly delete by convention
		path := filepath.Join("/var/lib/ngr", req.Unit)

		slog.Warn("trying to delete service data dir by convention", "path", path)

		if err := DeleteDir(path); err != nil {
			return err
		}

		if err := run.Command("systemctl", "start", req.Unit); err != nil {
			slog.Warn("failed to start service, ignoring", "service", req.Unit)
		}

		return nil
	}
}

func DeleteDir(path string) error {
	if resolved, err := resolveLink(path); err == nil {
		slog.Info("resolved data sym link", "path", path, "resolved", resolved)
		if err := os.RemoveAll(resolved); err != nil {
			slog.Error("failed to delete service data dir", "resolved", resolved)
		} else {
			slog.Info("service data dir deleted", "resolved", resolved)
		}
	}

	if err := os.RemoveAll(path); err != nil {
		slog.Error("failed to delete service data dir", "path", path)
	} else {
		slog.Info("service data dir deleted", "path", path)
	}

	return nil
}

func resolveLink(symLink string) (string, error) {
	resolvedLink, err := os.Readlink(symLink)
	if err != nil {
		return "", err
	}
	if !filepath.IsAbs(resolvedLink) { // Output of os.Readlink is OS-dependent...
		resolvedLink = path.Join(path.Dir(symLink), resolvedLink)
	}

	return resolvedLink, nil
}
