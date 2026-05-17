package tools

import "github.com/ilmich/emmai/internal/client"

// NewPhaseTools returns the phase management tools
func NewPhaseTools() []client.Tool {
	return []client.Tool{
		client.NewFunctionTool(
			"start_phase",
			"Transition to the next phase of the workflow state machine. Call this when you complete the current phase and are ready to proceed.",
			map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"phase": map[string]interface{}{
						"type":        "string",
						"description": "The name of the phase to start (e.g., 'explore', 'plan', 'execute', 'verify')",
					},
				},
				"required": []string{"phase"},
			},
		),
	}
}
