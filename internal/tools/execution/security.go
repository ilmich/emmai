package execution

import (
	"fmt"
	"strings"

	"github.com/ilmich/emmai/internal/config"
)

// SecurityValidator validates commands against security policy
type SecurityValidator struct {
	policy *config.CommandExecutionPolicy
}

// NewSecurityValidator creates a new security validator
func NewSecurityValidator(policy *config.CommandExecutionPolicy) *SecurityValidator {
	return &SecurityValidator{policy: policy}
}

// ValidateCommand checks if a command is allowed by the security policy
func (v *SecurityValidator) ValidateCommand(cmd string, currentPhase string) error {
	if !v.policy.Enabled {
		return fmt.Errorf("command execution disabled")
	}

	// Parse command into prefix and args
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return fmt.Errorf("empty command")
	}

	prefix := parts[0]
	args := parts[1:]

	// Find matching allowed command
	var allowedCmd *config.AllowedCommand
	for i := range v.policy.AllowedCommands {
		if v.policy.AllowedCommands[i].Prefix == prefix {
			allowedCmd = &v.policy.AllowedCommands[i]
			break
		}
	}

	if allowedCmd == nil {
		return fmt.Errorf("'%s' not in whitelist", prefix)
	}

	// Check phase restrictions
	if len(allowedCmd.AllowedPhases) > 0 {
		phaseAllowed := false
		for _, phase := range allowedCmd.AllowedPhases {
			if phase == currentPhase {
				phaseAllowed = true
				break
			}
		}
		if !phaseAllowed {
			return fmt.Errorf("'%s' not allowed in phase '%s'", prefix, currentPhase)
		}
	}

	// Check subcommands if specified
	if len(allowedCmd.Subcommands) > 0 && len(args) > 0 {
		subcommand := args[0]
		allowed := false
		for _, sc := range allowedCmd.Subcommands {
			if sc == subcommand {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("'%s %s' subcommand not allowed", prefix, subcommand)
		}
	}

	// Check for blocked arguments
	for _, blockedArg := range allowedCmd.BlockedArgs {
		for _, arg := range args {
			if strings.Contains(arg, blockedArg) {
				return fmt.Errorf("argument '%s' is blocked", blockedArg)
			}
		}
	}

	return nil
}

// ValidateTimeout checks if the timeout is within limits
func (v *SecurityValidator) ValidateTimeout(timeoutSec int) int {
	if timeoutSec <= 0 {
		return v.policy.DefaultTimeoutSec
	}
	if timeoutSec > v.policy.MaxTimeoutSec {
		return v.policy.MaxTimeoutSec
	}
	return timeoutSec
}
