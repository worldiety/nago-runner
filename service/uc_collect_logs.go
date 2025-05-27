// Copyright (c) 2025 worldiety GmbH
//
// This file is part of the NAGO Low-Code Platform.
// Licensed under the terms specified in the LICENSE file.
//
// SPDX-License-Identifier: Custom-License

package service

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/worldiety/nago-runner/service/event"
	"log/slog"
	"os/exec"
	"strconv"
)

func NewCollectLogs() CollectLogs {
	return func(request event.JournalCtlLogRequest) ([]event.JournalCtlEntry, error) {
		var args []string
		args = append(args, "--no-pager", "-o", "json")
		if request.LastN == 0 && request.Since == "" && request.Until == "" {
			request.LastN = 100
		}

		if request.LastN != 0 {
			args = append(args, "-n", strconv.Itoa(request.LastN))
		}

		if request.Unit != "" {
			args = append(args, "--unit", request.Unit)
		}

		cmd := exec.Command("journalctl", args...)
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return nil, fmt.Errorf("failed to get stdout pipe: %w", err)
		}

		var errBuf bytes.Buffer
		cmd.Stderr = &errBuf

		if err := cmd.Start(); err != nil {
			return nil, fmt.Errorf("failed to start journalctl: %w", err)
		}

		scanner := bufio.NewScanner(stdout)

		var res []event.JournalCtlEntry
		for scanner.Scan() {
			line := scanner.Text()
			var entry event.JournalCtlEntry
			if err := json.Unmarshal([]byte(line), &entry); err != nil {
				slog.Error("failed to unmarshal journalctl log entry", "entry", line, "err", err.Error())
				continue
			}

			res = append(res, entry)
		}

		/*	if err := cmd.Run(); err != nil {
			return nil, fmt.Errorf("failed to execute journalctl: %v: %w", buf.String(), err)
		}*/

		if err := scanner.Err(); err != nil {
			return nil, fmt.Errorf("failed to scan journalctl: %w", err)
		}

		if err := cmd.Wait(); err != nil {
			slog.Error(errBuf.String())
			return nil, fmt.Errorf("failed to wait journalctl: %w", err)
		}
		return res, nil
	}
}
