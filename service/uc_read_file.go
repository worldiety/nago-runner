// Copyright (c) 2025 worldiety GmbH
//
// This file is part of the NAGO Low-Code Platform.
// Licensed under the terms specified in the LICENSE file.
//
// SPDX-License-Identifier: Custom-License

package service

import (
	"github.com/worldiety/nago-runner/service/event"
	"io"
	"os"
)

func NewReadFile() ReadFile {
	return func(req event.ReadFileRequested) (event.ReadFileResponse, error) {
		if req.MaxSize == 0 {
			req.MaxSize = 1024 * 1024
		}

		var res event.ReadFileResponse

		info, err := os.Stat(req.Path)
		if err != nil {
			return res, err
		}

		tmp := make([]byte, req.MaxSize)
		f, err := os.Open(req.Path)
		if err != nil {
			return res, err
		}

		defer f.Close()

		n, err := f.Read(tmp)
		if err != nil {
			if err != io.EOF {
				return res, err
			}
		}

		res.RequestID = req.RequestID
		res.Path = req.Path
		res.Content = tmp[:n]
		res.File = event.File{
			Name:    info.Name(),
			Mode:    info.Mode(),
			ModTime: info.ModTime(),
			Size:    info.Size(),
		}

		return res, nil
	}
}
