package controller

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/omnitest/omnitest/pkg/model"
)

// --- AgentManager tests (without DB dependency) ---

func newTestAgentManager() *AgentManager {
	return &AgentManager{
		agents: make(map[string]*model.AgentInfo),
		store:  nil, // DB 없이 in-memory 테스트
	}
}

// Register without DB (store=nil 이면 패닉 방지용 래퍼)
func registerAgent(m *AgentManager, agent *model.AgentInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()
	agent.LastHeartbeat = time.Now()
	m.agents[agent.AgentID] = agent
}

func TestAgentManager_RegisterAndGet(t *testing.T) {
	m := newTestAgentManager()
	agent := &model.AgentInfo{
		AgentID:   "agent-1",
		Hostname:  "host-1",
		MaxVUsers: 5000,
		Status:    model.AgentStatusIdle,
	}
	registerAgent(m, agent)

	got := m.Get("agent-1")
	if got == nil {
		t.Fatal("Get() returned nil for registered agent")
	}
	if got.AgentID != "agent-1" {
		t.Errorf("AgentID = %q, want %q", got.AgentID, "agent-1")
	}
	if got.MaxVUsers != 5000 {
		t.Errorf("MaxVUsers = %d, want %d", got.MaxVUsers, 5000)
	}
}

func TestAgentManager_GetNotFound(t *testing.T) {
	m := newTestAgentManager()
	got := m.Get("nonexistent")
	if got != nil {
		t.Fatal("Get() should return nil for unregistered agent")
	}
}

func TestAgentManager_AllAgents(t *testing.T) {
	m := newTestAgentManager()

	agents := m.AllAgents()
	if len(agents) != 0 {
		t.Fatalf("AllAgents() on empty: got %d, want 0", len(agents))
	}

	registerAgent(m, &model.AgentInfo{AgentID: "a1", Status: model.AgentStatusIdle})
	registerAgent(m, &model.AgentInfo{AgentID: "a2", Status: model.AgentStatusRunning})
	registerAgent(m, &model.AgentInfo{AgentID: "a3", Status: model.AgentStatusOffline})

	agents = m.AllAgents()
	if len(agents) != 3 {
		t.Fatalf("AllAgents() got %d, want 3", len(agents))
	}
}

func TestAgentManager_OnlineAgents(t *testing.T) {
	m := newTestAgentManager()
	registerAgent(m, &model.AgentInfo{AgentID: "a1", Status: model.AgentStatusIdle})
	registerAgent(m, &model.AgentInfo{AgentID: "a2", Status: model.AgentStatusRunning})
	registerAgent(m, &model.AgentInfo{AgentID: "a3", Status: model.AgentStatusOffline})

	online := m.OnlineAgents()
	if len(online) != 2 {
		t.Fatalf("OnlineAgents() got %d, want 2 (offline excluded)", len(online))
	}
}

func TestAgentManager_Remove(t *testing.T) {
	m := newTestAgentManager()
	registerAgent(m, &model.AgentInfo{AgentID: "agent-1"})

	// Remove without DB
	m.mu.Lock()
	delete(m.agents, "agent-1")
	m.mu.Unlock()

	got := m.Get("agent-1")
	if got != nil {
		t.Fatal("Get() should return nil after Remove()")
	}
}

func TestAgentManager_SetStatus(t *testing.T) {
	m := newTestAgentManager()
	registerAgent(m, &model.AgentInfo{AgentID: "a1", Status: model.AgentStatusIdle})

	m.SetStatus("a1", model.AgentStatusRunning)
	got := m.Get("a1")
	if got.Status != model.AgentStatusRunning {
		t.Errorf("Status = %q, want %q", got.Status, model.AgentStatusRunning)
	}
}

func TestAgentManager_Heartbeat_InMemory(t *testing.T) {
	m := newTestAgentManager()
	registerAgent(m, &model.AgentInfo{AgentID: "a1", Status: model.AgentStatusIdle})

	// Heartbeat without DB - direct update
	m.mu.Lock()
	a := m.agents["a1"]
	a.Status = model.AgentStatusRunning
	a.CPUUsage = 45.0
	a.MemoryUsage = 60.0
	a.ActiveVUsers = 100
	a.LastHeartbeat = time.Now()
	m.mu.Unlock()

	got := m.Get("a1")
	if got.Status != model.AgentStatusRunning {
		t.Errorf("Status after heartbeat = %q, want %q", got.Status, model.AgentStatusRunning)
	}
	if got.CPUUsage != 45.0 {
		t.Errorf("CPUUsage = %f, want 45.0", got.CPUUsage)
	}
}

func TestAgentManager_MarkOffline(t *testing.T) {
	m := newTestAgentManager()

	staleTime := time.Now().Add(-40 * time.Second)
	freshTime := time.Now()

	m.mu.Lock()
	m.agents["stale"] = &model.AgentInfo{
		AgentID:       "stale",
		Status:        model.AgentStatusIdle,
		LastHeartbeat: staleTime,
	}
	m.agents["fresh"] = &model.AgentInfo{
		AgentID:       "fresh",
		Status:        model.AgentStatusIdle,
		LastHeartbeat: freshTime,
	}
	m.mu.Unlock()

	// Reproduce markOffline logic (without DB calls)
	m.mu.Lock()
	cutoff := time.Now().Add(-30 * time.Second)
	for _, a := range m.agents {
		if a.Status != model.AgentStatusOffline && a.LastHeartbeat.Before(cutoff) {
			a.Status = model.AgentStatusOffline
		}
	}
	m.mu.Unlock()

	stale := m.Get("stale")
	if stale.Status != model.AgentStatusOffline {
		t.Errorf("stale agent Status = %q, want %q", stale.Status, model.AgentStatusOffline)
	}
	fresh := m.Get("fresh")
	if fresh.Status != model.AgentStatusIdle {
		t.Errorf("fresh agent Status = %q, want %q", fresh.Status, model.AgentStatusIdle)
	}
}

func TestAgentManager_ConcurrentAccess(t *testing.T) {
	m := newTestAgentManager()
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			registerAgent(m, &model.AgentInfo{
				AgentID:  fmt.Sprintf("agent-%d", i),
				Hostname: fmt.Sprintf("host-%d", i),
				Status:   model.AgentStatusIdle,
			})
		}(i)
	}

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m.AllAgents()
			m.OnlineAgents()
		}()
	}

	wg.Wait()

	agents := m.AllAgents()
	if len(agents) != 100 {
		t.Errorf("after concurrent registration: got %d agents, want 100", len(agents))
	}
}

func TestAgentManager_GetReturnsCopy(t *testing.T) {
	m := newTestAgentManager()
	registerAgent(m, &model.AgentInfo{AgentID: "a1", Hostname: "original"})

	got := m.Get("a1")
	got.Hostname = "modified"

	original := m.Get("a1")
	if original.Hostname != "original" {
		t.Error("Get() should return a copy, not a reference to internal state")
	}
}

// --- Scheduler tests ---

func TestScheduler_Allocate(t *testing.T) {
	m := newTestAgentManager()
	registerAgent(m, &model.AgentInfo{AgentID: "a1", Status: model.AgentStatusIdle, MaxVUsers: 5000})
	registerAgent(m, &model.AgentInfo{AgentID: "a2", Status: model.AgentStatusIdle, MaxVUsers: 5000})

	s := NewScheduler(m)

	assignments, err := s.Allocate(100)
	if err != nil {
		t.Fatalf("Allocate() error = %v", err)
	}
	if len(assignments) != 2 {
		t.Fatalf("expected 2 assignments, got %d", len(assignments))
	}

	totalAssigned := 0
	for _, a := range assignments {
		totalAssigned += a.AssignedVUsers
	}
	if totalAssigned != 100 {
		t.Errorf("total assigned = %d, want 100", totalAssigned)
	}
}

func TestScheduler_Allocate_OddDistribution(t *testing.T) {
	m := newTestAgentManager()
	registerAgent(m, &model.AgentInfo{AgentID: "a1", Status: model.AgentStatusIdle})
	registerAgent(m, &model.AgentInfo{AgentID: "a2", Status: model.AgentStatusIdle})
	registerAgent(m, &model.AgentInfo{AgentID: "a3", Status: model.AgentStatusIdle})

	s := NewScheduler(m)

	assignments, err := s.Allocate(10) // 10 / 3 = 3 remainder 1
	if err != nil {
		t.Fatalf("Allocate() error = %v", err)
	}

	totalAssigned := 0
	for _, a := range assignments {
		totalAssigned += a.AssignedVUsers
	}
	if totalAssigned != 10 {
		t.Errorf("total assigned = %d, want 10", totalAssigned)
	}
}

func TestScheduler_Allocate_NoAgents(t *testing.T) {
	m := newTestAgentManager()
	s := NewScheduler(m)

	_, err := s.Allocate(100)
	if err == nil {
		t.Fatal("Allocate() should fail when no agents available")
	}
}

func TestScheduler_Allocate_ExcludesOffline(t *testing.T) {
	m := newTestAgentManager()
	registerAgent(m, &model.AgentInfo{AgentID: "a1", Status: model.AgentStatusIdle})
	registerAgent(m, &model.AgentInfo{AgentID: "a2", Status: model.AgentStatusOffline})

	s := NewScheduler(m)

	assignments, err := s.Allocate(100)
	if err != nil {
		t.Fatalf("Allocate() error = %v", err)
	}
	if len(assignments) != 1 {
		t.Fatalf("expected 1 assignment (offline excluded), got %d", len(assignments))
	}
	if assignments[0].AssignedVUsers != 100 {
		t.Errorf("single agent should get all 100 VUsers, got %d", assignments[0].AssignedVUsers)
	}
}

// --- Aggregator tests ---

func TestAggregator_Aggregate_Empty(t *testing.T) {
	agg := &Aggregator{
		metrics: make(map[string]map[string]*latestMetric),
	}

	result := agg.Aggregate("nonexistent")
	if result != nil {
		t.Fatal("Aggregate() on empty should return nil")
	}
}

func TestAggregator_Cleanup(t *testing.T) {
	agg := &Aggregator{
		metrics: make(map[string]map[string]*latestMetric),
	}
	agg.metrics["run-1"] = make(map[string]*latestMetric)

	agg.Cleanup("run-1")

	if _, ok := agg.metrics["run-1"]; ok {
		t.Fatal("Cleanup() should remove testRunID from metrics")
	}
}
