package phase

import (
	"fmt"
)

// PhaseClient is the interface for updating AI client with phase context
type PhaseClient interface {
	SetPhasePrompt(prompt string)
	SetPhaseAllowedTools(tools []string)
}

// SummaryProvider returns a codebase summary string to prepend to phase prompts.
// It returns "" when no index is available.
type SummaryProvider func() string

// Controller handles phase transitions with AI client integration
type Controller struct {
	manager  *Manager
	client   PhaseClient
	summary  SummaryProvider // may be nil
}

// NewController creates a new phase controller
func NewController(manager *Manager, client PhaseClient) *Controller {
	return &Controller{
		manager: manager,
		client:  client,
	}
}

// SetSummaryProvider attaches a summary provider that is called on every phase transition.
func (c *Controller) SetSummaryProvider(fn SummaryProvider) {
	c.summary = fn
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
	prompt := phaseResponse.Prompt
	if c.summary != nil {
		if s := c.summary(); s != "" {
			prompt = s + "\n\n" + prompt
		}
	}
	c.client.SetPhasePrompt(prompt)

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
