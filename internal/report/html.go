package report

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"time"

	"github.com/omnitest/omnitest/pkg/model"
)

func generateHTML(result *model.TestResult, outDir string) (string, error) {
	tmpl, err := template.New("report").Parse(htmlTemplate)
	if err != nil {
		return "", fmt.Errorf("parse HTML template: %w", err)
	}

	data := prepareHTMLData(result)

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute HTML template: %w", err)
	}

	filename := fmt.Sprintf("report-%s.html", time.Now().Format("20060102-150405"))
	path := filepath.Join(outDir, filename)

	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		return "", fmt.Errorf("write HTML report: %w", err)
	}

	return path, nil
}

type htmlData struct {
	ScenarioName string
	StartTime    string
	Duration     string
	TotalReqs    int64
	SuccessRate  string
	AvgLatency   string
	P50Latency   string
	P95Latency   string
	P99Latency   string
	MaxLatency   string
	AvgRPS       string
	MaxRPS       string
	Thresholds   []model.ThresholdResult
	RPSPoints    string
	LatencyPoints string
}

func prepareHTMLData(result *model.TestResult) htmlData {
	successRate := 100.0
	if result.TotalRequests > 0 {
		successRate = float64(result.SuccessCount) / float64(result.TotalRequests) * 100
	}

	// SVG 차트 데이터포인트 생성
	var rpsPoints, latencyPoints string
	if len(result.Snapshots) > 0 {
		maxRPS := 1.0
		maxLatency := int64(1)
		for _, s := range result.Snapshots {
			if s.RPS > maxRPS {
				maxRPS = s.RPS
			}
			if s.P99Latency.Milliseconds() > maxLatency {
				maxLatency = s.P99Latency.Milliseconds()
			}
		}

		width := 600.0
		height := 200.0
		for i, s := range result.Snapshots {
			x := float64(i) / float64(len(result.Snapshots)) * width
			yRPS := height - (s.RPS/maxRPS)*height
			yLat := height - (float64(s.P99Latency.Milliseconds())/float64(maxLatency))*height

			if i > 0 {
				rpsPoints += " "
				latencyPoints += " "
			}
			rpsPoints += fmt.Sprintf("%.1f,%.1f", x, yRPS)
			latencyPoints += fmt.Sprintf("%.1f,%.1f", x, yLat)
		}
	}

	return htmlData{
		ScenarioName:  result.ScenarioName,
		StartTime:     result.StartTime.Format("2006-01-02 15:04:05"),
		Duration:      result.Duration.Truncate(time.Second).String(),
		TotalReqs:     result.TotalRequests,
		SuccessRate:   fmt.Sprintf("%.2f%%", successRate),
		AvgLatency:    fmt.Sprintf("%dms", result.AvgLatency.Milliseconds()),
		P50Latency:    fmt.Sprintf("%dms", result.P50Latency.Milliseconds()),
		P95Latency:    fmt.Sprintf("%dms", result.P95Latency.Milliseconds()),
		P99Latency:    fmt.Sprintf("%dms", result.P99Latency.Milliseconds()),
		MaxLatency:    fmt.Sprintf("%dms", result.MaxLatency.Milliseconds()),
		AvgRPS:        fmt.Sprintf("%.0f", result.AvgRPS),
		MaxRPS:        fmt.Sprintf("%.0f", result.MaxRPS),
		Thresholds:    result.ThresholdResults,
		RPSPoints:     rpsPoints,
		LatencyPoints: latencyPoints,
	}
}

const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>OmniTest Report - {{.ScenarioName}}</title>
<style>
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; background: #f5f5f5; color: #333; padding: 2rem; }
  .container { max-width: 900px; margin: 0 auto; }
  h1 { font-size: 1.5rem; margin-bottom: 0.5rem; }
  .meta { color: #666; margin-bottom: 2rem; font-size: 0.9rem; }
  .grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 1rem; margin-bottom: 2rem; }
  .card { background: white; border-radius: 8px; padding: 1.2rem; box-shadow: 0 1px 3px rgba(0,0,0,0.1); }
  .card .label { font-size: 0.8rem; color: #888; text-transform: uppercase; }
  .card .value { font-size: 1.5rem; font-weight: 600; margin-top: 0.3rem; }
  .section { background: white; border-radius: 8px; padding: 1.5rem; margin-bottom: 1.5rem; box-shadow: 0 1px 3px rgba(0,0,0,0.1); }
  .section h2 { font-size: 1.1rem; margin-bottom: 1rem; }
  table { width: 100%%; border-collapse: collapse; }
  th, td { padding: 0.6rem; text-align: left; border-bottom: 1px solid #eee; }
  th { font-size: 0.8rem; color: #888; text-transform: uppercase; }
  .pass { color: #22c55e; font-weight: 600; }
  .fail { color: #ef4444; font-weight: 600; }
  svg { width: 100%%; height: auto; }
  .chart-line { fill: none; stroke-width: 2; }
  .rps-line { stroke: #3b82f6; }
  .latency-line { stroke: #f59e0b; }
</style>
</head>
<body>
<div class="container">
  <h1>OmniTest Report</h1>
  <p class="meta">{{.ScenarioName}} | {{.StartTime}} | Duration: {{.Duration}}</p>

  <div class="grid">
    <div class="card"><div class="label">Total Requests</div><div class="value">{{.TotalReqs}}</div></div>
    <div class="card"><div class="label">Success Rate</div><div class="value">{{.SuccessRate}}</div></div>
    <div class="card"><div class="label">Avg RPS</div><div class="value">{{.AvgRPS}}</div></div>
    <div class="card"><div class="label">Max RPS</div><div class="value">{{.MaxRPS}}</div></div>
  </div>

  <div class="section">
    <h2>Latency Distribution</h2>
    <table>
      <tr><th>Percentile</th><th>Value</th></tr>
      <tr><td>Avg</td><td>{{.AvgLatency}}</td></tr>
      <tr><td>P50</td><td>{{.P50Latency}}</td></tr>
      <tr><td>P95</td><td>{{.P95Latency}}</td></tr>
      <tr><td>P99</td><td>{{.P99Latency}}</td></tr>
      <tr><td>Max</td><td>{{.MaxLatency}}</td></tr>
    </table>
  </div>

  {{if .RPSPoints}}
  <div class="section">
    <h2>RPS Over Time</h2>
    <svg viewBox="0 0 600 200" preserveAspectRatio="xMidYMid meet">
      <polyline class="chart-line rps-line" points="{{.RPSPoints}}" />
    </svg>
  </div>

  <div class="section">
    <h2>P99 Latency Over Time</h2>
    <svg viewBox="0 0 600 200" preserveAspectRatio="xMidYMid meet">
      <polyline class="chart-line latency-line" points="{{.LatencyPoints}}" />
    </svg>
  </div>
  {{end}}

  {{if .Thresholds}}
  <div class="section">
    <h2>Thresholds</h2>
    <table>
      <tr><th>Metric</th><th>Condition</th><th>Actual</th><th>Result</th></tr>
      {{range .Thresholds}}
      <tr>
        <td>{{.Metric}}</td>
        <td>{{.Condition}}</td>
        <td>{{.Actual}}</td>
        <td>{{if .Passed}}<span class="pass">PASS</span>{{else}}<span class="fail">FAIL</span>{{end}}</td>
      </tr>
      {{end}}
    </table>
  </div>
  {{end}}

  <p class="meta" style="text-align:center; margin-top:2rem;">Generated by OmniTest</p>
</div>
</body>
</html>`
