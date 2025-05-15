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
	"github.com/worldiety/nago-runner/pkg/run"
	"github.com/worldiety/nago-runner/setup"
	"log/slog"
	"os"
	"slices"
)

const systemdConfDir = "/etc/systemd/system"

func Apply(logger *slog.Logger, settings setup.Settings, cfg configuration.Runner) error {
	keepServices, removeServices, err := categorizeServices(logger, cfg)
	if err != nil {
		return fmt.Errorf("cannot categorize services: %w", err)
	}

	if err := purgeServices(logger, removeServices); err != nil {
		return fmt.Errorf("cannot purge services: %w", err)
	}

	for _, service := range keepServices {
		logger.Info("apply service", "name", service.Name())
	}

	var requiresRestart []Service
	for _, app := range cfg.Applications {
		service, changed, err := createOrUpdateService(logger, settings, app)
		if err != nil {
			return fmt.Errorf("cannot create or update service: %w", err)
		}

		if !changed {
			logger.Info("service is unchanged", "service", service.Name())
			continue
		}

		requiresRestart = append(requiresRestart, service)
	}

	// this optimizes mass-updates to O(1) systemd reloads
	if len(requiresRestart) > 0 {
		if err := run.Command("systemctl", "daemon-reload"); err != nil {
			return fmt.Errorf("error reloading systemd daemon: %w", err)
		}

		for _, service := range requiresRestart {
			logger.Info("enable service", "service", service.Name())
			if err := run.Command("systemctl", "enable", service.Name()); err != nil {
				slog.Warn("failed to enable service, ignoring", "service", service.Name())
			}

			logger.Info("restart service", "service", service.Name())
			if err := run.Command("systemctl", "restart", service.Name()); err != nil {
				slog.Warn("failed to restart service, ignoring", "service", service.Name())
			}
		}
	}

	return nil
}

func categorizeServices(logger *slog.Logger, cfg configuration.Runner) (keep []Service, remove []Service, err error) {
	allServices, err := FindServices(logger)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find systemd units: %w", err)
	}

	for _, service := range allServices {
		if !service.Managed {
			logger.Debug("ignoring service", "service", service.Name())
			continue
		}

		stillAvailable := slices.ContainsFunc(cfg.Applications, func(app configuration.Application) bool {
			return string(app.Sandbox.Unit.Unit.Name) == service.Name()
		})

		if stillAvailable {
			keep = append(keep, service)
		} else {
			remove = append(remove, service)
		}
	}

	return keep, remove, nil
}

func purgeServices(logger *slog.Logger, toRemove []Service) error {
	var deletedServices int

	for _, service := range toRemove {

		// clean up stale and unwanted services
		logger.Warn("removing undeclared managed service", "service", service.Name())

		logger.Info("stopping service", "service", service.Name())
		if err := run.Command("systemctl", "stop", service.Name()); err != nil {
			slog.Warn("failed to stop service, ignoring", "service", service.Name())
		}

		paths := service.Paths()
		logger.Info("removing exec", "file", paths.ExecFilename)
		if err := os.RemoveAll(paths.ExecFilename); err != nil {
			return fmt.Errorf("failed to remove executable: %s: %w", paths.ExecFilename, err)
		}

		logger.Info("removing data", "file", paths.DataDirectory)
		if err := os.RemoveAll(paths.DataDirectory); err != nil {
			return fmt.Errorf("failed to remove data dir: %s: %w", paths.DataDirectory, err)
		}

		logger.Info("removing service", "file", service.UnitFilename)
		if err := os.RemoveAll(service.UnitFilename); err != nil {
			return fmt.Errorf("failed to remove service file:%s: %w", service.UnitFilename, err)
		}

		deletedServices++
	}

	if deletedServices > 0 {
		logger.Warn("removed unwanted or stale services", "count", deletedServices)
		if err := run.Command("systemctl", "daemon-reload"); err != nil {
			return fmt.Errorf("error reloading systemd daemon: %w", err)
		}
	}

	return nil
}
