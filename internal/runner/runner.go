package runner

import (
	"context"
	"fmt"
	"time"

	"github.com/omnitest/omnitest/internal/metrics"
	"github.com/omnitest/omnitest/internal/output"
	"github.com/omnitest/omnitest/internal/worker"
	"github.com/omnitest/omnitest/pkg/model"
)

// Options는 runner의 추가 옵션이다.
type Options struct {
	Quiet      bool
	NoColor    bool
	OnSnapshot func(snap model.MetricSnapshot) // Agent 모드: 스냅샷을 gRPC로 전송
}

// Run은 테스트의 전체 생명주기를 오케스트레이션한다.
func Run(ctx context.Context, cfg *model.TestConfig, opts Options) (*model.TestResult, error) {
	// MVP: 첫 번째 시나리오만 실행
	if len(cfg.Scenarios) == 0 {
		return nil, fmt.Errorf("no scenarios defined")
	}
	scenario := cfg.Scenarios[0]

	// target 찾기
	var target *model.Target
	for i := range cfg.Targets {
		if cfg.Targets[i].Name == scenario.Target {
			target = &cfg.Targets[i]
			break
		}
	}
	if target == nil {
		return nil, fmt.Errorf("target %q not found", scenario.Target)
	}

	if !opts.Quiet {
		fmt.Printf("→ Loading scenario: %s\n", scenario.Name)
		fmt.Printf("→ Target: %s (%s)\n", target.Name, target.BaseURL)
		fmt.Printf("→ VUsers: %d, Duration: %s", scenario.VUsers, scenario.Duration)
		if scenario.RampUp > 0 {
			fmt.Printf(", Ramp-up: %s", scenario.RampUp)
		}
		fmt.Println()
		fmt.Println()
	}

	// collector 생성
	collector := metrics.NewCollector()

	// resultCh: worker → collector
	bufSize := scenario.VUsers * 100
	if bufSize < 1000 {
		bufSize = 1000
	}
	resultCh := make(chan model.RequestResult, bufSize)

	// worker pool 생성
	pool := worker.NewPool(worker.WorkerConfig{
		VUsers:   scenario.VUsers,
		RampUp:   scenario.RampUp,
		BaseURL:  target.BaseURL,
		Requests: scenario.Requests,
		Headers:  target.Headers,
	}, resultCh)

	// metrics 수집 goroutine
	collectorDone := make(chan struct{})
	go func() {
		defer close(collectorDone)
		for r := range resultCh {
			collector.Record(r)
		}
	}()

	// snapshots 축적 (quiet 모드에서도 수집 — maxRPS/차트 데이터용)
	var snapshots []model.MetricSnapshot
	snapshotFn := func() model.MetricSnapshot {
		snap := collector.Snapshot(pool.ActiveVUsers(), scenario.VUsers)
		snapshots = append(snapshots, snap)
		if opts.OnSnapshot != nil {
			opts.OnSnapshot(snap)
		}
		return snap
	}

	// output printer
	var printer *output.Printer
	if !opts.Quiet {
		printer = output.NewPrinter(snapshotFn, opts.NoColor, scenario.Duration)
	}

	// 테스트 시작
	start := time.Now()
	testCtx, cancel := context.WithTimeout(ctx, scenario.Duration)
	defer cancel()

	snapshotDone := make(chan struct{})
	if printer != nil {
		go func() {
			defer close(snapshotDone)
			printer.Start(testCtx)
		}()
	} else {
		// quiet 모드: 1초 간격 snapshot 수집 (maxRPS 추적용)
		go func() {
			defer close(snapshotDone)
			ticker := time.NewTicker(1 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-testCtx.Done():
					return
				case <-ticker.C:
					snapshotFn()
				}
			}
		}()
	}

	// worker pool 시작 (ramp-up 포함)
	pool.Start(testCtx)

	// duration 만료 또는 ctx 취소 대기
	<-testCtx.Done()

	// snapshot goroutine 종료 대기 (snapshots 슬라이스 race 방지)
	<-snapshotDone

	// worker 종료 대기
	pool.Wait()

	// resultCh 닫기 → collector goroutine 종료
	close(resultCh)
	<-collectorDone

	end := time.Now()

	// 최종 결과 집계
	result := collector.Result(scenario.Name, start, end, snapshots)

	return result, nil
}
