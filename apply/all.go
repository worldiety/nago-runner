// Copyright (c) 2025 worldiety GmbH
//
// This file is part of the NAGO Low-Code Platform.
// Licensed under the terms specified in the LICENSE file.
//
// SPDX-License-Identifier: Custom-License

package apply

import (
	"fmt"
	"github.com/worldiety/nago-runner/configuration"
	"github.com/worldiety/nago-runner/setup"
	"log/slog"
)

func All(settings setup.Settings, cfg configuration.Runner) error {
	slog.Info("configure", "apps", len(cfg.Applications))
	for _, application := range cfg.Applications {
		/*if err := Debootstrap(application.Sandbox.Systemd.NSpawn.Debootstrap); err != nil {
			return fmt.Errorf("failed to configure debootstrap for app sandbox: %s: %w", application.ID, err)
		}*/

		if err := Systemd(settings.Token, application, NewS3Open()); err != nil {
			return fmt.Errorf("failed to configure systemd for app: %s: %w", application.ID, err)
		}
	}

	if err := Caddy(cfg); err != nil {
		return fmt.Errorf("failed to configure caddy: %w", err)
	}

	return nil
}
