package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// api.Server는 controller.Controller에 강하게 결합되어 있고,
// Controller는 DB(store)가 필요하므로 통합 테스트 없이는 완전한 테스트가 어렵다.
// 여기서는 미들웨어와 helper 함수들을 단위 테스트한다.

func TestCorsMiddleware(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := cors(inner)

	// Regular request
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("CORS header missing: Access-Control-Allow-Origin")
	}
	if w.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("CORS header missing: Access-Control-Allow-Methods")
	}

	// OPTIONS preflight
	req = httptest.NewRequest(http.MethodOptions, "/api/v1/health", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("OPTIONS response code = %d, want %d", w.Code, http.StatusNoContent)
	}
}

func TestRequestIDMiddleware(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := requestID(inner)

	// Without X-Request-ID header
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	id := w.Header().Get("X-Request-ID")
	if id == "" {
		t.Error("X-Request-ID should be auto-generated when not provided")
	}

	// With X-Request-ID header
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Request-ID", "custom-id-123")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	id = w.Header().Get("X-Request-ID")
	if id != "custom-id-123" {
		t.Errorf("X-Request-ID = %q, want %q", id, "custom-id-123")
	}
}

func TestLoggerMiddleware(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := logger(inner)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("logger middleware changed status code: got %d, want %d", w.Code, http.StatusOK)
	}
}

func TestGetPagination(t *testing.T) {
	tests := []struct {
		query   string
		page    int
		perPage int
	}{
		{"", 1, 20},                         // defaults
		{"?page=3&per_page=50", 3, 50},      // explicit
		{"?page=0&per_page=0", 1, 20},       // zero → defaults
		{"?page=-1&per_page=200", 1, 20},    // out of range
		{"?page=abc&per_page=xyz", 1, 20},   // invalid → defaults
	}

	for _, tt := range tests {
		req := httptest.NewRequest(http.MethodGet, "/test"+tt.query, nil)
		page, perPage := getPagination(req)
		if page != tt.page || perPage != tt.perPage {
			t.Errorf("getPagination(%q) = (%d, %d), want (%d, %d)",
				tt.query, page, perPage, tt.page, tt.perPage)
		}
	}
}

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	writeJSON(w, http.StatusOK, map[string]string{"key": "value"}, "req-123")

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json")
	}
	body := w.Body.String()
	if body == "" {
		t.Error("response body should not be empty")
	}
}

func TestWriteError(t *testing.T) {
	w := httptest.NewRecorder()
	writeError(w, http.StatusNotFound, "NOT_FOUND", "Resource not found", "details", "req-456")

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
	body := w.Body.String()
	if body == "" {
		t.Error("error response body should not be empty")
	}
}
