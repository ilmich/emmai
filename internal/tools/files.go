package tools

import "github.com/ilmich/emmai/internal/client"

// NewFileTools returns all file operation tools
func NewFileTools() []client.Tool {
	return []client.Tool{
		client.NewFunctionTool(
			"list_files",
			"Find files matching a glob pattern. Supports ** for recursive search. Examples: *.go, **/*.go, internal/**/*_test.go",
			map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"pattern": map[string]interface{}{
						"type":        "string",
						"description": "Glob pattern to match files (supports ** for recursive matching)",
					},
					"base_dir": map[string]interface{}{
						"type":        "string",
						"description": "Base directory to search from (optional, defaults to current working directory)",
					},
				},
				"required": []string{"pattern"},
			},
		),
	}
}
