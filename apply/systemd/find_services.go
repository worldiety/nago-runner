// Copyright (c) 2025 worldiety GmbH
//
// This file is part of the NAGO Low-Code Platform.
// Licensed under the terms specified in the LICENSE file.
//
// SPDX-License-Identifier: Custom-License

package systemd

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// FindServices inspects all available systemd system service files.
func FindServices(logger *slog.Logger) ([]Service, error) {
	files, err := os.ReadDir(systemdConfDir)
	if err != nil {
		return nil, fmt.Errorf("cannot find systemd conf files: %s: %w", systemdConfDir, err)
	}

	var units []Service
	for _, file := range files {
		if !file.Type().IsRegular() || !strings.HasSuffix(file.Name(), ".service") {
			continue
		}

		unit, err := ParseService(logger, filepath.Join(systemdConfDir, file.Name()))
		if err != nil {
			return nil, fmt.Errorf("cannot parse systemd conf file: %s: %w", file.Name(), err)
		}

		units = append(units, unit)
	}

	return units, nil
}

const (
	ngrMetaPrefix = "# ngr-meta: "
)
