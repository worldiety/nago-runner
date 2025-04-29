// Copyright (c) 2025 worldiety GmbH
//
// This file is part of the NAGO Low-Code Platform.
// Licensed under the terms specified in the LICENSE file.
//
// SPDX-License-Identifier: Custom-License

package linux

import (
	"fmt"
	"os/exec"
	"strings"
)

type Service struct {
	Description string
	Name        string
	User        string
	ExecStart   string
	Environment map[string]string
}

func ApplyService(service Service) Result {
	serviceContent := fmt.Sprintf(`[Unit]
Description=%s
After=network.target

[Service]
Type=simple
User=%s
ExecStart=%s
Restart=always
Environment=%s

[Install]
WantedBy=multi-user.target
`, service.Name, service.User, service.ExecStart, mapToEnvironmentLines(service.Environment))

	servicePath := fmt.Sprintf("/etc/systemd/system/%s.service", service.Name)
	if res := ApplyFile(File{
		Filename: servicePath,
		Data:     []byte(serviceContent),
		Mode:     0644,
		Status:   Present,
	}); res.Error != nil {
		return Result{Error: fmt.Errorf("failed to apply service file: %s", res.Error)}
	}

	// Reload systemd
	if err := exec.Command("systemctl", "daemon-reload").Run(); err != nil {
		return Result{Error: fmt.Errorf("failed to reload systemd: %w", err)}
	}

	// Enable the service
	if err := exec.Command("systemctl", "enable", service.Name).Run(); err != nil {
		return Result{Error: fmt.Errorf("failed to enable service: %w", err)}
	}

	return Result{Action: Updated}
}

func mapToEnvironmentLines(envVars map[string]string) []string {
	var lines []string
	for key, value := range envVars {
		// Escape quotes and backslashes in value
		escapedValue := strings.ReplaceAll(value, `"`, `\"`)
		escapedValue = strings.ReplaceAll(escapedValue, `\`, `\\`)
		line := fmt.Sprintf(`Environment="%s=%s"`, key, escapedValue)
		lines = append(lines, line)
	}
	return lines
}

func ServiceStart(name string) error {
	if err := exec.Command("systemctl", "start", name).Run(); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	return nil
}

func ServiceStop(name string) error {
	if err := exec.Command("systemctl", "stop", name).Run(); err != nil {
		return fmt.Errorf("failed to stop service: %w", err)
	}

	return nil
}
