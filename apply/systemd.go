// Copyright (c) 2025 worldiety GmbH
//
// This file is part of the NAGO Low-Code Platform.
// Licensed under the terms specified in the LICENSE file.
//
// SPDX-License-Identifier: Custom-License

package apply

import (
	"errors"
	"fmt"
	"github.com/worldiety/nago-runner/configuration"
	"github.com/worldiety/nago-runner/pkg/run"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
)

func Systemd(token string, appCfg configuration.Application, s3open S3Open) error {
	cfg := appCfg.Sandbox.Systemd
	if cfg.State == configuration.Disabled {
		// ignore
		return nil
	}

	serviceName := cfg.Name
	systemdFile := fmt.Sprintf("/etc/systemd/system/%s.service", serviceName)

	if cfg.State == configuration.Absent {

		if serviceName != "" {
			if err := run.Command("sudo", "systemctl", "stop", serviceName); err != nil {
				slog.Error("failed to stop systemd service", "service", serviceName, "err", err.Error())
			}

			if err := run.Command("sudo", "systemctl", "disable", serviceName); err != nil {
				slog.Error("systemctl disable failed", "service", serviceName, "err", err.Error())
			}
		}

		if _, err := os.Stat(systemdFile); os.IsNotExist(err) {
			return nil
		}

		slog.Info("removing existing systemd file", "file", systemdFile)
		if err := os.RemoveAll(systemdFile); err != nil {
			return fmt.Errorf("remove service file failed: %w", err)
		}

	}

	if cfg.State != configuration.Present {
		panic(fmt.Errorf("invalid state: %s", cfg.State))
	}

	var tmp string
	tmp += "[Unit]\n"
	tmp += fmt.Sprintf("Description=auto generated service file\n")
	tmp += "After=network.target\n\n"

	tmp += "[Service]\n"
	execCmd, err := systemdExecStart(token, appCfg, s3open)
	if err != nil {
		return fmt.Errorf("systemd exec start command generation failed: %w", err)
	}

	if !appCfg.Sandbox.Systemd.NSpawn.Enabled {
		// todo fixme
		for _, env := range appCfg.Sandbox.Systemd.NSpawn.Envs {
			tmp += fmt.Sprintf("Environment=%s=%s\n", env.Key, env.Value)
		}
	}

	tmp += "ExecStart=" + execCmd
	tmp += "\n\n"

	// sandbox
	if cfg.CapabilityBoundingSet != "" {
		tmp += fmt.Sprintf("CapabilityBoundingSet=%s\n", systemdFile)
	}

	if cfg.NoNewPrivileges {
		tmp += fmt.Sprintf("NewPrivileges=yes\n")
	}

	if cfg.ProtectSystem != "" {
		tmp += fmt.Sprintf("ProtectSystem=%s\n", cfg.ProtectSystem)
	}

	if cfg.ProtectHome {
		tmp += fmt.Sprintf("ProtectHome=yes\n")
	}

	if cfg.PrivateTmp {
		tmp += fmt.Sprintf("PrivateTmp=yes\n")
	}

	// cgroup
	if cfg.MemoryMax != 0 {
		tmp += fmt.Sprintf("MemoryLimit=%dM\n", cfg.MemoryMax)
	}

	if cfg.CPUQuota != 0 {
		tmp += fmt.Sprintf("CPUQuota=%d%%\n", cfg.CPUQuota)
	}

	// other stuff
	if cfg.KillMode != "" {
		tmp += fmt.Sprintf("KillMode=%s\n", cfg.KillMode)
	}

	if cfg.KillSignal != "" {
		tmp += fmt.Sprintf("KillSignal=%s\n", cfg.KillSignal)
	}

	if cfg.TimeoutStart != 0 {
		tmp += fmt.Sprintf("TimeoutStart=%d\n", cfg.TimeoutStart)
	}

	tmp += "[Install]\nWantedBy=multi-user.target\n"

	if EqualBuf(systemdFile, []byte(tmp)) {
		slog.Info("systemd service unchanged", "service", cfg.Name)
		return nil
	}

	if err := os.WriteFile(systemdFile, []byte(tmp), 0644); err != nil {
		return fmt.Errorf("error creating systemd file: %s, %w", systemdFile, err)
	}

	if err := run.Command("sudo", "systemctl", "daemon-reload"); err != nil {
		return fmt.Errorf("error reloading systemd daemon: %w", err)
	}

	if err := run.Command("sudo", "systemctl", "enable", serviceName); err != nil {
		return fmt.Errorf("error enabling systemd service: %s: %w", serviceName, err)
	}

	if err := run.Command("sudo", "systemctl", "restart", serviceName); err != nil {
		return fmt.Errorf("error restarting systemd daemon: %s: %w", serviceName, err)
	}

	slog.Info("service updated and restarted", "name", serviceName)

	return nil
}

func systemdExecStart(token string, appCfg configuration.Application, s3open S3Open) (string, error) {
	cfg := appCfg.Sandbox.Systemd

	deploymentDir := fmt.Sprintf("/opt/%s/%s/", appCfg.OrganizationSlug, appCfg.ApplicationSlug)
	var binaryFilePath string
	var binaryFile configuration.File
	switch appCfg.Artifacts.State {
	case configuration.Disabled:
	// do nothing
	case configuration.Present:
		// inspect
		for _, file := range appCfg.Artifacts.FileSet.Files {
			if err := file.Path.Validate(); err != nil {
				return "", fmt.Errorf("artifact file %s is not valid: %w", file.Path, err)
			}

			dstFile := filepath.Join(deploymentDir, string(file.Path))
			hash, err := Sha3(dstFile)
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				return "", fmt.Errorf("error hashing file %s: %w", file.Path, err)
			}

			if file.Executable && binaryFilePath == "" {
				binaryFilePath = dstFile
				binaryFile = file
			}

			if hash == file.Hash {
				slog.Info("artifact file %s is already up-to-date", file.Path)
				continue
			}

			// TODO still a good idea? What about just passing urls which is even more flexible?
			/*r, err := OpenByHash(s3open, appCfg.Artifacts.S3, hash)
			if err != nil {
				return "", fmt.Errorf("error opening file %s: %w", file.Path, err)
			}

			defer r.Close()*/

			req, err := http.NewRequest("GET", string(file.URL), nil)
			if err != nil {
				return "", fmt.Errorf("error creating file request %s: %w", file.Path, err)
			}
			req.Header.Set("Authorization", "Bearer "+token)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return "", fmt.Errorf("error downloading file %s: %w", file.Path, err)
			}

			if resp.StatusCode != 200 {
				return "", fmt.Errorf("error downloading file %s: status code %d", file.Path, resp.StatusCode)
			}

			defer resp.Body.Close()
			r := resp.Body

			_ = os.MkdirAll(filepath.Dir(dstFile), 0755)

			w, err := os.OpenFile(dstFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
			if err != nil {
				return "", fmt.Errorf("error creating file %s: %w", file.Path, err)
			}

			defer w.Close()

			if _, err := io.Copy(w, r); err != nil {
				return "", fmt.Errorf("error copying file %s: %w", file.Path, err)
			}

			if file.Executable {
				if err := os.Chmod(dstFile, 0777); err != nil {
					return "", fmt.Errorf("error setting executable bit file %s: %w", file.Path, err)
				}
			}

			slog.Info("updated artifact file", "file", dstFile, "hash", hash)
		}
	case configuration.Absent:
		if err := os.RemoveAll(deploymentDir); err != nil {
			return "", fmt.Errorf("removing application vendor files failed: %w", err)
		}

		return "", fmt.Errorf("everything is absent, cannot define service file")
	default:
		panic(fmt.Errorf("invalid state: %s", appCfg.Artifacts.State))
	}

	if binaryFilePath == "" {
		slog.Error("systemd exec start artifact set has not defined executable")
		return "", fmt.Errorf("artifacts do not contain any executable")
	}

	if !cfg.NSpawn.Enabled {
		// the no-nspawn case, where the binary is deployed and started without any isolation, which is
		// totally valid for embedded or one service per machine use cases
		return filepath.Join(deploymentDir, string(binaryFile.Path)), nil
	}

	nspawn := appCfg.Sandbox.Systemd.NSpawn
	container := nspawn.Debootstrap
	if container.State != configuration.Present {
		return "", fmt.Errorf("systemd debootstrap required but not present")
	}
	tmp := "/usr/bin/systemd-nspawn \\\n"
	tmp += fmt.Sprintf("    --machine=%s\\\n", appCfg.Sandbox.Systemd.Name)
	tmp += fmt.Sprintf("    --directory=%s\\\n", container.Target())
	tmp += fmt.Sprintf("    --bind=%s:/app\\\n", deploymentDir)
	for _, mount := range nspawn.BindMounts {
		tmp += fmt.Sprintf("    --bind=%s:%s\\\n", mount.Host, mount.Container)
	}

	for _, env := range nspawn.Envs {
		tmp += fmt.Sprintf("    --setenv=%s=%s\\\n", env.Key, env.Value)
	}

	if nspawn.ChDir != "" {
		tmp += fmt.Sprintf("    --chdir=%s\\\n", nspawn.ChDir)
	}

	//tmp += fmt.Sprintf("%s\n\n", filepath.Join("/app/", string(binaryFile.Path)))

	_ = binaryFile
	if true {
		tmp += fmt.Sprintf("    /bin/bash -c /app/testbin\\\n")
	}

	return tmp, nil
}
