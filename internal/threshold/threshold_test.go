package threshold

import (
	"testing"
	"time"

	"github.com/omnitest/omnitest/pkg/model"
)

func TestEvaluateCondition(t *testing.T) {
	tests := []struct {
		name      string
		actual    float64
		condition string
		want      bool
	}{
		{"less than ms pass", 180, "< 200ms", true},
		{"less than ms fail", 250, "< 200ms", false},
		{"less than ms equal", 200, "< 200ms", false},
		{"less equal ms pass", 200, "<= 200ms", true},
		{"greater than pass", 1500, "> 1000", true},
		{"greater than fail", 500, "> 1000", false},
		{"greater equal pass", 1000, ">= 1000", true},
		{"less than percent pass", 0.5, "< 1%", true},
		{"less than percent fail", 2.1, "< 1%", false},
		{"seconds conversion", 1500, "< 2s", true},
		{"seconds conversion fail", 2500, "< 2s", false},
		{"no operator", 100, "200ms", false},
		{"whitespace handling", 180, " <  200ms ", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EvaluateCondition(tt.actual, tt.condition)
			if got != tt.want {
				t.Errorf("EvaluateCondition(%f, %q) = %v, want %v", tt.actual, tt.condition, got, tt.want)
			}
		})
	}
}

func TestEvaluate(t *testing.T) {
	result := &model.TestResult{
		P99Latency:    180 * time.Millisecond,
		P95Latency:    120 * time.Millisecond,
		P50Latency:    50 * time.Millisecond,
		AvgLatency:    60 * time.Millisecond,
		ErrorRate:     0.005,
		TotalRequests: 5000,
	}

	thresholds := []model.Threshold{
		{Metric: "http_req_duration_p99", Condition: "< 200ms"},
		{Metric: "http_req_failed", Condition: "< 1%"},
		{Metric: "http_reqs", Condition: "> 1000"},
		{Metric: "unknown_metric", Condition: "< 100"},
	}

	results := Evaluate(thresholds, result)

	if len(results) != 4 {
		t.Fatalf("results count = %d, want 4", len(results))
	}

	// p99 < 200ms → 180ms → PASS
	if !results[0].Passed {
		t.Errorf("p99 threshold: got FAIL, want PASS (actual: %s)", results[0].Actual)
	}
	if results[0].Actual != "180ms" {
		t.Errorf("p99 actual = %q, want %q", results[0].Actual, "180ms")
	}

	// error rate < 1% → 0.50% → PASS
	if !results[1].Passed {
		t.Errorf("error rate threshold: got FAIL, want PASS (actual: %s)", results[1].Actual)
	}

	// http_reqs > 1000 → 5000 → PASS
	if !results[2].Passed {
		t.Errorf("http_reqs threshold: got FAIL, want PASS (actual: %s)", results[2].Actual)
	}

	// unknown metric → FAIL
	if results[3].Passed {
		t.Error("unknown metric threshold: got PASS, want FAIL")
	}
}

func TestEvaluate_FailingThreshold(t *testing.T) {
	result := &model.TestResult{
		P99Latency: 300 * time.Millisecond,
		ErrorRate:  0.05,
	}

	thresholds := []model.Threshold{
		{Metric: "http_req_duration_p99", Condition: "< 200ms"},
		{Metric: "http_req_failed", Condition: "< 1%"},
	}

	results := Evaluate(thresholds, result)

	// p99 = 300ms, threshold < 200ms → FAIL
	if results[0].Passed {
		t.Errorf("p99 should FAIL: actual %s", results[0].Actual)
	}

	// error rate = 5%, threshold < 1% → FAIL
	if results[1].Passed {
		t.Errorf("error rate should FAIL: actual %s", results[1].Actual)
	}
}
