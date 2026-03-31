package controller

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/omnitest/omnitest/internal/store"
	"github.com/omnitest/omnitest/pkg/model"
)

// TestCommand는 Agent에 전달할 테스트 명령이다.
type TestCommand struct {
	Type        string // "start" or "stop"
	TestRunID   string
	ScenarioYAML string
	AssignedVUsers int32
}

// AgentManager는 연결된 Agent를 관리한다.
type AgentManager struct {
	agents   map[string]*model.AgentInfo
	commands map[string]chan *TestCommand // agentID → pending commands
	mu       sync.RWMutex
	store    *store.Store
}

// NewAgentManager creates a new AgentManager.
func NewAgentManager(st *store.Store) *AgentManager {
	return &AgentManager{
		agents:   make(map[string]*model.AgentInfo),
		commands: make(map[string]chan *TestCommand),
		store:    st,
	}
}

// Register는 Agent를 등록한다.
func (m *AgentManager) Register(agent *model.AgentInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()

	agent.LastHeartbeat = time.Now()
	m.agents[agent.AgentID] = agent

	// command channel 생성 (이미 있으면 재사용)
	if _, ok := m.commands[agent.AgentID]; !ok {
		m.commands[agent.AgentID] = make(chan *TestCommand, 10)
	}

	// DB에도 저장
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := m.store.CreateAgent(ctx, agent); err != nil {
		log.Printf("[AgentManager] DB save error for agent %s: %v", agent.AgentID, err)
	}
}

// Heartbeat는 Agent 헬스 정보를 업데이트한다.
func (m *AgentManager) Heartbeat(agentID string, status model.AgentStatus, cpuUsage, memoryUsage float64, activeVUsers int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	agent, ok := m.agents[agentID]
	if !ok {
		return
	}

	agent.Status = status
	agent.CPUUsage = cpuUsage
	agent.MemoryUsage = memoryUsage
	agent.ActiveVUsers = activeVUsers
	agent.LastHeartbeat = time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	m.store.UpdateAgentHeartbeat(ctx, agentID, status, cpuUsage, memoryUsage, activeVUsers)
}

// EnqueueStartTest는 Agent에 StartTest 명령을 큐에 넣는다.
func (m *AgentManager) EnqueueStartTest(agentID, testRunID, scenarioYAML string, assignedVUsers int32) bool {
	m.mu.RLock()
	ch, ok := m.commands[agentID]
	m.mu.RUnlock()

	if !ok {
		return false
	}

	select {
	case ch <- &TestCommand{
		Type:           "start",
		TestRunID:      testRunID,
		ScenarioYAML:   scenarioYAML,
		AssignedVUsers: assignedVUsers,
	}:
		log.Printf("[AgentManager] Enqueued StartTest for agent %s (run: %s, vusers: %d)", agentID, testRunID, assignedVUsers)
		return true
	default:
		log.Printf("[AgentManager] Command queue full for agent %s", agentID)
		return false
	}
}

// EnqueueStopTest는 Agent에 StopTest 명령을 큐에 넣는다.
func (m *AgentManager) EnqueueStopTest(agentID, testRunID string) bool {
	m.mu.RLock()
	ch, ok := m.commands[agentID]
	m.mu.RUnlock()

	if !ok {
		return false
	}

	select {
	case ch <- &TestCommand{
		Type:      "stop",
		TestRunID: testRunID,
	}:
		return true
	default:
		return false
	}
}

// DequeueCommand는 Agent의 다음 pending command를 반환한다. 없으면 nil.
func (m *AgentManager) DequeueCommand(agentID string) *TestCommand {
	m.mu.RLock()
	ch, ok := m.commands[agentID]
	m.mu.RUnlock()

	if !ok {
		return nil
	}

	select {
	case cmd := <-ch:
		return cmd
	default:
		return nil
	}
}

// OnlineAgents는 현재 온라인 Agent 목록을 반환한다.
func (m *AgentManager) OnlineAgents() []model.AgentInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var agents []model.AgentInfo
	for _, a := range m.agents {
		if a.Status != model.AgentStatusOffline {
			agents = append(agents, *a)
		}
	}
	return agents
}

// AllAgents는 모든 등록된 Agent 목록을 반환한다.
func (m *AgentManager) AllAgents() []model.AgentInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	agents := make([]model.AgentInfo, 0, len(m.agents))
	for _, a := range m.agents {
		agents = append(agents, *a)
	}
	return agents
}

// Get는 특정 Agent를 반환한다.
func (m *AgentManager) Get(agentID string) *model.AgentInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if a, ok := m.agents[agentID]; ok {
		copied := *a
		return &copied
	}
	return nil
}

// Remove는 Agent를 제거한다.
func (m *AgentManager) Remove(agentID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.agents, agentID)
	delete(m.commands, agentID)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	m.store.DeleteAgent(ctx, agentID)
}

// SetStatus sets the status of a specific agent.
func (m *AgentManager) SetStatus(agentID string, status model.AgentStatus) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if a, ok := m.agents[agentID]; ok {
		a.Status = status
	}
}

// HealthCheckLoop는 주기적으로 헬스체크 타임아웃된 Agent를 offline으로 전환한다.
func (m *AgentManager) HealthCheckLoop(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.markOffline()
		}
	}
}

func (m *AgentManager) markOffline() {
	m.mu.Lock()
	defer m.mu.Unlock()

	threshold := time.Now().Add(-30 * time.Second)
	for _, a := range m.agents {
		if a.Status != model.AgentStatusOffline && a.LastHeartbeat.Before(threshold) {
			log.Printf("[AgentManager] Agent %s marked offline (last heartbeat: %v)", a.AgentID, a.LastHeartbeat)
			a.Status = model.AgentStatusOffline

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			m.store.UpdateAgentStatus(ctx, a.AgentID, model.AgentStatusOffline)
			cancel()
		}
	}
}
