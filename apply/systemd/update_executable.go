// Copyright (c) 2025 worldiety GmbH
//
// This file is part of the NAGO Low-Code Platform.
// Licensed under the terms specified in the LICENSE file.
//
// SPDX-License-Identifier: Custom-License

package systemd

import (
	"context"
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

	if err := download(logger, uri, settings.Token, cfg.Executable, paths.ExecFilename); err != nil {
		return false, err
	}

	return true, nil
}

var (
	// downloadStallTimeout fails a download which makes no progress at all. There is deliberately no limit for
	// the entire download, because large executables on slow links would otherwise never arrive.
	downloadStallTimeout = 60 * time.Second
	// downloadHeaderTimeout limits the wait for the response of the hub.
	downloadHeaderTimeout = 60 * time.Second
)

// download fetches the executable into a unique temporary file next to the target, verifies size and hash and
// renames it over the target. The temporary file is removed on any failure.
func download(logger *slog.Logger, uri string, token string, want configuration.Executable, target string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return fmt.Errorf("error creating http request for executable: %s: %w", uri, err)
	}

	// send our bearer secret to authorize us properly at the remote side
	req.Header.Add("Authorization", "Bearer "+token)

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.ResponseHeaderTimeout = downloadHeaderTimeout
	client := &http.Client{Transport: transport}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error executing http request for executable: %s: %w", uri, err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected http response when downloading executable: %s: %s", resp.Status, uri)
	}

	dir := filepath.Dir(target)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		_ = os.MkdirAll(dir, 0755)
	}

	// a unique name, because a stale file of an interrupted download must never be mixed into this one
	w, err := os.CreateTemp(dir, filepath.Base(target)+".*.tmp")
	if err != nil {
		return fmt.Errorf("error creating tmp file in %s: %w", dir, err)
	}

	tmpFile := w.Name()
	committed := false
	defer func() {
		if !committed {
			_ = w.Close()
			_ = os.Remove(tmpFile)
		}
	}()

	guard := newStallGuard(resp.Body, downloadStallTimeout, cancel)
	defer guard.Stop()

	downloadStart := time.Now()
	// never accept more than announced, a broken response must not fill the disk
	n, err := io.Copy(w, io.LimitReader(guard, want.Size+1))
	if err != nil {
		return fmt.Errorf("error downloading executable after %d bytes: %s: %w", n, uri, err)
	}

	logger.Info("downloaded executable", "size", n, "took", time.Since(downloadStart))

	if err := w.Close(); err != nil {
		return fmt.Errorf("error comitting/closing tmp file: %s: %w", tmpFile, err)
	}

	if n != want.Size {
		return fmt.Errorf("executable size mismatch: got %d, want %d", n, want.Size)
	}

	downloadedHash, err := linux.Sha3(tmpFile)
	if err != nil {
		return fmt.Errorf("error hashing downloaded executable: %s: %w", tmpFile, err)
	}

	if downloadedHash != want.Hash {
		return fmt.Errorf("executable hash mismatch for download: got %s, want %s", downloadedHash, want.Hash)
	}

	// set the bit before the rename, a restart in between must never find a non-executable file
	if err := os.Chmod(tmpFile, 0755); err != nil {
		return fmt.Errorf("cannot set executable bit: %w", err)
	}

	if err := os.Rename(tmpFile, target); err != nil {
		return fmt.Errorf("error renaming executable: %s: %w", tmpFile, err)
	}

	committed = true
	return nil
}

// stallGuard calls cancel, if the wrapped reader delivers no data for the given duration.
type stallGuard struct {
	r       io.Reader
	timeout time.Duration
	timer   *time.Timer
}

func newStallGuard(r io.Reader, timeout time.Duration, cancel func()) *stallGuard {
	return &stallGuard{r: r, timeout: timeout, timer: time.AfterFunc(timeout, cancel)}
}

func (g *stallGuard) Read(p []byte) (int, error) {
	n, err := g.r.Read(p)
	if n > 0 {
		g.timer.Reset(g.timeout)
	}

	return n, err
}

func (g *stallGuard) Stop() {
	g.timer.Stop()
}
