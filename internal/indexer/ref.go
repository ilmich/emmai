package indexer

import "sync"

// IndexRef is a thread-safe mutable reference to an Index.
// It allows the edit handler and phase controller to share a live pointer.
type IndexRef struct {
	mu  sync.RWMutex
	idx *Index
}

// NewIndexRef creates an IndexRef wrapping the given index (may be nil).
func NewIndexRef(idx *Index) *IndexRef {
	return &IndexRef{idx: idx}
}

// Get returns the current index (may be nil).
func (r *IndexRef) Get() *Index {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.idx
}

// Set replaces the current index.
func (r *IndexRef) Set(idx *Index) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.idx = idx
}
