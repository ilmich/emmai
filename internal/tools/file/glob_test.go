package file

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestGlobFiles(t *testing.T) {
	// Create temporary directory structure
	tempDir := t.TempDir()

	// Create test files
	files := []string{
		"file1.go",
		"file2.go",
		"test_file.go",
		"main.js",
		"styles.css",
		"dir1/nested.go",
		"dir1/nested.js",
		"dir2/deep/file.go",
		"README.md",
	}

	for _, file := range files {
		fullPath := filepath.Join(tempDir, file)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte("test content"), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
	}

	executor := NewGlobExecutor(tempDir)

	tests := []struct {
		name          string
		args          map[string]interface{}
		wantFiles     []string
		wantError     bool
		wantTruncated bool
	}{
		{
			name: "find all go files",
			args: map[string]interface{}{
				"pattern": "**/*.go",
			},
			wantFiles: []string{
				"file1.go",
				"file2.go",
				"test_file.go",
				"dir1/nested.go",
				"dir2/deep/file.go",
			},
			wantError: false,
		},
		{
			name: "find root level files",
			args: map[string]interface{}{
				"pattern": "*",
			},
			wantFiles: []string{
				"file1.go",
				"file2.go",
				"test_file.go",
				"main.js",
				"styles.css",
				"README.md",
				"dir1",
				"dir2",
			},
			wantError: false,
		},
		{
			name: "find multiple extensions",
			args: map[string]interface{}{
				"pattern": "**/*.{js,css}",
			},
			wantFiles: []string{
				"main.js",
				"styles.css",
				"dir1/nested.js",
			},
			wantError: false,
		},
		{
			name: "find with prefix pattern",
			args: map[string]interface{}{
				"pattern": "**/test_*",
			},
			wantFiles: []string{
				"test_file.go",
			},
			wantError: false,
		},
		{
			name: "find in specific directory",
			args: map[string]interface{}{
				"pattern": "dir1/*",
			},
			wantFiles: []string{
				"dir1/nested.go",
				"dir1/nested.js",
			},
			wantError: false,
		},
		{
			name: "no matches",
			args: map[string]interface{}{
				"pattern": "**/*.py",
			},
			wantFiles: []string{},
			wantError: false,
		},
		{
			name: "empty pattern",
			args: map[string]interface{}{
				"pattern": "",
			},
			wantFiles: nil,
			wantError: true,
		},
		{
			name: "max results truncation",
			args: map[string]interface{}{
				"pattern":     "**/*.go",
				"max_results": 2.0,
			},
			wantFiles:     []string{"file1.go", "file2.go"},
			wantError:     false,
			wantTruncated: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := executor.HandleGlobFiles(tt.args)
			if err != nil {
				t.Fatalf("HandleGlobFiles returned error: %v", err)
			}

			var response GlobResponse
			if err := json.Unmarshal([]byte(result), &response); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}

			if tt.wantError {
				if response.Success {
					t.Errorf("Expected error, got success")
				}
				return
			}

			if !response.Success {
				t.Errorf("Expected success, got error: %s", response.Error)
			}

			if tt.wantTruncated != response.Truncated {
				t.Errorf("Expected truncated=%v, got %v", tt.wantTruncated, response.Truncated)
			}

			if tt.wantFiles != nil {
				if len(response.Matches) != len(tt.wantFiles) {
					t.Errorf("Expected %d matches, got %d", len(tt.wantFiles), len(response.Matches))
				}

				foundFiles := make(map[string]bool)
				for _, match := range response.Matches {
					foundFiles[match.FilePath] = true
					// Verify metadata is present
					if match.Size == 0 && match.FilePath != "" {
						// Files we created have content, so size should be > 0
						if match.Size == 0 {
							t.Errorf("File %s has zero size", match.FilePath)
						}
					}
					if match.ModTime == "" {
						t.Errorf("File %s missing mod_time", match.FilePath)
					}
				}

				for _, expectedFile := range tt.wantFiles {
					if !foundFiles[expectedFile] {
						t.Errorf("Expected file %s not found in matches", expectedFile)
					}
				}
			}
		})
	}
}

func TestGlobFilesMaxResults(t *testing.T) {
	tempDir := t.TempDir()

	// Create many files
	for i := 0; i < 15; i++ {
		filename := filepath.Join(tempDir, fmt.Sprintf("file%d.go", i))
		if err := os.WriteFile(filename, []byte("content"), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
	}

	executor := NewGlobExecutor(tempDir)

	// Test default max_results
	result, _ := executor.HandleGlobFiles(map[string]interface{}{
		"pattern": "*.go",
	})

	var response GlobResponse
	json.Unmarshal([]byte(result), &response)

	if len(response.Matches) > defaultMaxGlobResults {
		t.Errorf("Expected max %d results, got %d", defaultMaxGlobResults, len(response.Matches))
	}

	// Test custom max_results
	result, _ = executor.HandleGlobFiles(map[string]interface{}{
		"pattern":     "*.go",
		"max_results": 5.0,
	})

	json.Unmarshal([]byte(result), &response)

	if len(response.Matches) > 5 {
		t.Errorf("Expected max 5 results, got %d", len(response.Matches))
	}

	if !response.Truncated {
		t.Error("Expected truncated=true when results exceed max")
	}
}

func TestGlobFilesMetadata(t *testing.T) {
	tempDir := t.TempDir()

	// Create a file
	testFile := filepath.Join(tempDir, "test.go")
	content := []byte("package main\n")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create a directory
	testDir := filepath.Join(tempDir, "testdir")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	executor := NewGlobExecutor(tempDir)

	result, _ := executor.HandleGlobFiles(map[string]interface{}{
		"pattern": "*",
	})

	var response GlobResponse
	json.Unmarshal([]byte(result), &response)

	if !response.Success {
		t.Fatalf("Expected success, got error: %s", response.Error)
	}

	// Check file metadata
	for _, match := range response.Matches {
		if match.FilePath == "test.go" {
			if match.IsDirectory {
				t.Error("test.go should not be marked as directory")
			}
			if match.Size != int64(len(content)) {
				t.Errorf("Expected size %d, got %d", len(content), match.Size)
			}
			if match.ModTime == "" {
				t.Error("ModTime should not be empty")
			}
		}
		if match.FilePath == "testdir" {
			if !match.IsDirectory {
				t.Error("testdir should be marked as directory")
			}
		}
	}
}
