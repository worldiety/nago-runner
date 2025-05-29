// Copyright (c) 2025 worldiety GmbH
//
// This file is part of the NAGO Low-Code Platform.
// Licensed under the terms specified in the LICENSE file.
//
// SPDX-License-Identifier: Custom-License

package service

import (
	"bytes"
	"errors"
	"github.com/worldiety/nago-runner/service/event"
	"log/slog"
	"os"
	"os/exec"
	"strings"
)

func NewExec() Exec {
	return func(req event.ExecRequest) (event.ExecResponse, error) {
		slog.Info("exec", req.Cmd, strings.Join(req.Args, " "))
		cmd := exec.Command(req.Cmd, req.Args...)

		var bufErr bytes.Buffer
		if !req.CollectErrOut {
			cmd.Stderr = os.Stderr
		} else {
			cmd.Stderr = &bufErr
		}

		var bufStd bytes.Buffer
		if !req.CollectStdOut {
			cmd.Stdout = os.Stdout
		} else {
			cmd.Stdout = &bufStd
		}

		res := event.ExecResponse{
			RequestID: req.RequestID,
			Cmd:       req.Cmd,
			Args:      req.Args,
		}

		if err := cmd.Run(); err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				res.ExitCode = exitErr.ExitCode()
				res.ExitCode = exitErr.ExitCode()
			}

			res.Error = err.Error()
			res.StdOut = bufStd.Bytes()
			res.ErrOut = bufErr.Bytes()
			return res, err
		}

		res.StdOut = bufStd.Bytes()
		res.ErrOut = bufErr.Bytes()

		return res, nil
	}
}
