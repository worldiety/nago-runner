// Copyright (c) 2025 worldiety GmbH
//
// This file is part of the NAGO Low-Code Platform.
// Licensed under the terms specified in the LICENSE file.
//
// SPDX-License-Identifier: Custom-License

package main

import (
	"fmt"
	"github.com/worldiety/nago-runner/setup"
)

func install() error {

	installRunnerFunc := setup.NewInstallRunner()
	if err := installRunnerFunc(); err != nil {
		return fmt.Errorf("cannot install runner: %w", err)
	}

	return nil
}
