// Copyright (c) 2025 worldiety GmbH
//
// This file is part of the NAGO Low-Code Platform.
// Licensed under the terms specified in the LICENSE file.
//
// SPDX-License-Identifier: Custom-License

package service

import (
	"fmt"
	"github.com/worldiety/nago-runner/pkg/run"
	"github.com/worldiety/nago-runner/service/event"
	"github.com/worldiety/nago-runner/setup"
	"log/slog"
	"net/http"
	"path/filepath"
	"time"
)

func NewDoRestore(settings setup.Settings, bus event.Bus) DoRestore {
	return func(req event.RestoreRequest) error {
		client := http.Client{
			Timeout: time.Minute * 5,
		}

		slog.Info("starting restore", "instance", req.InstanceID, "req", req.ReqID())
		bc := NewBackupClient(&client, settings, req.InstanceID)

		if err := run.Command("systemctl", "stop", req.InstanceID); err != nil {
			slog.Warn("failed to stop service, ignoring", "service", req.InstanceID)
		}

		slog.Info("awaiting service shutdown")
		time.Sleep(time.Second * 15)

		path := filepath.Join("/var/lib/ngr", req.InstanceID)

		slog.Warn("trying to delete service data dir by convention", "path", path)

		if err := DeleteDir(path); err != nil {
			return err
		}

		total := len(req.Data)
		filesProgress := 0
		if req.Exec.Sha3v512 != "" {
			if err := bc.DownloadIntoFile(execPrefix, req.Exec); err != nil {
				return fmt.Errorf("exec restore download failed: %w", err)
			}

			slog.Info("restored exec binary", "file", req.Exec.Name)
			filesProgress++
		}

		for _, file := range req.Data {
			if err := bc.DownloadIntoFile(filepath.Join(dataPrefix, req.InstanceID), file); err != nil {
				slog.Error("failed to restore download data file", "file", file.Name, "err", err)
			}

			filesProgress++
			bus.Publish(event.ProgressUpdated{
				ProgressID: req.ProgressID,
				Percent:    int(float64(filesProgress) / float64(total) * 100),
			})
		}

		if err := run.Command("systemctl", "start", req.InstanceID); err != nil {
			slog.Warn("failed to start service, ignoring", "service", req.InstanceID)
		}

		bus.Publish(event.ProgressUpdated{
			ProgressID: req.ProgressID,
			Percent:    100,
			Finished:   true,
		})

		return nil
	}
}
