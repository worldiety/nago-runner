// Copyright (c) 2025 worldiety GmbH
//
// This file is part of the NAGO Low-Code Platform.
// Licensed under the terms specified in the LICENSE file.
//
// SPDX-License-Identifier: Custom-License

package setup

import (
	"fmt"
	"github.com/worldiety/nago-runner/pkg/linux"
	"log/slog"
	"net/url"
	"strconv"
	"strings"
)

type Settings struct {
	URL   string `json:"url,omitempty"`
	Token string `json:"token,omitempty"`
}

func (s Settings) Endpoints() Endpoints {
	uri, err := url.Parse(s.URL)
	if err != nil {
		slog.Error("invalid settings, unable to parse URL", "url", s.URL)
		return Endpoints{}
	}

	var ep Endpoints
	ep.SSL = strings.ToLower(uri.Scheme) == "wss" || strings.ToLower(uri.Scheme) == "https"
	if uri.Port() != "" {
		p, err := strconv.Atoi(uri.Port())
		if err != nil {
			slog.Error("invalid settings, unable to parse Port", "url", s.URL)
		} else {
			ep.Port = p
		}
	}

	ep.Host = uri.Hostname()

	if ep.Port == 0 {
		if ep.SSL {
			ep.Port = 443
		} else {
			ep.Port = 80
		}
	}

	wsScheme := "ws"
	if ep.SSL {
		wsScheme = "wss"
	}

	httpSchema := "http"
	if ep.SSL {
		httpSchema = "https"
	}

	ep.RunnerWebsocket = fmt.Sprintf("%s://%s:%d/api/v1/runner", wsScheme, ep.Host, ep.Port)
	ep.RunnerConfiguration = fmt.Sprintf("%s://%s:%d/api/v1/configuration/runner", httpSchema, ep.Host, ep.Port)

	return ep
}

type Endpoints struct {
	SSL                 bool
	Host                string
	Port                int
	RunnerWebsocket     string
	RunnerConfiguration string
}

type ApplySettings func(Settings) linux.Result

type LoadSettings func() (Settings, error)

type InstallRunner func() error

type UseCases struct {
	ApplySettings ApplySettings
	LoadSettings  LoadSettings
	InstallRunner InstallRunner
}

func NewUseCases() UseCases {
	return UseCases{
		ApplySettings: NewApplySettings(),
		LoadSettings:  NewLoadSettings(),
		InstallRunner: NewInstallRunner(),
	}
}
