// Copyright (c) 2025 worldiety GmbH
//
// This file is part of the NAGO Low-Code Platform.
// Licensed under the terms specified in the LICENSE file.
//
// SPDX-License-Identifier: Custom-License

package apply

import (
	"encoding/json"
	"fmt"
	"github.com/worldiety/nago-runner/configuration"
	"github.com/worldiety/nago-runner/setup"
	"io"
	"mime"
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

	return decodeConfiguration(resp)
}

// decodeConfiguration only accepts a successful json response. This is critical, because the runner removes
// all managed services which are not declared, thus any error body (e.g. a problem+json document) which
// decodes into an empty configuration must never be applied.
func decodeConfiguration(resp *http.Response) (configuration.Runner, error) {
	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		return configuration.Runner{}, fmt.Errorf("unexpected status code from hub: %d", resp.StatusCode)
	}

	// note: older hubs do not declare a content type at all, thus we cannot require application/json
	if mediaType, _, err := mime.ParseMediaType(resp.Header.Get("Content-Type")); err == nil && mediaType == "application/problem+json" {
		return configuration.Runner{}, fmt.Errorf("hub responded with an error document")
	}

	var cfg configuration.Runner
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&cfg); err != nil {
		return configuration.Runner{}, err
	}

	return cfg, nil
}
