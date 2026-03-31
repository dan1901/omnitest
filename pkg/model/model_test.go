package model

import (
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func TestTestConfig_YAMLParsing(t *testing.T) {
	yamlData := `
version: "1"
targets:
  - name: "api"
    base_url: "https://api.example.com"
    headers:
      Authorization: "Bearer token"
scenarios:
  - name: "load test"
    target: "api"
    vusers: 100
    duration: "5m"
    ramp_up: "30s"
    requests:
      - method: GET
        path: "/users"
      - method: POST
        path: "/users"
        body:
          name: "test"
thresholds:
  - metric: "http_req_duration_p99"
    condition: "< 200ms"
`
	var cfg TestConfig
	if err := yaml.Unmarshal([]byte(yamlData), &cfg); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if cfg.Version != "1" {
		t.Errorf("version = %q, want %q", cfg.Version, "1")
	}
	if len(cfg.Targets) != 1 {
		t.Fatalf("targets count = %d, want 1", len(cfg.Targets))
	}
	if cfg.Targets[0].Name != "api" {
		t.Errorf("target name = %q, want %q", cfg.Targets[0].Name, "api")
	}
	if cfg.Targets[0].BaseURL != "https://api.example.com" {
		t.Errorf("base_url = %q", cfg.Targets[0].BaseURL)
	}
	if len(cfg.Scenarios) != 1 {
		t.Fatalf("scenarios count = %d, want 1", len(cfg.Scenarios))
	}
	if cfg.Scenarios[0].VUsers != 100 {
		t.Errorf("vusers = %d, want 100", cfg.Scenarios[0].VUsers)
	}
	if cfg.Scenarios[0].Duration != 5*time.Minute {
		t.Errorf("duration = %v, want 5m", cfg.Scenarios[0].Duration)
	}
	if cfg.Scenarios[0].RampUp != 30*time.Second {
		t.Errorf("ramp_up = %v, want 30s", cfg.Scenarios[0].RampUp)
	}
	if len(cfg.Scenarios[0].Requests) != 2 {
		t.Errorf("requests count = %d, want 2", len(cfg.Scenarios[0].Requests))
	}
	if len(cfg.Thresholds) != 1 {
		t.Errorf("thresholds count = %d, want 1", len(cfg.Thresholds))
	}
}
