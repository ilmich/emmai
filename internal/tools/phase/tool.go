package phase

import (
	"encoding/json"
	"fmt"

	"github.com/ilmich/emmai/internal/client"
	"github.com/ilmich/emmai/internal/phase"
)

// PhaseClient is the interface for updating phase context
type PhaseClient interface {
	SetPhasePrompt(prompt string)
	SetPhaseAllowedTools(tools []string)
}

// PhaseTransitionResponse is the response from start_phase tool
type PhaseTransitionResponse struct {
	Success   bool   `json:"success"`
	Phase     string `json:"phase"`
	NextPhase string `json:"next_phase,omitempty"`
	ReadOnly  bool   `json:"read_only"`
	Message   string `json:"message,omitempty"`
	Error     string `json:"error,omitempty"`
	Hint      string `json:"hint,omitempty"`
}

// PhaseExecutor handles phase transition operations
type PhaseExecutor struct {
	phaseManager *phase.Manager
	client       PhaseClient
}

// NewPhaseExecutor creates a new phase executor
func NewPhaseExecutor(manager *phase.Manager, aiClient PhaseClient) *PhaseExecutor {
	return &PhaseExecutor{
		phaseManager: manager,
		client:       aiClient,
	}
}

// NewPhaseTool returns the start_phase tool definition
func NewPhaseTool() client.Tool {
	return client.NewFunctionTool(
		"start_phase",
		"Transition to next workflow phase. Updates AI context and available tools automatically.",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"phase": map[string]interface{}{
					"type":        "string",
					"description": "Phase name (explore, plan, execute, verify)",
					"enum":        []string{"explore", "plan", "execute", "verify"},
				},
			},
			"required": []string{"phase"},
		},
	)
}

// HandleStartPhase handles the start_phase tool invocation
func (e *PhaseExecutor) HandleStartPhase(args map[string]interface{}) (string, error) {
	// Extract and validate phase name
	phaseName, ok := args["phase"].(string)
	if !ok || phaseName == "" {
		return e.errorResponse("", "phase parameter is required", "Provide a valid phase name (explore, plan, execute, verify)")
	}

	// Validate phase exists
	if err := e.validatePhase(phaseName); err != nil {
		return e.errorResponse(phaseName, err.Error(), "Use one of: explore, plan, execute, verify")
	}

	// Transition to new phase
	phaseResponse, err := e.phaseManager.StartPhase(phaseName)
	if err != nil {
		return e.errorResponse(phaseName, fmt.Sprintf("failed to transition: %s", err.Error()), "Check phase name and workflow state")
	}

	// Update client context (side effect - necessary for workflow management)
	e.updateClientContext(phaseResponse)

	// Build success response
	response := PhaseTransitionResponse{
		Success:   true,
		Phase:     phaseResponse.Phase,
		NextPhase: phaseResponse.NextPhase,
		ReadOnly:  phaseResponse.ReadOnly,
		Message:   fmt.Sprintf("Transitioned to %s phase", phaseName),
	}

	// Marshal to JSON
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		return e.errorResponse(phaseName, "failed to encode response", "Internal error")
	}

	return string(jsonResponse), nil
}

// ResetPhase resets the workflow to the initial phase
func (e *PhaseExecutor) ResetPhase() error {
	phaseResponse, err := e.phaseManager.ResetToInitial()
	if err != nil {
		return err
	}

	// Update client context
	e.updateClientContext(phaseResponse)

	return nil
}

// GetCurrentPhase returns the current phase name
func (e *PhaseExecutor) GetCurrentPhase() string {
	return e.phaseManager.GetCurrentPhase()
}

// updateClientContext updates the AI client with phase settings
func (e *PhaseExecutor) updateClientContext(phaseResponse *phase.PhaseResponse) {
	// Inject phase prompt into client
	e.client.SetPhasePrompt(phaseResponse.Prompt)

	// Update allowed tools for this phase
	allowedTools := e.phaseManager.GetCurrentPhaseAllowedTools()
	e.client.SetPhaseAllowedTools(allowedTools)
}

// validatePhase checks if the phase name is valid
func (e *PhaseExecutor) validatePhase(phaseName string) error {
	validPhases := []string{"explore", "plan", "execute", "verify"}
	
	for _, valid := range validPhases {
		if phaseName == valid {
			return nil
		}
	}

	return fmt.Errorf("invalid phase '%s'", phaseName)
}

// errorResponse creates an error response
func (e *PhaseExecutor) errorResponse(phaseName, errorMsg, hint string) (string, error) {
	response := PhaseTransitionResponse{
		Success: false,
		Phase:   phaseName,
		Error:   errorMsg,
		Hint:    hint,
	}

	jsonResponse, _ := json.Marshal(response)
	return string(jsonResponse), nil
}
