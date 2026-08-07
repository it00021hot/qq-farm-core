package hub

import (
	"encoding/json"
	"sync"
)

// Event is a WS broadcast payload.
type Event struct {
	Type      string          `json:"type"`
	AccountID uint64          `json:"accountId,omitempty"`
	TenantID  uint64          `json:"tenantId,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
}

type client struct {
	tenantID uint64
	ch       chan []byte
}

// Hub is an in-process pub/sub for farm realtime events.
type Hub struct {
	mu      sync.RWMutex
	clients map[uint64]*client // id -> client
	nextID  uint64
}

// Default global hub.
var Default = New()

func New() *Hub {
	return &Hub{clients: make(map[uint64]*client)}
}

// Subscribe registers a tenant-scoped listener; returns id and receive channel.
func (h *Hub) Subscribe(tenantID uint64) (uint64, <-chan []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.nextID++
	id := h.nextID
	ch := make(chan []byte, 64)
	h.clients[id] = &client{tenantID: tenantID, ch: ch}
	return id, ch
}

// Unsubscribe removes a listener.
func (h *Hub) Unsubscribe(id uint64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if c, ok := h.clients[id]; ok {
		close(c.ch)
		delete(h.clients, id)
	}
}

// Broadcast sends an event to matching tenants (0 = all).
func (h *Hub) Broadcast(ev Event) {
	b, err := json.Marshal(ev)
	if err != nil {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, c := range h.clients {
		if ev.TenantID != 0 && c.tenantID != 0 && c.tenantID != ev.TenantID {
			continue
		}
		select {
		case c.ch <- b:
		default:
			// drop if slow consumer
		}
	}
}

// PublishJSON helpers. Log-worthy types are also appended to the run-log ring.
func (h *Hub) PublishJSON(typ string, tenantID, accountID uint64, payload any) {
	AppendFromEvent(typ, accountID, payload)
	raw, _ := json.Marshal(payload)
	h.Broadcast(Event{Type: typ, TenantID: tenantID, AccountID: accountID, Payload: raw})
}
