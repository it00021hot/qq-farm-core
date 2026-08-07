package ws

import (
	"encoding/json"
	"log/slog"

	"github.com/MQEnergy/go-skeleton/internal/app/controller"
	"github.com/MQEnergy/go-skeleton/internal/farm/hub"
	"github.com/MQEnergy/go-skeleton/pkg/tenant"
	"github.com/gofiber/contrib/v3/websocket"
	"github.com/gofiber/fiber/v3"
	"github.com/spf13/cast"
)

type Controller struct {
	controller.Controller
}

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
	tid := cast.ToUint64(conn.Locals(tenant.LocalTenantID))
	if tid == 0 {
		tid = cast.ToUint64(conn.Query("tenantId"))
	}
	id, ch := hub.Default.Subscribe(tid)
	defer hub.Default.Unsubscribe(id)
	slog.Info("farm ws connected", "tenantId", tid, "sub", id)

	// Seed client with recent run-log ring snapshot (HTTP /farm/logs is primary).
	if snap := hub.Logs.Query(0, "", "", 300); len(snap) > 0 {
		raw, err := json.Marshal(snap)
		if err == nil {
			ev, err := json.Marshal(hub.Event{Type: "logs_snapshot", Payload: raw})
			if err == nil {
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
			if !ok {
				return
			}
			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		}
	}
}
