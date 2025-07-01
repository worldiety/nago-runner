// Copyright (c) 2025 worldiety GmbH
//
// This file is part of the NAGO Low-Code Platform.
// Licensed under the terms specified in the LICENSE file.
//
// SPDX-License-Identifier: Custom-License

package systemd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/worldiety/nago-runner/configuration"
	"github.com/worldiety/nago-runner/pkg/linux"
	"github.com/worldiety/nago-runner/setup"
	"log/slog"
	"strings"
)

// updateSystemd regenerates the entire systemd service unit file and rewrites and reloads systemd if required.
// If nothing has changed, this does nothing.
func updateSystemd(logger *slog.Logger, settings setup.Settings, cfg configuration.Application) (bool, error) {
	var f bytes.Buffer
	// header
	buf, err := json.Marshal(cfg)
	if err != nil {
		return false, fmt.Errorf("failed to marshal configuration: %w", err)
	}

	f.WriteString(ngrMetaPrefix)
	f.Write(buf)
	f.WriteString("\n\n")

	unit := cfg.Sandbox.Unit.Unit
	f.WriteString("[Unit]\n")
	f.WriteString("Description=" + unit.Description + "\n")
	if unit.After != "" {
		f.WriteString("After=" + string(unit.After) + "\n")
	}

	f.WriteString("\n")

	// ###
	service := cfg.Sandbox.Unit.Service
	f.WriteString("[Service]\n")
	if service.Type != "" {
		f.WriteString(fmt.Sprintf("Type=%s\n", service.Type))
	}

	if service.User != "" {
		f.WriteString(fmt.Sprintf("User=%s\n", service.User))
	}

	if service.Group != "" {
		f.WriteString(fmt.Sprintf("Group=%s\n", service.Group))
	}

	if service.BindPaths != "" {
		f.WriteString(fmt.Sprintf("BindPaths=%s\n", service.BindPaths))
	}

	if service.BindReadOnlyPaths != "" {
		f.WriteString(fmt.Sprintf("BindReadOnlyPaths=%s\n", service.BindReadOnlyPaths))
	}

	if service.ReadOnlyPaths != "" {
		f.WriteString(fmt.Sprintf("ReadOnlyPaths=%s\n", service.ReadOnlyPaths))
	}

	if service.InaccessiblePaths != "" {
		f.WriteString(fmt.Sprintf("InaccessiblePaths=%s\n", service.InaccessiblePaths))
	}

	if service.ExecPaths != "" {
		f.WriteString(fmt.Sprintf("ExecPaths=%s\n", service.ExecPaths))
	}

	if service.AppArmorProfile != "" {
		f.WriteString(fmt.Sprintf("AppArmorProfile=%s\n", service.AppArmorProfile))
	}

	if service.StateDirectory != "" {
		f.WriteString(fmt.Sprintf("StateDirectory=%s\n", service.StateDirectory))
	}

	if service.SystemCallFilter != "" {
		f.WriteString(fmt.Sprintf("SystemCallFilter=%s\n", service.SystemCallFilter))
	}

	if service.PrivateTmp {
		f.WriteString("PrivateTmp=yes\n")
	}

	if service.MemoryDenyWriteExecute {
		f.WriteString("MemoryDenyWriteExecute=yes\n")
	}

	if service.DynamicUser {
		f.WriteString("DynamicUser=yes\n")
	}

	if service.NoNewPrivileges {
		f.WriteString("NoNewPrivileges=yes\n")
	}

	if service.PrivateDevices {
		f.WriteString("PrivateDevices=yes\n")
	}

	if service.PrivateIPC {
		f.WriteString("PrivateIPC=yes\n")
	}

	if service.PrivatePIDs {
		f.WriteString("PrivatePIDs=yes\n")
	}

	if service.PrivateMounts {
		f.WriteString("PrivateMounts=yes\n")
	}

	if service.PrivateNetwork {
		f.WriteString("PrivateNetwork=yes\n")
	}

	if service.PrivateUsers != "" {
		f.WriteString(fmt.Sprintf("PrivateUsers=%s\n", service.PrivateUsers))
	}

	if service.ProtectKernelModules {
		f.WriteString("ProtectKernelModules=yes\n")
	}

	if service.ProtectKernelTunables {
		f.WriteString("ProtectKernelTunables=yes\n")
	}

	if service.ProtectClock {
		f.WriteString("ProtectClock=yes\n")
	}

	if service.ProtectKernelLogs {
		f.WriteString("ProtectKernelLogs=yes\n")
	}

	if service.ProtectHostname {
		f.WriteString("ProtectHostname=yes\n")
	}

	if service.SetLoginEnvironment {
		f.WriteString("SetLoginEnvironment=yes\n")
	}

	if service.RestrictSUIDSGID {
		f.WriteString("RestrictSUIDSGID=yes\n")
	}

	if service.RestrictRealtime {
		f.WriteString("RestrictRealtime=yes\n")
	}

	if service.MemoryDenyWriteExecute {
		f.WriteString("MemoryDenyWriteExecute=yes\n")
	}

	for _, ns := range service.RestrictNamespaces {
		f.WriteString(fmt.Sprintf("RestrictNamespaces=%s\n", ns))
	}

	if service.ProtectHome != "" {
		f.WriteString(fmt.Sprintf("ProtectHome=%s\n", service.ProtectHome))
	}

	if service.ProtectSystem != "" {
		f.WriteString(fmt.Sprintf("ProtectSystem=%s\n", service.ProtectSystem))
	}

	if service.ProtectControlGroups != "" {
		f.WriteString(fmt.Sprintf("ProtectControlGroups=%s\n", service.ProtectControlGroups))
	}

	if service.ProtectProc != "" {
		f.WriteString(fmt.Sprintf("ProtectProc=%s\n", service.ProtectProc))
	}

	f.WriteString(fmt.Sprintf("ExecStart=%s %s\n", service.ExecStart.Cmd, strings.Join(service.ExecStart.Args, " ")))
	for _, env := range service.Environment {
		f.WriteString(fmt.Sprintf("Environment=%s=%s\n", env.Key, env.Value))
	}

	for _, c := range service.CapabilityBoundingSet {
		f.WriteString(fmt.Sprintf("CapabilityBoundingSet=%s\n", c))
	}

	if service.Restart != "" {
		f.WriteString(fmt.Sprintf("Restart=%s\n", service.Restart))
	}

	if service.RestartSec != 0 {
		f.WriteString(fmt.Sprintf("RestartSec=%s\n", service.RestartSec.String()))
	}

	if service.MemoryHigh != "" {
		f.WriteString(fmt.Sprintf("MemoryHigh=%s\n", service.MemoryHigh))
	}

	if service.MemorySwapMax != "" {
		f.WriteString(fmt.Sprintf("MemorySwapMax=%s\n", service.MemorySwapMax))
	}

	if service.StartupMemoryHigh != "" {
		f.WriteString(fmt.Sprintf("StartupMemoryHigh=%s\n", service.StartupMemoryHigh))
	}

	if service.StartupMemorySwapMax != "" {
		f.WriteString(fmt.Sprintf("StartupMemorySwapMax=%s\n", service.StartupMemorySwapMax))
	}

	if service.OOMPolicy != "" {
		f.WriteString(fmt.Sprintf("OOMPolicy=%s\n", service.OOMPolicy))
	}

	if service.OOMScoreAdjust != 0 {
		f.WriteString(fmt.Sprintf("OOMScoreAdjust=%d\n", service.OOMScoreAdjust))
	}

	if service.CPUWeight != 0 {
		f.WriteString(fmt.Sprintf("CPUWeight=%d\n", service.CPUWeight))
	}

	if service.CPUQuota != 0 {
		f.WriteString(fmt.Sprintf("CPUQuota=%d\n", service.CPUQuota))
	}

	for _, sec := range service.SecureBits {
		f.WriteString(fmt.Sprintf("SecureBits=%s\n", sec))
	}

	for _, sec := range service.SocketBindAllow {
		f.WriteString(fmt.Sprintf("SocketBindAllow=%s\n", sec))
	}

	for _, sec := range service.SocketBindDeny {
		f.WriteString(fmt.Sprintf("SocketBindDeny=%s\n", sec))
	}

	if service.KillMode != "" {
		f.WriteString(fmt.Sprintf("KillMode=%s\n", service.KillMode))
	}

	if service.KillSignal != "" {
		f.WriteString(fmt.Sprintf("KillSignal=%s\n", service.KillSignal))
	}

	if service.TimeoutStopSec != 0 {
		f.WriteString(fmt.Sprintf("TimeoutStopSec=%s\n", service.TimeoutStopSec.String()))
	}

	// ###
	install := cfg.Sandbox.Unit.Install
	f.WriteString("[Install]\n")
	if install.WantedBy != "" {
		f.WriteString(fmt.Sprintf("WantedBy=%s\n", install.WantedBy))
	}

	// finally, check if changed

	fakeService := NewService(cfg.InstID)
	currentHash, err := linux.Sha3(fakeService.UnitFilename)
	if err != nil {
		return false, fmt.Errorf("failed to calculate current hash: %w", err)
	}

	expectedHash, err := linux.Sha3Bytes(f.Bytes())
	if err != nil {
		return false, fmt.Errorf("failed to calculate expected hash: %w", err)
	}

	if currentHash == expectedHash {
		slog.Info("systemd service unit file unchanged", "expected", expectedHash, "file", fakeService.UnitFilename)
		return false, nil
	}

	slog.Info("systemd service unit file expected hash does not match the current hash", "expected", expectedHash, "actual", currentHash)
	if err := linux.WriteFile(fakeService.UnitFilename, f.Bytes(), 0666); err != nil {
		return false, fmt.Errorf("failed to update systemd service unit file: %w", err)
	}

	return true, nil
}
