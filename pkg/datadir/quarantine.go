// Copyright (c) 2025 worldiety GmbH
//
// This file is part of the NAGO Low-Code Platform.
// Licensed under the terms specified in the LICENSE file.
//
// SPDX-License-Identifier: Custom-License

package datadir

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

// QuarantineDirName is the directory, which is created next to the actual data directory and which keeps all
// data directories which have been removed from service. Nothing in the runner ever deletes from it, so
// recovering or purging the data is always a manual decision by an operator.
const QuarantineDirName = ".quarantine"

// Quarantine moves the given data directory out of the way instead of deleting it. If path is a symlink
// (which systemd creates for StateDirectory in combination with DynamicUser, pointing into /var/lib/private),
// the resolved directory is moved and the link is removed afterward. The quarantine directory is always a
// sibling of the resolved directory, so that the rename stays on the same file system and is atomic.
// It returns the new location or the empty string, if there was nothing to move.
func Quarantine(path string) (string, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}

	if err != nil {
		return "", fmt.Errorf("cannot stat data dir: %s: %w", path, err)
	}

	isLink := info.Mode()&fs.ModeSymlink != 0
	resolved := path
	if isLink {
		resolved, err = filepath.EvalSymlinks(path)
		if errors.Is(err, fs.ErrNotExist) {
			// dangling link, there is no data to protect
			return "", os.Remove(path)
		}

		if err != nil {
			return "", fmt.Errorf("cannot resolve data dir link: %s: %w", path, err)
		}
	}

	qDir := filepath.Join(filepath.Dir(resolved), QuarantineDirName)
	if err := os.MkdirAll(qDir, 0700); err != nil {
		return "", fmt.Errorf("cannot create quarantine dir: %s: %w", qDir, err)
	}

	dst := filepath.Join(qDir, filepath.Base(resolved)+"-"+time.Now().UTC().Format("20060102T150405.000000000Z"))
	if err := os.Rename(resolved, dst); err != nil {
		return "", fmt.Errorf("cannot move data dir into quarantine: %s -> %s: %w", resolved, dst, err)
	}

	if isLink {
		if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
			// the data is already safe, a stale link is just cosmetic
			slog.Warn("cannot remove data dir link after quarantine", "path", path, "err", err.Error())
		}
	}

	slog.Warn("moved data dir into quarantine", "path", path, "quarantine", dst)
	return dst, nil
}
