// Copyright (c) 2025 worldiety GmbH
//
// This file is part of the NAGO Low-Code Platform.
// Licensed under the terms specified in the LICENSE file.
//
// SPDX-License-Identifier: Custom-License

package main

import (
	"context"
	"fmt"
	"github.com/worldiety/nago-runner/service"
	"github.com/worldiety/nago-runner/service/event/gorilla"
	"github.com/worldiety/nago-runner/setup"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func runService() error {
	ucSetup := setup.NewUseCases()

	cfg, err := ucSetup.LoadSettings()
	if err != nil {
		return fmt.Errorf("cannot load settings: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
	go func() {
		select {
		case <-interrupt:
			cancel()
		}
	}()

	bus := gorilla.NewWebsocketBus(cfg.URL, cfg.Token)
	ucService := service.NewUseCases(bus)

	go func() {
		if err := bus.Run(ctx); err != nil {
			slog.Error("cannot run bus", "err", err.Error())
			return
		}
	}()

	ucService.ScheduleStatistics(ctx)

	select {
	case <-ctx.Done():
		return nil
	}

}
