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
	"github.com/worldiety/nago-runner/setup"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	execPrefix = "/opt/ngr/"
	dataPrefix = "/var/lib/ngr/"
)

func NewDoBackup(settings setup.Settings, bus event.Bus) DoBackup {
	return func(req event.BackupRequest) error {
		client := http.Client{
			Timeout: time.Minute * 5,
		}

		slog.Info("starting backup", "instance", req.InstanceID, "req", req.ReqID())
		bc := NewBackupClient(&client, settings, req.InstanceID)

		backup := Backup{
			InstanceID: req.InstanceID,
		}

		var errs []error
		file, err := bc.BackupFile(os.DirFS(execPrefix), req.InstanceID)
		if err != nil {
			slog.Error("failed to backup exec file", "file", req.InstanceID, "err", err.Error())
			errs = append(errs, err) // do not fail entirely
		} else {
			backup.Exec = file
		}

		dataDir := filepath.Join(dataPrefix, req.InstanceID)
		if _, err := os.Stat(dataDir); os.IsNotExist(err) {
			slog.Warn("data dir does not exist", "dir", dataDir)
		}
		fsys := os.DirFS(dataDir)
		filesCount := countFiles(fsys) + 2 // 1 for executable and 1 for the commit
		filesProgress := 0

		err = fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				slog.Error("failed to walk dir", "path", path, "err", err.Error())
				return err
			}

			if !d.Type().IsRegular() {
				return nil
			}

			f, err := bc.BackupFile(fsys, path)
			if err != nil {
				slog.Error("failed to backup file", "file", path, "err", err.Error())
				errs = append(errs, err) // do not fail entirely
				return nil
			}

			backup.Data = append(backup.Data, f)

			filesProgress++
			bus.Publish(event.ProgressUpdated{
				ProgressID: req.ProgressID,
				Percent:    int(float64(filesProgress) / float64(filesCount) * 100),
			})

			return nil
		})

		if err != nil {
			slog.Error("failed to backup data dir", "dir", dataDir, "err", err.Error())
		}

		if err := bc.CommitBackup(backup); err != nil {
			slog.Error("failed to commit backup", "err", err.Error())
		} else {
			slog.Info("backup completed", "instance", req.InstanceID, "errors", len(errs))
		}

		if len(errs) > 0 {
			return fmt.Errorf("errors (%d) occured during backup: %w", len(errs), errs[0])
		}

		bus.Publish(event.ProgressUpdated{
			ProgressID: req.ProgressID,
			Percent:    100,
			Finished:   true,
		})

		return nil
	}
}

func countFiles(fsys fs.FS) int {
	count := 0
	err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			slog.Error("failed to walk dir", "path", path, "err", err.Error())
			return err
		}

		if !d.Type().IsRegular() {
			return nil
		}

		count++

		return nil
	})

	if err != nil {
		slog.Error("failed to count files in dir", "err", err.Error())
	}

	return count
}
