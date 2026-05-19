package file

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/ilmich/emmai/internal/client"
)

const (
	defaultMaxGlobResults = 1000
	maxGlobResults        = 10000
)

// GlobMatch represents a single file match
type GlobMatch struct {
	FilePath    string `json:"file_path"`
	IsDirectory bool   `json:"is_directory"`
	Size        int64  `json:"size"`
	ModTime     string `json:"mod_time"`
}

// GlobResponse is the response from glob_files tool
type GlobResponse struct {
	Success      bool        `json:"success"`
	Pattern      string      `json:"pattern"`
	Matches      []GlobMatch `json:"matches"`
	TotalMatches int         `json:"total_matches"`
	Truncated    bool        `json:"truncated"`
	Error        string      `json:"error,omitempty"`
	Hint         string      `json:"hint,omitempty"`
}

// GlobExecutor handles glob file operations
type GlobExecutor struct {
	workingDir string
}

// NewGlobExecutor creates a new glob executor
func NewGlobExecutor(workingDir string) *GlobExecutor {
	return &GlobExecutor{
		workingDir: workingDir,
	}
}

// NewGlobFilesTool returns the glob_files tool definition
func NewGlobFilesTool() client.Tool {
	return client.NewFunctionTool(
		"glob_files",
		"Find files by name using glob patterns. Returns list of matching file paths with metadata. Use this to discover files by name/extension, not for searching file content.",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"pattern": map[string]interface{}{
					"type":        "string",
					"description": "Glob pattern for file names (e.g., '**/*.html', 'src/**/*.{js,css}', '**/test_*.go')",
				},
				"max_results": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of results to return (default: 1000, max: 10000)",
					"default":     defaultMaxGlobResults,
					"minimum":     1,
					"maximum":     maxGlobResults,
				},
			},
			"required": []string{"pattern"},
		},
	)
}

// HandleGlobFiles handles the glob_files tool invocation
func (e *GlobExecutor) HandleGlobFiles(args map[string]interface{}) (string, error) {
	// Extract pattern (required)
	pattern, ok := args["pattern"].(string)
	if !ok || pattern == "" {
		return e.errorResponse("", "pattern parameter is required",
			"Provide a glob pattern like '**/*.html' or 'src/**/*.go'")
	}

	// Extract max_results (optional)
	maxResults := defaultMaxGlobResults
	if val, ok := args["max_results"].(float64); ok {
		maxResults = int(val)
		if maxResults < 1 {
			maxResults = 1
		}
		if maxResults > maxGlobResults {
			maxResults = maxGlobResults
		}
	}

	// Perform glob matching
	matches, truncated, err := e.globFiles(pattern, maxResults)

	// Build response
	response := GlobResponse{
		Success:      err == nil,
		Pattern:      pattern,
		Matches:      matches,
		TotalMatches: len(matches),
		Truncated:    truncated,
	}

	if err != nil {
		response.Error = err.Error()
		response.Hint = "Check pattern syntax or path accessibility"
	} else if len(matches) == 0 {
		response.Hint = "No files found. Try broader pattern or check directory"
	}

	jsonResponse, _ := json.Marshal(response)
	return string(jsonResponse), nil
}

func (e *GlobExecutor) globFiles(pattern string, maxResults int) ([]GlobMatch, bool, error) {
	// Use doublestar for glob matching (supports **)
	fsys := os.DirFS(e.workingDir)

	paths, err := doublestar.Glob(fsys, pattern)
	if err != nil {
		return nil, false, fmt.Errorf("glob pattern error: %w", err)
	}

	matches := make([]GlobMatch, 0, len(paths))
	truncated := false

	for _, path := range paths {
		if len(matches) >= maxResults {
			truncated = true
			break
		}

		// Get absolute path for stat
		absPath := filepath.Join(e.workingDir, path)

		info, err := os.Stat(absPath)
		if err != nil {
			// Skip files we can't stat
			continue
		}

		matches = append(matches, GlobMatch{
			FilePath:    path,
			IsDirectory: info.IsDir(),
			Size:        info.Size(),
			ModTime:     info.ModTime().Format(time.RFC3339),
		})
	}

	return matches, truncated, nil
}

func (e *GlobExecutor) errorResponse(pattern, error, hint string) (string, error) {
	response := GlobResponse{
		Success: false,
		Pattern: pattern,
		Error:   error,
		Hint:    hint,
	}
	jsonResponse, _ := json.Marshal(response)
	return string(jsonResponse), nil
}
