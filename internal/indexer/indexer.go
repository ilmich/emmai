package indexer

import (
	"bufio"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	indexFile    = ".emmai/index.json"
	freshnessTTL = 5 * time.Minute
)

var skipDirs = map[string]bool{
	".git": true, "vendor": true, "node_modules": true,
	"bin": true, "dist": true, "build": true, ".emmai": true,
	".cache": true, "__pycache__": true, ".idea": true, ".vscode": true,
}

var langByExt = map[string]string{
	".go": "go", ".py": "python", ".js": "javascript", ".ts": "typescript",
	".jsx": "javascript", ".tsx": "typescript", ".rs": "rust",
	".c": "c", ".h": "c", ".cpp": "cpp", ".cc": "cpp", ".hpp": "cpp",
	".java": "java", ".rb": "ruby", ".php": "php", ".lua": "lua",
	".swift": "swift", ".kt": "kotlin", ".sh": "shell", ".bash": "shell",
	".yaml": "yaml", ".yml": "yaml", ".toml": "toml", ".json": "json",
	".md": "markdown", ".txt": "text", ".html": "html", ".css": "css",
	".proto": "protobuf", ".sql": "sql",
}

// Build scans workDir and returns a fresh Index.
func Build(workDir string) (*Index, error) {
	idx := &Index{
		WorkDir:   workDir,
		IndexedAt: time.Now(),
	}

	err := filepath.WalkDir(workDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable entries
		}

		name := d.Name()

		if d.IsDir() {
			if skipDirs[name] || strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip generated/binary-like files
		if strings.HasSuffix(name, ".pb.go") || strings.HasSuffix(name, ".gen.go") {
			return nil
		}

		rel, err := filepath.Rel(workDir, path)
		if err != nil {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}

		ext := fileExt(name)
		lang := langByExt[ext]

		entry := FileEntry{
			Path:     rel,
			Language: lang,
			Size:     info.Size(),
		}

		lines, readErr := readLines(path)
		if readErr == nil {
			entry.Lines = len(lines)
		}

		var syms []Symbol
		if ext == ".go" {
			s, pkg, parseErr := parseGoFile(path, rel)
			if parseErr == nil {
				syms = s
				entry.Package = pkg
			}
		} else if readErr == nil {
			syms = parseFileCtags(rel, lines)
		}

		idx.Files = append(idx.Files, entry)
		idx.Symbols = append(idx.Symbols, syms...)
		return nil
	})

	return idx, err
}

// Load reads a previously saved index from <workDir>/.emmai/index.json.
func Load(workDir string) (*Index, error) {
	data, err := os.ReadFile(filepath.Join(workDir, indexFile))
	if err != nil {
		return nil, err
	}
	var idx Index
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, err
	}
	return &idx, nil
}

// Save persists the index to <workDir>/.emmai/index.json.
func Save(idx *Index) error {
	dir := filepath.Join(idx.WorkDir, ".emmai")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(idx.WorkDir, indexFile), data, 0o644)
}

// BuildOrLoad returns a cached index if fresh, otherwise rebuilds and saves.
func BuildOrLoad(workDir string) (*Index, error) {
	existing, err := Load(workDir)
	if err == nil && isFresh(workDir, existing) {
		return existing, nil
	}

	idx, err := Build(workDir)
	if err != nil {
		return nil, err
	}
	_ = Save(idx) // best-effort persist
	return idx, nil
}

// isFresh returns true if the index is recent and no Go file has been modified since.
func isFresh(workDir string, idx *Index) bool {
	if time.Since(idx.IndexedAt) > freshnessTTL {
		return false
	}
	// Check if any Go file is newer than the index
	stale := false
	_ = filepath.WalkDir(workDir, func(path string, d fs.DirEntry, err error) error {
		if stale || err != nil {
			return nil
		}
		if d.IsDir() {
			if skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, ".go") {
			info, err := d.Info()
			if err == nil && info.ModTime().After(idx.IndexedAt) {
				stale = true
			}
		}
		return nil
	})
	return !stale
}

func readLines(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var lines []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	return lines, sc.Err()
}
