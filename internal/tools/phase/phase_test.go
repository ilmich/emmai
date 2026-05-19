package phase

import (
	"encoding/json"
	"testing"

	"github.com/ilmich/emmai/internal/config"
	"github.com/ilmich/emmai/internal/phase"
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
func createTestPhaseManager() *phase.Manager {
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

	return phase.NewManager(phases, "explore")
}

func TestPhaseExecutor_StartPhase_Success(t *testing.T) {
	manager := createTestPhaseManager()
	mockCli := &mockClient{}
	executor := NewPhaseExecutor(manager, mockCli)

	args := map[string]interface{}{
		"phase": "execute",
	}

	result, err := executor.HandleStartPhase(args)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	var response PhaseTransitionResponse
	if err := json.Unmarshal([]byte(result), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if !response.Success {
		t.Errorf("Expected success=true, got false. Error: %s", response.Error)
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

	// Should have 3 tools: edit_file, run_command, and auto-injected start_phase
	if len(mockCli.phaseAllowedTools) != 3 {
		t.Errorf("Expected 3 allowed tools (including auto-injected start_phase), got %d: %v", len(mockCli.phaseAllowedTools), mockCli.phaseAllowedTools)
	}
}

func TestPhaseExecutor_StartPhase_InvalidPhase(t *testing.T) {
	manager := createTestPhaseManager()
	mockCli := &mockClient{}
	executor := NewPhaseExecutor(manager, mockCli)

	args := map[string]interface{}{
		"phase": "invalid_phase",
	}

	result, err := executor.HandleStartPhase(args)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	var response PhaseTransitionResponse
	if err := json.Unmarshal([]byte(result), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Success {
		t.Errorf("Expected success=false, got true")
	}

	if response.Error == "" {
		t.Errorf("Expected error message, got empty")
	}

	if response.Hint == "" {
		t.Errorf("Expected hint message, got empty")
	}
}

func TestPhaseExecutor_StartPhase_EmptyPhase(t *testing.T) {
	manager := createTestPhaseManager()
	mockCli := &mockClient{}
	executor := NewPhaseExecutor(manager, mockCli)

	args := map[string]interface{}{
		"phase": "",
	}

	result, err := executor.HandleStartPhase(args)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	var response PhaseTransitionResponse
	if err := json.Unmarshal([]byte(result), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Success {
		t.Errorf("Expected success=false, got true")
	}
}

func TestPhaseExecutor_StartPhase_MissingPhaseArg(t *testing.T) {
	manager := createTestPhaseManager()
	mockCli := &mockClient{}
	executor := NewPhaseExecutor(manager, mockCli)

	args := map[string]interface{}{}

	result, err := executor.HandleStartPhase(args)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	var response PhaseTransitionResponse
	if err := json.Unmarshal([]byte(result), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Success {
		t.Errorf("Expected success=false, got true")
	}
}

func TestPhaseExecutor_ResetPhase(t *testing.T) {
	manager := createTestPhaseManager()
	mockCli := &mockClient{}
	executor := NewPhaseExecutor(manager, mockCli)

	// Transition to execute phase first
	args := map[string]interface{}{
		"phase": "execute",
	}
	_, err := executor.HandleStartPhase(args)
	if err != nil {
		t.Fatalf("Failed to start execute phase: %v", err)
	}

	// Verify we're in execute phase
	if executor.GetCurrentPhase() != "execute" {
		t.Errorf("Expected current phase to be execute, got %s", executor.GetCurrentPhase())
	}

	// Reset to initial phase
	err = executor.ResetPhase()
	if err != nil {
		t.Fatalf("Failed to reset phase: %v", err)
	}

	// Verify we're back in explore phase
	if executor.GetCurrentPhase() != "explore" {
		t.Errorf("Expected current phase to be explore after reset, got %s", executor.GetCurrentPhase())
	}

	// Verify client state was updated
	if mockCli.phasePrompt != "Explore phase prompt" {
		t.Errorf("Expected prompt to be reset, got: %s", mockCli.phasePrompt)
	}
}

func TestPhaseExecutor_PhaseTransitionFlow(t *testing.T) {
	manager := createTestPhaseManager()
	mockCli := &mockClient{}
	executor := NewPhaseExecutor(manager, mockCli)

	// Test full phase flow: explore -> plan -> execute -> verify
	phases := []string{"explore", "plan", "execute", "verify"}
	
	for _, phaseName := range phases {
		args := map[string]interface{}{
			"phase": phaseName,
		}

		result, err := executor.HandleStartPhase(args)
		if err != nil {
			t.Fatalf("Failed to transition to %s: %v", phaseName, err)
		}

		var response PhaseTransitionResponse
		if err := json.Unmarshal([]byte(result), &response); err != nil {
			t.Fatalf("Failed to unmarshal response for %s: %v", phaseName, err)
		}

		if !response.Success {
			t.Errorf("Failed to transition to %s: %s", phaseName, response.Error)
		}

		if response.Phase != phaseName {
			t.Errorf("Expected phase=%s, got %s", phaseName, response.Phase)
		}

		if executor.GetCurrentPhase() != phaseName {
			t.Errorf("Expected current phase to be %s, got %s", phaseName, executor.GetCurrentPhase())
		}
	}
}

func TestNewPhaseTool(t *testing.T) {
	tool := NewPhaseTool()

	// Tool interface doesn't expose Name/Description directly
	// Just verify it was created without panicking
	_ = tool
}
