// Copyright (c) 2025 worldiety GmbH
//
// This file is part of the NAGO Low-Code Platform.
// Licensed under the terms specified in the LICENSE file.
//
// SPDX-License-Identifier: Custom-License

package systemd

import (
	"fmt"
	"github.com/worldiety/nago-runner/configuration"
	"github.com/worldiety/nago-runner/pkg/linux"
	"github.com/worldiety/nago-runner/setup"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// updateExecutable inspects the declared executable artifacts and creates or replaces any existing
// executable with the given version or does nothing if it already matches (returns false and no error).
func updateExecutable(logger *slog.Logger, settings setup.Settings, cfg configuration.Application) (bool, error) {
	if !configuration.Name(cfg.InstID).Valid() {
		return false, fmt.Errorf("invalid systemd unit name: %s", cfg.InstID)
	}

	fakeService := NewService(cfg.InstID)
	paths := fakeService.Paths()
	hash, err := linux.Sha3(paths.ExecFilename)
	if err != nil {
		return false, fmt.Errorf("error hashing executable: %s", paths.ExecFilename)
	}

	if hash == cfg.Executable.Hash {
		logger.Info("executable is unchanged", "expected", hash)
		return false, nil
	}

	logger.Info("executable hash is different", "expected", cfg.Executable.Hash, "got", hash)

	uri := string(cfg.Executable.URL)
	if !strings.HasPrefix(uri, "http") {
		uri = settings.Endpoints().Http(uri)
	}

	client := &http.Client{
		Timeout: 60 * time.Second,
	}

	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		return false, fmt.Errorf("error creating http request for executable: %s", uri)
	}

	// send our bearer secret to authorize us properly at the remote side
	req.Header.Add("Authorization", "Bearer "+settings.Token)

	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("error executing http request for executable: %s", uri)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("unexpected http response when downloading executable: %s: %s", resp.Status, uri)
	}

	tmpFile := paths.ExecFilename + ".tmp"
	if _, err := os.Stat(filepath.Dir(paths.ExecFilename)); os.IsNotExist(err) {
		_ = os.MkdirAll(filepath.Dir(paths.ExecFilename), 0755)
	}

	w, err := os.OpenFile(tmpFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return false, fmt.Errorf("error opening tmp file: %s", tmpFile)
	}

	downloadStart := time.Now()
	n, err := io.Copy(w, resp.Body)
	if err != nil {
		_ = w.Close()
		return false, fmt.Errorf("error downloading executable: %s", uri)
	}

	slog.Info("downloaded executable", "size", n, "took", time.Since(downloadStart))

	if err := w.Close(); err != nil {
		return false, fmt.Errorf("error comitting/closing tmp file: %s", tmpFile)
	}

	if n != cfg.Executable.Size {
		return false, fmt.Errorf("executable size mismatch: got %d, want %d", n, cfg.Executable.Size)
	}

	downloadedHash, err := linux.Sha3(tmpFile)
	if err != nil {
		return false, fmt.Errorf("error hashing downloaded executable: %s", tmpFile)
	}

	if downloadedHash != cfg.Executable.Hash {
		return false, fmt.Errorf("executable hash mismatch for download: got %s, want %s", downloadedHash, cfg.Executable.Hash)
	}

	if err := os.Rename(tmpFile, paths.ExecFilename); err != nil {
		return false, fmt.Errorf("error renaming executable: %s", tmpFile)
	}

	if err := os.Chmod(paths.ExecFilename, 0755); err != nil {
		return false, fmt.Errorf("cannot set executable bit: %w", err)
	}

	return true, nil
}
