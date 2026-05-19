package execution

import (
	"testing"

	"github.com/ilmich/emmai/internal/config"
)

func TestSecurityValidator_ValidateCommand(t *testing.T) {
	policy := &config.CommandExecutionPolicy{
		Enabled: true,
		AllowedCommands: []config.AllowedCommand{
			{
				Prefix:        "go",
				Subcommands:   []string{"test", "build"},
				BlockedArgs:   []string{"run"},
				AllowedPhases: []string{"execute", "verify"},
			},
			{
				Prefix:      "git",
				Subcommands: []string{"status", "diff"},
			},
		},
	}

	validator := NewSecurityValidator(policy)

	tests := []struct {
		name        string
		cmd         string
		phase       string
		expectError bool
	}{
		{"allowed command", "go test ./...", "execute", false},
		{"allowed command in phase", "go build", "execute", false},
		{"blocked subcommand", "go run main.go", "execute", true},
		{"wrong phase", "go test", "explore", true},
		{"not whitelisted", "rm -rf /", "execute", true},
		{"git allowed", "git status", "explore", false},
		{"blocked arg", "go test --run", "execute", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateCommand(tt.cmd, tt.phase)
			if tt.expectError && err == nil {
				t.Errorf("Expected error for cmd '%s' in phase '%s'", tt.cmd, tt.phase)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error for cmd '%s' in phase '%s': %v", tt.cmd, tt.phase, err)
			}
		})
	}
}

func TestSecurityValidator_ValidateTimeout(t *testing.T) {
	policy := &config.CommandExecutionPolicy{
		DefaultTimeoutSec: 30,
		MaxTimeoutSec:     120,
	}

	validator := NewSecurityValidator(policy)

	tests := []struct {
		input    int
		expected int
	}{
		{0, 30},      // default
		{-1, 30},     // negative -> default
		{60, 60},     // valid
		{200, 120},   // too high -> max
		{10, 10},     // valid
	}

	for _, tt := range tests {
		result := validator.ValidateTimeout(tt.input)
		if result != tt.expected {
			t.Errorf("ValidateTimeout(%d) = %d, expected %d", tt.input, result, tt.expected)
		}
	}
}
