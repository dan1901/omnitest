package output

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/omnitest/omnitest/pkg/model"
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
)

// Printer는 터미널 실시간 메트릭 출력기다.
type Printer struct {
	snapshotFn func() model.MetricSnapshot
	noColor    bool
	duration   time.Duration
	lines      int // 지난번 출력한 줄 수 (in-place 갱신용)
}

// NewPrinter는 새 출력기를 생성한다.
func NewPrinter(snapshotFn func() model.MetricSnapshot, noColor bool, duration time.Duration) *Printer {
	return &Printer{
		snapshotFn: snapshotFn,
		noColor:    noColor,
		duration:   duration,
	}
}

// Start는 1초 간격으로 터미널을 갱신하는 goroutine을 시작한다.
func (p *Printer) Start(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			snap := p.snapshotFn()
			p.render(snap)
		}
	}
}

func (p *Printer) render(snap model.MetricSnapshot) {
	// 이전 출력 지우기
	if p.lines > 0 {
		fmt.Printf("\033[%dA", p.lines)
	}

	var sb strings.Builder

	elapsed := time.Duration(snap.ElapsedSec * float64(time.Second))
	remaining := p.duration - elapsed
	if remaining < 0 {
		remaining = 0
	}

	// 헤더
	sb.WriteString(p.color(colorBold, "  Elapsed   VUsers   RPS      Avg      P50      P95      P99      Errors\n"))
	sb.WriteString("  ─────────────────────────────────────────────────────────────────────────\n")

	// 메트릭 행
	sb.WriteString(fmt.Sprintf("  %-9s %-8s %-8s %-8s %-8s %-8s %-8s %s\n",
		formatDuration(elapsed),
		fmt.Sprintf("%d/%d", snap.ActiveVUsers, snap.TotalVUsers),
		fmt.Sprintf("%.0f", snap.RPS),
		formatLatency(snap.AvgLatency),
		formatLatency(snap.P50Latency),
		formatLatency(snap.P95Latency),
		formatLatency(snap.P99Latency),
		p.colorizeErrorRate(snap.ErrorRate),
	))
	sb.WriteString("\n")

	// 프로그레스 바
	progress := snap.ElapsedSec / p.duration.Seconds()
	if progress > 1 {
		progress = 1
	}
	sb.WriteString(fmt.Sprintf("  %s %3.0f%% | %s remaining\n",
		p.progressBar(progress, 30),
		progress*100,
		formatDuration(remaining),
	))

	output := sb.String()
	fmt.Print(output)
	p.lines = strings.Count(output, "\n")
}

// PrintSummary는 테스트 완료 후 최종 요약을 출력한다.
func (p *Printer) PrintSummary(result *model.TestResult) {
	fmt.Println()
	fmt.Println(p.color(colorBold, "──────────────────── Test Summary ────────────────────"))
	fmt.Println()

	successRate := 100.0
	if result.TotalRequests > 0 {
		successRate = float64(result.SuccessCount) / float64(result.TotalRequests) * 100
	}

	fmt.Printf("  Total Requests:   %s\n", formatNumber(result.TotalRequests))
	fmt.Printf("  Total Duration:   %s\n", result.Duration.Truncate(time.Second))
	fmt.Printf("  Success Rate:     %s\n", p.colorizeSuccessRate(successRate))
	fmt.Println()
	fmt.Println("  Latency Distribution:")
	fmt.Printf("    P50:   %s\n", formatLatency(result.P50Latency))
	fmt.Printf("    P95:   %s\n", formatLatency(result.P95Latency))
	fmt.Printf("    P99:   %s\n", formatLatency(result.P99Latency))
	fmt.Printf("    Max:   %s\n", formatLatency(result.MaxLatency))
	fmt.Println()
	fmt.Println("  RPS:")
	fmt.Printf("    Avg:   %.0f\n", result.AvgRPS)
	fmt.Printf("    Max:   %.0f\n", result.MaxRPS)
	fmt.Println()
	fmt.Println(p.color(colorBold, "──────────────────────────────────────────────────────"))
}

// PrintThresholds는 threshold 평가 결과를 출력한다.
func (p *Printer) PrintThresholds(results []model.ThresholdResult) {
	if len(results) == 0 {
		return
	}
	fmt.Println()
	fmt.Println("  Thresholds:")
	for _, r := range results {
		if r.Passed {
			fmt.Printf("    %s %s (%s) ... %s\n",
				p.color(colorGreen, "✓"),
				r.Metric,
				r.Actual,
				p.color(colorGreen, "PASS"),
			)
		} else {
			fmt.Printf("    %s %s (%s) ... %s (%s)\n",
				p.color(colorRed, "✗"),
				r.Metric,
				r.Actual,
				p.color(colorRed, "FAIL"),
				r.Condition,
			)
		}
	}
}

func (p *Printer) color(code, text string) string {
	if p.noColor {
		return text
	}
	return code + text + colorReset
}

func (p *Printer) colorizeErrorRate(rate float64) string {
	text := fmt.Sprintf("%.1f%%", rate*100)
	if rate == 0 {
		return p.color(colorGreen, text)
	}
	if rate < 0.01 {
		return p.color(colorYellow, text)
	}
	return p.color(colorRed, text)
}

func (p *Printer) colorizeSuccessRate(rate float64) string {
	text := fmt.Sprintf("%.2f%%", rate)
	if rate >= 99.9 {
		return p.color(colorGreen, text)
	}
	if rate >= 99.0 {
		return p.color(colorYellow, text)
	}
	return p.color(colorRed, text)
}

func (p *Printer) progressBar(progress float64, width int) string {
	filled := int(progress * float64(width))
	if filled > width {
		filled = width
	}
	empty := width - filled
	return p.color(colorCyan, "["+strings.Repeat("█", filled)+strings.Repeat("░", empty)+"]")
}

func formatDuration(d time.Duration) string {
	d = d.Truncate(time.Second)
	m := int(d.Minutes())
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d", m, s)
}

func formatLatency(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%dμs", d.Microseconds())
	}
	return fmt.Sprintf("%dms", d.Milliseconds())
}

func formatNumber(n int64) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var result []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(c))
	}
	return string(result)
}
