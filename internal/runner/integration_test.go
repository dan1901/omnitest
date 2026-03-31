package runner

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/omnitest/omnitest/pkg/model"
)

func newTestServer() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/get", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/post", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"created": "true"})
	})
	mux.HandleFunc("/slow", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/error", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	return httptest.NewServer(mux)
}

func TestIntegration_GetRequest(t *testing.T) {
	server := newTestServer()
	defer server.Close()

	cfg := &model.TestConfig{
		Version: "1",
		Targets: []model.Target{
			{Name: "local", BaseURL: server.URL},
		},
		Scenarios: []model.Scenario{
			{
				Name:     "integration-get",
				Target:   "local",
				VUsers:   5,
				Duration: 2 * time.Second,
				Requests: []model.Request{
					{Method: "GET", Path: "/get"},
				},
			},
		},
	}

	result, err := Run(context.Background(), cfg, Options{Quiet: true})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if result.TotalRequests == 0 {
		t.Error("expected requests to be made")
	}
	if result.SuccessCount == 0 {
		t.Error("expected successful requests")
	}
	if result.ErrorRate > 0.01 {
		t.Errorf("unexpected error rate: %f (want < 1%%)", result.ErrorRate)
	}
	if result.P50Latency <= 0 {
		t.Error("expected P50 latency > 0")
	}
	if result.AvgRPS <= 0 {
		t.Error("expected AvgRPS > 0")
	}
}

func TestIntegration_PostRequest(t *testing.T) {
	server := newTestServer()
	defer server.Close()

	cfg := &model.TestConfig{
		Version: "1",
		Targets: []model.Target{
			{Name: "local", BaseURL: server.URL},
		},
		Scenarios: []model.Scenario{
			{
				Name:     "integration-post",
				Target:   "local",
				VUsers:   2,
				Duration: 2 * time.Second,
				Requests: []model.Request{
					{Method: "POST", Path: "/post", Body: map[string]any{"key": "value"}},
				},
			},
		},
	}

	result, err := Run(context.Background(), cfg, Options{Quiet: true})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if result.TotalRequests == 0 {
		t.Error("expected requests to be made")
	}
}

func TestIntegration_MultipleRequests(t *testing.T) {
	server := newTestServer()
	defer server.Close()

	cfg := &model.TestConfig{
		Version: "1",
		Targets: []model.Target{
			{Name: "local", BaseURL: server.URL},
		},
		Scenarios: []model.Scenario{
			{
				Name:     "integration-multi",
				Target:   "local",
				VUsers:   3,
				Duration: 2 * time.Second,
				Requests: []model.Request{
					{Method: "GET", Path: "/get"},
					{Method: "POST", Path: "/post", Body: map[string]any{"test": true}},
				},
			},
		},
	}

	result, err := Run(context.Background(), cfg, Options{Quiet: true})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if result.TotalRequests == 0 {
		t.Error("expected requests to be made")
	}
	if result.Duration < 1*time.Second {
		t.Errorf("Duration = %v, want >= 1s", result.Duration)
	}
}

func TestIntegration_WithHeaders(t *testing.T) {
	var mu sync.Mutex
	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotAuth = r.Header.Get("Authorization")
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := &model.TestConfig{
		Version: "1",
		Targets: []model.Target{
			{
				Name:    "auth-server",
				BaseURL: server.URL,
				Headers: map[string]string{"Authorization": "Bearer test-token"},
			},
		},
		Scenarios: []model.Scenario{
			{
				Name:     "integration-headers",
				Target:   "auth-server",
				VUsers:   1,
				Duration: 1 * time.Second,
				Requests: []model.Request{
					{Method: "GET", Path: "/"},
				},
			},
		},
	}

	_, err := Run(context.Background(), cfg, Options{Quiet: true})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	mu.Lock()
	auth := gotAuth
	mu.Unlock()
	if auth != "Bearer test-token" {
		t.Errorf("Authorization = %q, want %q", auth, "Bearer test-token")
	}
}

func TestIntegration_ContextCancellation(t *testing.T) {
	server := newTestServer()
	defer server.Close()

	cfg := &model.TestConfig{
		Version: "1",
		Targets: []model.Target{
			{Name: "local", BaseURL: server.URL},
		},
		Scenarios: []model.Scenario{
			{
				Name:     "integration-cancel",
				Target:   "local",
				VUsers:   5,
				Duration: 30 * time.Second,
				Requests: []model.Request{
					{Method: "GET", Path: "/get"},
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	result, err := Run(ctx, cfg, Options{Quiet: true})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if result.Duration > 3*time.Second {
		t.Errorf("Duration = %v, expected early termination", result.Duration)
	}
}

func TestIntegration_WithRampUp(t *testing.T) {
	server := newTestServer()
	defer server.Close()

	cfg := &model.TestConfig{
		Version: "1",
		Targets: []model.Target{
			{Name: "local", BaseURL: server.URL},
		},
		Scenarios: []model.Scenario{
			{
				Name:     "integration-rampup",
				Target:   "local",
				VUsers:   4,
				Duration: 3 * time.Second,
				RampUp:   1 * time.Second,
				Requests: []model.Request{
					{Method: "GET", Path: "/get"},
				},
			},
		},
	}

	result, err := Run(context.Background(), cfg, Options{Quiet: true})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if result.TotalRequests == 0 {
		t.Error("expected requests")
	}
}
