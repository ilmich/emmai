package index

import (
	"encoding/json"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/ilmich/emmai/internal/client"
	"github.com/ilmich/emmai/internal/indexer"
)

const maxResults = 200

// NewQueryIndexTool returns the OpenAI tool definition for query_index.
func NewQueryIndexTool() client.Tool {
	return client.NewFunctionTool(
		"query_index",
		"Query the pre-built codebase index for files and symbols. Much faster than grep or glob for finding where functions, types, and structs are defined. query_type: 'files' lists indexed files; 'symbols' lists symbols (functions, types, etc.).",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query_type": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"files", "symbols"},
					"description": "What to query: 'files' for file list, 'symbols' for code symbols.",
				},
				"name": map[string]interface{}{
					"type":        "string",
					"description": "Optional substring filter on symbol or file name.",
				},
				"kind": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"func", "method", "type", "struct", "interface", "const", "var"},
					"description": "Optional kind filter (symbols only).",
				},
				"file": map[string]interface{}{
					"type":        "string",
					"description": "Optional file path filter. Supports glob patterns (e.g. 'internal/tools/**', '**/*_test.go'). Falls back to substring match when no glob metacharacters are present.",
				},
				"package": map[string]interface{}{
					"type":        "string",
					"description": "Optional package name filter (Go only).",
				},
				"language": map[string]interface{}{
					"type":        "string",
					"description": "Optional language filter for files query (e.g. 'go', 'python').",
				},
			},
			"required": []string{"query_type"},
		},
	)
}

// QueryExecutor handles query_index tool calls.
type QueryExecutor struct {
	ref *indexer.IndexRef
}

// NewQueryExecutor creates a QueryExecutor backed by a live IndexRef.
// The ref is read on each call so it always reflects the latest index.
func NewQueryExecutor(ref *indexer.IndexRef) *QueryExecutor {
	return &QueryExecutor{ref: ref}
}

type queryResponse struct {
	Success   bool        `json:"success"`
	QueryType string      `json:"query_type"`
	Results   interface{} `json:"results"`
	Total     int         `json:"total"`
	Truncated bool        `json:"truncated"`
	Error     string      `json:"error,omitempty"`
	Hint      string      `json:"hint,omitempty"`
}

// HandleQueryIndex is the tool handler registered with SimpleToolExecutor.
func (q *QueryExecutor) HandleQueryIndex(args map[string]interface{}) (string, error) {
	idx := q.ref.Get()
	if idx == nil {
		return marshal(queryResponse{
			Success: false,
			Error:   "index not available",
			Hint:    "The codebase index could not be built. Use glob_files or search_files instead.",
		})
	}

	queryType, _ := args["query_type"].(string)
	nameFilter, _ := args["name"].(string)
	kindFilter, _ := args["kind"].(string)
	fileFilter, _ := args["file"].(string)
	pkgFilter, _ := args["package"].(string)
	langFilter, _ := args["language"].(string)

	switch queryType {
	case "files":
		return q.queryFiles(idx, nameFilter, fileFilter, langFilter)
	case "symbols":
		return q.querySymbols(idx, nameFilter, kindFilter, fileFilter, pkgFilter)
	default:
		return marshal(queryResponse{
			Success: false,
			Error:   "unknown query_type: " + queryType,
			Hint:    "Use 'files' or 'symbols'.",
		})
	}
}

func (q *QueryExecutor) queryFiles(idx *indexer.Index, name, file, lang string) (string, error) {
	var results []indexer.FileEntry
	for _, f := range idx.Files {
		if name != "" && !containsCI(f.Path, name) {
			continue
		}
		if file != "" && !matchPath(file, f.Path) {
			continue
		}
		if lang != "" && !strings.EqualFold(f.Language, lang) {
			continue
		}
		results = append(results, f)
		if len(results) >= maxResults {
			break
		}
	}
	truncated := len(results) == maxResults && len(idx.Files) > maxResults
	return marshal(queryResponse{
		Success:   true,
		QueryType: "files",
		Results:   results,
		Total:     len(results),
		Truncated: truncated,
	})
}

func (q *QueryExecutor) querySymbols(idx *indexer.Index, name, kind, file, pkg string) (string, error) {
	var results []indexer.Symbol
	for _, s := range idx.Symbols {
		if name != "" && !containsCI(s.Name, name) {
			continue
		}
		if kind != "" && s.Kind != kind {
			continue
		}
		if file != "" && !matchPath(file, s.File) {
			continue
		}
		if pkg != "" && !containsCI(s.Package, pkg) {
			continue
		}
		results = append(results, s)
		if len(results) >= maxResults {
			break
		}
	}
	truncated := len(results) == maxResults && len(idx.Symbols) > maxResults
	return marshal(queryResponse{
		Success:   true,
		QueryType: "symbols",
		Results:   results,
		Total:     len(results),
		Truncated: truncated,
	})
}

func containsCI(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// matchPath matches a file path against a pattern.
// If the pattern contains glob metacharacters (* ? { [) it uses doublestar.Match,
// otherwise it falls back to case-insensitive substring match.
func matchPath(pattern, path string) bool {
	if strings.ContainsAny(pattern, "*?{[") {
		matched, err := doublestar.Match(pattern, path)
		return err == nil && matched
	}
	return containsCI(path, pattern)
}

func marshal(v interface{}) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
