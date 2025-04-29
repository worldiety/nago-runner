// Copyright (c) 2025 worldiety GmbH
//
// This file is part of the NAGO Low-Code Platform.
// Licensed under the terms specified in the LICENSE file.
//
// SPDX-License-Identifier: Custom-License

package setup

import (
	"fmt"
	"github.com/worldiety/nago-runner/pkg/linux"
	"os"
	"os/exec"
	"path/filepath"
)

const unixRunnerUserName = "nago-runner"
const systemdNagoRunnerName = "nago-runner"
const locationNagoRunnerBin = "/usr/local/bin/nago-runner"

func NewInstallRunner() InstallRunner {
	return func() error {
		if err := linux.AptUpdate(); err != nil {
			return fmt.Errorf("apt update failed: %w", err)
		}

		if err := linux.AptInstall("golang", "git"); err != nil {
			return fmt.Errorf("apt install failed: %w", err)
		}

		// build and deploy

		// we are not idempotent here, better clean than sorry
		if res := linux.ApplyUser(linux.User{
			Name:   unixRunnerUserName,
			Status: linux.Absent,
		}); res.Error != nil {
			return fmt.Errorf("cannot remove ng-runner user: %w", res.Error)
		}

		// clean create of the user
		if res := linux.ApplyUser(linux.User{
			Name:   unixRunnerUserName,
			Status: linux.Present,
			Sudo:   true,
			System: true,
		}); res.Error != nil {
			return fmt.Errorf("cannot add ng-runner system user: %w", res.Error)
		}

		// systemd stuff
		if res := linux.ApplyService(linux.Service{
			Description: "nago runner service",
			Name:        systemdNagoRunnerName,
			User:        unixRunnerUserName,
			ExecStart:   locationNagoRunnerBin,
		}); res.Error != nil {
			return fmt.Errorf("cannot configure nago-runner service: %w", res.Error)
		}

		// simple go build of this nago-runner
		cmd := exec.Command("go", "install", "github.com/worldiety/nago-runner/cmd/nago-runner@latest")
		cmd.Env = append(cmd.Env, "GOPROXY=direct")
		if buf, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("go install failed: %w: %s", err, string(buf))
		}

		myHome, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("cannot determine user home directory: %w", err)
		}

		execGoPath := filepath.Join(myHome, "go", "bin", "nago-runner")

		_ = linux.ServiceStop(systemdNagoRunnerName)

		if err := os.Rename(execGoPath, locationNagoRunnerBin); err != nil {
			return fmt.Errorf("cannot rename nago-runner binary: %w: %s->%s", err, execGoPath, locationNagoRunnerBin)
		}

		if err := linux.ServiceStart(systemdNagoRunnerName); err != nil {
			return fmt.Errorf("cannot start nago-runner service: %w", err)
		}

		return nil
	}
}
