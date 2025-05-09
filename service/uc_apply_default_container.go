// Copyright (c) 2025 worldiety GmbH
//
// This file is part of the NAGO Low-Code Platform.
// Licensed under the terms specified in the LICENSE file.
//
// SPDX-License-Identifier: Custom-License

package service

import (
	"fmt"
	"github.com/worldiety/nago-runner/pkg/run"
	"log/slog"
	"os"
	"runtime"
)

const containerDir = "/var/lib/machines/nago-container"

func NewApplyDefaultContainer() ApplyDefaultContainer {
	return func() (dir string, err error) {

		if _, err := os.Stat(containerDir); os.IsNotExist(err) {
			slog.Info("create container", "containerDir", containerDir)
			// bullseye
			//if err := run.Command("sudo", "debootstrap", "--arch", debootstrapArch(), "bullseye", containerDir, "http://deb.debian.org/debian"); err != nil {
			//	return "", fmt.Errorf("debootstrap failed: %w", err)
			//}

			//"http://archive.ubuntu.com/ubuntu/"
			if err := run.Command("sudo", "debootstrap", "--variant", "minbase", "plucky", containerDir, "http://ports.ubuntu.com/ubuntu-ports/"); err != nil {
				_ = os.RemoveAll(containerDir)
				return "", fmt.Errorf("debootstrap failed: %w", err)
			}
		} else {
			slog.Info("use container", "containerDir", containerDir)
		}

		slog.Info("updating container...")
		if err := run.Command("sudo", "chroot", containerDir, "apt", "update"); err != nil {
			return "", fmt.Errorf("apt update failed: %w", err)
		}

		if false {
			// TODO typst is missing, not available through apt
			// TODO textlive-full does not build at all
			if err := installPackages(containerDir, "typst"); err != nil {
				return "", fmt.Errorf("install failed: %w", err)
			}
		}

		slog.Info("container available", "containerDir", containerDir)
		return containerDir, nil
	}
}

func installPackages(containerDir string, packages ...string) error {
	args := append([]string{"chroot", containerDir, "apt", "install", "-y"}, packages...)
	return run.Command("sudo", args...)
}

// returns the arch string as required by debootstrap
func debootstrapArch() string {
	return runtime.GOARCH
}
