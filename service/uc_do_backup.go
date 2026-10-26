// Copyright (c) 2025 worldiety GmbH
//
// This file is part of the NAGO Low-Code Platform.
// Licensed under the terms specified in the LICENSE file.
//
// SPDX-License-Identifier: Custom-License

package service

import (
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/worldiety/nago-runner/service/event"
	"github.com/worldiety/nago-runner/setup"
)

const (
	execPrefix = "/opt/ngr/"
	dataPrefix = "/var/lib/ngr/"
)

// NewDoBackup implements the v2 backup protocol:
//  1. begin a pending backup at the hub
//  2. hash all files locally and ask the hub in batches which blobs are missing
//  3. upload only the missing blobs, the hub returns the authoritative hash of the stored content
//  4. commit the manifest
func NewDoBackup(settings setup.Settings, bus event.Bus) DoBackup {
	return func(req event.BackupRequest) error {
		client := http.Client{
			Timeout: time.Hour, // a single large database file may take a while
		}

		fail := func(err error) error {
			slog.Error("backup failed", "instance", req.InstanceID, "err", err.Error())
			bus.Publish(event.ProgressUpdated{ProgressID: req.ProgressID, Error: err.Error()})
			return err
		}

		progress := func(percent int) {
			bus.Publish(event.ProgressUpdated{ProgressID: req.ProgressID, Percent: percent})
		}

		slog.Info("starting backup", "instance", req.InstanceID, "req", req.ReqID())
		bc := NewBackupClientV2(&client, settings, req.InstanceID)

		id, err := bc.Begin()
		if err != nil {
			return fail(fmt.Errorf("cannot begin backup: %w", err))
		}

		// collect and hash everything first, so that we can ask the hub in batches
		type candidate struct {
			fsys fs.FS
			name string
			file File
		}

		var exec *candidate
		if f, err := statAndHash(os.DirFS(execPrefix), req.InstanceID); err != nil {
			slog.Error("failed to hash exec file", "file", req.InstanceID, "err", err.Error())
		} else {
			exec = &candidate{fsys: os.DirFS(execPrefix), name: req.InstanceID, file: f}
		}

		dataDir := filepath.Join(dataPrefix, req.InstanceID)
		if _, err := os.Stat(dataDir); os.IsNotExist(err) {
			slog.Warn("data dir does not exist", "dir", dataDir)
		}

		fsys := os.DirFS(dataDir)
		var data []candidate
		var errs []error
		err = fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				slog.Error("failed to walk dir", "path", path, "err", err.Error())
				errs = append(errs, err)
				return nil
			}

			if !d.Type().IsRegular() {
				return nil
			}

			f, err := statAndHash(fsys, path)
			if err != nil {
				slog.Error("failed to hash file", "file", path, "err", err.Error())
				errs = append(errs, err)
				return nil
			}

			data = append(data, candidate{fsys: fsys, name: path, file: f})
			return nil
		})

		if err != nil {
			return fail(fmt.Errorf("cannot walk data dir: %w", err))
		}

		all := data
		if exec != nil {
			all = append([]candidate{*exec}, data...)
		}

		hashes := make([]Sha3V512, 0, len(all))
		for _, c := range all {
			hashes = append(hashes, c.file.Hash)
		}

		existing, err := bc.HasBlobs(hashes)
		if err != nil {
			return fail(fmt.Errorf("cannot check blobs: %w", err))
		}

		slog.Info("backup hashed files", "instance", req.InstanceID, "files", len(all), "existing", len(existing))

		// upload missing blobs
		total := len(all) + 1 // +1 for the commit
		for i := range all {
			c := &all[i]
			if existing[c.file.Hash] {
				progress(i * 100 / total)
				continue
			}

			res, err := bc.Upload(c.fsys, c.name)
			if err != nil {
				slog.Error("failed to upload file", "file", c.name, "err", err.Error())
				errs = append(errs, err)
				continue
			}

			if res.Hash != c.file.Hash {
				slog.Warn("backup file changed while in transit", "file", c.name, "hash", c.file.Hash, "server-hash", res.Hash)
			}

			// what the hub has stored is authoritative
			c.file.Hash = res.Hash
			c.file.Size = res.Size
			c.file.UploadedAt = time.Now()
			progress(i * 100 / total)
		}

		var execFile File
		var dataFiles []File
		for i, c := range all {
			if exec != nil && i == 0 {
				execFile = c.file
				continue
			}

			dataFiles = append(dataFiles, c.file)
		}

		if err := bc.Commit(id, execFile, dataFiles); err != nil {
			return fail(fmt.Errorf("cannot commit backup: %w", err))
		}

		slog.Info("backup completed", "instance", req.InstanceID, "backup", id, "files", len(all), "errors", len(errs))

		if len(errs) > 0 {
			// the backup is committed but incomplete, make that visible
			err := fmt.Errorf("backup committed with %d errors, first: %w", len(errs), errs[0])
			bus.Publish(event.ProgressUpdated{ProgressID: req.ProgressID, Error: err.Error()})
			return err
		}

		bus.Publish(event.ProgressUpdated{ProgressID: req.ProgressID, Percent: 100, Finished: true})
		return nil
	}
}

func statAndHash(fsys fs.FS, filename string) (File, error) {
	stat, err := fs.Stat(fsys, filename)
	if err != nil {
		return File{}, fmt.Errorf("failed to stat file %s: %w", filename, err)
	}

	hash, n, err := sha3v512File(fsys, filename)
	if err != nil {
		return File{}, fmt.Errorf("failed to hash file %s: %w", filename, err)
	}

	return File{
		Hash:         hash,
		Size:         n,
		LastModified: stat.ModTime(),
		Mode:         stat.Mode(),
		Name:         filename,
	}, nil
}
