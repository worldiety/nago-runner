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
	return func(settings Settings) error {

		buf, err := json.MarshalIndent(settings, "", "  ")
		if err != nil {
			return fmt.Errorf("could not marshal settings: %w", err)
		}

		if err := linux.WriteFile(cfgJson, buf, 0600); err != nil {
			return fmt.Errorf("could not write settings: %w", err)
		}

		return nil
	}
}
