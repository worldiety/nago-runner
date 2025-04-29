// Copyright (c) 2025 worldiety GmbH
//
// This file is part of the NAGO Low-Code Platform.
// Licensed under the terms specified in the LICENSE file.
//
// SPDX-License-Identifier: Custom-License

package setup

import (
	"encoding/json"
	"fmt"
	"github.com/worldiety/nago-runner/pkg/linux"
)

const (
	cfgPath = "/etc/nago-runner/"
	cfgJson = cfgPath + "config.json"
)

func NewApplySettings() ApplySettings {
	return func(settings Settings) linux.Result {

		buf, err := json.MarshalIndent(settings, "", "  ")
		if err != nil {
			return linux.Result{Error: fmt.Errorf("failed marshalling settings: %w", err)}
		}

		return linux.ApplyFile(linux.File{
			Filename: cfgJson,
			Data:     buf,
			// security note: 0600 means that only the owner can read or write the file
			Mode: 0600,
		})

	}
}
