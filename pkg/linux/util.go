// Copyright (c) 2025 worldiety GmbH
//
// This file is part of the NAGO Low-Code Platform.
// Licensed under the terms specified in the LICENSE file.
//
// SPDX-License-Identifier: Custom-License

package linux

import (
	"bytes"
	"crypto/sha3"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/worldiety/nago-runner/configuration"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
)

func EqualJSON[T any](path string, other T) bool {
	buf, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return false
	}

	if err != nil {
		slog.Error("failed to read file to compare as json object", "file", path, "err", err.Error())
		return false
	}

	var obj T
	if err := json.Unmarshal(buf, &obj); err != nil {
		slog.Error("failed to unmarshal json to compare as json object", "file", path, "err", err.Error())
		return false
	}

	return reflect.DeepEqual(obj, other)
}

func EqualBuf(path string, other []byte) bool {
	buf, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return false
	}

	if err != nil {
		slog.Error("failed to read file to compare buf", "file", path, "err", err.Error())
		return false
	}

	return bytes.Equal(buf, other)
}

func WriteJSON[T any](path string, obj T) error {
	buf, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal json: %w", err)
	}

	// TODO write in tmp and do atomic rename
	if err := os.WriteFile(path, buf, 0644); err != nil {
		return fmt.Errorf("failed to write json: %w", err)
	}

	return nil
}

func Sha3Bytes(buf []byte) (configuration.Sha3V512, error) {
	h := sha3.New512()
	_, _ = h.Write(buf)
	return configuration.Sha3V512(hex.EncodeToString(h.Sum(nil))), nil
}

func Sha3(file string) (configuration.Sha3V512, error) {
	r, err := os.Open(file)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}

		return "", fmt.Errorf("failed to open file: %s: %w", file, err)
	}

	defer r.Close()

	h := sha3.New512()
	if _, err := io.Copy(h, r); err != nil {
		return "", fmt.Errorf("failed to hash file: %s: %w", file, err)
	}

	return configuration.Sha3V512(hex.EncodeToString(h.Sum(nil))), nil
}

func WriteFile(name string, data []byte, perm os.FileMode) error {
	if _, err := os.Stat(filepath.Dir(name)); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(name), 0755); err != nil {
			return err
		}
	}

	tmpF := name + ".tmp"

	if err := os.WriteFile(tmpF, data, perm); err != nil {
		_ = os.Remove(tmpF)
		return err
	}

	if err := os.Rename(tmpF, name); err != nil {
		_ = os.Remove(tmpF)
		return err
	}

	return nil
}
