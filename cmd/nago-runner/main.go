// Copyright (c) 2025 worldiety GmbH
//
// This file is part of the NAGO Low-Code Platform.
// Licensed under the terms specified in the LICENSE file.
//
// SPDX-License-Identifier: Custom-License

package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/worldiety/nago-runner/apply"
	"github.com/worldiety/nago-runner/apply/caddy"
	"github.com/worldiety/nago-runner/apply/systemd"
	"github.com/worldiety/nago-runner/service"
	"github.com/worldiety/nago-runner/service/event"
	"github.com/worldiety/nago-runner/service/event/gorilla"
	"github.com/worldiety/nago-runner/setup"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	if err := realMain(); err != nil {
		log.Fatal(err)
	}
}

func realMain() error {
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
			slog.Info("interrupt received")
			cancel()
		}
	}()

	endpoints := cfg.Endpoints()

	bus := gorilla.NewWebsocketBus(endpoints.RunnerWebsocket, cfg.Token)

	launch(ctx, bus, cfg)

	go func() {
		if err := bus.Run(ctx); err != nil {
			slog.Error("cannot run bus", "err", err.Error())
			return
		}
	}()

	select {
	case <-ctx.Done():
		return nil
	}

}

func launch(ctx context.Context, bus *gorilla.WebsocketBus, settings setup.Settings) {
	ucService := service.NewUseCases(bus, settings)
	ucService.ScheduleStatistics(ctx)

	// the bus notifies every event on its own goroutine, thus configuration changes must be serialized here. A
	// failed apply is retried, see apply.Serial.
	caddyApplier := caddy.NewApplier()
	systemdApplier := systemd.NewApplier()
	applier := apply.NewSerial(ctx, func() error {
		return applyConfiguration(settings, caddyApplier, systemdApplier)
	})

	bus.Subscribe(func(obj event.Event) {
		switch obj.(type) {
		case event.RunnerConfigurationChanged:
			slog.Info("configuration change announced by the hub")
			applier.Request()
		case event.ConnectionCreated:
			// changes announced while the connection was down are lost, thus apply after every (re)connect
			slog.Info("connected to the hub, applying the current configuration")
			applier.Request()
		}
	})

}

// applyConfiguration fetches and applies the current configuration. A returned error means that the apply should
// be retried later.
func applyConfiguration(settings setup.Settings, caddyApplier *caddy.Applier, systemdApplier *systemd.Applier) error {
	cfg, err := apply.QueryConfiguration(settings)
	if err != nil {
		slog.Error("cannot load configuration", "err", err.Error())
		return fmt.Errorf("cannot load configuration: %w", err)
	}

	slog.Info("applying configuration", "instances", len(cfg.Applications))

	// an empty configuration is far more likely a hub failure than the intention to undeploy
	// everything. Applying it would remove all reverse proxy rules and services at once.
	if len(cfg.Applications) == 0 {
		n, err := systemd.CountManagedServices(slog.Default())
		if err != nil {
			slog.Error("cannot count managed services, refusing to apply an empty configuration", "err", err.Error())
			return fmt.Errorf("cannot count managed services: %w", err)
		}

		if n > 0 {
			slog.Error("refusing to apply an empty configuration while managed services exist, remove them manually if intended", "count", n)
			return fmt.Errorf("refusing to apply an empty configuration while %d managed services exist", n)
		}
	}

	var errs []error
	if err := caddyApplier.Apply(slog.Default(), settings, cfg); err != nil {
		slog.Error("cannot apply caddy configuration", "err", err.Error())
		errs = append(errs, err)
	}

	if err := systemdApplier.Apply(slog.Default(), settings, cfg); err != nil {
		slog.Error("cannot apply systemd configuration", "err", err.Error())
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}
