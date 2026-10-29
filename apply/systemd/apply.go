// Copyright (c) 2025 worldiety GmbH
//
// This file is part of the NAGO Low-Code Platform.
// Licensed under the terms specified in the LICENSE file.
//
// SPDX-License-Identifier: Custom-License

package systemd

import (
	"errors"
	"fmt"
	"github.com/worldiety/nago-runner/configuration"
	"github.com/worldiety/nago-runner/pkg/datadir"
	"github.com/worldiety/nago-runner/pkg/run"
	"github.com/worldiety/nago-runner/setup"
	"log/slog"
	"os"
	"slices"
	"strings"
)

const systemdConfDir = "/etc/systemd/system"

func Apply(logger *slog.Logger, settings setup.Settings, cfg configuration.Runner) error {
	keepServices, removeServices, err := categorizeServices(logger, cfg)
	if err != nil {
		return fmt.Errorf("cannot categorize services: %w", err)
	}

	if refusePurge(cfg, removeServices) {
		// an empty configuration is far more likely a hub failure than the intention to undeploy everything
		logger.Error("refusing to remove all managed services due to an empty configuration, remove them manually if intended", "count", len(removeServices))
		removeServices = nil
	}

	// a failed removal of an undeclared service must not block the updates of the declared ones
	var errs []error
	if err := purgeServices(logger, removeServices); err != nil {
		logger.Error("cannot purge services, continuing with the declared ones", "err", err.Error())
		errs = append(errs, fmt.Errorf("cannot purge services: %w", err))
	}

	for _, service := range keepServices {
		logger.Info("apply service", "name", service.Name())
	}

	// one broken instance must neither block the updates of the other instances nor the restarts of the
	// executables which have already been replaced
	var requiresRestart []Service
	for _, app := range cfg.Applications {
		service, changed, err := createOrUpdateService(logger, settings, app)
		if err != nil {
			logger.Error("cannot create or update service, continuing with the others", "instance", app.InstID, "err", err.Error())
			errs = append(errs, fmt.Errorf("cannot create or update service %s: %w", app.InstID, err))
			continue
		}

		if !changed {
			stale, err := runsReplacedExecutable(service)
			if err != nil {
				logger.Warn("cannot inspect the running executable", "service", service.Name(), "err", err.Error())
			}

			if !stale {
				logger.Info("service is unchanged", "service", service.Name())
				continue
			}

			logger.Warn("service still runs a replaced executable", "service", service.Name())
		}

		requiresRestart = append(requiresRestart, service)
	}

	// this optimizes mass-updates to O(1) systemd reloads
	if len(requiresRestart) > 0 {
		if err := run.Command("systemctl", "daemon-reload"); err != nil {
			errs = append(errs, fmt.Errorf("error reloading systemd daemon: %w", err))
			return errors.Join(errs...)
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

	return errors.Join(errs...)
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

		stillAvailable := isDeclared(cfg, service)

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

		// security note: never delete the data of an undeclared service, a wrong configuration must not
		// be able to destroy any data. An operator has to decide what to do with the quarantined data.
		logger.Info("moving data into quarantine", "file", paths.DataDirectory)
		if _, err := datadir.Quarantine(paths.DataDirectory); err != nil {
			return fmt.Errorf("failed to quarantine data dir: %s: %w", paths.DataDirectory, err)
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

// refusePurge returns true, if the given configuration would remove all managed services at once because it
// declares no applications at all.
func refusePurge(cfg configuration.Runner, toRemove []Service) bool {
	return len(cfg.Applications) == 0 && len(toRemove) > 0
}

// isDeclared matches case-insensitive, because the service name is derived from the lower-cased unit file name.
func isDeclared(cfg configuration.Runner, service Service) bool {
	return slices.ContainsFunc(cfg.Applications, func(app configuration.Application) bool {
		return strings.EqualFold(app.InstID, service.Name())
	})
}

// CountManagedServices returns the amount of services which are managed by this runner.
func CountManagedServices(logger *slog.Logger) (int, error) {
	allServices, err := FindServices(logger)
	if err != nil {
		return 0, err
	}

	n := 0
	for _, service := range allServices {
		if service.Managed {
			n++
		}
	}

	return n, nil
}
