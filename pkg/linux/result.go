// Copyright (c) 2025 worldiety GmbH
//
// This file is part of the NAGO Low-Code Platform.
// Licensed under the terms specified in the LICENSE file.
//
// SPDX-License-Identifier: Custom-License

package linux

type Status string

const (
	Absent  Status = "absent"
	Present Status = "present"
)

type Action string

const (
	Updated Action = "updated"
	Deleted Action = "deleted"
	Created Action = "created"
	Ignored Action = "ignored"
)

type Result struct {
	Error  error
	Action Action
}
