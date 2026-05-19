package phase

import (
	"fmt"

	"github.com/ilmich/emmai/internal/config"
)

// Manager handles phase transitions and state
type Manager struct {
	phases       map[string]*config.PhaseConfig
	currentPhase string
	initialPhase string
}

// NewManager creates a new phase manager with the given configuration
func NewManager(phases []config.PhaseConfig, initialPhase string) *Manager {
	phaseMap := make(map[string]*config.PhaseConfig)
	for i := range phases {
		phaseMap[phases[i].Name] = &phases[i]
	}

	return &Manager{
		phases:       phaseMap,
		currentPhase: "",
		initialPhase: initialPhase,
	}
}

// GetInitialPhase returns the first phase to start with
func (m *Manager) GetInitialPhase() string {
	return m.initialPhase
}

// GetCurrentPhase returns the active phase name
func (m *Manager) GetCurrentPhase() string {
	return m.currentPhase
}

// StartPhase transitions to a new phase and returns its prompt
func (m *Manager) StartPhase(phaseName string) (*PhaseResponse, error) {
	phase, exists := m.phases[phaseName]
	if !exists {
		availablePhases := make([]string, 0, len(m.phases))
		for name := range m.phases {
			availablePhases = append(availablePhases, name)
		}
		return nil, fmt.Errorf("phase '%s' not found. Available phases: %v", phaseName, availablePhases)
	}

	m.currentPhase = phaseName

	return &PhaseResponse{
		Phase:     phaseName,
		Prompt:    phase.Prompt,
		NextPhase: phase.NextPhase,
		ReadOnly:  phase.ReadOnly,
	}, nil
}

// ResetToInitial resets the workflow to the initial phase
func (m *Manager) ResetToInitial() (*PhaseResponse, error) {
	return m.StartPhase(m.initialPhase)
}

// IsReadOnly returns whether the current phase allows modifications
func (m *Manager) IsReadOnly() bool {
	if m.currentPhase == "" {
		return true // Safe default
	}

	phase, exists := m.phases[m.currentPhase]
	if !exists {
		return true
	}

	return phase.ReadOnly
}

// GetPhaseConfig returns the configuration for a specific phase
func (m *Manager) GetPhaseConfig(phaseName string) (*config.PhaseConfig, bool) {
	phase, exists := m.phases[phaseName]
	return phase, exists
}

// GetCurrentPhaseAllowedTools returns the allowed tools for the current phase
func (m *Manager) GetCurrentPhaseAllowedTools() []string {
	if m.currentPhase == "" {
		return []string{} // No phase active, no filtering
	}

	phase, exists := m.phases[m.currentPhase]
	if !exists {
		return []string{} // Phase doesn't exist, no filtering
	}

	return phase.GetAllowedTools()
}

// PhaseResponse is the structured response returned when starting a phase
type PhaseResponse struct {
	Phase     string `json:"phase"`
	Prompt    string `json:"prompt"`
	NextPhase string `json:"next_phase,omitempty"`
	ReadOnly  bool   `json:"read_only"`
}
