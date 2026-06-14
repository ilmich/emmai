package indexer

import "time"

// Index is the full codebase index for a working directory.
type Index struct {
	WorkDir   string      `json:"work_dir"`
	IndexedAt time.Time   `json:"indexed_at"`
	Files     []FileEntry `json:"files"`
	Symbols   []Symbol    `json:"symbols"`
}

// FileEntry holds metadata for a single indexed file.
type FileEntry struct {
	Path     string `json:"path"`               // relative to WorkDir
	Language string `json:"language"`            // "go", "python", "typescript", etc.
	Size     int64  `json:"size"`
	Lines    int    `json:"lines"`
	Package  string `json:"package,omitempty"`   // Go package name
}

// Symbol is a named code entity extracted from a file.
type Symbol struct {
	Name      string `json:"name"`
	Kind      string `json:"kind"`               // func, method, type, struct, interface, const, var
	File      string `json:"file"`               // relative path
	Line      int    `json:"line"`
	Package   string `json:"package,omitempty"`
	Receiver  string `json:"receiver,omitempty"` // non-empty for methods
	Signature string `json:"signature,omitempty"`
}
