package base

import "fmt"

// ToolError represents a structured tool error
type ToolError struct {
	Type    ErrorType
	Message string
	Hint    string
}

func (e *ToolError) Error() string {
	return e.Message
}

// NewValidationError creates a validation error
func NewValidationError(field, reason string) *ToolError {
	return &ToolError{
		Type:    ErrorValidation,
		Message: fmt.Sprintf("invalid %s: %s", field, reason),
		Hint:    "Check parameter format and requirements",
	}
}

// NewPermissionError creates a permission error
func NewPermissionError(resource string) *ToolError {
	return &ToolError{
		Type:    ErrorPermission,
		Message: fmt.Sprintf("access denied: %s", resource),
		Hint:    "Check security policy and path restrictions",
	}
}

// NewNotFoundError creates a not found error
func NewNotFoundError(resource string) *ToolError {
	return &ToolError{
		Type:    ErrorNotFound,
		Message: fmt.Sprintf("not found: %s", resource),
		Hint:    "Verify path and file existence",
	}
}

// NewExecutionError creates an execution error
func NewExecutionError(message string) *ToolError {
	return &ToolError{
		Type:    ErrorExecution,
		Message: message,
		Hint:    "Check command syntax and permissions",
	}
}

// NewTimeoutError creates a timeout error
func NewTimeoutError(operation string, timeoutSec int) *ToolError {
	return &ToolError{
		Type:    ErrorTimeout,
		Message: fmt.Sprintf("%s timed out after %ds", operation, timeoutSec),
		Hint:    "Try with longer timeout or simpler operation",
	}
}
