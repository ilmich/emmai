package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

const maxFilesLimit = 50

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
