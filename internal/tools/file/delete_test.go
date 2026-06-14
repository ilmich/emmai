package file

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDeleteFile(t *testing.T) {
	dir := t.TempDir()
	exec := NewDeleteExecutor(dir)

	// Create a file to delete
	target := filepath.Join(dir, "target.txt")
	if err := os.WriteFile(target, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := exec.HandleDeleteFile(map[string]interface{}{"file_path": "target.txt"})
	if err != nil {
		t.Fatal(err)
	}

	var resp DeleteResponse
	if err := json.Unmarshal([]byte(result), &resp); err != nil {
		t.Fatal(err)
	}
	if !resp.Success {
		t.Fatalf("expected success, got error: %s", resp.Error)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Error("file should have been deleted")
	}
}

func TestDeleteFile_NotFound(t *testing.T) {
	exec := NewDeleteExecutor(t.TempDir())
	result, err := exec.HandleDeleteFile(map[string]interface{}{"file_path": "nope.txt"})
	if err != nil {
		t.Fatal(err)
	}
	var resp DeleteResponse
	json.Unmarshal([]byte(result), &resp)
	if resp.Success {
		t.Error("expected failure for missing file")
	}
}

func TestDeleteFile_Traversal(t *testing.T) {
	exec := NewDeleteExecutor(t.TempDir())
	result, err := exec.HandleDeleteFile(map[string]interface{}{"file_path": "../../etc/passwd"})
	if err != nil {
		t.Fatal(err)
	}
	var resp DeleteResponse
	json.Unmarshal([]byte(result), &resp)
	if resp.Success {
		t.Error("expected failure for path traversal")
	}
}

func TestDeleteFile_Directory(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "subdir")
	os.Mkdir(sub, 0o755)
	exec := NewDeleteExecutor(dir)
	result, _ := exec.HandleDeleteFile(map[string]interface{}{"file_path": "subdir"})
	var resp DeleteResponse
	json.Unmarshal([]byte(result), &resp)
	if resp.Success {
		t.Error("expected failure when target is a directory")
	}
}
