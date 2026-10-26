// Copyright (c) 2025 worldiety GmbH
//
// This file is part of the NAGO Low-Code Platform.
// Licensed under the terms specified in the LICENSE file.
//
// SPDX-License-Identifier: Custom-License

package service

import (
	"fmt"
	"github.com/worldiety/nago-runner/configuration"
	"github.com/worldiety/nago-runner/pkg/datadir"
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

		if !configuration.Name(req.InstanceID).Valid() {
			err := fmt.Errorf("invalid instance id: %q", req.InstanceID)
			bus.Publish(event.ProgressUpdated{
				ProgressID: req.ProgressID,
				Error:      err.Error(),
			})
			return err
		}

		slog.Info("starting restore", "instance", req.InstanceID, "req", req.ReqID())
		bc := NewBackupClient(&client, settings, req.InstanceID)

		if err := run.Command("systemctl", "stop", req.InstanceID); err != nil {
			slog.Warn("failed to stop service, ignoring", "service", req.InstanceID)
		}

		slog.Info("awaiting service shutdown")
		time.Sleep(time.Second * 15)

		path := filepath.Join("/var/lib/ngr", req.InstanceID)

		// security note: the restore may fail or be partial, thus never delete the current data
		slog.Warn("moving current service data dir into quarantine by convention", "path", path)

		if _, err := datadir.Quarantine(path); err != nil {
			err = fmt.Errorf("failed to quarantine dir: %w", err)
			bus.Publish(event.ProgressUpdated{
				ProgressID: req.ProgressID,
				Error:      err.Error(),
			})
			return err
		}

		total := len(req.Data)
		filesProgress := 0
		if req.Exec.Sha3v512 != "" {
			if err := bc.DownloadIntoFile(execPrefix, req.Exec); err != nil {
				err = fmt.Errorf("exec restore download failed: %w", err)
				bus.Publish(event.ProgressUpdated{
					ProgressID: req.ProgressID,
					Error:      err.Error(),
				})

				return err
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
