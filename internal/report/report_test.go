package report

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/omnitest/omnitest/pkg/model"
)

func testResult() *model.TestResult {
	return &model.TestResult{
		ScenarioName:  "test-scenario",
		StartTime:     time.Now().Add(-10 * time.Second),
		EndTime:       time.Now(),
		Duration:      10 * time.Second,
		TotalRequests: 1000,
		SuccessCount:  990,
		ErrorCount:    10,
		ErrorRate:     0.01,
		AvgLatency:    50 * time.Millisecond,
		P50Latency:    40 * time.Millisecond,
		P95Latency:    100 * time.Millisecond,
		P99Latency:    200 * time.Millisecond,
		MaxLatency:    500 * time.Millisecond,
		MinLatency:    5 * time.Millisecond,
		AvgRPS:        100,
		MaxRPS:        150,
	}
}

func TestGenerateJSON(t *testing.T) {
	dir := t.TempDir()
	result := testResult()

	err := Generate(result, []string{"json"}, dir)
	if err != nil {
		t.Fatalf("Generate JSON failed: %v", err)
	}

	files, _ := filepath.Glob(filepath.Join(dir, "report-*.json"))
	if len(files) != 1 {
		t.Fatalf("expected 1 JSON file, got %d", len(files))
	}

	data, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "test-scenario") {
		t.Error("JSON should contain scenario name")
	}
}

func TestGenerateHTML(t *testing.T) {
	dir := t.TempDir()
	result := testResult()
	result.Snapshots = []model.MetricSnapshot{
		{RPS: 100, P99Latency: 200 * time.Millisecond},
		{RPS: 120, P99Latency: 180 * time.Millisecond},
	}

	err := Generate(result, []string{"html"}, dir)
	if err != nil {
		t.Fatalf("Generate HTML failed: %v", err)
	}

	files, _ := filepath.Glob(filepath.Join(dir, "report-*.html"))
	if len(files) != 1 {
		t.Fatalf("expected 1 HTML file, got %d", len(files))
	}

	data, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "test-scenario") {
		t.Error("HTML should contain scenario name")
	}
}

func TestGenerate_NoFormats(t *testing.T) {
	err := Generate(testResult(), nil, t.TempDir())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGenerate_UnknownFormat(t *testing.T) {
	err := Generate(testResult(), []string{"pdf"}, t.TempDir())
	if err == nil {
		t.Error("expected error for unknown format")
	}
}
