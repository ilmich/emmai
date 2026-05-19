package tools

import (
	"encoding/json"
	"fmt"

	"github.com/ilmich/emmai/internal/client"
	"github.com/ilmich/emmai/internal/phase"
)

// PhaseToolsExecutor handles phase management tool execution
type PhaseToolsExecutor struct {
	phaseManager *phase.Manager
	client       *client.OpenAIClient
}

// NewPhaseToolsExecutor creates a new phase tools executor
func NewPhaseToolsExecutor(manager *phase.Manager, aiClient *client.OpenAIClient) *PhaseToolsExecutor {
	return &PhaseToolsExecutor{
		phaseManager: manager,
		client:       aiClient,
	}
}

// RegisterHandlers registers all phase tool handlers with the executor
func (e *PhaseToolsExecutor) RegisterHandlers(executor interface {
	RegisterHandler(name string, handler func(args map[string]interface{}) (string, error))
}) {
	executor.RegisterHandler("start_phase", e.handleStartPhase)
}

// handleStartPhase handles the start_phase tool invocation
func (e *PhaseToolsExecutor) handleStartPhase(args map[string]interface{}) (string, error) {
	// Extract phase name
	phaseName, ok := args["phase"].(string)
	if !ok || phaseName == "" {
		return "", fmt.Errorf("phase is required and must be a non-empty string")
	}

	// Transition to new phase
	response, err := e.phaseManager.StartPhase(phaseName)
	if err != nil {
		return "", err
	}

	// Inject phase prompt into client (not returned to LLM)
	e.client.SetPhasePrompt(response.Prompt)

	// Update allowed tools for this phase
	allowedTools := e.phaseManager.GetCurrentPhaseAllowedTools()
	e.client.SetPhaseAllowedTools(allowedTools)

	// Return simple OK response
	simpleResponse := map[string]interface{}{
		"status":     "ok",
		"phase":      response.Phase,
		"next_phase": response.NextPhase,
		"read_only":  response.ReadOnly,
	}

	jsonResponse, err := json.MarshalIndent(simpleResponse, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to encode response: %w", err)
	}

	return string(jsonResponse), nil
}

// ResetPhase resets the workflow to the initial phase
func (e *PhaseToolsExecutor) ResetPhase() error {
	response, err := e.phaseManager.ResetToInitial()
	if err != nil {
		return err
	}

	// Update client with initial phase settings
	e.client.SetPhasePrompt(response.Prompt)
	allowedTools := e.phaseManager.GetCurrentPhaseAllowedTools()
	e.client.SetPhaseAllowedTools(allowedTools)

	return nil
}
