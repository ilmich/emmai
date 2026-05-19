package file

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/ilmich/emmai/internal/client"
)

const (
	defaultMaxResults   = 100
	maxContextLines     = 10
	searchTimeoutSec    = 10
	maxSearchFileSizeMB = 5
)

// SearchOptions contains search configuration
type SearchOptions struct {
	Pattern       string
	Path          string
	Regex         bool
	CaseSensitive bool
	ContextLines  int
	MaxResults    int
}

// SearchMatch represents a single search match
type SearchMatch struct {
	FilePath      string   `json:"file_path"`
	LineNumber    int      `json:"line_number"`
	LineContent   string   `json:"line_content"`
	ContextBefore []string `json:"context_before"`
	ContextAfter  []string `json:"context_after"`
}

// SearchResponse is the response from search_files tool
type SearchResponse struct {
	Success       bool          `json:"success"`
	Pattern       string        `json:"pattern"`
	Matches       []SearchMatch `json:"matches"`
	TotalMatches  int           `json:"total_matches"`
	FilesSearched int           `json:"files_searched"`
	SearchTimeMs  int64         `json:"search_time_ms"`
	Truncated     bool          `json:"truncated"`
	Error         string        `json:"error,omitempty"`
	Hint          string        `json:"hint,omitempty"`
}

// SearchExecutor handles search operations
type SearchExecutor struct {
	workingDir string
}

// NewSearchExecutor creates a new search executor
func NewSearchExecutor(workingDir string) *SearchExecutor {
	return &SearchExecutor{
		workingDir: workingDir,
	}
}

// NewSearchFilesTool returns the search_files tool definition
func NewSearchFilesTool() client.Tool {
	return client.NewFunctionTool(
		"search_files",
		"Search for text patterns in files using regex. Returns matches with context lines.",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"pattern": map[string]interface{}{
					"type":        "string",
					"description": "Search pattern (supports regex if regex=true)",
				},
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Path or glob pattern to search in (e.g., '**/*.go', 'internal/'). Defaults to '**/*'",
				},
				"regex": map[string]interface{}{
					"type":        "boolean",
					"description": "Treat pattern as regex (default: false for plain text search)",
				},
				"case_sensitive": map[string]interface{}{
					"type":        "boolean",
					"description": "Case sensitive search (default: false)",
				},
				"context_lines": map[string]interface{}{
					"type":        "integer",
					"description": "Number of context lines before/after match (default: 2, max: 10)",
					"minimum":     0,
					"maximum":     maxContextLines,
				},
				"max_results": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of results to return (default: 100, max: 500)",
					"minimum":     1,
					"maximum":     500,
				},
			},
			"required": []string{"pattern"},
		},
	)
}

// HandleSearchFiles handles the search_files tool invocation
func (e *SearchExecutor) HandleSearchFiles(args map[string]interface{}) (string, error) {
	startTime := time.Now()

	// Extract and validate parameters
	opts, err := e.extractSearchOptions(args)
	if err != nil {
		return e.errorResponse("", err.Error(), "Check parameter format")
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), searchTimeoutSec*time.Second)
	defer cancel()

	// Perform search
	matches, filesSearched, err := e.search(ctx, opts)
	
	executionTime := time.Since(startTime)
	timedOut := ctx.Err() == context.DeadlineExceeded

	// Build response
	response := SearchResponse{
		Success:       err == nil && !timedOut,
		Pattern:       opts.Pattern,
		Matches:       matches,
		TotalMatches:  len(matches),
		FilesSearched: filesSearched,
		SearchTimeMs:  executionTime.Milliseconds(),
		Truncated:     len(matches) >= opts.MaxResults,
	}

	if timedOut {
		response.Error = fmt.Sprintf("search timed out after %ds", searchTimeoutSec)
		response.Hint = "Try narrowing search path or pattern"
	} else if err != nil {
		response.Error = err.Error()
	}

	if len(matches) == 0 && err == nil && !timedOut {
		response.Hint = "No matches found. Try broadening pattern or checking path"
	}

	jsonResponse, _ := json.Marshal(response)
	return string(jsonResponse), nil
}

func (e *SearchExecutor) extractSearchOptions(args map[string]interface{}) (SearchOptions, error) {
	opts := SearchOptions{
		Path:          "**/*",
		Regex:         false,
		CaseSensitive: false,
		ContextLines:  2,
		MaxResults:    defaultMaxResults,
	}

	// Extract pattern (required)
	pattern, ok := args["pattern"].(string)
	if !ok || pattern == "" {
		return opts, fmt.Errorf("pattern is required")
	}
	opts.Pattern = pattern

	// Extract optional parameters
	if path, ok := args["path"].(string); ok && path != "" {
		opts.Path = path
	}

	if regex, ok := args["regex"].(bool); ok {
		opts.Regex = regex
	}

	if caseSensitive, ok := args["case_sensitive"].(bool); ok {
		opts.CaseSensitive = caseSensitive
	}

	if contextLines, ok := args["context_lines"].(float64); ok {
		lines := int(contextLines)
		if lines < 0 {
			lines = 0
		} else if lines > maxContextLines {
			lines = maxContextLines
		}
		opts.ContextLines = lines
	}

	if maxResults, ok := args["max_results"].(float64); ok {
		max := int(maxResults)
		if max < 1 {
			max = 1
		} else if max > 500 {
			max = 500
		}
		opts.MaxResults = max
	}

	return opts, nil
}

func (e *SearchExecutor) search(ctx context.Context, opts SearchOptions) ([]SearchMatch, int, error) {
	// Compile regex pattern
	pattern, err := e.compilePattern(opts.Pattern, opts.Regex, opts.CaseSensitive)
	if err != nil {
		return nil, 0, fmt.Errorf("invalid pattern: %w", err)
	}

	// Find files matching path pattern
	fsys := os.DirFS(e.workingDir)
	files, err := doublestar.Glob(fsys, opts.Path)
	if err != nil {
		return nil, 0, fmt.Errorf("invalid path pattern: %w", err)
	}

	var matches []SearchMatch
	filesSearched := 0

	for _, file := range files {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return matches, filesSearched, ctx.Err()
		default:
		}

		// Stop if we have enough results
		if len(matches) >= opts.MaxResults {
			break
		}

		absPath := filepath.Join(e.workingDir, file)

		// Check if it's a file
		info, err := os.Stat(absPath)
		if err != nil || info.IsDir() {
			continue
		}

		// Skip large files
		if info.Size() > maxSearchFileSizeMB*1024*1024 {
			continue
		}

		// Search file
		fileMatches, err := e.searchFile(absPath, file, pattern, opts.ContextLines)
		if err != nil {
			// Skip files that can't be read or are binary
			continue
		}

		filesSearched++

		// Add matches up to max limit
		for _, match := range fileMatches {
			if len(matches) >= opts.MaxResults {
				break
			}
			matches = append(matches, match)
		}
	}

	return matches, filesSearched, nil
}

func (e *SearchExecutor) searchFile(absPath, relPath string, pattern *regexp.Regexp, contextLines int) ([]SearchMatch, error) {
	// Read file
	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil, err
	}

	// Validate UTF-8
	if !utf8.Valid(content) {
		return nil, fmt.Errorf("binary file")
	}

	// Split into lines
	lines := strings.Split(string(content), "\n")

	var matches []SearchMatch

	// Search each line
	for i, line := range lines {
		if pattern.MatchString(line) {
			match := SearchMatch{
				FilePath:      relPath,
				LineNumber:    i + 1,
				LineContent:   line,
				ContextBefore: e.getContextLines(lines, i, -contextLines, 0),
				ContextAfter:  e.getContextLines(lines, i, 1, contextLines+1),
			}
			matches = append(matches, match)
		}
	}

	return matches, nil
}

func (e *SearchExecutor) getContextLines(lines []string, currentLine, startOffset, endOffset int) []string {
	var context []string

	start := currentLine + startOffset
	end := currentLine + endOffset

	if start < 0 {
		start = 0
	}
	if end > len(lines) {
		end = len(lines)
	}

	for i := start; i < end; i++ {
		if i != currentLine {
			context = append(context, lines[i])
		}
	}

	return context
}

func (e *SearchExecutor) compilePattern(pattern string, regex, caseSensitive bool) (*regexp.Regexp, error) {
	// If not regex, escape special characters for literal search
	if !regex {
		pattern = regexp.QuoteMeta(pattern)
	}

	// Add case-insensitive flag if needed
	if !caseSensitive {
		pattern = "(?i)" + pattern
	}

	return regexp.Compile(pattern)
}

func (e *SearchExecutor) errorResponse(pattern, error, hint string) (string, error) {
	response := SearchResponse{
		Success: false,
		Pattern: pattern,
		Error:   error,
		Hint:    hint,
	}
	jsonResponse, _ := json.Marshal(response)
	return string(jsonResponse), nil
}
