// demo-server는 OmniTest의 기능을 데모하기 위한 샘플 HTTP 서버입니다.
// 다양한 응답 패턴(지연, 에러, JSON 등)을 제공하여 부하 테스트 시나리오를 검증할 수 있습니다.
//
// 실행:
//
//	go run examples/demo-server/main.go
//	# 서버가 :8888 에서 시작됩니다
package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"
)

var requestCount atomic.Int64

func main() {
	mux := http.NewServeMux()

	// GET /health - 헬스체크 (항상 즉시 응답)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status": "healthy",
			"uptime": time.Since(startTime).String(),
		})
	})

	// GET /api/users - 사용자 목록 (정상 응답, 20-50ms 지연)
	mux.HandleFunc("GET /api/users", func(w http.ResponseWriter, r *http.Request) {
		delay := 20 + rand.Intn(30)
		time.Sleep(time.Duration(delay) * time.Millisecond)
		requestCount.Add(1)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"users": []map[string]any{
				{"id": 1, "name": "Alice", "email": "alice@example.com"},
				{"id": 2, "name": "Bob", "email": "bob@example.com"},
				{"id": 3, "name": "Charlie", "email": "charlie@example.com"},
			},
			"total":      3,
			"request_id": requestCount.Load(),
		})
	})

	// POST /api/users - 사용자 생성 (30-80ms 지연)
	mux.HandleFunc("POST /api/users", func(w http.ResponseWriter, r *http.Request) {
		delay := 30 + rand.Intn(50)
		time.Sleep(time.Duration(delay) * time.Millisecond)
		requestCount.Add(1)

		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"id":      rand.Intn(10000),
			"created": true,
			"data":    body,
		})
	})

	// GET /api/products - 상품 목록 (50-150ms 지연, DB 쿼리 시뮬레이션)
	mux.HandleFunc("GET /api/products", func(w http.ResponseWriter, r *http.Request) {
		delay := 50 + rand.Intn(100)
		time.Sleep(time.Duration(delay) * time.Millisecond)
		requestCount.Add(1)

		w.Header().Set("Content-Type", "application/json")
		products := make([]map[string]any, 20)
		for i := range products {
			products[i] = map[string]any{
				"id":    i + 1,
				"name":  fmt.Sprintf("Product %d", i+1),
				"price": 10.0 + float64(rand.Intn(990))/10.0,
				"stock": rand.Intn(100),
			}
		}
		json.NewEncoder(w).Encode(map[string]any{
			"products": products,
			"total":    20,
		})
	})

	// GET /api/search?q=xxx - 검색 (100-300ms 지연, 느린 응답 시뮬레이션)
	mux.HandleFunc("GET /api/search", func(w http.ResponseWriter, r *http.Request) {
		delay := 100 + rand.Intn(200)
		time.Sleep(time.Duration(delay) * time.Millisecond)
		requestCount.Add(1)

		q := r.URL.Query().Get("q")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"query":   q,
			"results": rand.Intn(50),
			"took_ms": delay,
		})
	})

	// POST /api/orders - 주문 생성 (50-100ms, 5% 확률로 에러)
	mux.HandleFunc("POST /api/orders", func(w http.ResponseWriter, r *http.Request) {
		delay := 50 + rand.Intn(50)
		time.Sleep(time.Duration(delay) * time.Millisecond)
		requestCount.Add(1)

		// 5% 확률로 500 에러
		if rand.Intn(100) < 5 {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]any{
				"error": "internal server error",
				"code":  "ORDER_FAILED",
			})
			return
		}

		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"order_id": fmt.Sprintf("ORD-%d", rand.Intn(100000)),
			"status":   "confirmed",
			"data":     body,
		})
	})

	// GET /api/slow - 의도적으로 느린 엔드포인트 (500-2000ms)
	mux.HandleFunc("GET /api/slow", func(w http.ResponseWriter, r *http.Request) {
		delay := 500 + rand.Intn(1500)
		time.Sleep(time.Duration(delay) * time.Millisecond)
		requestCount.Add(1)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"message":  "slow response",
			"delay_ms": delay,
		})
	})

	// GET /api/flaky - 불안정한 엔드포인트 (30% 에러율)
	mux.HandleFunc("GET /api/flaky", func(w http.ResponseWriter, r *http.Request) {
		delay := 20 + rand.Intn(30)
		time.Sleep(time.Duration(delay) * time.Millisecond)
		requestCount.Add(1)

		if rand.Intn(100) < 30 {
			codes := []int{500, 502, 503}
			w.WriteHeader(codes[rand.Intn(len(codes))])
			json.NewEncoder(w).Encode(map[string]any{
				"error": "service unavailable",
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"status": "ok"})
	})

	// GET /api/stats - 서버 통계 (요청 카운터)
	mux.HandleFunc("GET /api/stats", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"total_requests": requestCount.Load(),
			"uptime":         time.Since(startTime).String(),
		})
	})

	// GET /api/echo - 헤더/쿼리 에코 (인증 토큰 검증 데모)
	mux.HandleFunc("GET /api/echo", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond)
		requestCount.Add(1)

		auth := r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")

		if auth == "" {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]any{
				"error": "missing authorization header",
			})
			return
		}

		json.NewEncoder(w).Encode(map[string]any{
			"auth":    auth,
			"headers": r.Header,
			"query":   r.URL.Query(),
		})
	})

	// GET /api/variable-load - 부하에 따라 응답 시간 증가
	mux.HandleFunc("GET /api/variable-load", func(w http.ResponseWriter, r *http.Request) {
		current := requestCount.Add(1)
		// 동시 요청이 많을수록 느려지는 시뮬레이션
		baseDelay := 10
		loadFactor := int(current%100) / 10
		delay := baseDelay + loadFactor*20
		time.Sleep(time.Duration(delay) * time.Millisecond)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"delay_ms":    delay,
			"load_factor": loadFactor,
		})
	})

	// PUT /api/users/{id} - 사용자 수정
	mux.HandleFunc("PUT /api/users/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		delay := 30 + rand.Intn(40)
		time.Sleep(time.Duration(delay) * time.Millisecond)
		requestCount.Add(1)

		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":      id,
			"updated": true,
			"data":    body,
		})
	})

	// DELETE /api/users/{id} - 사용자 삭제
	mux.HandleFunc("DELETE /api/users/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		time.Sleep(20 * time.Millisecond)
		requestCount.Add(1)

		idNum, _ := strconv.Atoi(id)
		if idNum > 1000 {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]any{"error": "user not found"})
			return
		}

		w.WriteHeader(http.StatusNoContent)
	})

	fmt.Println("╔══════════════════════════════════════════════╗")
	fmt.Println("║  OmniTest Demo Server                       ║")
	fmt.Println("║  http://localhost:8888                       ║")
	fmt.Println("╠══════════════════════════════════════════════╣")
	fmt.Println("║  Endpoints:                                 ║")
	fmt.Println("║    GET  /health            (즉시)            ║")
	fmt.Println("║    GET  /api/users         (20-50ms)        ║")
	fmt.Println("║    POST /api/users         (30-80ms)        ║")
	fmt.Println("║    GET  /api/products      (50-150ms)       ║")
	fmt.Println("║    GET  /api/search?q=     (100-300ms)      ║")
	fmt.Println("║    POST /api/orders        (50-100ms, err5%) ║")
	fmt.Println("║    GET  /api/slow          (500-2000ms)     ║")
	fmt.Println("║    GET  /api/flaky         (30% error rate) ║")
	fmt.Println("║    GET  /api/echo          (인증 필요)       ║")
	fmt.Println("║    GET  /api/variable-load (부하 비례 지연)   ║")
	fmt.Println("║    PUT  /api/users/{id}    (30-70ms)        ║")
	fmt.Println("║    DELETE /api/users/{id}  (20ms)           ║")
	fmt.Println("║    GET  /api/stats         (서버 통계)       ║")
	fmt.Println("╚══════════════════════════════════════════════╝")

	http.ListenAndServe(":8888", mux)
}

var startTime = time.Now()
