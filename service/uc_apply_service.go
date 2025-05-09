// Copyright (c) 2025 worldiety GmbH
//
// This file is part of the NAGO Low-Code Platform.
// Licensed under the terms specified in the LICENSE file.
//
// SPDX-License-Identifier: Custom-License

package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/worldiety/nago-runner/pkg/run"
	"log/slog"
	"os"
	"strconv"
	"text/template"
)

func NewApplyService() ApplyService {
	return func(deployment Deployment) error {
		jsonStr, err := json.Marshal(deployment)
		if err != nil {
			return fmt.Errorf("error marshalling deployment: %w", err)
		}

		serviceName := fmt.Sprintf("%s-%s", deployment.OrgSlug, deployment.AppSlug)
		model := serviceTplModel{
			Name:            serviceName,
			Description:     string(jsonStr),
			MemoryMax:       fmt.Sprintf("%dM", deployment.MaxMemoryMiB),
			CPUQuota:        fmt.Sprintf("%d%%", deployment.MaxCPUQuota),
			TimeoutStartSec: strconv.Itoa(deployment.TimeoutStartSec),
			RootFsDir:       containerDir,
			BinaryPath:      fmt.Sprintf("/bin/%s-%s", deployment.OrgSlug, deployment.AppSlug),
			Port:            strconv.Itoa(deployment.Port),
		}

		tpl, err := template.New("serviceTpl").Parse(serviceTpl)
		if err != nil {
			return fmt.Errorf("error parsing template: %w", err)
		}

		var buf bytes.Buffer
		if err := tpl.Execute(&buf, model); err != nil {
			return fmt.Errorf("error executing template: %w", err)
		}

		systemdFile := fmt.Sprintf("/etc/systemd/system/%s.service", serviceName)
		existingBuf, err := os.ReadFile(systemdFile)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("error reading systemd file: %s, %w", systemdFile, err)
		}

		if bytes.Equal(existingBuf, buf.Bytes()) {
			slog.Info("systemd file already up-2-date", "file", systemdFile)
			return nil
		}

		if err := os.WriteFile(systemdFile, buf.Bytes(), 0644); err != nil {
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
}

type serviceTplModel struct {
	Name            string // e.g. myorg-myserver
	Description     string // a json string as one-liner
	MemoryMax       string // e.g. 512M
	CPUQuota        string // e.g 50%
	TimeoutStartSec string // e.g. 30
	RootFsDir       string // e.g. /var/lib/machines/myorg/myserver must contains /bin/myserver /home and /tmp
	BinaryPath      string // e.g. /bin/myserver
	Port            string // e.g. 3000
}

const serviceTpl = `[Unit]
Description={{.Description}}
After=network.target

[Service]
ExecStart=/usr/bin/systemd-nspawn \
    --machine={{.Name}} \
    --directory={{.RootFsDir}} \
    --bind=/srv/builds/app123:/app \
    --setenv=PORT={{.Port}} \
	--setenv=HOME=/home \
	--setenv=TMPDIR=/tmp \
    --chdir=/home \
    --private-users=pick \
    {{.BinaryPath}}

# security sandbox
CapabilityBoundingSet=~CAP_SYS_ADMIN CAP_SETUID CAP_SETGID CAP_NET_ADMIN
NoNewPrivileges=yes
ProtectSystem=strict
ProtectHome=yes
PrivateTmp=yes

# cgroup
MemoryMax={{.MemoryMax}}
CPUQuota={{.CPUQuota}}


KillMode=control-group
TimeoutStartSec={{.TimeoutStartSec}}
KillSignal=SIGTERM

[Install]
WantedBy=multi-user.target`
