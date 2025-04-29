// Copyright (c) 2025 worldiety GmbH
//
// This file is part of the NAGO Low-Code Platform.
// Licensed under the terms specified in the LICENSE file.
//
// SPDX-License-Identifier: Custom-License

package setup

import (
	"github.com/worldiety/nago-runner/pkg/linux"
)

type Settings struct {
	URL   string `json:"url,omitempty"`
	Token string `json:"token,omitempty"`
}

type ApplySettings func(Settings) linux.Result

type LoadSettings func() (Settings, error)

type InstallRunner func() error

type UseCases struct {
	ApplySettings ApplySettings
	LoadSettings  LoadSettings
	InstallRunner InstallRunner
}

func NewUseCases() UseCases {
	return UseCases{
		ApplySettings: NewApplySettings(),
		LoadSettings:  NewLoadSettings(),
		InstallRunner: NewInstallRunner(),
	}
}
