// Copyright (c) 2025 worldiety GmbH
//
// This file is part of the NAGO Low-Code Platform.
// Licensed under the terms specified in the LICENSE file.
//
// SPDX-License-Identifier: Custom-License

package linux

import (
	"fmt"
	"os/exec"
)

type User struct {
	Name   string
	Status Status
	System bool
	Sudo   bool
}

func ApplyUser(user User) Result {
	exists := userExists(user.Name)
	if !exists && user.Status == Absent {
		return Result{Action: Ignored}
	}

	if exists && user.Status == Present {
		// well, this is incorrect, because we do not check the other properties properly
		// but that makes everything really complicated and we probably don't need it yet.
		return Result{Action: Ignored}
	}

	if exists && user.Status == Absent {
		if err := deleteUser(user.Name); err != nil {
			return Result{Error: err}
		}

		return Result{Action: Deleted}
	}

	if user.System {
		if err := exec.Command("useradd", "--system", "--no-create-home", "--shell", "/usr/sbin/nologin", user.Name).Run(); err != nil {
			return Result{Error: fmt.Errorf("failed to add system user: %w", err)}
		}
	} else {
		if err := exec.Command("useradd", user.Name).Run(); err != nil {
			return Result{Error: fmt.Errorf("failed to add user: %w", err)}
		}
	}

	if user.Sudo {
		sudoersEntry := user.Name + " ALL=(ALL) NOPASSWD: ALL\n"
		if res := ApplyFile(File{
			Filename: "/etc/sudoers.d/" + user.Name,
			Data:     []byte(sudoersEntry),
			Mode:     0440,
			Status:   Present,
		}); res.Error != nil {
			return Result{Error: res.Error}
		}
	}

	return Result{Action: Created}
}

func userExists(username string) bool {
	cmd := exec.Command("id", username)
	err := cmd.Run()
	return err == nil
}

func deleteUser(username string) error {
	cmd := exec.Command("userdel", "--force", "--remove", username)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error deleting user %s: %v, output: %s", username, err, string(output))
	}

	return nil
}
