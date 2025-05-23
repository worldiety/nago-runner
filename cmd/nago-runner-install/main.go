// Copyright (c) 2025 worldiety GmbH
//
// This file is part of the NAGO Low-Code Platform.
// Licensed under the terms specified in the LICENSE file.
//
// SPDX-License-Identifier: Custom-License

package main

import (
	"flag"
	"fmt"
	"github.com/worldiety/nago-runner/setup"
	"log"
	"runtime"
)

func main() {
	if err := realMain(); err != nil {
		log.Fatal(err)
	}
}

func realMain() error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	var cfg setup.Settings

	flag.StringVar(&cfg.URL, "url", "https://nago.app", "URL to a worldiety hub instance. Keep empty, to not overwrite existing configuration.")
	flag.StringVar(&cfg.Token, "token", "", "Token to a worldiety hub instance. Keep empty, to not overwrite existing configuration.")
	flag.Parse()

	ucSetup := setup.NewUseCases()
	loadedCfg, err := ucSetup.LoadSettings()
	if err != nil {
		return fmt.Errorf("could not load settings: %w", err)
	}

	changed := cfg.Token != "" || cfg.URL != ""

	if cfg.URL == "" {
		cfg.URL = loadedCfg.URL
	}

	if cfg.Token == "" {
		cfg.Token = loadedCfg.Token
	}

	if changed {
		if err := ucSetup.ApplySettings(loadedCfg); err != nil {
			return fmt.Errorf("could not apply settings: %w", err)
		}
	}

	if err := ucSetup.InstallRunner(); err != nil {
		return fmt.Errorf("could not install runner: %w", err)
	}

	return nil
}
