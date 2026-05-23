package phase

import (
	"fmt"
)

// PhaseClient is the interface for updating AI client with phase context
type PhaseClient interface {
	SetPhasePrompt(prompt string)
	SetPhaseAllowedTools(tools []string)
}

// Controller handles phase transitions with AI client integration
type Controller struct {
	manager *Manager
	client  PhaseClient
}

// NewController creates a new phase controller
func NewController(manager *Manager, client PhaseClient) *Controller {
	return &Controller{
		manager: manager,
		client:  client,
	}
}

// TransitionToPhase transitions to a specified phase and updates the AI client
func (c *Controller) TransitionToPhase(phaseName string) (*PhaseResponse, error) {
	// Validate phase exists
	if err := c.validatePhase(phaseName); err != nil {
		return nil, fmt.Errorf("invalid phase: %w", err)
	}

	// Transition to new phase
	phaseResponse, err := c.manager.StartPhase(phaseName)
	if err != nil {
		return nil, fmt.Errorf("failed to transition: %w", err)
	}

	// Update client context (inject prompt and tools)
	c.updateClientContext(phaseResponse)

	return phaseResponse, nil
}

// ResetToInitial resets the workflow to the initial phase
func (c *Controller) ResetToInitial() (*PhaseResponse, error) {
	phaseResponse, err := c.manager.ResetToInitial()
	if err != nil {
		return nil, err
	}

	// Update client context
	c.updateClientContext(phaseResponse)

	return phaseResponse, nil
}

// GetCurrentPhase returns the current phase name
func (c *Controller) GetCurrentPhase() string {
	return c.manager.GetCurrentPhase()
}

// updateClientContext updates the AI client with phase settings
func (c *Controller) updateClientContext(phaseResponse *PhaseResponse) {
	// Inject phase prompt into client
	c.client.SetPhasePrompt(phaseResponse.Prompt)

	// Update allowed tools for this phase
	allowedTools := c.manager.GetCurrentPhaseAllowedTools()
	c.client.SetPhaseAllowedTools(allowedTools)
}

// validatePhase checks if the phase name is valid
func (c *Controller) validatePhase(phaseName string) error {
	validPhases := []string{"explore", "plan", "execute", "verify"}

	for _, valid := range validPhases {
		if phaseName == valid {
			return nil
		}
	}

	return fmt.Errorf("invalid phase '%s'", phaseName)
}
