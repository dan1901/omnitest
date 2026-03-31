package runner

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/omnitest/omnitest/pkg/model"
)

func TestRun_BasicExecution(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	cfg := &model.TestConfig{
		Version: "1",
		Targets: []model.Target{
			{Name: "test", BaseURL: server.URL},
		},
		Scenarios: []model.Scenario{
			{
				Name:     "basic",
				Target:   "test",
				VUsers:   2,
				Duration: 2 * time.Second,
				Requests: []model.Request{
					{Method: "GET", Path: "/"},
				},
			},
		},
	}

	result, err := Run(context.Background(), cfg, Options{Quiet: true})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if result.TotalRequests == 0 {
		t.Error("expected at least some requests")
	}
	if result.ScenarioName != "basic" {
		t.Errorf("ScenarioName = %q, want %q", result.ScenarioName, "basic")
	}
	if result.Duration < 1*time.Second {
		t.Errorf("Duration = %v, want >= 1s", result.Duration)
	}
}

func TestRun_TargetNotFound(t *testing.T) {
	cfg := &model.TestConfig{
		Targets: []model.Target{
			{Name: "a", BaseURL: "http://localhost"},
		},
		Scenarios: []model.Scenario{
			{
				Name:   "test",
				Target: "nonexistent",
				VUsers: 1,
				Duration: 1 * time.Second,
				Requests: []model.Request{
					{Method: "GET", Path: "/"},
				},
			},
		},
	}

	_, err := Run(context.Background(), cfg, Options{Quiet: true})
	if err == nil {
		t.Error("expected error for nonexistent target")
	}
}
