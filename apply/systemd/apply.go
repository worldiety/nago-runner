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
	"github.com/worldiety/nago-runner/pkg/instlock"
	"github.com/worldiety/nago-runner/pkg/run"
	"github.com/worldiety/nago-runner/setup"
	"log/slog"
	"os"
	"slices"
	"strings"
)

const systemdConfDir = "/etc/systemd/system"

// Applier applies the declared services. It is not safe for concurrent use, the runner calls it from a single
// goroutine, see apply.Serial.
type Applier struct {
	// pendingEnable and pendingRestart remember the services whose enable or restart has failed. Their unit file
	// and executable are already up to date, thus a later apply sees no change and nothing else would ever enable
	// or restart them again.
	pendingEnable  map[string]bool
	pendingRestart map[string]bool
}

func NewApplier() *Applier {
	return &Applier{
		pendingEnable:  map[string]bool{},
		pendingRestart: map[string]bool{},
	}
}

// Apply updates, enables and restarts the declared services and removes the undeclared ones. A returned error
// means that the apply should be retried later.
func (a *Applier) Apply(logger *slog.Logger, settings setup.Settings, cfg configuration.Runner) error {
	keepServices, removeServices, err := categorizeServices(logger, cfg)
	if err != nil {
		return fmt.Errorf("cannot categorize services: %w", err)
	}

	if refusePurge(cfg, removeServices) {
		// an empty configuration is far more likely a hub failure than the intention to undeploy everything
		logger.Error("refusing to remove all managed services due to an empty configuration, remove them manually if intended", "count", len(removeServices))
		removeServices = nil
	}

	var errs []error

	// security note: an instance which is being restored is stopped and its data is replaced. It must neither be
	// removed, updated nor restarted now, a retry applies it after the restore.
	var locked []string
	defer func() {
		for _, id := range locked {
			instlock.Unlock(id)
		}
	}()

	lock := func(id string) bool {
		if !instlock.TryLock(id) {
			return false
		}

		locked = append(locked, id)
		return true
	}

	removeServices = slices.DeleteFunc(removeServices, func(service Service) bool {
		if lock(service.Name()) {
			return false
		}

		logger.Warn("service is being restored, removing it later", "service", service.Name())
		errs = append(errs, fmt.Errorf("service %s is being restored", service.Name()))
		return true
	})

	// a failed removal of an undeclared service must not block the updates of the declared ones
	if err := purgeServices(logger, removeServices); err != nil {
		logger.Error("cannot purge services, continuing with the declared ones", "err", err.Error())
		errs = append(errs, fmt.Errorf("cannot purge services: %w", err))
	}

	a.forgetUndeclared(cfg)

	for _, service := range keepServices {
		logger.Info("apply service", "name", service.Name())
	}

	// one broken instance must neither block the updates of the other instances nor the restarts of the
	// executables which have already been replaced
	var requiresRestart, requiresEnable []Service
	for _, app := range cfg.Applications {
		if !lock(app.InstID) {
			logger.Warn("instance is being restored, applying it later", "instance", app.InstID)
			errs = append(errs, fmt.Errorf("instance %s is being restored", app.InstID))
			continue
		}

		service, changed, err := createOrUpdateService(logger, settings, app)
		if err != nil {
			if errors.Is(err, ErrNoExecutable) {
				// not retried: only a new configuration can change this and the hub announces it
				logger.Error("cannot create or update service, continuing with the others", "instance", app.InstID, "err", err.Error())
				continue
			}

			logger.Error("cannot create or update service, continuing with the others", "instance", app.InstID, "err", err.Error())
			errs = append(errs, fmt.Errorf("cannot create or update service %s: %w", app.InstID, err))
			continue
		}

		name := service.Name()
		switch {
		case changed:
		case a.pendingRestart[name]:
			logger.Warn("retrying the failed restart of the service", "service", name)
		default:
			stale, err := runsReplacedExecutable(service)
			if err != nil {
				logger.Warn("cannot inspect the running executable", "service", name, "err", err.Error())
			}

			if !stale {
				if a.pendingEnable[name] {
					logger.Warn("retrying the failed enable of the service", "service", name)
					requiresEnable = append(requiresEnable, service)
					continue
				}

				logger.Info("service is unchanged", "service", name)
				continue
			}

			logger.Warn("service still runs a replaced executable", "service", name)
		}

		requiresRestart = append(requiresRestart, service)
	}

	// this optimizes mass-updates to O(1) systemd reloads
	if len(requiresRestart) > 0 {
		if err := run.Command("systemctl", "daemon-reload"); err != nil {
			for _, service := range requiresRestart {
				a.pendingEnable[service.Name()] = true
				a.pendingRestart[service.Name()] = true
			}

			errs = append(errs, fmt.Errorf("error reloading systemd daemon: %w", err))
			return errors.Join(errs...)
		}
	}

	for _, service := range requiresRestart {
		if err := a.enable(logger, service); err != nil {
			errs = append(errs, err)
		}

		if err := a.restart(logger, service); err != nil {
			errs = append(errs, err)
		}
	}

	for _, service := range requiresEnable {
		if err := a.enable(logger, service); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

func (a *Applier) enable(logger *slog.Logger, service Service) error {
	name := service.Name()
	logger.Info("enable service", "service", name)
	if err := run.Command("systemctl", "enable", name); err != nil {
		logger.Error("failed to enable service, retrying later", "service", name, "err", err.Error())
		a.pendingEnable[name] = true
		return fmt.Errorf("cannot enable service %s: %w", name, err)
	}

	delete(a.pendingEnable, name)
	return nil
}

func (a *Applier) restart(logger *slog.Logger, service Service) error {
	name := service.Name()
	logger.Info("restart service", "service", name)
	if err := run.Command("systemctl", "restart", name); err != nil {
		logger.Error("failed to restart service, retrying later", "service", name, "err", err.Error())
		a.pendingRestart[name] = true
		return fmt.Errorf("cannot restart service %s: %w", name, err)
	}

	delete(a.pendingRestart, name)
	return nil
}

// forgetUndeclared drops the pending actions of services which are not declared anymore.
func (a *Applier) forgetUndeclared(cfg configuration.Runner) {
	for _, pending := range []map[string]bool{a.pendingEnable, a.pendingRestart} {
		for name := range pending {
			if !isDeclared(cfg, NewService(name)) {
				delete(pending, name)
			}
		}
	}
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
