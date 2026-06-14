package file

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/ilmich/emmai/internal/client"
)

// DeleteResponse is the response from delete_file tool
type DeleteResponse struct {
	Success  bool   `json:"success"`
	FilePath string `json:"file_path"`
	Error    string `json:"error,omitempty"`
	Hint     string `json:"hint,omitempty"`
}

// DeleteExecutor handles file deletion operations
type DeleteExecutor struct {
	workingDir string
}

// NewDeleteExecutor creates a new delete executor
func NewDeleteExecutor(workingDir string) *DeleteExecutor {
	return &DeleteExecutor{workingDir: workingDir}
}

// NewDeleteFileTool returns the delete_file tool definition
func NewDeleteFileTool() client.Tool {
	return client.NewFunctionTool(
		"delete_file",
		"Delete a file from the filesystem.",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"file_path": map[string]interface{}{
					"type":        "string",
					"description": "Relative path to the file to delete",
				},
			},
			"required": []string{"file_path"},
		},
	)
}

// HandleDeleteFile handles the delete_file tool invocation
func (e *DeleteExecutor) HandleDeleteFile(args map[string]interface{}) (string, error) {
	filePath, ok := args["file_path"].(string)
	if !ok || filePath == "" {
		return e.respond(DeleteResponse{
			Success: false,
			Error:   "file_path is required",
			Hint:    "Provide a valid file path relative to the working directory",
		})
	}

	filePath = filepath.Clean(filePath)
	absPath := filepath.Join(e.workingDir, filePath)

	// Path traversal check
	relPath, err := filepath.Rel(e.workingDir, absPath)
	if err != nil || strings.HasPrefix(relPath, "..") {
		return e.respond(DeleteResponse{
			Success:  false,
			FilePath: filePath,
			Error:    "file_path must be within the working directory",
			Hint:     "Use a relative path without '..'",
		})
	}

	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return e.respond(DeleteResponse{
				Success:  false,
				FilePath: filePath,
				Error:    "file not found",
				Hint:     "Check the file path",
			})
		}
		return e.respond(DeleteResponse{
			Success:  false,
			FilePath: filePath,
			Error:    err.Error(),
		})
	}

	if info.IsDir() {
		return e.respond(DeleteResponse{
			Success:  false,
			FilePath: filePath,
			Error:    "path is a directory, not a file",
			Hint:     "delete_file only removes files, not directories",
		})
	}

	if err := os.Remove(absPath); err != nil {
		return e.respond(DeleteResponse{
			Success:  false,
			FilePath: filePath,
			Error:    err.Error(),
		})
	}

	return e.respond(DeleteResponse{
		Success:  true,
		FilePath: filePath,
	})
}

func (e *DeleteExecutor) respond(r DeleteResponse) (string, error) {
	b, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
