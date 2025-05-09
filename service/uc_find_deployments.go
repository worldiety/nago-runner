// Copyright (c) 2025 worldiety GmbH
//
// This file is part of the NAGO Low-Code Platform.
// Licensed under the terms specified in the LICENSE file.
//
// SPDX-License-Identifier: Custom-License

package service

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

const (
	systemdServiceDir       = "/etc/systemd/system/"
	nagoRunnerServicePrefix = "ngr-"
)

func NewFindDeployments() FindDeployments {
	return func() ([]Deployment, error) {
		files, err := os.ReadDir(systemdServiceDir)
		if err != nil {
			return nil, fmt.Errorf("cannot read systemd service directory: %s: %w", systemdServiceDir, err)
		}

		var res []Deployment
		for _, file := range files {
			if !strings.HasPrefix(file.Name(), nagoRunnerServicePrefix) || !strings.HasSuffix(file.Name(), ".service") {
				continue
			}

			buf, err := os.ReadFile(filepath.Join(systemdServiceDir, file.Name()))
			if err != nil {
				slog.Error("cannot read file", "file", file.Name(), "err", err.Error())
				continue
			}

			for line := range strings.Lines(string(buf)) {
				if strings.HasPrefix(strings.TrimSpace(line), "Description") {
					jsonStr := line[strings.Index(line, "="):]
					var d Deployment
					if err := json.Unmarshal([]byte(jsonStr), &d); err != nil {
						slog.Error("cannot unmarshal deployment from service description", "file", file.Name(), "line", line, "err", err.Error())
						break
					}

					res = append(res, d)
					break
				}
			}
		}

		return res, nil

	}
}
