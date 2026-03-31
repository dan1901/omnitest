package report

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/omnitest/omnitest/pkg/model"
)

// Generate는 TestResult를 지정된 형식으로 리포트 파일을 생성한다.
func Generate(result *model.TestResult, formats []string, outDir string) error {
	if len(formats) == 0 {
		return nil
	}

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	for _, format := range formats {
		switch format {
		case "json":
			path, err := generateJSON(result, outDir)
			if err != nil {
				return err
			}
			fmt.Printf("✓ JSON data: %s\n", path)
		case "html":
			path, err := generateHTML(result, outDir)
			if err != nil {
				return err
			}
			fmt.Printf("✓ Report saved: %s\n", path)
		default:
			return fmt.Errorf("unknown output format: %s", format)
		}
	}

	return nil
}

type jsonReport struct {
	ScenarioName string  `json:"scenario_name"`
	StartTime    string  `json:"start_time"`
	EndTime      string  `json:"end_time"`
	Duration     string  `json:"duration"`
	TotalReqs    int64   `json:"total_requests"`
	SuccessCount int64   `json:"success_count"`
	ErrorCount   int64   `json:"error_count"`
	ErrorRate    float64 `json:"error_rate"`
	AvgLatencyMs float64 `json:"avg_latency_ms"`
	P50LatencyMs float64 `json:"p50_latency_ms"`
	P95LatencyMs float64 `json:"p95_latency_ms"`
	P99LatencyMs float64 `json:"p99_latency_ms"`
	MaxLatencyMs float64 `json:"max_latency_ms"`
	MinLatencyMs float64 `json:"min_latency_ms"`
	AvgRPS       float64 `json:"avg_rps"`
	MaxRPS       float64 `json:"max_rps"`
	Thresholds   []jsonThreshold `json:"thresholds,omitempty"`
}

type jsonThreshold struct {
	Metric    string `json:"metric"`
	Condition string `json:"condition"`
	Actual    string `json:"actual"`
	Passed    bool   `json:"passed"`
}

func generateJSON(result *model.TestResult, outDir string) (string, error) {
	report := jsonReport{
		ScenarioName: result.ScenarioName,
		StartTime:    result.StartTime.Format(time.RFC3339),
		EndTime:      result.EndTime.Format(time.RFC3339),
		Duration:     result.Duration.String(),
		TotalReqs:    result.TotalRequests,
		SuccessCount: result.SuccessCount,
		ErrorCount:   result.ErrorCount,
		ErrorRate:    result.ErrorRate,
		AvgLatencyMs: float64(result.AvgLatency.Microseconds()) / 1000,
		P50LatencyMs: float64(result.P50Latency.Microseconds()) / 1000,
		P95LatencyMs: float64(result.P95Latency.Microseconds()) / 1000,
		P99LatencyMs: float64(result.P99Latency.Microseconds()) / 1000,
		MaxLatencyMs: float64(result.MaxLatency.Microseconds()) / 1000,
		MinLatencyMs: float64(result.MinLatency.Microseconds()) / 1000,
		AvgRPS:       result.AvgRPS,
		MaxRPS:       result.MaxRPS,
	}

	for _, t := range result.ThresholdResults {
		report.Thresholds = append(report.Thresholds, jsonThreshold{
			Metric:    t.Metric,
			Condition: t.Condition,
			Actual:    t.Actual,
			Passed:    t.Passed,
		})
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal JSON report: %w", err)
	}

	filename := fmt.Sprintf("report-%s.json", time.Now().Format("20060102-150405"))
	path := filepath.Join(outDir, filename)

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", fmt.Errorf("write JSON report: %w", err)
	}

	return path, nil
}
