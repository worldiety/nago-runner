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
)

func configure() error {
	var cfg setup.Settings
	flags := flag.NewFlagSet("configure", flag.ExitOnError)

	flags.StringVar(&cfg.URL, "url", "ws://localhost:3000/api/v1/runner", "URL to a worldiety hub instance")
	flags.StringVar(&cfg.Token, "token", "", "Token to a worldiety hub instance")
	if err := flags.Parse(flag.Args()[1:]); err != nil {
		log.Fatal(err)
	}

	applySettingsFunc := setup.NewApplySettings()
	if res := applySettingsFunc(cfg); res.Error != nil {
		return fmt.Errorf("cannot apply settings: %w", res.Error)
	}

	return nil
}
