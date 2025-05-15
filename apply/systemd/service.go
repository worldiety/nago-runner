// Copyright (c) 2025 worldiety GmbH
//
// This file is part of the NAGO Low-Code Platform.
// Licensed under the terms specified in the LICENSE file.
//
// SPDX-License-Identifier: Custom-License

package systemd

import (
	"encoding/json"
	"fmt"
	"github.com/worldiety/nago-runner/configuration"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// Service represents a systemd service unit on disk.
type Service struct {
	UnitFilename  string
	Configuration configuration.ServiceUnit
	Managed       bool
}

// NewService creates a managed instance without a configuration.
func NewService(name string) Service {
	return Service{
		UnitFilename: filepath.Join(systemdConfDir, name+".service"),
		Managed:      true,
	}
}

// ParseService inspects a given service file and loads the embedded configuration from it if it is managed.
func ParseService(logger *slog.Logger, filename string) (Service, error) {
	buf, err := os.ReadFile(filename)
	if err != nil {
		return Service{}, fmt.Errorf("cannot read systemd conf file: %s: %w", filename, err)
	}

	if !utf8.Valid(buf) {
		return Service{}, fmt.Errorf("invalid non-utf8 systemd conf file: %s", filename)
	}

	res := Service{UnitFilename: filename}
	for line := range strings.Lines(string(buf)) {
		if strings.HasPrefix(line, ngrMetaPrefix) {
			var tmp configuration.ServiceUnit
			if err := json.Unmarshal([]byte(line[len(ngrMetaPrefix):]), &tmp); err != nil {
				logger.Error("failed to parse managed systemd conf file: %s: %s", filename, err)
				break
			}
			res.Managed = true
			res.Configuration = tmp
		}
	}

	return res, nil
}

func (s Service) Name() string {
	return strings.TrimSuffix(strings.ToLower(filepath.Base(s.UnitFilename)), ".service")
}

func (s Service) Paths() Paths {
	p := Paths{
		DataDirectory: "/var/lib/ngr/" + s.Name(),
		ExecFilename:  "/opt/ngr/" + s.Name(),
	}

	if s.Configuration.Service.ExecStart.Cmd != "" {
		p.ExecFilename = s.Configuration.Service.ExecStart.Cmd
	}

	if s.Configuration.Service.StateDirectory != "" {
		// state directory must be relative and is placed inside /var/lib/
		p.DataDirectory = filepath.Join("/var/lib/", s.Configuration.Service.StateDirectory)
	}

	return p
}

type Paths struct {
	DataDirectory string
	ExecFilename  string
}
