package threshold

import (
	"fmt"
	"strings"

	"github.com/omnitest/omnitest/pkg/model"
)

// Evaluate는 threshold 목록을 테스트 결과에 대해 평가하여 각 결과를 반환한다.
func Evaluate(thresholds []model.Threshold, result *model.TestResult) []model.ThresholdResult {
	var results []model.ThresholdResult

	for _, t := range thresholds {
		tr := model.ThresholdResult{
			Metric:    t.Metric,
			Condition: t.Condition,
		}

		var actualValue float64
		var actualStr string

		switch t.Metric {
		case "http_req_duration_p50":
			actualValue = float64(result.P50Latency.Milliseconds())
			actualStr = fmt.Sprintf("%dms", result.P50Latency.Milliseconds())
		case "http_req_duration_p95":
			actualValue = float64(result.P95Latency.Milliseconds())
			actualStr = fmt.Sprintf("%dms", result.P95Latency.Milliseconds())
		case "http_req_duration_p99":
			actualValue = float64(result.P99Latency.Milliseconds())
			actualStr = fmt.Sprintf("%dms", result.P99Latency.Milliseconds())
		case "http_req_duration_avg":
			actualValue = float64(result.AvgLatency.Milliseconds())
			actualStr = fmt.Sprintf("%dms", result.AvgLatency.Milliseconds())
		case "http_req_failed":
			actualValue = result.ErrorRate * 100
			actualStr = fmt.Sprintf("%.2f%%", result.ErrorRate*100)
		case "http_reqs":
			actualValue = float64(result.TotalRequests)
			actualStr = fmt.Sprintf("%d", result.TotalRequests)
		default:
			tr.Actual = "unknown metric"
			tr.Passed = false
			results = append(results, tr)
			continue
		}

		tr.Actual = actualStr
		tr.Passed = EvaluateCondition(actualValue, t.Condition)
		results = append(results, tr)
	}

	return results
}

// EvaluateCondition은 실제 값을 조건 문자열에 대해 평가한다.
// 조건 형식: "< 200ms", "> 1000", "< 1%"
func EvaluateCondition(actual float64, condition string) bool {
	condition = strings.TrimSpace(condition)

	var op string

	if strings.HasPrefix(condition, "<=") {
		op = "<="
		condition = strings.TrimPrefix(condition, "<=")
	} else if strings.HasPrefix(condition, ">=") {
		op = ">="
		condition = strings.TrimPrefix(condition, ">=")
	} else if strings.HasPrefix(condition, "<") {
		op = "<"
		condition = strings.TrimPrefix(condition, "<")
	} else if strings.HasPrefix(condition, ">") {
		op = ">"
		condition = strings.TrimPrefix(condition, ">")
	} else {
		return false
	}

	condition = strings.TrimSpace(condition)

	var threshold float64
	if strings.HasSuffix(condition, "ms") {
		fmt.Sscanf(strings.TrimSuffix(condition, "ms"), "%f", &threshold)
	} else if strings.HasSuffix(condition, "s") {
		fmt.Sscanf(strings.TrimSuffix(condition, "s"), "%f", &threshold)
		threshold *= 1000
	} else if strings.HasSuffix(condition, "%") {
		fmt.Sscanf(strings.TrimSuffix(condition, "%"), "%f", &threshold)
	} else {
		fmt.Sscanf(condition, "%f", &threshold)
	}

	switch op {
	case "<":
		return actual < threshold
	case "<=":
		return actual <= threshold
	case ">":
		return actual > threshold
	case ">=":
		return actual >= threshold
	}
	return false
}
