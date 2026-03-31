package output

import (
	"context"
	"testing"
	"time"

	"github.com/omnitest/omnitest/pkg/model"
)

func TestFormatLatency(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{500 * time.Microsecond, "500μs"},
		{10 * time.Millisecond, "10ms"},
		{1500 * time.Millisecond, "1500ms"},
		{0, "0μs"},
	}
	for _, tt := range tests {
		got := formatLatency(tt.d)
		if got != tt.want {
			t.Errorf("formatLatency(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{0, "00:00"},
		{30 * time.Second, "00:30"},
		{90 * time.Second, "01:30"},
		{5 * time.Minute, "05:00"},
	}
	for _, tt := range tests {
		got := formatDuration(tt.d)
		if got != tt.want {
			t.Errorf("formatDuration(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		n    int64
		want string
	}{
		{0, "0"},
		{999, "999"},
		{1000, "1,000"},
		{1234567, "1,234,567"},
	}
	for _, tt := range tests {
		got := formatNumber(tt.n)
		if got != tt.want {
			t.Errorf("formatNumber(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}

func TestColor(t *testing.T) {
	tests := []struct {
		name    string
		noColor bool
		code    string
		text    string
		want    string
	}{
		{"with color", false, colorGreen, "ok", "\033[32mok\033[0m"},
		{"no color", true, colorGreen, "ok", "ok"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewPrinter(nil, tt.noColor, 0)
			got := p.color(tt.code, tt.text)
			if got != tt.want {
				t.Errorf("color() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestColorizeErrorRate(t *testing.T) {
	tests := []struct {
		name    string
		rate    float64
		noColor bool
		wantSub string
	}{
		{"zero errors no-color", 0, true, "0.0%"},
		{"low errors no-color", 0.005, true, "0.5%"},
		{"high errors no-color", 0.05, true, "5.0%"},
		{"zero errors with color", 0, false, "0.0%"},
		{"low errors with color", 0.005, false, "0.5%"},
		{"high errors with color", 0.05, false, "5.0%"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewPrinter(nil, tt.noColor, 0)
			got := p.colorizeErrorRate(tt.rate)
			if !containsStr(got, tt.wantSub) {
				t.Errorf("colorizeErrorRate(%f) = %q, want substring %q", tt.rate, got, tt.wantSub)
			}
			if !tt.noColor {
				if !containsStr(got, "\033[") {
					t.Errorf("expected ANSI color codes in output %q", got)
				}
			}
		})
	}
}

func TestColorizeSuccessRate(t *testing.T) {
	tests := []struct {
		name    string
		rate    float64
		noColor bool
		wantSub string
	}{
		{"perfect", 100.0, true, "100.00%"},
		{"high", 99.95, true, "99.95%"},
		{"medium", 99.5, true, "99.50%"},
		{"low", 95.0, true, "95.00%"},
		{"with color green", 100.0, false, "100.00%"},
		{"with color yellow", 99.5, false, "99.50%"},
		{"with color red", 95.0, false, "95.00%"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewPrinter(nil, tt.noColor, 0)
			got := p.colorizeSuccessRate(tt.rate)
			if !containsStr(got, tt.wantSub) {
				t.Errorf("colorizeSuccessRate(%f) = %q, want substring %q", tt.rate, got, tt.wantSub)
			}
		})
	}
}

func TestProgressBar(t *testing.T) {
	tests := []struct {
		name     string
		progress float64
		width    int
	}{
		{"zero", 0.0, 20},
		{"half", 0.5, 20},
		{"full", 1.0, 20},
		{"overflow", 1.5, 20},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewPrinter(nil, true, 10*time.Second)
			got := p.progressBar(tt.progress, tt.width)
			if !containsStr(got, "[") || !containsStr(got, "]") {
				t.Errorf("progressBar() = %q, want brackets", got)
			}
		})
	}
}

func TestRender(t *testing.T) {
	p := NewPrinter(nil, true, 10*time.Second)
	snap := model.MetricSnapshot{
		ElapsedSec:   5.0,
		ActiveVUsers: 10,
		TotalVUsers:  10,
		RPS:          100,
		AvgLatency:   50 * time.Millisecond,
		P50Latency:   40 * time.Millisecond,
		P95Latency:   80 * time.Millisecond,
		P99Latency:   100 * time.Millisecond,
		ErrorRate:    0.01,
	}
	// render가 panic 없이 동작하는지 확인
	p.render(snap)
	if p.lines == 0 {
		t.Error("expected lines to be set after render")
	}
}

func TestStart_CancelContext(t *testing.T) {
	callCount := 0
	snapFn := func() model.MetricSnapshot {
		callCount++
		return model.MetricSnapshot{
			ElapsedSec:   float64(callCount),
			ActiveVUsers: 5,
			TotalVUsers:  5,
			RPS:          50,
		}
	}

	p := NewPrinter(snapFn, true, 5*time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() {
		p.Start(ctx)
		close(done)
	}()

	select {
	case <-done:
		// Start가 ctx 취소 시 정상 종료됨
	case <-time.After(5 * time.Second):
		t.Fatal("Start did not return after context cancellation")
	}
}

func TestPrintSummary_VariousRates(t *testing.T) {
	tests := []struct {
		name    string
		success int64
		total   int64
	}{
		{"perfect", 1000, 1000},
		{"high", 999, 1000},
		{"low", 950, 1000},
		{"zero requests", 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewPrinter(nil, true, 0)
			result := &model.TestResult{
				ScenarioName:  tt.name,
				TotalRequests: tt.total,
				SuccessCount:  tt.success,
				Duration:      10 * time.Second,
				P50Latency:    10 * time.Millisecond,
				P95Latency:    50 * time.Millisecond,
				P99Latency:    100 * time.Millisecond,
				MaxLatency:    500 * time.Millisecond,
				AvgRPS:        100,
				MaxRPS:        120,
			}
			// panic 없이 동작하는지 확인
			p.PrintSummary(result)
		})
	}
}

func TestPrintThresholds_Empty(t *testing.T) {
	p := NewPrinter(nil, true, 0)
	// empty slice - should return immediately
	p.PrintThresholds(nil)
	p.PrintThresholds([]model.ThresholdResult{})
}

func TestPrintThresholds_Mixed(t *testing.T) {
	p := NewPrinter(nil, false, 0) // with color
	results := []model.ThresholdResult{
		{Metric: "p99", Condition: "< 200ms", Actual: "180ms", Passed: true},
		{Metric: "errors", Condition: "< 1%", Actual: "2.1%", Passed: false},
	}
	p.PrintThresholds(results)
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
