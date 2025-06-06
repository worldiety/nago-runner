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
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
)

func NewLoadSettings() LoadSettings {
	return func() (Settings, error) {
		var cfgJsonPath string
		if runtime.GOOS != "linux" {
			slog.Info("running on unsupported operating system")
			myHome, err := os.UserHomeDir()
			if err != nil {
				return Settings{}, fmt.Errorf("cannot determine user home directory: %w", err)
			}

			cfgJsonPath = filepath.Join(myHome, ".nago-runner/config.json")
		} else {
			cfgJsonPath = cfgJson
		}

		if _, err := os.Stat(cfgJsonPath); os.IsNotExist(err) {
			return Settings{}, nil
		}

		var cfg Settings
		buf, err := os.ReadFile(cfgJsonPath)
		if err != nil {
			return Settings{}, fmt.Errorf("cannot read config file: %w: %s", err, cfgJsonPath)
		}

		if err := json.Unmarshal(buf, &cfg); err != nil {
			return Settings{}, fmt.Errorf("cannot parse config file: %w: %s", err, string(buf))
		}

		return cfg, nil
	}
}
