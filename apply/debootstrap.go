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
	"github.com/worldiety/nago-runner/pkg/run"
	"log/slog"
	"os"
)

func Debootstrap(cfg configuration.Debootstrap) error {
	if cfg.State == configuration.Disabled {
		return nil
	}

	slog := slog.With("configure", "debootstrap")

	configFile := string(cfg.Target() + ".json")
	containerDir := string(cfg.Target())
	if err := os.MkdirAll(containerDir, 0755); err != nil {
		return fmt.Errorf("couldn't create directory %s: %w", containerDir, err)
	}

	if cfg.State == configuration.Absent {
		if err := os.RemoveAll(configFile); err != nil {
			return fmt.Errorf("cannot remove config file: %s: %w", configFile, err)
		}

		if err := os.RemoveAll(containerDir); err != nil {
			return fmt.Errorf("cannot remove machine dir: %s: %w", containerDir, err)
		}

		return nil
	}

	if cfg.State != configuration.Present {
		panic(fmt.Errorf("invalid state: %s", cfg.State))
	}

	if EqualJSON(configFile, cfg) {
		if len(cfg.UpgradeCommands) > 0 {
			slog.Info("running upgrade commands", "machine", containerDir)
		}

		for _, command := range cfg.UpgradeCommands {
			if err := chrootRun(containerDir, command.Cmd, command.Args...); err != nil {
				return fmt.Errorf("upgrade command failed: %w", err)
			}
		}

		slog.Info("machine is latest", "machine", containerDir)

		return nil
	}

	// something has changed, restart building the image from scratch
	if err := os.RemoveAll(containerDir); err != nil {
		return fmt.Errorf("cannot remove container dir: %s: %w", containerDir, err)
	}

	slog.Info("creating image", "machine", containerDir)
	var args []string
	args = append(args, "debootstrap")
	if cfg.Variant != "" {
		args = append(args, "--variant", cfg.Variant)
	}

	args = append(args, cfg.Suite)
	args = append(args, containerDir)
	args = append(args, string(cfg.Mirror))

	if err := run.Command("sudo", args...); err != nil {
		_ = os.RemoveAll(containerDir) // try some cleanup on fatal error
		return fmt.Errorf("debootstrap failed: %s, %w", containerDir, err)
	}

	for _, command := range cfg.PostCommands {
		_ = os.RemoveAll(containerDir) // try some cleanup on fatal error
		if err := chrootRun(containerDir, command.Cmd, command.Args...); err != nil {
			return fmt.Errorf("post command failed: %w", err)
		}
	}

	// update cfg file
	if err := WriteJSON(configFile, cfg); err != nil {
		return fmt.Errorf("cannot write config file: %w", err)
	}

	slog.Info("image installed", "machine", containerDir)
	return nil
}

func chrootRun(containerDir string, cmd string, argsv ...string) error {
	args := append([]string{"chroot", containerDir}, cmd)
	args = append(args, argsv...)
	return run.Command("sudo", args...)
}
