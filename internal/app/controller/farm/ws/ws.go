package ws

import (
	"encoding/json"
	"log/slog"

	"github.com/it00021hot/qq-farm-core/internal/app/controller"
	"github.com/it00021hot/qq-farm-core/internal/farm/hub"
	"github.com/gofiber/contrib/v3/websocket"
	"github.com/gofiber/fiber/v3"
)

type Controller struct{ controller.Controller }

var WS = &Controller{}

// Upgrade marks the request for websocket upgrade.
func (c *Controller) Upgrade(ctx fiber.Ctx) error {
	if websocket.IsWebSocketUpgrade(ctx) {
		return ctx.Next()
	}
	return fiber.ErrUpgradeRequired
}

// Handle streams hub events to the client.
func (c *Controller) Handle(conn *websocket.Conn) {
	id, ch := hub.Default.Subscribe()
	defer hub.Default.Unsubscribe(id)
	slog.Info("farm ws connected", "sub", id)
	if snap := hub.Logs.Query(0, "", "", 300); len(snap) > 0 {
		raw, err := json.Marshal(snap)
		if err == nil {
			if ev, err := json.Marshal(hub.Event{Type: "logs_snapshot", Payload: raw}); err == nil {
				_ = conn.WriteMessage(websocket.TextMessage, ev)
			}
		}
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()
	for {
		select {
		case <-done:
			return
		case msg, ok := <-ch:
			if !ok || conn.WriteMessage(websocket.TextMessage, msg) != nil {
				return
			}
		}
	}
}
