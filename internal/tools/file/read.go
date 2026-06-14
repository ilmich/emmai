package file

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/ilmich/emmai/internal/client"
)

const (
	maxReadFileSize = 1024 * 1024 // 1MB
)

// Line represents a single line with hash for identification
type Line struct {
	Number  int    `json:"num"`
	Hash    string `json:"hash"`
	Content string `json:"content"`
}

// ReadResponse is the response from read_file tool
type ReadResponse struct {
	Success      bool   `json:"success"`
	FilePath     string `json:"file_path"`
	Size         int64  `json:"size"`
	LastModified string `json:"last_modified,omitempty"` // RFC3339 format
	LineCount    int    `json:"line_count,omitempty"`
	Lines        []Line `json:"lines,omitempty"`
	Error        string `json:"error,omitempty"`
	Hint         string `json:"hint,omitempty"`
}

// ReadExecutor handles file reading operations
type ReadExecutor struct {
	workingDir string
}

// NewReadExecutor creates a new read executor
func NewReadExecutor(workingDir string) *ReadExecutor {
	return &ReadExecutor{
		workingDir: workingDir,
	}
}

// NewReadFileTool returns the read_file tool definition
func NewReadFileTool() client.Tool {
	return client.NewFunctionTool(
		"read_file",
		"Read a file. Returns lines with 8-char hashes required by edit_file.",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"file_path": map[string]interface{}{
					"type":        "string",
					"description": "Relative path to the file",
				},
			},
			"required": []string{"file_path"},
		},
	)
}

// HandleReadFile handles the read_file tool invocation
func (e *ReadExecutor) HandleReadFile(args map[string]interface{}) (string, error) {
	// Extract file_path (required)
	filePath, ok := args["file_path"].(string)
	if !ok || filePath == "" {
		return e.errorResponse("", "file_path parameter is required", "Provide a valid file path relative to working directory")
	}

	// Clean the path
	filePath = filepath.Clean(filePath)

	// Convert to absolute path
	var absPath string
	if filepath.IsAbs(filePath) {
		absPath = filePath
	} else {
		absPath = filepath.Join(e.workingDir, filePath)
	}
	absPath = filepath.Clean(absPath)

	// Security check: Ensure file path is within working directory
	relPath, err := filepath.Rel(e.workingDir, absPath)
	if err != nil {
		return e.errorResponse(filePath, "invalid file path", "Use relative paths within the working directory")
	}
	if strings.HasPrefix(relPath, "..") || strings.HasPrefix(relPath, string(filepath.Separator)) {
		return e.errorResponse(filePath, "path traversal not allowed", "File path must be within working directory")
	}

	// Check if file exists and get info
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return e.errorResponse(relPath, "file not found", "Verify the file path and ensure the file exists")
		}
		return e.errorResponse(relPath, fmt.Sprintf("failed to access file: %s", err.Error()), "Check file permissions")
	}

	// Check if it's a regular file (not directory)
	if info.IsDir() {
		return e.errorResponse(relPath, "path is a directory, not a file", "Specify a file path, not a directory")
	}

	// Check file size
	if info.Size() > maxReadFileSize {
		sizeInMB := float64(info.Size()) / (1024 * 1024)
		return e.errorResponse(relPath, fmt.Sprintf("file too large (%.1fMB, max 1MB)", sizeInMB), "File exceeds size limit. Consider reading in chunks or processing externally")
	}

	// Read file contents
	contents, err := os.ReadFile(absPath)
	if err != nil {
		return e.errorResponse(relPath, fmt.Sprintf("failed to read file: %s", err.Error()), "Check file permissions and try again")
	}

	// Validate UTF-8
	if !utf8.Valid(contents) {
		return e.errorResponse(relPath, "file contains non-UTF-8 data (likely a binary file)", "Binary files cannot be read as text")
	}

	// Normalize line endings and split into lines
	content := strings.ReplaceAll(string(contents), "\r\n", "\n")
	lines := strings.Split(content, "\n")

	// Build lines with hashes
	lineStructs := make([]Line, len(lines))
	for i, lineContent := range lines {
		lineStructs[i] = Line{
			Number:  i + 1,
			Hash:    hashLine(lineContent),
			Content: lineContent,
		}
	}

	// Build success response
	return e.successResponseHashlines(relPath, info.Size(), info.ModTime().Format(time.RFC3339), len(lines), lineStructs)
}

// errorResponse creates an error response
func (e *ReadExecutor) errorResponse(filePath, errorMsg, hint string) (string, error) {
	response := ReadResponse{
		Success:  false,
		FilePath: filePath,
		Size:     0,
		Error:    errorMsg,
		Hint:     hint,
	}

	jsonResponse, err := json.Marshal(response)
	if err != nil {
		return "", fmt.Errorf("failed to encode error response: %w", err)
	}

	return string(jsonResponse), nil
}

// successResponseHashlines creates a success response with hashlines
func (e *ReadExecutor) successResponseHashlines(filePath string, size int64, lastModified string, lineCount int, lines []Line) (string, error) {
	response := ReadResponse{
		Success:      true,
		FilePath:     filePath,
		Size:         size,
		LastModified: lastModified,
		LineCount:    lineCount,
		Lines:        lines,
	}

	jsonResponse, err := json.Marshal(response)
	if err != nil {
		return "", fmt.Errorf("failed to encode response: %w", err)
	}

	return string(jsonResponse), nil
}

// hashLine generates an 8-character hash for a line
func hashLine(content string) string {
	h := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", h[:4])
}
