package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

type SSEBroker struct {
	mu      sync.RWMutex
	clients map[chan []byte]bool
}

func NewSSEBroker() *SSEBroker {
	return &SSEBroker{
		clients: make(map[chan []byte]bool),
	}
}

func (b *SSEBroker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	// Allowing CORS for development if we run UI separately
	w.Header().Set("Access-Control-Allow-Origin", "*")

	msgChan := make(chan []byte, 100)
	
	b.mu.Lock()
	b.clients[msgChan] = true
	b.mu.Unlock()

	defer func() {
		b.mu.Lock()
		delete(b.clients, msgChan)
		b.mu.Unlock()
		close(msgChan)
	}()

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Send an initial connected message
	fmt.Fprintf(w, "data: {\"event\":\"connected\"}\n\n")
	flusher.Flush()

	for {
		select {
		case msg := <-msgChan:
			fmt.Fprintf(w, "data: %s\n\n", msg)
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

func (b *SSEBroker) Broadcast(event string, data any) {
	payload, err := json.Marshal(map[string]any{
		"event": event,
		"data":  data,
	})
	if err != nil {
		return
	}
	
	b.mu.RLock()
	defer b.mu.RUnlock()
	
	for client := range b.clients {
		select {
		case client <- payload:
		default:
			// If client channel is full, drop the message
		}
	}
}

type SSELogWriter struct {
	Broker *SSEBroker
}

func (w *SSELogWriter) Write(p []byte) (n int, err error) {
	if w.Broker == nil {
		return len(p), nil
	}
	
	// Zap may output JSON (production) or plaintext (development)
	var logData map[string]any
	if err := json.Unmarshal(p, &logData); err == nil {
		w.Broker.Broadcast("syslog", logData)
	} else {
		// Fallback for plain text logs
		w.Broker.Broadcast("syslog", string(p))
	}
	return len(p), nil
}
