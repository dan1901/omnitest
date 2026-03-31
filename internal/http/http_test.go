package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/omnitest/omnitest/pkg/model"
)

func TestClient_Do_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	client := NewClient()
	result := client.Do(context.Background(), server.URL, model.Request{
		Method: "GET",
		Path:   "/test",
	}, nil)

	if result.Error != nil {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	if result.StatusCode != 200 {
		t.Errorf("status = %d, want 200", result.StatusCode)
	}
	if result.Latency <= 0 {
		t.Error("latency should be > 0")
	}
	if result.BytesIn <= 0 {
		t.Error("bytes should be > 0")
	}
}

func TestClient_Do_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient()
	result := client.Do(context.Background(), server.URL, model.Request{
		Method: "GET",
		Path:   "/error",
	}, nil)

	if result.StatusCode != 500 {
		t.Errorf("status = %d, want 500", result.StatusCode)
	}
	if result.Error == nil {
		t.Error("expected error for 5xx response")
	}
}

func TestClient_Do_WithHeaders(t *testing.T) {
	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient()
	client.Do(context.Background(), server.URL, model.Request{
		Method: "GET",
		Path:   "/",
	}, map[string]string{"Authorization": "Bearer test"})

	if gotAuth != "Bearer test" {
		t.Errorf("Authorization header = %q, want %q", gotAuth, "Bearer test")
	}
}

func TestClient_Do_PostWithBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method = %s, want POST", r.Method)
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	client := NewClient()
	result := client.Do(context.Background(), server.URL, model.Request{
		Method: "POST",
		Path:   "/create",
		Body:   map[string]any{"name": "test"},
	}, nil)

	if result.StatusCode != 201 {
		t.Errorf("status = %d, want 201", result.StatusCode)
	}
}

func TestClient_Do_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
	}))
	defer server.Close()

	client := NewClient()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	result := client.Do(ctx, server.URL, model.Request{
		Method: "GET",
		Path:   "/slow",
	}, nil)

	if result.Error == nil {
		t.Error("expected timeout error")
	}
}
