package agent

import (
	"testing"
)

func TestNew(t *testing.T) {
	a := New(Config{
		ControllerAddr: "localhost:9090",
		Name:           "test-agent",
		MaxVUsers:      5000,
	})

	if a == nil {
		t.Fatal("New() returned nil")
	}
	if a.name != "test-agent" {
		t.Errorf("name = %q, want %q", a.name, "test-agent")
	}
	if a.maxVUsers != 5000 {
		t.Errorf("maxVUsers = %d, want %d", a.maxVUsers, 5000)
	}
	if a.status != "idle" {
		t.Errorf("initial status = %q, want %q", a.status, "idle")
	}
	if a.controllerAddr != "localhost:9090" {
		t.Errorf("controllerAddr = %q, want %q", a.controllerAddr, "localhost:9090")
	}
}

func TestNew_DefaultMaxVUsers(t *testing.T) {
	a := New(Config{
		ControllerAddr: "localhost:9090",
		Name:           "test-agent",
		MaxVUsers:      0, // should default to 1000
	})

	if a.maxVUsers != 1000 {
		t.Errorf("maxVUsers = %d, want 1000 (default)", a.maxVUsers)
	}
}

func TestNew_DefaultName(t *testing.T) {
	a := New(Config{
		ControllerAddr: "localhost:9090",
		// Name is empty, should use hostname
	})

	if a.name == "" {
		t.Error("name should default to hostname when not specified")
	}
}

func TestNew_AgentID_Format(t *testing.T) {
	a := New(Config{
		ControllerAddr: "localhost:9090",
		Name:           "myagent",
	})

	if a.agentID == "" {
		t.Error("agentID should be auto-generated")
	}
	// Agent ID format: "agent-{name}-{timestamp}"
	if len(a.agentID) < 10 {
		t.Errorf("agentID seems too short: %q", a.agentID)
	}
}

func TestStopTest(t *testing.T) {
	a := New(Config{
		ControllerAddr: "localhost:9090",
		Name:           "test-agent",
	})

	// No active test - should return false
	stopped := a.StopTest("nonexistent-run")
	if stopped {
		t.Error("StopTest() should return false when no active test")
	}
}
