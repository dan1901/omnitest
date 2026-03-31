// Package ws implements a WebSocket hub for broadcasting real-time metrics.
package ws

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/omnitest/omnitest/pkg/model"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(_ *http.Request) bool {
		return true // Allow all origins in development
	},
}

const (
	writeWait  = 10 * time.Second
	pongWait   = 30 * time.Second
	pingPeriod = 25 * time.Second
	sendBufSz  = 256
)

// Client는 개별 WebSocket 연결이다.
type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
}

// Hub manages WebSocket connections and broadcasts messages.
type Hub struct {
	mu      sync.RWMutex
	clients map[*Client]struct{}
}

// NewHub creates a new WebSocket hub.
func NewHub() *Hub {
	return &Hub{
		clients: make(map[*Client]struct{}),
	}
}

// HandleWebSocket upgrades HTTP connections to WebSocket.
func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WS] upgrade error: %v", err)
		return
	}

	client := &Client{
		hub:  h,
		conn: conn,
		send: make(chan []byte, sendBufSz),
	}

	h.mu.Lock()
	h.clients[client] = struct{}{}
	h.mu.Unlock()

	// WritePump: send 채널에서 읽어 conn.WriteMessage (단일 goroutine에서 write)
	go client.writePump()

	// ReadPump: 클라이언트 메시지 읽기 (ping/pong 유지)
	client.readPump()
}

func (c *Client) readPump() {
	defer func() {
		c.hub.mu.Lock()
		delete(c.hub.clients, c)
		c.hub.mu.Unlock()
		close(c.send)
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			break
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// send 채널 닫힘 → close message 전송
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// BroadcastMetrics sends aggregated metrics to all connected clients.
func (h *Hub) BroadcastMetrics(data *model.AggregatedMetrics) {
	if data == nil {
		return
	}
	h.broadcast(map[string]any{
		"type": "metrics_update",
		"data": data,
	})
}

// BroadcastEvent sends a system event to all connected clients.
func (h *Hub) BroadcastEvent(eventType string, payload any) {
	h.broadcast(map[string]any{
		"type": eventType,
		"data": payload,
	})
}

func (h *Hub) broadcast(data any) {
	msg, err := json.Marshal(data)
	if err != nil {
		log.Printf("[WS] marshal error: %v", err)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		select {
		case client.send <- msg:
		default:
			// send 버퍼 초과 → 느린 클라이언트 제거
			go func(c *Client) {
				h.mu.Lock()
				delete(h.clients, c)
				h.mu.Unlock()
				close(c.send)
				c.conn.Close()
			}(client)
		}
	}
}

// ClientCount returns the number of connected clients.
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
