package file

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadExecutor_ReadFile_Success(t *testing.T) {
	tmpDir := t.TempDir()
	executor := NewReadExecutor(tmpDir)

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "Hello, World!\nThis is a test file.\n"
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	args := map[string]interface{}{
		"file_path": "test.txt",
	}

	result, err := executor.HandleReadFile(args)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	var response ReadResponse
	if err := json.Unmarshal([]byte(result), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if !response.Success {
		t.Errorf("Expected success=true, got false. Error: %s", response.Error)
	}

	if response.FilePath != "test.txt" {
		t.Errorf("Expected file_path=test.txt, got %s", response.FilePath)
	}

	// Check hashline format - split matches how read.go splits (includes trailing empty line)
	expectedLines := strings.Split(testContent, "\n")
	if response.LineCount != len(expectedLines) {
		t.Errorf("Line count mismatch. Expected: %d, Got: %d", len(expectedLines), response.LineCount)
	}

	if len(response.Lines) != len(expectedLines) {
		t.Errorf("Lines array length mismatch. Expected: %d, Got: %d", len(expectedLines), len(response.Lines))
	}

	// Verify each line has proper structure
	for i, line := range response.Lines {
		if line.Number != i+1 {
			t.Errorf("Line %d: Expected num=%d, got %d", i, i+1, line.Number)
		}
		if line.Hash == "" {
			t.Errorf("Line %d: Expected hash, got empty", i)
		}
		if len(line.Hash) != 8 {
			t.Errorf("Line %d: Expected 8-char hash, got %d chars: %s", i, len(line.Hash), line.Hash)
		}
		if i < len(expectedLines) && line.Content != expectedLines[i] {
			t.Errorf("Line %d: Content mismatch. Expected: %q, Got: %q", i, expectedLines[i], line.Content)
		}
	}

	// Verify hash consistency (same content = same hash)
	if len(response.Lines) >= 2 {
		hash1 := hashLine(response.Lines[0].Content)
		if response.Lines[0].Hash != hash1 {
			t.Errorf("Hash mismatch for line 1. Expected: %s, Got: %s", hash1, response.Lines[0].Hash)
		}
	}

	if response.Size != int64(len(testContent)) {
		t.Errorf("Size mismatch. Expected: %d, Got: %d", len(testContent), response.Size)
	}

	if response.LastModified == "" {
		t.Errorf("Expected last_modified timestamp, got empty")
	}
}

func TestReadExecutor_ReadFile_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	executor := NewReadExecutor(tmpDir)

	args := map[string]interface{}{
		"file_path": "nonexistent.txt",
	}

	result, err := executor.HandleReadFile(args)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	var response ReadResponse
	if err := json.Unmarshal([]byte(result), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response.Success {
		t.Errorf("Expected success=false for missing file")
	}

	if !strings.Contains(response.Error, "file not found") {
		t.Errorf("Expected 'file not found' error, got: %s", response.Error)
	}

	if response.Hint == "" {
		t.Errorf("Expected hint for missing file")
	}
}

func TestReadExecutor_ReadFile_Directory(t *testing.T) {
	tmpDir := t.TempDir()
	executor := NewReadExecutor(tmpDir)

	// Create a subdirectory
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	args := map[string]interface{}{
		"file_path": "subdir",
	}

	result, err := executor.HandleReadFile(args)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	var response ReadResponse
	if err := json.Unmarshal([]byte(result), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response.Success {
		t.Errorf("Expected success=false for directory")
	}

	if !strings.Contains(response.Error, "directory") {
		t.Errorf("Expected 'directory' error, got: %s", response.Error)
	}
}

func TestReadExecutor_ReadFile_TooLarge(t *testing.T) {
	tmpDir := t.TempDir()
	executor := NewReadExecutor(tmpDir)

	// Create a file larger than 1MB
	testFile := filepath.Join(tmpDir, "large.txt")
	largeContent := make([]byte, maxReadFileSize+1)
	for i := range largeContent {
		largeContent[i] = 'A'
	}
	if err := os.WriteFile(testFile, largeContent, 0644); err != nil {
		t.Fatal(err)
	}

	args := map[string]interface{}{
		"file_path": "large.txt",
	}

	result, err := executor.HandleReadFile(args)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	var response ReadResponse
	if err := json.Unmarshal([]byte(result), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response.Success {
		t.Errorf("Expected success=false for large file")
	}

	if !strings.Contains(response.Error, "too large") {
		t.Errorf("Expected 'too large' error, got: %s", response.Error)
	}

	if !strings.Contains(response.Hint, "size limit") {
		t.Errorf("Expected size limit hint, got: %s", response.Hint)
	}
}

func TestReadExecutor_ReadFile_BinaryFile(t *testing.T) {
	tmpDir := t.TempDir()
	executor := NewReadExecutor(tmpDir)

	// Create a binary file (non-UTF-8)
	testFile := filepath.Join(tmpDir, "binary.dat")
	binaryContent := []byte{0xFF, 0xFE, 0xFD, 0xFC, 0xFB}
	if err := os.WriteFile(testFile, binaryContent, 0644); err != nil {
		t.Fatal(err)
	}

	args := map[string]interface{}{
		"file_path": "binary.dat",
	}

	result, err := executor.HandleReadFile(args)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	var response ReadResponse
	if err := json.Unmarshal([]byte(result), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response.Success {
		t.Errorf("Expected success=false for binary file")
	}

	if !strings.Contains(response.Error, "non-UTF-8") || !strings.Contains(response.Error, "binary") {
		t.Errorf("Expected 'non-UTF-8' or 'binary' error, got: %s", response.Error)
	}
}

func TestReadExecutor_ReadFile_PathTraversal(t *testing.T) {
	tmpDir := t.TempDir()
	executor := NewReadExecutor(tmpDir)

	// Try to read a file outside working directory
	args := map[string]interface{}{
		"file_path": "../../../etc/passwd",
	}

	result, err := executor.HandleReadFile(args)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	var response ReadResponse
	if err := json.Unmarshal([]byte(result), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response.Success {
		t.Errorf("Expected success=false for path traversal attempt")
	}

	if !strings.Contains(response.Error, "not allowed") && !strings.Contains(response.Error, "working directory") {
		t.Errorf("Expected path traversal error, got: %s", response.Error)
	}
}

func TestReadExecutor_ReadFile_EmptyFilePath(t *testing.T) {
	tmpDir := t.TempDir()
	executor := NewReadExecutor(tmpDir)

	args := map[string]interface{}{
		"file_path": "",
	}

	result, err := executor.HandleReadFile(args)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	var response ReadResponse
	if err := json.Unmarshal([]byte(result), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response.Success {
		t.Errorf("Expected success=false for empty file path")
	}

	if !strings.Contains(response.Error, "required") {
		t.Errorf("Expected 'required' error, got: %s", response.Error)
	}
}
