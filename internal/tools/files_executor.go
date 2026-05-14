package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/bmatcuk/doublestar/v4"
)

const maxFilesLimit = 50
const maxFileSize = 1024 * 1024 // 1MB

// isHiddenPath checks if a path contains any hidden component (starts with .)
func isHiddenPath(path string) bool {
	parts := strings.Split(filepath.ToSlash(path), "/")
	for _, part := range parts {
		if strings.HasPrefix(part, ".") && part != "." && part != ".." {
			return true
		}
	}
	return false
}

// FileContent represents the response from read_file tool
type FileContent struct {
	FilePath     string `json:"file_path"`
	Size         int64  `json:"size"`
	LastModified string `json:"last_modified"` // RFC3339 format
	Contents     string `json:"contents"`
}

// WriteFileResponse represents the response from write_file tool
type WriteFileResponse struct {
	FilePath     string `json:"file_path"`
	Size         int64  `json:"size"`
	Operation    string `json:"operation"`     // "created" or "overwritten"
	LastModified string `json:"last_modified"` // RFC3339 format
}

// FileToolsExecutor handles file operation tool execution
type FileToolsExecutor struct {
	workingDir string
}

// NewFileToolsExecutor creates a new file tools executor
func NewFileToolsExecutor() *FileToolsExecutor {
	wd, _ := os.Getwd()
	return &FileToolsExecutor{
		workingDir: wd,
	}
}

// RegisterHandlers registers all file tool handlers with the executor
func (e *FileToolsExecutor) RegisterHandlers(executor interface {
	RegisterHandler(name string, handler func(args map[string]interface{}) (string, error))
}) {
	executor.RegisterHandler("list_files", e.handleListFiles)
	executor.RegisterHandler("read_file", e.handleReadFile)
	executor.RegisterHandler("write_file", e.handleWriteFile)
}

// handleListFiles finds files matching a glob pattern
func (e *FileToolsExecutor) handleListFiles(args map[string]interface{}) (string, error) {
	// Extract pattern (required)
	pattern, ok := args["pattern"].(string)
	if !ok || pattern == "" {
		return "", fmt.Errorf("pattern is required and must be a non-empty string")
	}

	// Extract optional base directory
	baseDir := e.workingDir
	if dir, ok := args["base_dir"].(string); ok && dir != "" {
		// Convert to absolute path
		var absDir string
		if filepath.IsAbs(dir) {
			absDir = dir
		} else {
			absDir = filepath.Join(e.workingDir, dir)
		}
		absDir = filepath.Clean(absDir)

		// Security check: Ensure base_dir is within working directory
		relPath, err := filepath.Rel(e.workingDir, absDir)
		if err != nil {
			return "", fmt.Errorf("invalid base directory: %w", err)
		}
		if strings.HasPrefix(relPath, "..") || strings.HasPrefix(relPath, string(filepath.Separator)) {
			return "", fmt.Errorf("base_dir must be within working directory")
		}

		// Verify directory exists
		info, err := os.Stat(absDir)
		if err != nil {
			return "", fmt.Errorf("base_dir does not exist: %w", err)
		}
		if !info.IsDir() {
			return "", fmt.Errorf("base_dir must be a directory")
		}

		baseDir = absDir
	}

	// Find matching files using doublestar
	fsys := os.DirFS(baseDir)
	matches, err := doublestar.Glob(fsys, pattern)
	if err != nil {
		return "", fmt.Errorf("invalid glob pattern: %w", err)
	}

	// Filter to only files (not directories) and build result list
	var results []string
	for _, match := range matches {
		if len(results) >= maxFilesLimit {
			break
		}

		// Skip hidden files/directories
		if isHiddenPath(match) {
			continue
		}

		// Construct absolute path
		absPath := filepath.Join(baseDir, match)

		// Check if it's a file (not directory)
		info, err := os.Stat(absPath)
		if err != nil {
			continue // Skip inaccessible files
		}
		if info.IsDir() {
			continue // Skip directories
		}

		// Convert to relative path from working directory
		relPath, err := filepath.Rel(e.workingDir, absPath)
		if err != nil {
			// Fallback to absolute path if relative conversion fails
			relPath = absPath
		}

		results = append(results, relPath)
	}

	// Return as JSON array
	response, err := json.Marshal(results)
	if err != nil {
		return "", fmt.Errorf("failed to encode results: %w", err)
	}

	return string(response), nil
}

// handleReadFile reads the contents of a file
func (e *FileToolsExecutor) handleReadFile(args map[string]interface{}) (string, error) {
	// Extract file_path (required)
	filePath, ok := args["file_path"].(string)
	if !ok || filePath == "" {
		return "", fmt.Errorf("file_path is required and must be a non-empty string")
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
		return "", fmt.Errorf("invalid file path: %w", err)
	}
	if strings.HasPrefix(relPath, "..") || strings.HasPrefix(relPath, string(filepath.Separator)) {
		return "", fmt.Errorf("file_path must be within working directory (attempted to access: %s)", filePath)
	}

	// Check if file exists and get info
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("file not found: %s", relPath)
		}
		return "", fmt.Errorf("failed to access file: %w", err)
	}

	// Check if it's a regular file (not directory)
	if info.IsDir() {
		return "", fmt.Errorf("path is a directory, not a file: %s", relPath)
	}

	// Check file size
	if info.Size() > maxFileSize {
		return "", fmt.Errorf("file size (%d bytes) exceeds maximum limit of %d bytes (1MB)", info.Size(), maxFileSize)
	}

	// Read file contents
	contents, err := os.ReadFile(absPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// Validate UTF-8
	if !utf8.Valid(contents) {
		return "", fmt.Errorf("file contains invalid UTF-8 data (likely a binary file)")
	}

	// Build response
	fileContent := FileContent{
		FilePath:     relPath,
		Size:         info.Size(),
		LastModified: info.ModTime().Format(time.RFC3339),
		Contents:     string(contents),
	}

	// Marshal to JSON
	response, err := json.Marshal(fileContent)
	if err != nil {
		return "", fmt.Errorf("failed to encode response: %w", err)
	}

	return string(response), nil
}

// handleWriteFile writes content to a file
func (e *FileToolsExecutor) handleWriteFile(args map[string]interface{}) (string, error) {
	// Extract file_path (required)
	filePath, ok := args["file_path"].(string)
	if !ok || filePath == "" {
		return "", fmt.Errorf("file_path is required and must be a non-empty string")
	}

	// Extract content (required)
	content, ok := args["content"].(string)
	if !ok {
		return "", fmt.Errorf("content is required and must be a string")
	}

	// Validate content is not empty
	if len(content) == 0 {
		return "", fmt.Errorf("content cannot be empty")
	}

	// Validate UTF-8
	if !utf8.ValidString(content) {
		return "", fmt.Errorf("content must be valid UTF-8 text")
	}

	// Check content size
	contentBytes := []byte(content)
	if len(contentBytes) > maxFileSize {
		return "", fmt.Errorf("content size (%d bytes) exceeds maximum limit of %d bytes (1MB)", len(contentBytes), maxFileSize)
	}

	// Clean the path
	filePath = filepath.Clean(filePath)

	// Block hidden files/directories
	if isHiddenPath(filePath) {
		return "", fmt.Errorf("cannot write to hidden files or directories (path: %s)", filePath)
	}

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
		return "", fmt.Errorf("invalid file path: %w", err)
	}
	if strings.HasPrefix(relPath, "..") || strings.HasPrefix(relPath, string(filepath.Separator)) {
		return "", fmt.Errorf("file_path must be within working directory (attempted to access: %s)", filePath)
	}

	// Check if file exists to determine operation
	operation := "created"
	if _, err := os.Stat(absPath); err == nil {
		operation = "overwritten"
	}

	// Create parent directories if they don't exist
	parentDir := filepath.Dir(absPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create parent directories: %w", err)
	}

	// Write file with 0644 permissions
	if err := os.WriteFile(absPath, contentBytes, 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	// Get file info for response
	info, err := os.Stat(absPath)
	if err != nil {
		return "", fmt.Errorf("failed to stat written file: %w", err)
	}

	// Build response
	writeResponse := WriteFileResponse{
		FilePath:     relPath,
		Size:         info.Size(),
		Operation:    operation,
		LastModified: info.ModTime().Format(time.RFC3339),
	}

	// Marshal to JSON
	jsonResponse, err := json.Marshal(writeResponse)
	if err != nil {
		return "", fmt.Errorf("failed to encode response: %w", err)
	}

	return string(jsonResponse), nil
}
