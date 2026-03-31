package controller

import (
	"fmt"

	"github.com/omnitest/omnitest/pkg/model"
)

// Scheduler는 테스트 실행 시 Agent에 VUser를 분배한다.
type Scheduler struct {
	agentManager *AgentManager
}

// NewScheduler creates a new Scheduler.
func NewScheduler(am *AgentManager) *Scheduler {
	return &Scheduler{agentManager: am}
}

// Allocate는 온라인 Agent들에게 총 VUser를 균등 분배한다.
func (s *Scheduler) Allocate(totalVUsers int) ([]model.AgentAssignment, error) {
	agents := s.agentManager.OnlineAgents()
	if len(agents) == 0 {
		return nil, fmt.Errorf("no agents available")
	}

	perAgent := totalVUsers / len(agents)
	remainder := totalVUsers % len(agents)

	assignments := make([]model.AgentAssignment, len(agents))
	for i, a := range agents {
		assigned := perAgent
		if i < remainder {
			assigned++
		}
		assignments[i] = model.AgentAssignment{
			AgentID:        a.AgentID,
			AssignedVUsers: assigned,
		}
	}

	return assignments, nil
}
