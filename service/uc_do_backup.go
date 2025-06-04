// Copyright (c) 2025 worldiety GmbH
//
// This file is part of the NAGO Low-Code Platform.
// Licensed under the terms specified in the LICENSE file.
//
// SPDX-License-Identifier: Custom-License

package service

import (
	"bytes"
	"crypto/sha3"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/worldiety/nago-runner/service/event"
	"github.com/worldiety/nago-runner/setup"
	"io"
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

func NewDoBackup(settings setup.Settings) DoBackup {
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

		return nil
	}
}

type BackupClient struct {
	client     *http.Client
	settings   setup.Settings
	instanceId string
}

func NewBackupClient(client *http.Client, settings setup.Settings, instanceId string) *BackupClient {
	return &BackupClient{client: client, settings: settings, instanceId: instanceId}
}

func (c *BackupClient) BackupFile(fsys fs.FS, filename string) (File, error) {
	slog.Info("backup file", "filename", filename, "instance", c.instanceId)
	stat, err := fs.Stat(fsys, filename)
	if err != nil {
		return File{}, fmt.Errorf("failed to stat file %s: %w", filename, err)
	}

	hash, n, err := sha3v512File(fsys, filename)
	if err != nil {
		return File{}, fmt.Errorf("failed to calculate sha3 hash: %w", err)
	}

	ok, err := c.hasBlob(hash)
	if err != nil {
		return File{}, fmt.Errorf("failed to check blob existence: %w", err)
	}

	if ok {
		slog.Info("backup file already exists at remote", "filename", filename, "hash", hash, "instance", c.instanceId)

		return File{
			Hash:         hash,
			Size:         n,
			LastModified: stat.ModTime(),
			UploadedAt:   time.Now(),
			Mode:         stat.Mode(),
			Name:         filename,
		}, nil
	}

	file, err := c.uploadRemote(fsys, filename)
	if err != nil {
		return File{}, fmt.Errorf("failed to upload file %s: %w", filename, err)
	}

	if hash != file.Hash {
		slog.Warn("backup file changed while in transit", "file", filename, "hash", hash, "server-hash", file.Hash)
	}

	slog.Info("backup file successfully uploaded", "filename", filename, "hash", hash, "instance", c.instanceId)

	return File{
		Hash:       file.Hash, // note, that this is what has been stored
		Size:       file.Size,
		UploadedAt: time.Now(),
		Mode:       stat.Mode(),
		Name:       filename,
	}, nil
}

func (c *BackupClient) hasBlob(sha3 Sha3V512) (bool, error) {
	url := c.settings.Endpoints().Http("api/v1/backup/blob/exists?hash=" + string(sha3))
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create http request: %w", err)
	}

	req.Header.Add("Authorization", "Bearer "+c.settings.Token)
	resp, err := c.client.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to execute http request: %w", err)
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("failed to execute http request: http status %d", resp.StatusCode)
	}

	var res struct {
		Exists bool `json:"exists"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return false, fmt.Errorf("failed to decode http response: %w", err)
	}

	return res.Exists, nil
}

func (c *BackupClient) uploadRemote(fsys fs.FS, filename string) (FileStoreResult, error) {
	f, err := fsys.Open(filename)
	if err != nil {
		return FileStoreResult{}, fmt.Errorf("failed to open file %s: %w", filename, err)
	}

	defer f.Close()

	url := c.settings.Endpoints().Http("api/v1/backup/blob/upload")
	req, err := http.NewRequest(http.MethodPost, url, f)
	if err != nil {
		return FileStoreResult{}, fmt.Errorf("failed to create http request: %w", err)
	}

	req.Header.Add("Authorization", "Bearer "+c.settings.Token)
	res, err := c.client.Do(req)
	if err != nil {
		return FileStoreResult{}, fmt.Errorf("failed to execute http request: %w", err)
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return FileStoreResult{}, fmt.Errorf("failed to execute http request: http status %d", res.StatusCode)
	}

	var result FileStoreResult
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return FileStoreResult{}, fmt.Errorf("failed to decode http response: %w", err)
	}

	return result, nil
}

func (c *BackupClient) CommitBackup(backup Backup) error {
	buf, err := json.Marshal(backup)
	if err != nil {
		return fmt.Errorf("failed to marshal backup: %w", err)
	}

	url := c.settings.Endpoints().Http("api/v1/backup/create")

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(buf))
	if err != nil {
		return fmt.Errorf("failed to create http request: %w", err)
	}

	req.Header.Add("Authorization", "Bearer "+c.settings.Token)
	req.Header.Add("Content-Type", "application/json")
	res, err := c.client.Do(req)

	if err != nil {
		return fmt.Errorf("failed to execute http request: %w", err)
	}

	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to execute http request: http status %d", res.StatusCode)
	}

	return nil
}

func sha3v512File(fsys fs.FS, filename string) (Sha3V512, int64, error) {
	f, err := fsys.Open(filename)
	if err != nil {
		return "", 0, err
	}

	defer f.Close()

	hasher := sha3.New512()
	n, err := io.Copy(hasher, f)
	if err != nil {
		return "", 0, fmt.Errorf("failed to hash file: %w", err)
	}

	return Sha3V512(hex.EncodeToString(hasher.Sum(nil))), n, err
}

// Sha3V512 is the hex encoded sha3 512 hashcode.
type Sha3V512 string

type FileStoreResult struct {
	Size int64
	Hash Sha3V512
}
type File struct {
	Hash         Sha3V512    `json:"hash,omitempty"`
	Size         int64       `json:"size,omitempty"`
	LastModified time.Time   `json:"lastModified"`
	UploadedAt   time.Time   `json:"uploadedAt"`
	Mode         os.FileMode `json:"mode,omitempty"`
	// Name is always relative to the backup root and contains all relevant directory fragments.
	// E.g. /var/lib/ngr/123456/files/a/b/cdefg.bin becomes files/a/b/cdefg.bin
	Name string `json:"name,omitempty"`
}

type Backup struct {
	InstanceID string `json:"instanceId,omitempty"`

	// Backup-Root is <runner>/opt/ngr/ which is implicit and may change between runner versions.
	Exec File `json:"exec,omitempty"`

	// Backup-Root is <runner>/var/lib/ngr/<instance id> which is implicit and may change between runner versions.
	Data []File `json:"data,omitempty"`
}
