package file

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestEditExecutor_ReplaceLines_SingleLine(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.go")
	content := `package main

func oldName() {
	println("test")
}
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	executor := NewEditExecutor(tmpDir)

	// Get hash for the line we want to replace
	lines := []string{"package main", "", "func oldName() {", "\tprintln(\"test\")", "}"}
	targetHash := hashLine(lines[2]) // "func oldName() {"

	args := map[string]interface{}{
		"file_path": "test.go",
		"edits": []interface{}{
			map[string]interface{}{
				"type":        "replace_lines",
				"start_hash":  targetHash,
				"new_content": "func newName() {",
			},
		},
	}

	result, err := executor.HandleEditFile(args)
	if err != nil {
		t.Errorf("HandleEditFile failed: %v", err)
	}

	var response EditResponse
	if err := json.Unmarshal([]byte(result), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if !response.Success {
		t.Errorf("Expected success, got error: %s", response.Error)
	}

	if response.EditsApplied != 1 {
		t.Errorf("Expected 1 edit applied, got %d", response.EditsApplied)
	}

	// Verify file was modified
	modifiedContent, _ := os.ReadFile(testFile)
	if !contains(string(modifiedContent), "func newName()") {
		t.Error("Expected file to contain 'func newName()'")
	}

	t.Logf("Replace result: %+v", response)
}

func TestEditExecutor_InsertAfterHash(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.go")
	content := `package main

import (
	"fmt"
)

func main() {}
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	executor := NewEditExecutor(tmpDir)

	// Get hash for the line after which we want to insert
	fmtLineHash := hashLine("\t\"fmt\"")

	args := map[string]interface{}{
		"file_path": "test.go",
		"edits": []interface{}{
			map[string]interface{}{
				"type":       "insert_after_hash",
				"after_hash": fmtLineHash,
				"content":    "\n\t\"os\"",
			},
		},
	}

	result, err := executor.HandleEditFile(args)
	if err != nil {
		t.Errorf("HandleEditFile failed: %v", err)
	}

	var response EditResponse
	if err := json.Unmarshal([]byte(result), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if !response.Success {
		t.Errorf("Expected success, got error: %s", response.Error)
	}

	modifiedContent, _ := os.ReadFile(testFile)
	if !contains(string(modifiedContent), "\"os\"") {
		t.Error("Expected file to contain '\"os\"' import")
	}

	t.Logf("Insert result: %+v", response)
}

func TestEditExecutor_DeleteByHash(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.go")
	content := `line 1
line 2
line 3
line 4
line 5
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	executor := NewEditExecutor(tmpDir)

	// Calculate hashes for lines 2-4
	line2Hash := hashLine("line 2")
	line4Hash := hashLine("line 4")

	args := map[string]interface{}{
		"file_path": "test.go",
		"edits": []interface{}{
			map[string]interface{}{
				"type":       "delete_by_hash",
				"start_hash": line2Hash,
				"end_hash":   line4Hash,
			},
		},
	}

	result, err := executor.HandleEditFile(args)
	if err != nil {
		t.Errorf("HandleEditFile failed: %v", err)
	}

	var response EditResponse
	if err := json.Unmarshal([]byte(result), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if !response.Success {
		t.Errorf("Expected success, got error: %s", response.Error)
	}

	modifiedContent, _ := os.ReadFile(testFile)
	contentStr := string(modifiedContent)
	
	if contains(contentStr, "line 2") || contains(contentStr, "line 3") || contains(contentStr, "line 4") {
		t.Error("Expected lines 2-4 to be deleted")
	}

	if !contains(contentStr, "line 1") || !contains(contentStr, "line 5") {
		t.Error("Expected lines 1 and 5 to remain")
	}

	t.Logf("Delete result: %+v", response)
}

func TestEditExecutor_MultipleEdits_Hashline(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.go")
	content := `package main

import "fmt"

func oldFunc() {
	fmt.Println("test")
}
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	executor := NewEditExecutor(tmpDir)

	// Calculate hashes
	importLineHash := hashLine("import \"fmt\"")
	funcLineHash := hashLine("func oldFunc() {")

	args := map[string]interface{}{
		"file_path": "test.go",
		"edits": []interface{}{
			map[string]interface{}{
				"type":       "insert_after_hash",
				"after_hash": importLineHash,
				"content":    "\nimport \"os\"",
			},
			map[string]interface{}{
				"type":        "replace_lines",
				"start_hash":  funcLineHash,
				"new_content": "func newFunc() {",
			},
		},
	}

	result, err := executor.HandleEditFile(args)
	if err != nil {
		t.Errorf("HandleEditFile failed: %v", err)
	}

	var response EditResponse
	if err := json.Unmarshal([]byte(result), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if !response.Success {
		t.Errorf("Expected success, got error: %s", response.Error)
	}

	if response.EditsApplied != 2 {
		t.Errorf("Expected 2 edits applied, got %d", response.EditsApplied)
	}

	t.Logf("Multiple edits result: %+v", response)
}

func TestEditExecutor_HashNotFound(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.go")
	content := `package main`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	executor := NewEditExecutor(tmpDir)

	// Use a hash that doesn't exist in the file
	args := map[string]interface{}{
		"file_path": "test.go",
		"edits": []interface{}{
			map[string]interface{}{
				"type":        "replace_lines",
				"start_hash":  "deadbeef", // Non-existent hash
				"new_content": "something",
			},
		},
	}

	result, err := executor.HandleEditFile(args)
	if err != nil {
		t.Errorf("HandleEditFile failed: %v", err)
	}

	var response EditResponse
	if err := json.Unmarshal([]byte(result), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response.Success {
		t.Errorf("Expected failure for non-existent hash")
	}

	if !contains(response.Error, "hash") && !contains(response.Error, "not found") {
		t.Errorf("Expected hash not found error, got: %s", response.Error)
	}

	t.Logf("Hash not found result: %+v", response)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || 
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestEditExecutor_CreateFile_Success(t *testing.T) {
	tmpDir := t.TempDir()
	executor := NewEditExecutor(tmpDir)

	args := map[string]interface{}{
		"file_path": "newfile.txt",
		"edits": []interface{}{
			map[string]interface{}{
				"type":    "create_file",
				"content": "Hello, World!\n",
			},
		},
	}

	result, err := executor.HandleEditFile(args)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	var response EditResponse
	if err := json.Unmarshal([]byte(result), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if !response.Success {
		t.Errorf("Expected success, got error: %s", response.Error)
	}

	if response.EditsApplied != 1 {
		t.Errorf("Expected 1 edit applied, got %d", response.EditsApplied)
	}

	// Verify file was created
	content, err := os.ReadFile(filepath.Join(tmpDir, "newfile.txt"))
	if err != nil {
		t.Fatalf("File was not created: %v", err)
	}

	if string(content) != "Hello, World!\n" {
		t.Errorf("Content mismatch. Got: %s", string(content))
	}
}

func TestEditExecutor_CreateFile_AlreadyExists(t *testing.T) {
	tmpDir := t.TempDir()
	executor := NewEditExecutor(tmpDir)

	// Create a file first
	existingFile := filepath.Join(tmpDir, "existing.txt")
	os.WriteFile(existingFile, []byte("existing"), 0644)

	// Try to create it again
	args := map[string]interface{}{
		"file_path": "existing.txt",
		"edits": []interface{}{
			map[string]interface{}{
				"type":    "create_file",
				"content": "new content",
			},
		},
	}

	result, err := executor.HandleEditFile(args)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	var response EditResponse
	json.Unmarshal([]byte(result), &response)

	if response.Success {
		t.Errorf("Expected failure when file exists")
	}

	if !contains(response.Error, "already exists") {
		t.Errorf("Expected 'already exists' error, got: %s", response.Error)
	}
}

func TestEditExecutor_CreateFile_WithDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	executor := NewEditExecutor(tmpDir)

	// Create file in nested directory
	args := map[string]interface{}{
		"file_path": "path/to/newfile.txt",
		"edits": []interface{}{
			map[string]interface{}{
				"type":    "create_file",
				"content": "nested content",
			},
		},
	}

	result, err := executor.HandleEditFile(args)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	var response EditResponse
	json.Unmarshal([]byte(result), &response)

	if !response.Success {
		t.Errorf("Expected success, got: %s", response.Error)
	}

	// Verify file and directories were created
	content, err := os.ReadFile(filepath.Join(tmpDir, "path/to/newfile.txt"))
	if err != nil {
		t.Fatalf("File was not created: %v", err)
	}

	if string(content) != "nested content" {
		t.Errorf("Content mismatch")
	}
}

func TestEditExecutor_CreateFile_EmptyContent(t *testing.T) {
	tmpDir := t.TempDir()
	executor := NewEditExecutor(tmpDir)

	args := map[string]interface{}{
		"file_path": "empty.txt",
		"edits": []interface{}{
			map[string]interface{}{
				"type":    "create_file",
				"content": "",
			},
		},
	}

	result, err := executor.HandleEditFile(args)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	var response EditResponse
	json.Unmarshal([]byte(result), &response)

	if response.Success {
		t.Errorf("Expected failure with empty content")
	}
}

func TestEditExecutor_CreateFile_MixedWithEdits(t *testing.T) {
	tmpDir := t.TempDir()
	executor := NewEditExecutor(tmpDir)

	// Try to mix create_file with other operations
	args := map[string]interface{}{
		"file_path": "test.txt",
		"edits": []interface{}{
			map[string]interface{}{
				"type":    "create_file",
				"content": "initial",
			},
			map[string]interface{}{
				"type":     "replace",
				"old_text": "initial",
				"new_text": "modified",
			},
		},
	}

	result, err := executor.HandleEditFile(args)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	var response EditResponse
	json.Unmarshal([]byte(result), &response)

	if response.Success {
		t.Errorf("Expected failure when mixing create_file with other operations")
	}
}
