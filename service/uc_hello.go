// Copyright (c) 2025 worldiety GmbH
//
// This file is part of the NAGO Low-Code Platform.
// Licensed under the terms specified in the LICENSE file.
//
// SPDX-License-Identifier: Custom-License

package service

import (
	"github.com/worldiety/nago-runner/service/event"
	"os"
)

func NewHello() Hello {
	return func(evt event.ConnectionCreated) event.RunnerLaunched {
		hostname, _ := os.Hostname()

		return event.RunnerLaunched{
			Hostname: hostname,
		}
	}
}
