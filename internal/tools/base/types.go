package base

import "time"

// ToolResponse is the standard response envelope for all tools
type ToolResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
	ErrorType ErrorType   `json:"error_type,omitempty"`
	Hint      string      `json:"hint,omitempty"`
}

// ErrorType categorizes tool errors
type ErrorType string

const (
	ErrorValidation ErrorType = "validation"
	ErrorPermission ErrorType = "permission"
	ErrorNotFound   ErrorType = "not_found"
	ErrorExecution  ErrorType = "execution"
	ErrorTimeout    ErrorType = "timeout"
)

// ExecutionMetadata tracks tool execution details
type ExecutionMetadata struct {
	ExecutionTimeMs int64     `json:"execution_time_ms"`
	Timestamp       time.Time `json:"timestamp"`
}
