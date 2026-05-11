package client

import "encoding/json"

// Tool represents a tool/function that can be called by the AI
type Tool struct {
	Type     string       `json:"type"`     // "function"
	Function FunctionDef  `json:"function"` // Function definition
}

// FunctionDef defines a function that can be called
type FunctionDef struct {
	Name        string                 `json:"name"`                  // Function name
	Description string                 `json:"description,omitempty"` // What the function does
	Parameters  map[string]interface{} `json:"parameters,omitempty"`  // JSON Schema for parameters
	Strict      bool                   `json:"strict,omitempty"`      // Whether to use strict schema adherence
}

// ToolChoice specifies how the model should use tools
// Can be "none", "auto", "required", or {"type": "function", "function": {"name": "my_function"}}
type ToolChoice interface{}

// ToolRegistry manages available tools
type ToolRegistry struct {
	tools map[string]Tool
}

// NewToolRegistry creates a new tool registry
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]Tool),
	}
}

// RegisterTool adds a tool to the registry
func (r *ToolRegistry) RegisterTool(tool Tool) {
	r.tools[tool.Function.Name] = tool
}

// GetTool retrieves a tool by name
func (r *ToolRegistry) GetTool(name string) (Tool, bool) {
	tool, exists := r.tools[name]
	return tool, exists
}

// GetAllTools returns all registered tools
func (r *ToolRegistry) GetAllTools() []Tool {
	tools := make([]Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}
	return tools
}

// HasTools returns true if any tools are registered
func (r *ToolRegistry) HasTools() bool {
	return len(r.tools) > 0
}

// NewFunctionTool creates a new function tool with the given parameters
func NewFunctionTool(name, description string, parameters map[string]interface{}) Tool {
	return Tool{
		Type: "function",
		Function: FunctionDef{
			Name:        name,
			Description: description,
			Parameters:  parameters,
		},
	}
}

// NewStrictFunctionTool creates a new function tool with strict schema adherence
func NewStrictFunctionTool(name, description string, parameters map[string]interface{}) Tool {
	return Tool{
		Type: "function",
		Function: FunctionDef{
			Name:        name,
			Description: description,
			Parameters:  parameters,
			Strict:      true,
		},
	}
}

// ToolExecutor is an interface for executing tools
type ToolExecutor interface {
	Execute(name string, arguments string) (string, error)
}

// SimpleToolExecutor provides a basic implementation of ToolExecutor
type SimpleToolExecutor struct {
	handlers map[string]func(args map[string]interface{}) (string, error)
}

// NewSimpleToolExecutor creates a new simple tool executor
func NewSimpleToolExecutor() *SimpleToolExecutor {
	return &SimpleToolExecutor{
		handlers: make(map[string]func(args map[string]interface{}) (string, error)),
	}
}

// RegisterHandler registers a handler function for a tool
func (e *SimpleToolExecutor) RegisterHandler(name string, handler func(args map[string]interface{}) (string, error)) {
	e.handlers[name] = handler
}

// Execute executes a tool with the given arguments
func (e *SimpleToolExecutor) Execute(name string, arguments string) (string, error) {
	handler, exists := e.handlers[name]
	if !exists {
		return "", nil // Tool not found - return empty result
	}

	// Parse arguments JSON
	var args map[string]interface{}
	if arguments != "" {
		if err := json.Unmarshal([]byte(arguments), &args); err != nil {
			return "", err
		}
	}

	// Execute the handler
	return handler(args)
}
