// Copyright (c) 2025 worldiety GmbH
//
// This file is part of the NAGO Low-Code Platform.
// Licensed under the terms specified in the LICENSE file.
//
// SPDX-License-Identifier: Custom-License

package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"

	"github.com/worldiety/nago-runner/setup"
)

// hasBatchSize is the amount of hashes sent per existence check.
const hasBatchSize = 500

// BackupClientV2 speaks the v2 backup protocol: begin, has (batched), blob upload, commit. Blobs are
// deduplicated per instance by the hub.
type BackupClientV2 struct {
	client     *http.Client
	settings   setup.Settings
	instanceID string
}

func NewBackupClientV2(client *http.Client, settings setup.Settings, instanceID string) *BackupClientV2 {
	return &BackupClientV2{client: client, settings: settings, instanceID: instanceID}
}

func (c *BackupClientV2) postJSON(path string, in any, out any) error {
	buf, err := json.Marshal(in)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.settings.Endpoints().Http(path), bytes.NewReader(buf))
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
		msg, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
		return fmt.Errorf("%s: http status %d: %s", path, res.StatusCode, string(msg))
	}

	if out == nil {
		return nil
	}

	if err := json.NewDecoder(res.Body).Decode(out); err != nil {
		return fmt.Errorf("failed to decode http response: %w", err)
	}

	return nil
}

func (c *BackupClientV2) Begin() (string, error) {
	var res struct {
		ID string `json:"id"`
	}

	err := c.postJSON("api/v2/backup/begin", map[string]string{"instanceId": c.instanceID}, &res)
	return res.ID, err
}

// HasBlobs returns the set of hashes which the hub already stores for this instance.
func (c *BackupClientV2) HasBlobs(hashes []Sha3V512) (map[Sha3V512]bool, error) {
	existing := map[Sha3V512]bool{}
	for start := 0; start < len(hashes); start += hasBatchSize {
		end := min(start+hasBatchSize, len(hashes))

		var res struct {
			Existing []Sha3V512 `json:"existing"`
		}

		req := map[string]any{"instanceId": c.instanceID, "hashes": hashes[start:end]}
		if err := c.postJSON("api/v2/backup/has", req, &res); err != nil {
			return nil, err
		}

		for _, h := range res.Existing {
			existing[h] = true
		}
	}

	return existing, nil
}

// Upload streams the file to the hub, which calculates the final hash. The file may be mutated while being
// read, thus the returned hash is authoritative.
func (c *BackupClientV2) Upload(fsys fs.FS, filename string) (FileStoreResult, error) {
	f, err := fsys.Open(filename)
	if err != nil {
		return FileStoreResult{}, fmt.Errorf("failed to open file %s: %w", filename, err)
	}

	defer f.Close()

	req, err := http.NewRequest(http.MethodPost, c.settings.Endpoints().Http("api/v2/backup/blob?instance="+url.QueryEscape(c.instanceID)), f)
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
		msg, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
		return FileStoreResult{}, fmt.Errorf("blob upload: http status %d: %s", res.StatusCode, string(msg))
	}

	var result FileStoreResult
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return FileStoreResult{}, fmt.Errorf("failed to decode http response: %w", err)
	}

	return result, nil
}

func (c *BackupClientV2) Commit(id string, exec File, data []File) error {
	req := map[string]any{
		"id":         id,
		"instanceId": c.instanceID,
		"exec":       exec,
		"data":       data,
	}

	return c.postJSON("api/v2/backup/commit", req, nil)
}

// BlobURL returns the v2 download url of a blob of the given instance.
func (c *BackupClientV2) BlobURL(backupInstance string, hash Sha3V512) string {
	return c.settings.Endpoints().Http("api/v2/backup/blob?instance=" + url.QueryEscape(backupInstance) + "&hash=" + url.QueryEscape(string(hash)))
}
