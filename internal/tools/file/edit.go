package file

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/ilmich/emmai/internal/client"
)

const (
	maxEditFileSize = 1024 * 1024 // 1MB
)

// EditType defines the type of edit operation
type EditType string

const (
	EditReplaceLines     EditType = "replace_lines"
	EditInsertAfterHash  EditType = "insert_after_hash"
	EditInsertBeforeHash EditType = "insert_before_hash"
	EditDeleteByHash     EditType = "delete_by_hash"
	EditCreateFile       EditType = "create_file"
)

// EditOperation represents a single edit operation
type EditOperation struct {
	Type       EditType `json:"type"`
	StartHash  string   `json:"start_hash,omitempty"`
	EndHash    string   `json:"end_hash,omitempty"`
	AfterHash  string   `json:"after_hash,omitempty"`
	BeforeHash string   `json:"before_hash,omitempty"`
	NewContent string   `json:"new_content,omitempty"`
	Content    string   `json:"content,omitempty"`
}

// EditDetail provides details about an edit operation result
type EditDetail struct {
	EditIndex    int      `json:"edit_index"`
	Type         EditType `json:"type"`
	Status       string   `json:"status"` // "success" or "failed"
	MatchesFound int      `json:"matches_found,omitempty"`
	Error        string   `json:"error,omitempty"`
}

// EditResponse is the response from edit_file tool
type EditResponse struct {
	Success        bool         `json:"success"`
	FilePath       string       `json:"file_path"`
	PreviewOnly    bool         `json:"preview_only,omitempty"`
	Diff           string       `json:"diff,omitempty"`
	EditsApplied   int          `json:"edits_applied"`
	EditsFailed    int          `json:"edits_failed"`
	FileSizeBefore int64        `json:"file_size_before"`
	FileSizeAfter  int64        `json:"file_size_after"`
	Details        []EditDetail `json:"details,omitempty"`
	Error          string       `json:"error,omitempty"`
	Hint           string       `json:"hint,omitempty"`
}

// EditExecutor handles file editing operations
type EditExecutor struct {
	workingDir string
}

// NewEditExecutor creates a new edit executor
func NewEditExecutor(workingDir string) *EditExecutor {
	return &EditExecutor{
		workingDir: workingDir,
	}
}

// NewEditFileTool returns the edit_file tool definition
func NewEditFileTool() client.Tool {
	return client.NewFunctionTool(
		"edit_file",
		"Edit a file using hash-based line addressing. Hashes come from read_file. Operations: replace_lines, insert_after_hash, insert_before_hash, delete_by_hash, create_file. Set preview_only=true to see a unified diff without writing the file.",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"file_path": map[string]interface{}{
					"type":        "string",
					"description": "Relative path to the file",
				},
				"edits": map[string]interface{}{
					"type":        "array",
					"description": "Edit operations to apply sequentially",
					"items": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"type": map[string]interface{}{
								"type": "string",
								"enum": []string{"replace_lines", "insert_after_hash", "insert_before_hash", "delete_by_hash", "create_file"},
							},
							"start_hash": map[string]interface{}{
								"type":        "string",
								"description": "Hash of first line to replace/delete",
							},
							"end_hash": map[string]interface{}{
								"type":        "string",
								"description": "Hash of last line to replace/delete (defaults to start_hash)",
							},
							"after_hash": map[string]interface{}{
								"type":        "string",
								"description": "Hash of line to insert after",
							},
							"before_hash": map[string]interface{}{
								"type":        "string",
								"description": "Hash of line to insert before",
							},
							"new_content": map[string]interface{}{
								"type":        "string",
								"description": "Replacement content for replace_lines",
							},
							"content": map[string]interface{}{
								"type":        "string",
								"description": "Content for insert or create_file operations",
							},
						},
						"required": []string{"type"},
					},
				},
				"preview_only": map[string]interface{}{
					"type":        "boolean",
					"description": "If true, compute and return a unified diff but do not write to disk. Use to verify changes before applying.",
				},
			},
			"required": []string{"file_path", "edits"},
		},
	)
}

// HandleEditFile handles the edit_file tool invocation
func (e *EditExecutor) HandleEditFile(args map[string]interface{}) (string, error) {
	// Extract file path
	filePath, ok := args["file_path"].(string)
	if !ok || filePath == "" {
		return e.errorResponse("", "file_path is required", "Provide valid file path")
	}

	previewOnly, _ := args["preview_only"].(bool)

	// Clean and validate path
	filePath = filepath.Clean(filePath)
	absPath := filepath.Join(e.workingDir, filePath)

	// Security check
	relPath, err := filepath.Rel(e.workingDir, absPath)
	if err != nil || strings.HasPrefix(relPath, "..") {
		return e.errorResponse(filePath, "file_path must be within working directory", "Use relative path")
	}

	// Extract edits early to check for create_file operation
	editsRaw, ok := args["edits"].([]interface{})
	if !ok || len(editsRaw) == 0 {
		return e.errorResponse(filePath, "edits array is required", "Provide at least one edit operation")
	}

	// Check if this is a create_file operation
	isCreateOperation := false
	if len(editsRaw) > 0 {
		if editMap, ok := editsRaw[0].(map[string]interface{}); ok {
			if editType, ok := editMap["type"].(string); ok && editType == "create_file" {
				isCreateOperation = true
			}
		}
	}

	// Handle create vs edit differently
	if isCreateOperation {
		return e.handleCreateFile(filePath, absPath, relPath, editsRaw)
	}

	// Check file exists (for edit operations)
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return e.errorResponse(filePath, "file not found. Use create_file operation to create new files", "Verify file path or use create_file")
		}
		return e.errorResponse(filePath, err.Error(), "Check file permissions")
	}

	if info.IsDir() {
		return e.errorResponse(filePath, "path is a directory", "Specify a file")
	}

	// Check file size
	if info.Size() > maxEditFileSize {
		return e.errorResponse(filePath, "file too large (max 1MB)", "Split into smaller files or handle manually")
	}

	sizeBefore := info.Size()

	// Read file content
	content, err := os.ReadFile(absPath)
	if err != nil {
		return e.errorResponse(filePath, "failed to read file", "Check permissions")
	}

	// Validate UTF-8
	if !utf8.Valid(content) {
		return e.errorResponse(filePath, "file contains invalid UTF-8 (binary file?)", "Binary files not supported")
	}

	originalContent := string(content)

	// Parse edit operations
	var edits []EditOperation
	for i, editRaw := range editsRaw {
		editMap, ok := editRaw.(map[string]interface{})
		if !ok {
			return e.errorResponse(filePath, fmt.Sprintf("invalid edit at index %d", i), "Check edit format")
		}

		edit, err := e.parseEditOperation(editMap)
		if err != nil {
			return e.errorResponse(filePath, fmt.Sprintf("edit %d: %s", i, err.Error()), "Check edit parameters")
		}

		edits = append(edits, edit)
	}

	// Validate all edits first (dry run)
	var details []EditDetail
	testContent := originalContent

	for i, edit := range edits {
		detail := EditDetail{
			EditIndex: i,
			Type:      edit.Type,
		}

		if err := e.validateEdit(testContent, edit); err != nil {
			detail.Status = "failed"
			detail.Error = err.Error()
			details = append(details, detail)

			// Return error - transaction failed
			return e.errorResponseWithDetails(filePath, fmt.Sprintf("validation failed at edit %d: %s", i, err.Error()), "Fix edit parameters or make pattern more specific", details)
		}

		// Apply edit to test content for next validation
		var applyErr error
		testContent, applyErr = e.applyEdit(testContent, edit)
		if applyErr != nil {
			detail.Status = "failed"
			detail.Error = applyErr.Error()
			details = append(details, detail)
			return e.errorResponseWithDetails(filePath, fmt.Sprintf("apply failed at edit %d: %s", i, applyErr.Error()), "Check edit logic", details)
		}

		detail.Status = "success"
		details = append(details, detail)
	}

	// Compute diff before writing
	diff := unifiedDiff(originalContent, testContent, relPath)

	if previewOnly {
		response := EditResponse{
			Success:        true,
			FilePath:       relPath,
			PreviewOnly:    true,
			Diff:           diff,
			EditsApplied:   len(edits),
			EditsFailed:    0,
			FileSizeBefore: sizeBefore,
			Details:        details,
			Hint:           "Preview only — no changes written. Call again without preview_only=true to apply.",
		}
		jsonResponse, _ := json.Marshal(response)
		return string(jsonResponse), nil
	}

	// Write the modified content
	if err := os.WriteFile(absPath, []byte(testContent), info.Mode()); err != nil {
		return e.errorResponse(filePath, "failed to write file", "Check permissions")
	}

	// Get new file size
	newInfo, _ := os.Stat(absPath)
	sizeAfter := newInfo.Size()

	// Build success response
	response := EditResponse{
		Success:        true,
		FilePath:       relPath,
		Diff:           diff,
		EditsApplied:   len(edits),
		EditsFailed:    0,
		FileSizeBefore: sizeBefore,
		FileSizeAfter:  sizeAfter,
		Details:        details,
	}

	jsonResponse, _ := json.Marshal(response)
	return string(jsonResponse), nil
}

// handleCreateFile handles file creation operations
func (e *EditExecutor) handleCreateFile(filePath, absPath, relPath string, editsRaw []interface{}) (string, error) {
	// Validate only one create_file operation allowed
	if len(editsRaw) != 1 {
		return e.errorResponse(filePath, "create_file must be the only operation", "Use edit_file separately for modifications")
	}

	// Parse the create operation
	editMap, ok := editsRaw[0].(map[string]interface{})
	if !ok {
		return e.errorResponse(filePath, "invalid create_file operation", "Check format")
	}

	edit, err := e.parseEditOperation(editMap)
	if err != nil {
		return e.errorResponse(filePath, fmt.Sprintf("parse error: %s", err.Error()), "Check parameters")
	}

	// Validate content field exists and is not empty
	if edit.Content == "" {
		return e.errorResponse(filePath, "content field is required for create_file", "Provide file content")
	}

	// Validate UTF-8
	if !utf8.ValidString(edit.Content) {
		return e.errorResponse(filePath, "content must be valid UTF-8 text", "Check file encoding")
	}

	// Check content size
	contentBytes := []byte(edit.Content)
	if len(contentBytes) > maxEditFileSize {
		return e.errorResponse(filePath, fmt.Sprintf("content too large (%d bytes, max 1MB)", len(contentBytes)), "Split into smaller files")
	}

	// Check file does NOT already exist
	if _, err := os.Stat(absPath); err == nil {
		return e.errorResponse(filePath, "file already exists. Use edit operations to modify", "Use replace/insert_after for existing files")
	}

	// Create parent directories if needed
	parentDir := filepath.Dir(absPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return e.errorResponse(filePath, fmt.Sprintf("failed to create directories: %s", err.Error()), "Check permissions")
	}

	// Write the file
	if err := os.WriteFile(absPath, contentBytes, 0644); err != nil {
		return e.errorResponse(filePath, fmt.Sprintf("failed to write file: %s", err.Error()), "Check permissions")
	}

	// Get file info for response
	info, err := os.Stat(absPath)
	if err != nil {
		return e.errorResponse(filePath, "file created but stat failed", "File may still exist")
	}

	// Build success response
	detail := EditDetail{
		EditIndex: 0,
		Type:      EditCreateFile,
		Status:    "success",
	}

	response := EditResponse{
		Success:        true,
		FilePath:       relPath,
		EditsApplied:   1,
		EditsFailed:    0,
		FileSizeBefore: 0,
		FileSizeAfter:  info.Size(),
		Details:        []EditDetail{detail},
		Hint:           fmt.Sprintf("Created new file: %s (%d bytes)", relPath, info.Size()),
	}

	jsonResponse, err := json.Marshal(response)
	if err != nil {
		return e.errorResponse(filePath, "failed to encode response", "Internal error")
	}

	return string(jsonResponse), nil
}

func (e *EditExecutor) parseEditOperation(editMap map[string]interface{}) (EditOperation, error) {
	edit := EditOperation{}

	// Extract type
	typeStr, ok := editMap["type"].(string)
	if !ok {
		return edit, fmt.Errorf("type is required")
	}
	edit.Type = EditType(typeStr)

	// Validate and extract parameters based on type
	switch edit.Type {
	case EditReplaceLines:
		if startHash, ok := editMap["start_hash"].(string); ok {
			edit.StartHash = startHash
		} else {
			return edit, fmt.Errorf("start_hash is required for replace_lines")
		}
		if newContent, ok := editMap["new_content"].(string); ok {
			edit.NewContent = newContent
		} else {
			return edit, fmt.Errorf("new_content is required for replace_lines")
		}
		// end_hash is optional
		if endHash, ok := editMap["end_hash"].(string); ok {
			edit.EndHash = endHash
		}

	case EditInsertAfterHash:
		if afterHash, ok := editMap["after_hash"].(string); ok {
			edit.AfterHash = afterHash
		} else {
			return edit, fmt.Errorf("after_hash is required for insert_after_hash")
		}
		if content, ok := editMap["content"].(string); ok {
			edit.Content = content
		} else {
			return edit, fmt.Errorf("content is required for insert_after_hash")
		}

	case EditInsertBeforeHash:
		if beforeHash, ok := editMap["before_hash"].(string); ok {
			edit.BeforeHash = beforeHash
		} else {
			return edit, fmt.Errorf("before_hash is required for insert_before_hash")
		}
		if content, ok := editMap["content"].(string); ok {
			edit.Content = content
		} else {
			return edit, fmt.Errorf("content is required for insert_before_hash")
		}

	case EditDeleteByHash:
		if startHash, ok := editMap["start_hash"].(string); ok {
			edit.StartHash = startHash
		} else {
			return edit, fmt.Errorf("start_hash is required for delete_by_hash")
		}
		// end_hash is optional
		if endHash, ok := editMap["end_hash"].(string); ok {
			edit.EndHash = endHash
		}

	case EditCreateFile:
		if content, ok := editMap["content"].(string); ok {
			edit.Content = content
		} else {
			return edit, fmt.Errorf("content is required for create_file")
		}

	default:
		return edit, fmt.Errorf("invalid edit type: %s (valid: replace_lines, insert_after_hash, insert_before_hash, delete_by_hash, create_file)", typeStr)
	}

	return edit, nil
}

func (e *EditExecutor) validateEdit(content string, edit EditOperation) error {
	lines := strings.Split(content, "\n")

	switch edit.Type {
	case EditReplaceLines:
		_, _, err := e.validateHashRange(lines, edit.StartHash, edit.EndHash)
		return err

	case EditInsertAfterHash:
		_, err := e.findLineByHash(lines, edit.AfterHash)
		return err

	case EditInsertBeforeHash:
		_, err := e.findLineByHash(lines, edit.BeforeHash)
		return err

	case EditDeleteByHash:
		_, _, err := e.validateHashRange(lines, edit.StartHash, edit.EndHash)
		return err

	case EditCreateFile:
		// No validation needed
		return nil
	}

	return nil
}

func (e *EditExecutor) applyEdit(content string, edit EditOperation) (string, error) {
	lines := strings.Split(content, "\n")

	var result []string
	var err error

	switch edit.Type {
	case EditReplaceLines:
		result, err = e.applyReplaceLines(lines, edit)
	case EditInsertAfterHash:
		result, err = e.applyInsertAfterHash(lines, edit)
	case EditInsertBeforeHash:
		result, err = e.applyInsertBeforeHash(lines, edit)
	case EditDeleteByHash:
		result, err = e.applyDeleteByHash(lines, edit)
	default:
		return "", fmt.Errorf("unsupported edit type: %s", edit.Type)
	}

	if err != nil {
		return "", err
	}

	return strings.Join(result, "\n"), nil
}

// findLineByHash finds the line index for a given hash
func (e *EditExecutor) findLineByHash(lines []string, hash string) (int, error) {
	for i, line := range lines {
		if hashLine(line) == hash {
			return i, nil // 0-indexed
		}
	}
	return -1, fmt.Errorf("hash %s not found (file may have changed since you read it - please re-read and retry)", hash)
}

// validateHashRange validates a hash range and returns indices
func (e *EditExecutor) validateHashRange(lines []string, startHash, endHash string) (int, int, error) {
	startIdx, err := e.findLineByHash(lines, startHash)
	if err != nil {
		return -1, -1, fmt.Errorf("start_hash: %w", err)
	}

	if endHash == "" {
		// Single line operation
		return startIdx, startIdx, nil
	}

	endIdx, err := e.findLineByHash(lines, endHash)
	if err != nil {
		return -1, -1, fmt.Errorf("end_hash: %w", err)
	}

	if endIdx < startIdx {
		return -1, -1, fmt.Errorf("end_hash line appears before start_hash line")
	}

	return startIdx, endIdx, nil
}

// applyReplaceLines replaces lines identified by hash range
func (e *EditExecutor) applyReplaceLines(lines []string, edit EditOperation) ([]string, error) {
	startIdx, endIdx, err := e.validateHashRange(lines, edit.StartHash, edit.EndHash)
	if err != nil {
		return nil, err
	}

	// Split new content into lines
	newLines := strings.Split(edit.NewContent, "\n")

	// Build result: before + new + after
	result := make([]string, 0, len(lines)-(endIdx-startIdx)+len(newLines))
	result = append(result, lines[:startIdx]...)
	result = append(result, newLines...)
	result = append(result, lines[endIdx+1:]...)

	return result, nil
}

// applyInsertAfterHash inserts content after a line
func (e *EditExecutor) applyInsertAfterHash(lines []string, edit EditOperation) ([]string, error) {
	idx, err := e.findLineByHash(lines, edit.AfterHash)
	if err != nil {
		return nil, err
	}

	insertLines := strings.Split(edit.Content, "\n")

	result := make([]string, 0, len(lines)+len(insertLines))
	result = append(result, lines[:idx+1]...)
	result = append(result, insertLines...)
	result = append(result, lines[idx+1:]...)

	return result, nil
}

// applyInsertBeforeHash inserts content before a line
func (e *EditExecutor) applyInsertBeforeHash(lines []string, edit EditOperation) ([]string, error) {
	idx, err := e.findLineByHash(lines, edit.BeforeHash)
	if err != nil {
		return nil, err
	}

	insertLines := strings.Split(edit.Content, "\n")

	result := make([]string, 0, len(lines)+len(insertLines))
	result = append(result, lines[:idx]...)
	result = append(result, insertLines...)
	result = append(result, lines[idx:]...)

	return result, nil
}

// applyDeleteByHash deletes lines by hash range
func (e *EditExecutor) applyDeleteByHash(lines []string, edit EditOperation) ([]string, error) {
	startIdx, endIdx, err := e.validateHashRange(lines, edit.StartHash, edit.EndHash)
	if err != nil {
		return nil, err
	}

	result := make([]string, 0, len(lines)-(endIdx-startIdx+1))
	result = append(result, lines[:startIdx]...)
	result = append(result, lines[endIdx+1:]...)

	return result, nil
}

func (e *EditExecutor) errorResponse(filePath, error, hint string) (string, error) {
	response := EditResponse{
		Success:  false,
		FilePath: filePath,
		Error:    error,
		Hint:     hint,
	}
	jsonResponse, _ := json.Marshal(response)
	return string(jsonResponse), nil
}

func (e *EditExecutor) errorResponseWithDetails(filePath, error, hint string, details []EditDetail) (string, error) {
	failedCount := 0
	successCount := 0
	for _, d := range details {
		if d.Status == "failed" {
			failedCount++
		} else {
			successCount++
		}
	}

	response := EditResponse{
		Success:      false,
		FilePath:     filePath,
		EditsApplied: successCount,
		EditsFailed:  failedCount,
		Details:      details,
		Error:        error,
		Hint:         hint,
	}
	jsonResponse, _ := json.Marshal(response)
	return string(jsonResponse), nil
}
