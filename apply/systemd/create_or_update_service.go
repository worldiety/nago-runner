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
	"github.com/worldiety/nago-runner/setup"
	"log/slog"
	"path/filepath"
)

func createOrUpdateService(logger *slog.Logger, settings setup.Settings, cfg configuration.Application) (Service, bool, error) {
	execUpdated, err := updateExecutable(logger, settings, cfg)
	if err != nil {
		return Service{}, false, fmt.Errorf("failed to update executable: %w", err)
	}

	unitUpdated, err := updateSystemd(logger, settings, cfg)
	if err != nil {
		return Service{}, false, fmt.Errorf("failed to update systemd unit: %w", err)
	}

	service, err := ParseService(logger, filepath.Join(systemdConfDir, string(cfg.Sandbox.Unit.Unit.Name)+".service"))
	if err != nil {
		return Service{}, false, fmt.Errorf("cannot parse systemd conf file: %s: %w", cfg.Sandbox.Unit.Unit.Name, err)
	}

	if !unitUpdated && !execUpdated {
		return service, false, nil
	}

	return service, true, nil
}
