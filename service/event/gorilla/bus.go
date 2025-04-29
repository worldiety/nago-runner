// Copyright (c) 2025 worldiety GmbH
//
// This file is part of the NAGO Low-Code Platform.
// Licensed under the terms specified in the LICENSE file.
//
// SPDX-License-Identifier: Custom-License

package gorilla

import (
	"context"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/worldiety/enum/json"
	"github.com/worldiety/nago-runner/service/event"
	"log"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

const retryDelay = 5 * time.Second

type WebsocketBus struct {
	url         string
	token       string
	conn        atomic.Pointer[websocket.Conn]
	mutex       sync.Mutex
	subscribers map[int]func(obj event.Event)
	lastHnd     int
}

func NewWebsocketBus(url string, token string) *WebsocketBus {
	return &WebsocketBus{url: url, token: token, subscribers: make(map[int]func(obj event.Event))}
}

func (b *WebsocketBus) Publish(obj event.Event) {
	buf, err := json.MarshalFor[event.Event](obj)
	if err != nil {
		slog.Error("failed to marshal websocket json message", "err", err.Error())
		return
	}

	c := b.conn.Load()
	if c == nil {
		slog.Error("websocket connection is gone")
		return
	}

	if err := c.WriteMessage(websocket.TextMessage, buf); err != nil {
		slog.Error("failed to write websocket json message", "err", err.Error())
	}
}

func (b *WebsocketBus) Subscribe(fn func(obj event.Event)) (close func()) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	b.lastHnd = b.lastHnd + 1
	myHandle := b.lastHnd
	b.subscribers[myHandle] = fn
	return func() {
		b.mutex.Lock()
		defer b.mutex.Unlock()
		delete(b.subscribers, myHandle)
	}
}

func (b *WebsocketBus) notify(msg []byte) {
	var evt event.Event
	if err := json.Unmarshal(msg, &evt); err != nil {
		slog.Error("failed to unmarshal websocket event", "err", err.Error())
		return
	}

	b.mutex.Lock()
	defer b.mutex.Unlock()
	for _, f := range b.subscribers {
		go f(evt)
	}
}

func (b *WebsocketBus) Run(ctx context.Context) error {
	for {
		conn, err := b.connect(ctx)
		if err != nil {
			slog.Error("connection failed", "err", err.Error())
			log.Printf("retrying in %s...", retryDelay)
			select {
			case <-time.After(retryDelay):
				continue
			case <-ctx.Done():
				if conn != nil {
					err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
					if err != nil {
						slog.Error("write close error:", "err", err.Error())
					}
					_ = conn.Close()

				}

				slog.Error("interrupt received during retry. Exiting...")
				return ctx.Err()
			}
		}

		b.readMessages(ctx, conn)

		slog.Info("connected. Listening for messages... Press Ctrl+C to exit.")

	}
}

func (b *WebsocketBus) readMessages(ctx context.Context, conn *websocket.Conn) {
	b.conn.Store(conn)
	defer b.conn.Store(nil)

	buf, err := json.MarshalFor[event.Event](event.ConnectionCreated{})
	if err != nil {
		slog.Error("failed to marshal websocket dummy json message", "err", err.Error())
		return
	}

	b.notify(buf)

	for {
		select {
		case <-ctx.Done():
			slog.Error("stopping message reader...")
			return
		default:
			_, message, err := conn.ReadMessage()
			if err != nil {
				slog.Error("read error:", "err", err.Error())
				return
			}

			b.notify(message)
		}
	}
}

func (b *WebsocketBus) connect(ctx context.Context) (*websocket.Conn, error) {
	slog.Info("connecting to websocket", "url", b.url)

	conn, res, err := websocket.DefaultDialer.DialContext(ctx, b.url, map[string][]string{
		"Authorization": {"Bearer " + b.token},
	})

	if res != nil && res.StatusCode != http.StatusOK && res.StatusCode != http.StatusNoContent && res.StatusCode != http.StatusSwitchingProtocols {
		slog.Error("unexpected status code", "code", res.Status)
	}

	if err != nil {
		return nil, fmt.Errorf("websocket dial error: %w", err)
	}

	return conn, nil
}
