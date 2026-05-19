package execution

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ilmich/emmai/internal/client"
	"github.com/ilmich/emmai/internal/config"
	"github.com/ilmich/emmai/internal/phase"
)

// CommandResponse is the response from run_command tool
type CommandResponse struct {
	Success         bool   `json:"success"`
	Command         string `json:"command"`
	ExitCode        int    `json:"exit_code"`
	Stdout          string `json:"stdout"`
	Stderr          string `json:"stderr"`
	ExecutionTimeMs int64  `json:"execution_time_ms"`
	WorkingDir      string `json:"working_dir"`
	TimedOut        bool   `json:"timed_out"`
	Error           string `json:"error,omitempty"`
	Reason          string `json:"reason,omitempty"`
}

// CommandExecutor handles command execution
type CommandExecutor struct {
	workingDir     string
	securityPolicy *config.CommandExecutionPolicy
	validator      *SecurityValidator
	phaseManager   *phase.Manager
}

// NewCommandExecutor creates a new command executor
func NewCommandExecutor(workingDir string, policy *config.CommandExecutionPolicy, phaseManager *phase.Manager) *CommandExecutor {
	return &CommandExecutor{
		workingDir:     workingDir,
		securityPolicy: policy,
		validator:      NewSecurityValidator(policy),
		phaseManager:   phaseManager,
	}
}

// NewRunCommandTool returns the run_command tool definition
func NewRunCommandTool() client.Tool {
	return client.NewFunctionTool(
		"run_command",
		"Execute a shell command (tests, builds, linters, git). Commands are restricted by security policy and phase.",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"command": map[string]interface{}{
					"type":        "string",
					"description": "Shell command to execute (must be whitelisted)",
				},
				"working_dir": map[string]interface{}{
					"type":        "string",
					"description": "Working directory for command (optional, relative to project root)",
				},
				"timeout_seconds": map[string]interface{}{
					"type":        "integer",
					"description": "Max execution time in seconds (default: 30, max: 120)",
					"minimum":     1,
					"maximum":     120,
				},
			},
			"required": []string{"command"},
		},
	)
}

// HandleRunCommand handles the run_command tool invocation
func (e *CommandExecutor) HandleRunCommand(args map[string]interface{}) (string, error) {
	startTime := time.Now()

	// Extract command
	cmd, ok := args["command"].(string)
	if !ok || cmd == "" {
		return e.errorResponse("", "command is required", "blocked")
	}

	// Get current phase
	currentPhase := e.phaseManager.GetCurrentPhase()

	// Validate command against security policy
	if err := e.validator.ValidateCommand(cmd, currentPhase); err != nil {
		return e.securityErrorResponse(cmd, err)
	}

	// Extract working directory
	workingDir := e.workingDir
	if dir, ok := args["working_dir"].(string); ok && dir != "" {
		workingDir = filepath.Join(e.workingDir, filepath.Clean(dir))
		
		// Security check: ensure within project directory
		relPath, err := filepath.Rel(e.workingDir, workingDir)
		if err != nil || strings.HasPrefix(relPath, "..") {
			return e.errorResponse(cmd, "working_dir must be within project directory", "blocked")
		}
	}

	// Extract and validate timeout
	timeoutSec := e.securityPolicy.DefaultTimeoutSec
	if t, ok := args["timeout_seconds"].(float64); ok {
		timeoutSec = e.validator.ValidateTimeout(int(t))
	}

	// Execute command
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSec)*time.Second)
	defer cancel()

	// Split command safely (no shell injection)
	parts := strings.Fields(cmd)
	execCmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	execCmd.Dir = workingDir

	// Capture output
	var stdout, stderr bytes.Buffer
	execCmd.Stdout = &stdout
	execCmd.Stderr = &stderr

	// Run command
	err := execCmd.Run()

	executionTime := time.Since(startTime)
	timedOut := ctx.Err() == context.DeadlineExceeded

	// Build response
	exitCode := 0
	if execCmd.ProcessState != nil {
		exitCode = execCmd.ProcessState.ExitCode()
	}

	response := CommandResponse{
		Success:         err == nil && !timedOut,
		Command:         cmd,
		ExitCode:        exitCode,
		Stdout:          stdout.String(),
		Stderr:          stderr.String(),
		ExecutionTimeMs: executionTime.Milliseconds(),
		WorkingDir:      workingDir,
		TimedOut:        timedOut,
	}

	// Handle errors
	if timedOut {
		response.Error = fmt.Sprintf("timeout after %ds", timeoutSec)
	} else if err != nil {
		response.Error = fmt.Sprintf("exit code %d", exitCode)
	}

	// Truncate output if too large
	if len(response.Stdout) > e.securityPolicy.MaxOutputSizeBytes {
		response.Stdout = response.Stdout[:e.securityPolicy.MaxOutputSizeBytes] + "\n...(truncated)"
	}
	if len(response.Stderr) > e.securityPolicy.MaxOutputSizeBytes {
		response.Stderr = response.Stderr[:e.securityPolicy.MaxOutputSizeBytes] + "\n...(truncated)"
	}

	jsonResponse, _ := json.Marshal(response)
	return string(jsonResponse), nil
}

func (e *CommandExecutor) securityErrorResponse(cmd string, validationErr error) (string, error) {
	response := CommandResponse{
		Success: false,
		Command: cmd,
		Error:   "command blocked by security policy",
		Reason:  validationErr.Error(),
	}
	jsonResponse, _ := json.Marshal(response)
	return string(jsonResponse), nil
}

func (e *CommandExecutor) errorResponse(cmd, error, reason string) (string, error) {
	response := CommandResponse{
		Success: false,
		Command: cmd,
		Error:   error,
		Reason:  reason,
	}
	jsonResponse, _ := json.Marshal(response)
	return string(jsonResponse), nil
}
