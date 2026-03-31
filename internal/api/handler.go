// Package api implements REST API handlers for the OmniTest Controller.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/omnitest/omnitest/internal/config"
	"github.com/omnitest/omnitest/internal/controller"
	"github.com/omnitest/omnitest/internal/store"
	"github.com/omnitest/omnitest/internal/ws"
	"github.com/omnitest/omnitest/pkg/model"
)

type contextKey string

const requestIDKey contextKey = "request_id"

// Server는 REST API HTTP 서버다.
type Server struct {
	mux       *http.ServeMux
	store     *store.Store
	ctrl      *controller.Controller
	wsHub     *ws.Hub
	startTime time.Time
}

// NewServer creates a new REST API server.
func NewServer(ctrl *controller.Controller) *Server {
	s := &Server{
		mux:       http.NewServeMux(),
		store:     ctrl.Store(),
		ctrl:      ctrl,
		wsHub:     ctrl.WSHub(),
		startTime: time.Now(),
	}
	s.registerRoutes()
	return s
}

func (s *Server) registerRoutes() {
	// System
	s.mux.HandleFunc("GET /api/v1/health", s.health)
	s.mux.HandleFunc("GET /api/v1/version", s.version)

	// Agents
	s.mux.HandleFunc("GET /api/v1/agents", s.listAgents)
	s.mux.HandleFunc("GET /api/v1/agents/{id}", s.getAgent)
	s.mux.HandleFunc("DELETE /api/v1/agents/{id}", s.deleteAgent)

	// Tests
	s.mux.HandleFunc("GET /api/v1/tests", s.listTests)
	s.mux.HandleFunc("POST /api/v1/tests", s.createTest)
	s.mux.HandleFunc("GET /api/v1/tests/{id}", s.getTest)
	s.mux.HandleFunc("PUT /api/v1/tests/{id}", s.updateTest)
	s.mux.HandleFunc("DELETE /api/v1/tests/{id}", s.deleteTest)

	// Test runs
	s.mux.HandleFunc("POST /api/v1/tests/{id}/run", s.runTest)
	s.mux.HandleFunc("POST /api/v1/tests/{id}/stop", s.stopTest)
	s.mux.HandleFunc("GET /api/v1/tests/{id}/runs", s.listTestRuns)

	// Agent internal (command polling)
	s.mux.HandleFunc("GET /api/v1/internal/agents/{id}/command", s.pollAgentCommand)

	// Results
	s.mux.HandleFunc("GET /api/v1/runs/{id}", s.getTestRun)
	s.mux.HandleFunc("GET /api/v1/runs/{id}/metrics", s.getRunMetrics)
	s.mux.HandleFunc("POST /api/v1/runs/{id}/stop", s.stopTestByRunID)

	// WebSocket
	s.mux.HandleFunc("/ws/metrics/{runId}", s.wsHub.HandleWebSocket)
	s.mux.HandleFunc("/ws/agents", s.wsHub.HandleWebSocket)
	s.mux.HandleFunc("/ws/events", s.wsHub.HandleWebSocket)
}

// Handler returns the HTTP handler with middleware applied.
func (s *Server) Handler() http.Handler {
	return cors(requestID(logger(s.mux)))
}

// --- Middleware ---

func requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = fmt.Sprintf("req-%d", time.Now().UnixNano()%1000000000)
		}
		w.Header().Set("X-Request-ID", id)
		ctx := context.WithValue(r.Context(), requestIDKey, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("[API] %s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Request-ID")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// --- Response helpers ---

type envelope struct {
	Data  any    `json:"data,omitempty"`
	Error *apiError `json:"error,omitempty"`
	Meta  meta   `json:"meta"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

type meta struct {
	RequestID string `json:"request_id"`
	Total     int    `json:"total,omitempty"`
	Page      int    `json:"page,omitempty"`
	PerPage   int    `json:"per_page,omitempty"`
}

func getRequestID(w http.ResponseWriter) string {
	return w.Header().Get("X-Request-ID")
}

func writeJSON(w http.ResponseWriter, status int, data any, reqID string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(envelope{
		Data: data,
		Meta: meta{RequestID: reqID},
	})
}

func writeJSONList(w http.ResponseWriter, data any, total, page, perPage int, reqID string) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(envelope{
		Data: data,
		Meta: meta{RequestID: reqID, Total: total, Page: page, PerPage: perPage},
	})
}

func writeError(w http.ResponseWriter, status int, code, message, details string, reqID string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(envelope{
		Error: &apiError{Code: code, Message: message, Details: details},
		Meta:  meta{RequestID: reqID},
	})
}

func getPagination(r *http.Request) (int, int) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	return page, perPage
}

// --- System handlers ---

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	reqID := getRequestID(w)
	agents := s.ctrl.AgentManager.OnlineAgents()
	uptime := time.Since(s.ctrl.StartTime()).Seconds()
	writeJSON(w, http.StatusOK, map[string]any{
		"status":        "ok",
		"db":            "connected",
		"agents_online": len(agents),
		"uptime_seconds": int(uptime),
	}, reqID)
}

func (s *Server) version(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"version": "0.2.0",
	}, getRequestID(w))
}

// --- Agent handlers ---

func (s *Server) listAgents(w http.ResponseWriter, r *http.Request) {
	agents := s.ctrl.AgentManager.AllAgents()
	if agents == nil {
		agents = []model.AgentInfo{}
	}
	writeJSON(w, http.StatusOK, agents, getRequestID(w))
}

func (s *Server) getAgent(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	agent := s.ctrl.AgentManager.Get(id)
	if agent == nil {
		writeError(w, http.StatusNotFound, "AGENT_NOT_FOUND",
			fmt.Sprintf("Agent with ID '%s' not found", id), "", getRequestID(w))
		return
	}
	writeJSON(w, http.StatusOK, agent, getRequestID(w))
}

func (s *Server) deleteAgent(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	agent := s.ctrl.AgentManager.Get(id)
	if agent == nil {
		writeError(w, http.StatusNotFound, "AGENT_NOT_FOUND",
			fmt.Sprintf("Agent with ID '%s' not found", id), "", getRequestID(w))
		return
	}
	s.ctrl.AgentManager.Remove(id)
	writeJSON(w, http.StatusOK, map[string]bool{"disconnected": true}, getRequestID(w))
}

// --- Test handlers ---

func (s *Server) listTests(w http.ResponseWriter, r *http.Request) {
	page, perPage := getPagination(r)
	tests, total, err := s.store.ListTests(r.Context(), page, perPage)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list tests", err.Error(), getRequestID(w))
		return
	}
	if tests == nil {
		tests = []model.TestDefinition{}
	}
	writeJSONList(w, tests, total, page, perPage, getRequestID(w))
}

func (s *Server) createTest(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name         string `json:"name"`
		ScenarioYAML string `json:"scenario_yaml"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_PARAMETER", "Invalid request body", err.Error(), getRequestID(w))
		return
	}

	// Validate YAML
	if _, err := config.LoadFromString(body.ScenarioYAML); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_SCENARIO", "YAML scenario validation failed", err.Error(), getRequestID(w))
		return
	}

	test := &model.TestDefinition{
		Name:         body.Name,
		ScenarioYAML: body.ScenarioYAML,
	}
	if err := s.store.CreateTest(r.Context(), test); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create test", err.Error(), getRequestID(w))
		return
	}
	writeJSON(w, http.StatusCreated, test, getRequestID(w))
}

func (s *Server) getTest(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	test, err := s.store.GetTest(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get test", err.Error(), getRequestID(w))
		return
	}
	if test == nil {
		writeError(w, http.StatusNotFound, "TEST_NOT_FOUND",
			fmt.Sprintf("Test with ID '%s' not found", id), "", getRequestID(w))
		return
	}
	writeJSON(w, http.StatusOK, test, getRequestID(w))
}

func (s *Server) updateTest(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body struct {
		Name         string `json:"name"`
		ScenarioYAML string `json:"scenario_yaml"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_PARAMETER", "Invalid request body", err.Error(), getRequestID(w))
		return
	}

	if body.ScenarioYAML != "" {
		if _, err := config.LoadFromString(body.ScenarioYAML); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_SCENARIO", "YAML scenario validation failed", err.Error(), getRequestID(w))
			return
		}
	}

	test, err := s.store.UpdateTest(r.Context(), id, body.Name, body.ScenarioYAML)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update test", err.Error(), getRequestID(w))
		return
	}
	if test == nil {
		writeError(w, http.StatusNotFound, "TEST_NOT_FOUND",
			fmt.Sprintf("Test with ID '%s' not found", id), "", getRequestID(w))
		return
	}
	writeJSON(w, http.StatusOK, test, getRequestID(w))
}

func (s *Server) deleteTest(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.store.DeleteTest(r.Context(), id); err != nil {
		if strings.Contains(err.Error(), "no rows") {
			writeError(w, http.StatusNotFound, "TEST_NOT_FOUND",
				fmt.Sprintf("Test with ID '%s' not found", id), "", getRequestID(w))
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete test", err.Error(), getRequestID(w))
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"deleted": true}, getRequestID(w))
}

// --- Test Run handlers ---

func (s *Server) runTest(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	reqID := getRequestID(w)

	// 테스트 조회
	test, err := s.store.GetTest(r.Context(), id)
	if err != nil || test == nil {
		writeError(w, http.StatusNotFound, "TEST_NOT_FOUND",
			fmt.Sprintf("Test with ID '%s' not found", id), "", reqID)
		return
	}

	// 이미 실행 중인지 확인
	existing, _ := s.store.GetRunningTestRun(r.Context(), id)
	if existing != nil {
		writeError(w, http.StatusConflict, "TEST_ALREADY_RUNNING",
			"Test is already running", "", reqID)
		return
	}

	// 온라인 에이전트 확인
	agents := s.ctrl.AgentManager.OnlineAgents()
	if len(agents) == 0 {
		writeError(w, http.StatusConflict, "NO_AGENTS_AVAILABLE",
			"Cannot start test: no agents are connected",
			"Start at least one agent with 'omnitest agent --controller=host:9090'", reqID)
		return
	}

	// 시나리오 파싱하여 VUser/Duration 추출
	cfg, err := config.LoadFromString(test.ScenarioYAML)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_SCENARIO",
			"Stored scenario YAML is invalid", err.Error(), reqID)
		return
	}

	// 오버라이드 처리
	var override struct {
		VUsersOverride   int    `json:"vusers_override"`
		DurationOverride string `json:"duration_override"`
	}
	json.NewDecoder(r.Body).Decode(&override)

	totalVUsers := cfg.Scenarios[0].VUsers
	if override.VUsersOverride > 0 {
		totalVUsers = override.VUsersOverride
	}
	durationSec := int(cfg.Scenarios[0].Duration.Seconds())

	// VUser 분배
	assignments, err := s.ctrl.Scheduler.Allocate(totalVUsers)
	if err != nil {
		writeError(w, http.StatusConflict, "NO_AGENTS_AVAILABLE", err.Error(), "", reqID)
		return
	}

	// TestRun 생성
	run := &model.TestRun{
		TestID:          id,
		Status:          model.TestRunPending,
		TotalVUsers:     totalVUsers,
		DurationSeconds: durationSec,
	}
	if err := s.store.CreateTestRun(r.Context(), run); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create test run", err.Error(), reqID)
		return
	}

	// 상태를 running으로
	_ = s.store.UpdateTestRunStatus(r.Context(), run.ID, model.TestRunRunning)
	run.Status = model.TestRunRunning

	// WebSocket 이벤트
	s.wsHub.BroadcastEvent("test_started", map[string]any{
		"test_run_id": run.ID,
		"test_id":     id,
		"assignments": assignments,
	})

	// 응답은 먼저 반환하고 백그라운드에서 실행
	writeJSON(w, http.StatusCreated, map[string]any{
		"test_run":    run,
		"assignments": assignments,
	}, reqID)

	// Agent들에게 StartTest 명령 큐잉
	for _, a := range assignments {
		ok := s.ctrl.AgentManager.EnqueueStartTest(
			a.AgentID,
			run.ID,
			test.ScenarioYAML,
			int32(a.AssignedVUsers),
		)
		if !ok {
			log.Printf("[API] Failed to enqueue StartTest for agent %s", a.AgentID)
		}
	}
}

func (s *Server) stopTest(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	reqID := getRequestID(w)

	run, _ := s.store.GetRunningTestRun(r.Context(), id)
	if run == nil {
		writeError(w, http.StatusBadRequest, "INVALID_PARAMETER",
			"No running test found for this test ID", "", reqID)
		return
	}

	_ = s.store.UpdateTestRunStatus(r.Context(), run.ID, model.TestRunStopped)

	// 모든 Agent에 StopTest 명령 큐잉
	for _, agent := range s.ctrl.AgentManager.AllAgents() {
		s.ctrl.AgentManager.EnqueueStopTest(agent.AgentID, run.ID)
	}

	s.wsHub.BroadcastEvent("test_stopped", map[string]any{
		"test_run_id": run.ID,
		"test_id":     id,
	})

	writeJSON(w, http.StatusOK, map[string]bool{"stopped": true}, reqID)
}

func (s *Server) listTestRuns(w http.ResponseWriter, r *http.Request) {
	testID := r.PathValue("id")
	page, perPage := getPagination(r)

	runs, total, err := s.store.ListTestRuns(r.Context(), testID, page, perPage)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list runs", err.Error(), getRequestID(w))
		return
	}
	if runs == nil {
		runs = []model.TestRun{}
	}
	writeJSONList(w, runs, total, page, perPage, getRequestID(w))
}

func (s *Server) getTestRun(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	run, err := s.store.GetTestRun(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get run", err.Error(), getRequestID(w))
		return
	}
	if run == nil {
		writeError(w, http.StatusNotFound, "RESULT_NOT_FOUND",
			fmt.Sprintf("Test run with ID '%s' not found", id), "", getRequestID(w))
		return
	}
	writeJSON(w, http.StatusOK, run, getRequestID(w))
}

// pollAgentCommand는 Agent가 pending command를 폴링하는 내부 엔드포인트다.
func (s *Server) pollAgentCommand(w http.ResponseWriter, r *http.Request) {
	agentID := r.PathValue("id")
	cmd := s.ctrl.AgentManager.DequeueCommand(agentID)
	if cmd == nil {
		writeJSON(w, http.StatusOK, map[string]any{"command": nil}, getRequestID(w))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"command": map[string]any{
			"type":            cmd.Type,
			"test_run_id":     cmd.TestRunID,
			"scenario_yaml":   cmd.ScenarioYAML,
			"assigned_vusers": cmd.AssignedVUsers,
		},
	}, getRequestID(w))
}

// stopTestByRunID는 run ID로 테스트를 중지하는 엔드포인트다 (클라이언트 호환용).
func (s *Server) stopTestByRunID(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")
	reqID := getRequestID(w)

	run, err := s.store.GetTestRun(r.Context(), runID)
	if err != nil || run == nil {
		writeError(w, http.StatusNotFound, "RUN_NOT_FOUND",
			fmt.Sprintf("Test run with ID '%s' not found", runID), "", reqID)
		return
	}

	_ = s.store.UpdateTestRunStatus(r.Context(), runID, model.TestRunStopped)

	for _, agent := range s.ctrl.AgentManager.AllAgents() {
		s.ctrl.AgentManager.EnqueueStopTest(agent.AgentID, runID)
	}

	s.wsHub.BroadcastEvent("test_stopped", map[string]any{
		"test_run_id": runID,
		"test_id":     run.TestID,
	})

	writeJSON(w, http.StatusOK, map[string]bool{"stopped": true}, reqID)
}

func (s *Server) getRunMetrics(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	metrics, err := s.store.ListMetrics(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get metrics", err.Error(), getRequestID(w))
		return
	}
	if metrics == nil {
		metrics = []map[string]interface{}{}
	}
	writeJSON(w, http.StatusOK, metrics, getRequestID(w))
}
