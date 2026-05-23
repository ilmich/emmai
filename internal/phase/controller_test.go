package phase

import (
	"testing"

	"github.com/ilmich/emmai/internal/config"
)

// mockClient is a mock client for testing
type mockClient struct {
	phasePrompt       string
	phaseAllowedTools []string
}

func (m *mockClient) SetPhasePrompt(prompt string) {
	m.phasePrompt = prompt
}

func (m *mockClient) SetPhaseAllowedTools(tools []string) {
	m.phaseAllowedTools = tools
}

// Helper to create a test phase manager
func createTestPhaseManager() *Manager {
	phases := []config.PhaseConfig{
		{
			Name:         "explore",
			Prompt:       "Explore phase prompt",
			NextPhase:    "plan",
			ReadOnly:     true,
			AllowedTools: []string{"search_files"},
		},
		{
			Name:         "plan",
			Prompt:       "Plan phase prompt",
			NextPhase:    "execute",
			ReadOnly:     true,
			AllowedTools: []string{"search_files"},
		},
		{
			Name:         "execute",
			Prompt:       "Execute phase prompt",
			NextPhase:    "verify",
			ReadOnly:     false,
			AllowedTools: []string{"edit_file", "run_command"},
		},
		{
			Name:      "verify",
			Prompt:    "Verify phase prompt",
			NextPhase: "",
			ReadOnly:  true,
			AllowedTools: []string{"search_files", "run_command"},
		},
	}

	return NewManager(phases, "explore")
}

func TestController_TransitionToPhase_Success(t *testing.T) {
	manager := createTestPhaseManager()
	mockCli := &mockClient{}
	controller := NewController(manager, mockCli)

	response, err := controller.TransitionToPhase("execute")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if response.Phase != "execute" {
		t.Errorf("Expected phase=execute, got %s", response.Phase)
	}

	if response.NextPhase != "verify" {
		t.Errorf("Expected next_phase=verify, got %s", response.NextPhase)
	}

	if response.ReadOnly {
		t.Errorf("Expected read_only=false, got true")
	}

	// Verify client state was updated
	if mockCli.phasePrompt != "Execute phase prompt" {
		t.Errorf("Expected prompt to be updated, got: %s", mockCli.phasePrompt)
	}

	// Should have 2 tools: edit_file, run_command
	if len(mockCli.phaseAllowedTools) != 2 {
		t.Errorf("Expected 2 allowed tools, got %d: %v", len(mockCli.phaseAllowedTools), mockCli.phaseAllowedTools)
	}
}

func TestController_TransitionToPhase_InvalidPhase(t *testing.T) {
	manager := createTestPhaseManager()
	mockCli := &mockClient{}
	controller := NewController(manager, mockCli)

	_, err := controller.TransitionToPhase("invalid_phase")
	if err == nil {
		t.Fatalf("Expected error for invalid phase, got nil")
	}
}

func TestController_TransitionToPhase_EmptyPhase(t *testing.T) {
	manager := createTestPhaseManager()
	mockCli := &mockClient{}
	controller := NewController(manager, mockCli)

	_, err := controller.TransitionToPhase("")
	if err == nil {
		t.Fatalf("Expected error for empty phase, got nil")
	}
}

func TestController_ResetToInitial(t *testing.T) {
	manager := createTestPhaseManager()
	mockCli := &mockClient{}
	controller := NewController(manager, mockCli)

	// Transition to execute phase first
	_, err := controller.TransitionToPhase("execute")
	if err != nil {
		t.Fatalf("Failed to start execute phase: %v", err)
	}

	// Verify we're in execute phase
	if controller.GetCurrentPhase() != "execute" {
		t.Errorf("Expected current phase to be execute, got %s", controller.GetCurrentPhase())
	}

	// Reset to initial phase
	resp, err := controller.ResetToInitial()
	if err != nil {
		t.Fatalf("Failed to reset phase: %v", err)
	}

	// Verify we're back in explore phase
	if controller.GetCurrentPhase() != "explore" {
		t.Errorf("Expected current phase to be explore after reset, got %s", controller.GetCurrentPhase())
	}

	if resp.Phase != "explore" {
		t.Errorf("Expected response phase to be explore, got %s", resp.Phase)
	}

	// Verify client state was updated
	if mockCli.phasePrompt != "Explore phase prompt" {
		t.Errorf("Expected prompt to be reset, got: %s", mockCli.phasePrompt)
	}
}

func TestController_PhaseTransitionFlow(t *testing.T) {
	manager := createTestPhaseManager()
	mockCli := &mockClient{}
	controller := NewController(manager, mockCli)

	// Test full phase flow: explore -> plan -> execute -> verify
	phases := []string{"explore", "plan", "execute", "verify"}
	
	for _, phaseName := range phases {
		response, err := controller.TransitionToPhase(phaseName)
		if err != nil {
			t.Fatalf("Failed to transition to %s: %v", phaseName, err)
		}

		if response.Phase != phaseName {
			t.Errorf("Expected phase=%s, got %s", phaseName, response.Phase)
		}

		if controller.GetCurrentPhase() != phaseName {
			t.Errorf("Expected current phase to be %s, got %s", phaseName, controller.GetCurrentPhase())
		}
	}
}
