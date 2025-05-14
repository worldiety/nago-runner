// Copyright (c) 2025 worldiety GmbH
//
// This file is part of the NAGO Low-Code Platform.
// Licensed under the terms specified in the LICENSE file.
//
// SPDX-License-Identifier: Custom-License

package linux

import "github.com/worldiety/nago-runner/pkg/run"

func Which(name string) (string, error) {
	return run.CommandString("which", name)
}
