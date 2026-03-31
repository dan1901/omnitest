package controller

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/omnitest/omnitest/internal/grpc/omnitestv1"
	"github.com/omnitest/omnitest/internal/store"
	"github.com/omnitest/omnitest/internal/ws"
	"github.com/omnitest/omnitest/pkg/model"
)

// Aggregator는 Agent별 메트릭을 실시간 병합한다.
type Aggregator struct {
	metrics map[string]map[string]*latestMetric // testRunID → agentID → latest
	mu      sync.RWMutex
	wsHub   *ws.Hub
	store   *store.Store

	// 테스트 완료 콜백
	OnTestComplete func(testRunID string, aggregated *model.AggregatedMetrics)
}

type latestMetric struct {
	Report    *omnitestv1.MetricReport
	UpdatedAt time.Time
}

// NewAggregator creates a new Aggregator.
func NewAggregator(hub *ws.Hub, st *store.Store) *Aggregator {
	return &Aggregator{
		metrics: make(map[string]map[string]*latestMetric),
		wsHub:   hub,
		store:   st,
	}
}

// OnMetricReport는 Agent에서 MetricReport를 수신했을 때 호출된다.
func (a *Aggregator) OnMetricReport(report *omnitestv1.MetricReport) error {
	a.mu.Lock()
	testRunID := report.GetTestRunId()
	agentID := report.GetAgentId()

	if a.metrics[testRunID] == nil {
		a.metrics[testRunID] = make(map[string]*latestMetric)
	}
	a.metrics[testRunID][agentID] = &latestMetric{
		Report:    report,
		UpdatedAt: time.Now(),
	}
	a.mu.Unlock()

	// Aggregate and broadcast
	aggregated := a.Aggregate(testRunID)
	a.wsHub.BroadcastMetrics(aggregated)

	// Store metric to DB
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ts := time.Now()
	if report.GetTimestamp() != nil {
		ts = report.GetTimestamp().AsTime()
	}

	if err := a.store.InsertMetric(ctx, testRunID, agentID, ts,
		report.GetTotalRequests(), report.GetTotalErrors(),
		report.GetRps(), report.GetAvgLatencyMs(),
		report.GetP50LatencyMs(), report.GetP95LatencyMs(), report.GetP99LatencyMs(),
		int(report.GetActiveVusers())); err != nil {
		log.Printf("[Aggregator] DB insert error: %v", err)
	}

	return nil
}

// Aggregate는 특정 testRunID의 전체 Agent 메트릭을 병합한다.
func (a *Aggregator) Aggregate(testRunID string) *model.AggregatedMetrics {
	a.mu.RLock()
	defer a.mu.RUnlock()

	agentMetrics, ok := a.metrics[testRunID]
	if !ok {
		return nil
	}

	agg := &model.AggregatedMetrics{
		TestRunID: testRunID,
		Timestamp: time.Now(),
		PerAgent:  make(map[string]*model.MetricSnapshot),
	}

	var totalWeightedAvg, totalWeightedP50, totalWeightedP95, totalWeightedP99 float64
	var totalReqsForWeight int64

	for agentID, lm := range agentMetrics {
		r := lm.Report
		agg.TotalRPS += r.GetRps()
		agg.TotalReqs += r.GetTotalRequests()
		agg.TotalErrors += r.GetTotalErrors()
		agg.ActiveVUsers += int(r.GetActiveVusers())

		// 가중 평균 계산용
		reqs := r.GetTotalRequests()
		if reqs > 0 {
			totalWeightedAvg += r.GetAvgLatencyMs() * float64(reqs)
			totalWeightedP50 += r.GetP50LatencyMs() * float64(reqs)
			totalWeightedP95 += r.GetP95LatencyMs() * float64(reqs)
			totalWeightedP99 += r.GetP99LatencyMs() * float64(reqs)
			totalReqsForWeight += reqs
		}

		// Per-agent snapshot
		snap := &model.MetricSnapshot{
			Timestamp:    time.Now(),
			RPS:          r.GetRps(),
			ActiveVUsers: int(r.GetActiveVusers()),
			TotalReqs:    r.GetTotalRequests(),
			TotalErrors:  r.GetTotalErrors(),
		}
		agg.PerAgent[agentID] = snap
	}

	if totalReqsForWeight > 0 {
		agg.AvgLatencyMs = totalWeightedAvg / float64(totalReqsForWeight)
		agg.P50LatencyMs = totalWeightedP50 / float64(totalReqsForWeight)
		agg.P95LatencyMs = totalWeightedP95 / float64(totalReqsForWeight)
		agg.P99LatencyMs = totalWeightedP99 / float64(totalReqsForWeight)
	}

	return agg
}

// Cleanup은 완료된 testRun의 메트릭 캐시를 정리한다.
func (a *Aggregator) Cleanup(testRunID string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.metrics, testRunID)
}
