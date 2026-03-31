package ws

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/omnitest/omnitest/pkg/model"
)

func TestNewHub(t *testing.T) {
	hub := NewHub()
	if hub == nil {
		t.Fatal("NewHub() returned nil")
	}
	if hub.ClientCount() != 0 {
		t.Errorf("ClientCount() = %d, want 0", hub.ClientCount())
	}
}

func TestHub_WebSocketConnection(t *testing.T) {
	hub := NewHub()

	server := httptest.NewServer(http.HandlerFunc(hub.HandleWebSocket))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	if hub.ClientCount() != 1 {
		t.Errorf("ClientCount() = %d, want 1", hub.ClientCount())
	}

	ws.Close()
	time.Sleep(100 * time.Millisecond)

	if hub.ClientCount() != 0 {
		t.Errorf("ClientCount() after close = %d, want 0", hub.ClientCount())
	}
}

func TestHub_BroadcastEvent(t *testing.T) {
	hub := NewHub()

	server := httptest.NewServer(http.HandlerFunc(hub.HandleWebSocket))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer ws.Close()

	time.Sleep(50 * time.Millisecond)

	hub.BroadcastEvent("test_event", map[string]string{"key": "value"})

	ws.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read message: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(msg, &parsed); err != nil {
		t.Fatalf("Failed to parse message: %v", err)
	}

	if parsed["type"] != "test_event" {
		t.Errorf("event type = %v, want %q", parsed["type"], "test_event")
	}
}

func TestHub_MultipleClients(t *testing.T) {
	hub := NewHub()

	server := httptest.NewServer(http.HandlerFunc(hub.HandleWebSocket))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	clients := make([]*websocket.Conn, 3)
	for i := 0; i < 3; i++ {
		ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatalf("Client %d failed to connect: %v", i, err)
		}
		clients[i] = ws
	}

	time.Sleep(50 * time.Millisecond)

	if hub.ClientCount() != 3 {
		t.Errorf("ClientCount() = %d, want 3", hub.ClientCount())
	}

	hub.BroadcastEvent("multi_test", map[string]int{"count": 42})

	for i, ws := range clients {
		ws.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, msg, err := ws.ReadMessage()
		if err != nil {
			t.Fatalf("Client %d failed to read: %v", i, err)
		}
		var parsed map[string]interface{}
		json.Unmarshal(msg, &parsed)
		if parsed["type"] != "multi_test" {
			t.Errorf("Client %d: type = %v, want %q", i, parsed["type"], "multi_test")
		}
	}

	for _, ws := range clients {
		ws.Close()
	}

	time.Sleep(100 * time.Millisecond)

	if hub.ClientCount() != 0 {
		t.Errorf("ClientCount() after all close = %d, want 0", hub.ClientCount())
	}
}

func TestHub_BroadcastMetrics_Nil(t *testing.T) {
	hub := NewHub()
	hub.BroadcastMetrics(nil)
}

func TestHub_BroadcastMetrics_WithData(t *testing.T) {
	hub := NewHub()

	server := httptest.NewServer(http.HandlerFunc(hub.HandleWebSocket))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer ws.Close()

	time.Sleep(50 * time.Millisecond)

	hub.BroadcastMetrics(&model.AggregatedMetrics{
		TestRunID:    "run-1",
		TotalRPS:     1500.0,
		AvgLatencyMs: 12.5,
		P99LatencyMs: 45.0,
		TotalReqs:    50000,
		ActiveVUsers: 200,
	})

	ws.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read metrics: %v", err)
	}

	var parsed map[string]interface{}
	json.Unmarshal(msg, &parsed)
	if parsed["type"] != "metrics_update" {
		t.Errorf("type = %v, want %q", parsed["type"], "metrics_update")
	}
}

// WritePump 패턴 동시성 테스트: 여러 goroutine에서 동시에 broadcast
func TestHub_WritePump_ConcurrentBroadcast(t *testing.T) {
	hub := NewHub()

	server := httptest.NewServer(http.HandlerFunc(hub.HandleWebSocket))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// 5 clients 연결
	clients := make([]*websocket.Conn, 5)
	for i := 0; i < 5; i++ {
		ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatalf("Client %d failed to connect: %v", i, err)
		}
		clients[i] = ws
	}
	time.Sleep(50 * time.Millisecond)

	if hub.ClientCount() != 5 {
		t.Fatalf("ClientCount() = %d, want 5", hub.ClientCount())
	}

	// 10 goroutines가 동시에 broadcast (race condition 검증)
	var wg sync.WaitGroup
	broadcastCount := 10
	for i := 0; i < broadcastCount; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			hub.BroadcastEvent("concurrent_test", map[string]int{"seq": i})
		}(i)
	}
	wg.Wait()

	// 각 클라이언트가 모든 메시지를 수신했는지 확인
	for ci, ws := range clients {
		received := 0
		for received < broadcastCount {
			ws.SetReadDeadline(time.Now().Add(2 * time.Second))
			_, _, err := ws.ReadMessage()
			if err != nil {
				break
			}
			received++
		}
		if received != broadcastCount {
			t.Errorf("Client %d received %d messages, want %d", ci, received, broadcastCount)
		}
	}

	for _, ws := range clients {
		ws.Close()
	}
}

// 클라이언트 연결/해제 동시성 테스트
func TestHub_ConcurrentConnectDisconnect(t *testing.T) {
	hub := NewHub()

	server := httptest.NewServer(http.HandlerFunc(hub.HandleWebSocket))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
			if err != nil {
				return
			}
			time.Sleep(10 * time.Millisecond)
			ws.Close()
		}()
	}
	wg.Wait()

	time.Sleep(200 * time.Millisecond)

	if hub.ClientCount() != 0 {
		t.Errorf("ClientCount() after all concurrent connect/disconnect = %d, want 0", hub.ClientCount())
	}
}
