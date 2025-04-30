// Copyright (c) 2025 worldiety GmbH
//
// This file is part of the NAGO Low-Code Platform.
// Licensed under the terms specified in the LICENSE file.
//
// SPDX-License-Identifier: Custom-License

package linux

import (
	"fmt"
	"os"
	"os/user"
	"strconv"
)

func Chown(path string, username string) error {
	u, err := user.Lookup(username)
	if err != nil {
		return fmt.Errorf("benutzer '%s' nicht gefunden: %w", username, err)
	}

	uid, err := strconv.Atoi(u.Uid)
	if err != nil {
		return fmt.Errorf("ungültige UID: %w", err)
	}

	gid, err := strconv.Atoi(u.Gid)
	if err != nil {
		return fmt.Errorf("ungültige GID: %w", err)
	}

	if err := os.Chown(path, uid, gid); err != nil {
		return fmt.Errorf("chown fehlgeschlagen: %w", err)
	}

	return nil
}
