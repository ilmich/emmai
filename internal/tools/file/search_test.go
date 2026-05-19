package file

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSearchExecutor_BasicSearch(t *testing.T) {
	// Create temp directory with test files
	tmpDir := t.TempDir()

	// Create test file
	testFile := filepath.Join(tmpDir, "test.go")
	content := `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}

func helper() {
	fmt.Println("Helper function")
}
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	executor := NewSearchExecutor(tmpDir)

	// Test basic text search
	args := map[string]interface{}{
		"pattern": "fmt.Println",
		"path":    "**/*.go",
	}

	result, err := executor.HandleSearchFiles(args)
	if err != nil {
		t.Errorf("HandleSearchFiles failed: %v", err)
	}

	if result == "" {
		t.Error("Expected non-empty result")
	}

	t.Logf("Search result: %s", result)
}

func TestSearchExecutor_RegexSearch(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.go")
	content := `func handleRequest() {}
func handleResponse() {}
func processData() {}
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	executor := NewSearchExecutor(tmpDir)

	// Test regex search
	args := map[string]interface{}{
		"pattern": "func handle.*",
		"regex":   true,
	}

	result, err := executor.HandleSearchFiles(args)
	if err != nil {
		t.Errorf("HandleSearchFiles failed: %v", err)
	}

	if result == "" {
		t.Error("Expected non-empty result")
	}

	t.Logf("Regex search result: %s", result)
}

func TestSearchExecutor_CaseSensitive(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.txt")
	content := `Hello World
hello world
HELLO WORLD
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	executor := NewSearchExecutor(tmpDir)

	// Test case-sensitive search
	args := map[string]interface{}{
		"pattern":        "Hello",
		"case_sensitive": true,
	}

	result, err := executor.HandleSearchFiles(args)
	if err != nil {
		t.Errorf("HandleSearchFiles failed: %v", err)
	}

	t.Logf("Case-sensitive result: %s", result)
}
