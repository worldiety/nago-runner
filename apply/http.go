// Copyright (c) 2025 worldiety GmbH
//
// This file is part of the NAGO Low-Code Platform.
// Licensed under the terms specified in the LICENSE file.
//
// SPDX-License-Identifier: Custom-License

package apply

import (
	"encoding/json"
	"github.com/worldiety/nago-runner/configuration"
	"github.com/worldiety/nago-runner/setup"
	"net/http"
	"time"
)

func QueryConfiguration(settings setup.Settings) (configuration.Runner, error) {

	req, err := http.NewRequest("GET", settings.Endpoints().RunnerConfiguration, nil)
	if err != nil {
		return configuration.Runner{}, err
	}

	req.Header.Set("Authorization", "Bearer "+settings.Token)

	client := &http.Client{
		Timeout: time.Second * 30,
	}

	resp, err := client.Do(req)
	if err != nil {
		return configuration.Runner{}, err
	}

	defer resp.Body.Close()

	var cfg configuration.Runner
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&cfg); err != nil {
		return configuration.Runner{}, err
	}

	return cfg, nil
}
