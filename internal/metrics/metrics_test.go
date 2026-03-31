package metrics

import (
	"fmt"
	"testing"
	"time"

	"github.com/omnitest/omnitest/pkg/model"
)

func TestCollector_RecordAndSnapshot(t *testing.T) {
	c := NewCollector()

	for i := 0; i < 100; i++ {
		c.Record(model.RequestResult{
			StatusCode: 200,
			Latency:    10 * time.Millisecond,
			BytesIn:    100,
			Timestamp:  time.Now(),
		})
	}

	snap := c.Snapshot(10, 10)

	if snap.TotalReqs != 100 {
		t.Errorf("TotalReqs = %d, want 100", snap.TotalReqs)
	}
	if snap.TotalErrors != 0 {
		t.Errorf("TotalErrors = %d, want 0", snap.TotalErrors)
	}
	if snap.ErrorRate != 0 {
		t.Errorf("ErrorRate = %f, want 0", snap.ErrorRate)
	}
	if snap.P50Latency < 9*time.Millisecond || snap.P50Latency > 11*time.Millisecond {
		t.Errorf("P50Latency = %v, want ~10ms", snap.P50Latency)
	}
}

func TestCollector_ErrorRate(t *testing.T) {
	c := NewCollector()

	for i := 0; i < 90; i++ {
		c.Record(model.RequestResult{
			StatusCode: 200,
			Latency:    5 * time.Millisecond,
			Timestamp:  time.Now(),
		})
	}
	for i := 0; i < 10; i++ {
		c.Record(model.RequestResult{
			StatusCode: 500,
			Latency:    5 * time.Millisecond,
			Error:      fmt.Errorf("HTTP 500"),
			Timestamp:  time.Now(),
		})
	}

	snap := c.Snapshot(10, 10)

	if snap.ErrorRate < 0.09 || snap.ErrorRate > 0.11 {
		t.Errorf("ErrorRate = %f, want ~0.1", snap.ErrorRate)
	}
}

func TestCollector_Result(t *testing.T) {
	c := NewCollector()

	for i := 0; i < 50; i++ {
		c.Record(model.RequestResult{
			StatusCode: 200,
			Latency:    20 * time.Millisecond,
			BytesIn:    200,
			Timestamp:  time.Now(),
		})
	}

	start := time.Now().Add(-10 * time.Second)
	end := time.Now()

	result := c.Result("test-scenario", start, end, nil)

	if result.ScenarioName != "test-scenario" {
		t.Errorf("ScenarioName = %q, want %q", result.ScenarioName, "test-scenario")
	}
	if result.TotalRequests != 50 {
		t.Errorf("TotalRequests = %d, want 50", result.TotalRequests)
	}
	if result.SuccessCount != 50 {
		t.Errorf("SuccessCount = %d, want 50", result.SuccessCount)
	}
	if result.AvgRPS < 4 || result.AvgRPS > 6 {
		t.Errorf("AvgRPS = %f, want ~5", result.AvgRPS)
	}
}

func TestCollector_EmptyHistogram(t *testing.T) {
	c := NewCollector()
	snap := c.Snapshot(0, 0)

	if snap.TotalReqs != 0 {
		t.Errorf("TotalReqs = %d, want 0", snap.TotalReqs)
	}
	if snap.ErrorRate != 0 {
		t.Errorf("ErrorRate = %f, want 0", snap.ErrorRate)
	}
}
