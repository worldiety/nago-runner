// Copyright (c) 2025 worldiety GmbH
//
// This file is part of the NAGO Low-Code Platform.
// Licensed under the terms specified in the LICENSE file.
//
// SPDX-License-Identifier: Custom-License

package run

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"log/slog"
	"os/exec"
	"strings"
)

func Command(command string, args ...string) error {
	slog.Info("exec", command, strings.Join(args, " "))
	cmd := exec.Command(command, args...)
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	go logOutput(stdoutPipe, slog.LevelInfo)
	go logOutput(stderrPipe, slog.LevelError)

	return cmd.Wait()
}

func CommandString(command string, args ...string) (string, error) {
	slog.Info("exec", command, strings.Join(args, " "))
	cmd := exec.Command(command, args...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	if err := cmd.Run(); err != nil {
		return buf.String(), err
	}

	return strings.TrimSpace(buf.String()), nil
}

func logOutput(pipe io.ReadCloser, level slog.Level) {
	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		line := scanner.Text()
		slog.Log(context.Background(), level, line)
	}
}
