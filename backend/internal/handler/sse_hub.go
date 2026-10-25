package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

type SSEEvent struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

type SSEHub struct {
	clients    map[chan SSEEvent]bool
	register   chan chan SSEEvent
	unregister chan chan SSEEvent
	broadcast  chan SSEEvent
	mu         sync.Mutex
}

func NewSSEHub() *SSEHub {
	hub := &SSEHub{
		clients:    make(map[chan SSEEvent]bool),
		register:   make(chan chan SSEEvent),
		unregister: make(chan chan SSEEvent),
		broadcast:  make(chan SSEEvent, 64),
	}
	go hub.run()
	return hub
}

func (h *SSEHub) run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client)
			}
			h.mu.Unlock()
		case event := <-h.broadcast:
			h.mu.Lock()
			for client := range h.clients {
				select {
				case client <- event:
				default:
					// Slow client
				}
			}
			h.mu.Unlock()
		}
	}
}

func (h *SSEHub) Broadcast(eventType string, data interface{}) {
	h.broadcast <- SSEEvent{
		Type: eventType,
		Data: data,
	}
}

func (h *SSEHub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	clientChan := make(chan SSEEvent, 16)
	h.register <- clientChan

	defer func() {
		h.unregister <- clientChan
	}()

	// Send initial ping
	fmt.Fprintf(w, "event: ping\ndata: connected\n\n")
	flusher.Flush()

	notify := r.Context().Done()
	for {
		select {
		case <-notify:
			return
		case event, ok := <-clientChan:
			if !ok {
				return
			}
			payload, err := json.Marshal(event.Data)
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, payload)
			flusher.Flush()
		}
	}
}
