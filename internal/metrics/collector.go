package metrics

import (
	"sync"
	"sync/atomic"
	"time"

	hdrhistogram "github.com/HdrHistogram/hdrhistogram-go"

	"github.com/omnitest/omnitest/pkg/model"
)

// Collector는 HDR Histogram 기반 실시간 메트릭 수집기다.
type Collector struct {
	mu        sync.Mutex
	histogram *hdrhistogram.Histogram
	startTime time.Time

	totalReqs   atomic.Int64
	totalErrors atomic.Int64
	totalBytes  atomic.Int64

	// 1초 윈도우용
	windowReqs atomic.Int64
	lastRPS    float64
	maxRPS     float64
}

// NewCollector는 새 메트릭 수집기를 생성한다.
func NewCollector() *Collector {
	return &Collector{
		// 1μs ~ 1시간 범위, 유효 숫자 3자리
		histogram: hdrhistogram.New(1, 3600000000, 3),
		startTime: time.Now(),
	}
}

// Record는 개별 요청 결과를 기록한다.
// channel을 통해 단일 goroutine에서 호출되므로 histogram은 직렬화됨.
// Snapshot과의 동기화를 위해 mu 사용.
func (c *Collector) Record(r model.RequestResult) {
	c.totalReqs.Add(1)
	c.windowReqs.Add(1)

	if r.Error != nil {
		c.totalErrors.Add(1)
	}

	c.totalBytes.Add(r.BytesIn)

	latencyUs := r.Latency.Microseconds()
	if latencyUs < 1 {
		latencyUs = 1
	}

	c.mu.Lock()
	_ = c.histogram.RecordValue(latencyUs)
	c.mu.Unlock()
}

// Snapshot은 현재 시점의 메트릭 스냅샷을 반환한다.
func (c *Collector) Snapshot(activeVUsers, totalVUsers int) model.MetricSnapshot {
	c.mu.Lock()
	defer c.mu.Unlock()

	elapsed := time.Since(c.startTime).Seconds()
	total := c.totalReqs.Load()
	errors := c.totalErrors.Load()

	windowReqs := c.windowReqs.Swap(0)
	rps := float64(windowReqs)
	if rps == 0 && elapsed > 0 {
		rps = float64(total) / elapsed
	}
	c.lastRPS = rps
	if rps > c.maxRPS {
		c.maxRPS = rps
	}

	var errorRate float64
	if total > 0 {
		errorRate = float64(errors) / float64(total)
	}

	return model.MetricSnapshot{
		Timestamp:    time.Now(),
		ElapsedSec:   elapsed,
		ActiveVUsers: activeVUsers,
		TotalVUsers:  totalVUsers,
		RPS:          rps,
		AvgLatency:   microToDuration(int64(c.histogram.Mean())),
		P50Latency:   microToDuration(c.histogram.ValueAtPercentile(50)),
		P95Latency:   microToDuration(c.histogram.ValueAtPercentile(95)),
		P99Latency:   microToDuration(c.histogram.ValueAtPercentile(99)),
		ErrorRate:    errorRate,
		TotalReqs:    total,
		TotalErrors:  errors,
		BytesIn:      c.totalBytes.Load(),
	}
}

// Result는 최종 집계 결과를 반환한다.
func (c *Collector) Result(scenarioName string, start, end time.Time, snapshots []model.MetricSnapshot) *model.TestResult {
	c.mu.Lock()
	defer c.mu.Unlock()

	total := c.totalReqs.Load()
	errors := c.totalErrors.Load()
	duration := end.Sub(start)

	var errorRate float64
	if total > 0 {
		errorRate = float64(errors) / float64(total)
	}

	var avgRPS float64
	if duration.Seconds() > 0 {
		avgRPS = float64(total) / duration.Seconds()
	}
	maxRPS := c.maxRPS

	return &model.TestResult{
		ScenarioName:  scenarioName,
		StartTime:     start,
		EndTime:       end,
		Duration:      duration,
		TotalRequests: total,
		SuccessCount:  total - errors,
		ErrorCount:    errors,
		ErrorRate:     errorRate,
		AvgLatency:    microToDuration(int64(c.histogram.Mean())),
		P50Latency:    microToDuration(c.histogram.ValueAtPercentile(50)),
		P95Latency:    microToDuration(c.histogram.ValueAtPercentile(95)),
		P99Latency:    microToDuration(c.histogram.ValueAtPercentile(99)),
		MaxLatency:    microToDuration(c.histogram.ValueAtPercentile(100)),
		MinLatency:    microToDuration(c.histogram.ValueAtPercentile(0)),
		AvgRPS:        avgRPS,
		MaxRPS:        maxRPS,
		Snapshots:     snapshots,
	}
}

func microToDuration(us int64) time.Duration {
	return time.Duration(us) * time.Microsecond
}
