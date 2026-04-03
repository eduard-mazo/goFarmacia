package backend

import (
	"encoding/json"
	"sync"
)

// EventBus is a global SSE broadcaster. Services call EventBus.Emit(event, data)
// instead of wailsruntime.EventsEmit. HTTP handler in api/handlers/sse.go
// subscribes clients and forwards messages.
var EventBus = newEventBroadcaster()

type eventBroadcaster struct {
	mu      sync.RWMutex
	clients map[chan []byte]struct{}
}

func newEventBroadcaster() *eventBroadcaster {
	return &eventBroadcaster{clients: make(map[chan []byte]struct{})}
}

// Subscribe returns a channel that will receive SSE-formatted messages.
func (e *eventBroadcaster) Subscribe() chan []byte {
	ch := make(chan []byte, 32)
	e.mu.Lock()
	e.clients[ch] = struct{}{}
	e.mu.Unlock()
	return ch
}

// Unsubscribe removes the channel and closes it.
func (e *eventBroadcaster) Unsubscribe(ch chan []byte) {
	e.mu.Lock()
	delete(e.clients, ch)
	e.mu.Unlock()
	close(ch)
}

// Emit broadcasts an SSE event to all connected clients.
func (e *eventBroadcaster) Emit(event string, data any) {
	payload, _ := json.Marshal(map[string]any{"event": event, "data": data})
	// SSE wire format: "data: <json>\n\n"
	msg := append([]byte("data: "), append(payload, '\n', '\n')...)
	e.mu.RLock()
	for ch := range e.clients {
		select {
		case ch <- msg:
		default: // drop if client buffer full (slow consumer)
		}
	}
	e.mu.RUnlock()
}
