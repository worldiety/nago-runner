// Copyright (c) 2025 worldiety GmbH
//
// This file is part of the NAGO Low-Code Platform.
// Licensed under the terms specified in the LICENSE file.
//
// SPDX-License-Identifier: Custom-License

package caddy

import (
	"fmt"
	"github.com/worldiety/nago-runner/configuration"
	"github.com/worldiety/nago-runner/pkg/linux"
	"github.com/worldiety/nago-runner/pkg/run"
	"github.com/worldiety/nago-runner/setup"
	"log/slog"
)

func Apply(logger *slog.Logger, settings setup.Settings, cfg configuration.Runner) error {
	if err := installCaddy(logger); err != nil {
		return fmt.Errorf("cannot install caddy: %w", err)
	}

	updated, err := updateCaddyfile(logger, settings, cfg)
	if err != nil {
		return fmt.Errorf("cannot update caddyfile: %w", err)
	}

	if updated {
		if err := run.Command("systemctl", "reload", "caddy"); err != nil {
			return fmt.Errorf("error reloading caddy: %w", err)
		}
	} else {
		logger.Info("caddy configuration is up to date")
	}

	return nil
}

func installCaddy(logger *slog.Logger) error {
	cpath, err := linux.Which("caddy")
	if err != nil {
		// exit early, we will cause trouble with existing files, if
		return fmt.Errorf("cannot find caddy executable: %w", err)
	}

	if cpath == "" {
		logger.Warn("caddy executable not found in $PATH")
		if err := run.Command("apt", "install", "-y", "debian-keyring", "debian-archive-keyring", "apt-transport-https", "curl"); err != nil {
			return err
		}

		if err := run.Command("/bin/bash", "-c", "curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | sudo gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg"); err != nil {
			return err
		}

		if err := run.Command("/bin/bash", "-c", "curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | sudo tee /etc/apt/sources.list.d/caddy-stable.list"); err != nil {
			return err
		}

		if err := run.Command("apt", "update"); err != nil {
			return err
		}

		if err := run.Command("apt", "install", "-y", "caddy"); err != nil {
			return fmt.Errorf("cannot install caddy: %w", err)
		}

		// after installing freshly, caddy does not yet run
		if err := run.Command("systemctl", "start", "caddy"); err != nil {
			return fmt.Errorf("error starting caddy: %w", err)
		}
	}

	return nil
}
