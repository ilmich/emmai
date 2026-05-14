package tools

import "github.com/ilmich/emmai/internal/client"

// NewFileTools returns all file operation tools
func NewFileTools() []client.Tool {
	return []client.Tool{
		client.NewFunctionTool(
			"list_files",
			"Find files matching a glob pattern in the codebase for code exploration and understanding project structure.",
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
		client.NewFunctionTool(
			"read_file",
			"Read content of a file for inspection, analysis, and understanding.",
			map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"file_path": map[string]interface{}{
						"type":        "string",
						"description": "Relative path to the file to read (from working directory)",
					},
				},
				"required": []string{"file_path"},
			},
		),
		client.NewFunctionTool(
			"write_file",
			"Write content to a file, creating it if doesn't exist.",
			map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"file_path": map[string]interface{}{
						"type":        "string",
						"description": "Relative path to the file to write (from working directory)",
					},
					"content": map[string]interface{}{
						"type":        "string",
						"description": "UTF-8 text content to write to the file",
					},
				},
				"required": []string{"file_path", "content"},
			},
		),
	}
}
