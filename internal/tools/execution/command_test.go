package execution

import (
	"encoding/json"
	"testing"

	"github.com/ilmich/emmai/internal/config"
	"github.com/ilmich/emmai/internal/phase"
)

func TestCommandExecutor_HandleRunCommand(t *testing.T) {
	// Create phase manager
	phaseManager := phase.NewManager([]config.PhaseConfig{
		{Name: "execute", ReadOnly: false},
	}, "execute")

	// Start execute phase
	_, err := phaseManager.StartPhase("execute")
	if err != nil {
		t.Fatal(err)
	}

	// Create policy
	policy := &config.CommandExecutionPolicy{
		Enabled: true,
		AllowedCommands: []config.AllowedCommand{
			{
				Prefix:        "echo",
				AllowedPhases: []string{"execute"},
			},
		},
		DefaultTimeoutSec:  30,
		MaxTimeoutSec:      120,
		MaxOutputSizeBytes: 1048576,
	}

	executor := NewCommandExecutor(t.TempDir(), policy, phaseManager)

	// Test successful command
	args := map[string]interface{}{
		"command": "echo Hello World",
	}

	result, err := executor.HandleRunCommand(args)
	if err != nil {
		t.Errorf("HandleRunCommand failed: %v", err)
	}

	var response CommandResponse
	if err := json.Unmarshal([]byte(result), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if !response.Success {
		t.Errorf("Expected success, got error: %s", response.Error)
	}

	if response.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", response.ExitCode)
	}

	t.Logf("Command result: %+v", response)
}

func TestCommandExecutor_SecurityBlock(t *testing.T) {
	phaseManager := phase.NewManager([]config.PhaseConfig{
		{Name: "execute", ReadOnly: false},
	}, "execute")

	_, _ = phaseManager.StartPhase("execute")

	policy := &config.CommandExecutionPolicy{
		Enabled: true,
		AllowedCommands: []config.AllowedCommand{
			{
				Prefix:        "go",
				Subcommands:   []string{"test"},
				AllowedPhases: []string{"execute"},
			},
		},
		DefaultTimeoutSec:  30,
		MaxTimeoutSec:      120,
		MaxOutputSizeBytes: 1048576,
	}

	executor := NewCommandExecutor(t.TempDir(), policy, phaseManager)

	// Test blocked command
	args := map[string]interface{}{
		"command": "rm -rf /",
	}

	result, err := executor.HandleRunCommand(args)
	if err != nil {
		t.Errorf("HandleRunCommand failed: %v", err)
	}

	var response CommandResponse
	if err := json.Unmarshal([]byte(result), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response.Success {
		t.Error("Expected command to be blocked")
	}

	if response.Error != "command blocked by security policy" {
		t.Errorf("Expected security error, got: %s", response.Error)
	}

	t.Logf("Security block result: %+v", response)
}
