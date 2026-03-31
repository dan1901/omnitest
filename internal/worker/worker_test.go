package worker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/omnitest/omnitest/pkg/model"
)

func TestPool_BasicExecution(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	resultCh := make(chan model.RequestResult, 1000)
	pool := NewPool(WorkerConfig{
		VUsers:  5,
		BaseURL: server.URL,
		Requests: []model.Request{
			{Method: "GET", Path: "/"},
		},
	}, resultCh)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go pool.Start(ctx)

	<-ctx.Done()
	pool.Wait()
	close(resultCh)

	count := 0
	for range resultCh {
		count++
	}

	if count == 0 {
		t.Error("expected at least some requests")
	}
}

func TestPool_ActiveVUsers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	resultCh := make(chan model.RequestResult, 1000)
	pool := NewPool(WorkerConfig{
		VUsers:  3,
		BaseURL: server.URL,
		Requests: []model.Request{
			{Method: "GET", Path: "/"},
		},
	}, resultCh)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	go pool.Start(ctx)

	// 워커가 시작될 시간
	time.Sleep(200 * time.Millisecond)

	active := pool.ActiveVUsers()
	if active != 3 {
		t.Errorf("ActiveVUsers = %d, want 3", active)
	}

	cancel()
	pool.Wait()
}

func TestPool_GracefulShutdown(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	resultCh := make(chan model.RequestResult, 1000)
	pool := NewPool(WorkerConfig{
		VUsers:  10,
		BaseURL: server.URL,
		Requests: []model.Request{
			{Method: "GET", Path: "/"},
		},
	}, resultCh)

	ctx, cancel := context.WithCancel(context.Background())

	go pool.Start(ctx)

	time.Sleep(200 * time.Millisecond)
	cancel()

	done := make(chan struct{})
	go func() {
		pool.Wait()
		close(done)
	}()

	select {
	case <-done:
		// OK
	case <-time.After(5 * time.Second):
		t.Fatal("pool.Wait() did not return within timeout")
	}

	if pool.ActiveVUsers() != 0 {
		t.Errorf("ActiveVUsers = %d after shutdown, want 0", pool.ActiveVUsers())
	}
}

func TestPool_RampUp(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	resultCh := make(chan model.RequestResult, 1000)
	pool := NewPool(WorkerConfig{
		VUsers:  4,
		RampUp:  2 * time.Second,
		BaseURL: server.URL,
		Requests: []model.Request{
			{Method: "GET", Path: "/"},
		},
	}, resultCh)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	go pool.Start(ctx)

	// 500ms 후에는 1개만 시작되어야 함 (interval = 2s/4 = 500ms)
	time.Sleep(300 * time.Millisecond)
	active := pool.ActiveVUsers()
	if active > 2 {
		t.Errorf("after 300ms, ActiveVUsers = %d, want <= 2", active)
	}

	cancel()
	pool.Wait()
}
