package worker

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	gohttp "github.com/omnitest/omnitest/internal/http"
	"github.com/omnitest/omnitest/pkg/model"
)

// WorkerConfig는 워커 풀 설정이다.
type WorkerConfig struct {
	VUsers   int
	RampUp   time.Duration
	BaseURL  string
	Requests []model.Request
	Headers  map[string]string
}

// Pool은 goroutine 기반 VUser 워커 풀이다.
type Pool struct {
	cfg       WorkerConfig
	resultCh  chan<- model.RequestResult
	client    *gohttp.Client
	wg        sync.WaitGroup
	activeVUs atomic.Int32
}

// NewPool은 새 워커 풀을 생성한다.
// 단일 HTTP Client를 생성하여 모든 VUser가 공유한다 (http.Client/Transport는 goroutine-safe).
func NewPool(cfg WorkerConfig, resultCh chan<- model.RequestResult) *Pool {
	return &Pool{
		cfg:      cfg,
		resultCh: resultCh,
		client:   gohttp.NewClient(),
	}
}

// ActiveVUsers는 현재 활성 VUser 수를 반환한다.
func (p *Pool) ActiveVUsers() int {
	return int(p.activeVUs.Load())
}

// Start는 워커 풀을 시작한다. Ramp-up에 따라 점진적으로 VUser를 증가시킨다.
func (p *Pool) Start(ctx context.Context) {
	if p.cfg.RampUp <= 0 || p.cfg.VUsers <= 1 {
		// Ramp-up 없음: 모든 VUser 즉시 시작
		for i := 0; i < p.cfg.VUsers; i++ {
			p.wg.Add(1)
			go p.runVUser(ctx)
		}
		return
	}

	interval := p.cfg.RampUp / time.Duration(p.cfg.VUsers)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for i := 0; i < p.cfg.VUsers; i++ {
		p.wg.Add(1)
		go p.runVUser(ctx)
		if i < p.cfg.VUsers-1 {
			select {
			case <-ticker.C:
			case <-ctx.Done():
				return
			}
		}
	}
}

// Wait는 모든 워커가 종료될 때까지 대기한다.
func (p *Pool) Wait() {
	p.wg.Wait()
}

func (p *Pool) runVUser(ctx context.Context) {
	defer p.wg.Done()

	p.activeVUs.Add(1)
	defer p.activeVUs.Add(-1)

	for {
		for _, req := range p.cfg.Requests {
			select {
			case <-ctx.Done():
				return
			default:
			}

			result := p.client.Do(ctx, p.cfg.BaseURL, req, p.cfg.Headers)

			select {
			case p.resultCh <- result:
			case <-ctx.Done():
				return
			}
		}
	}
}
