package embedder

import (
	"testing"
)

// TestEmbedderInterface verifies that E5Embedder implements the Embedder interface.
func TestEmbedderInterface(t *testing.T) {
	// Compile-time check: E5Embedder must implement Embedder
	var _ Embedder = (*E5Embedder)(nil)
}

// TestE5Embedder_Dimensions_FromMock verifies dimensions are read from the worker.
func TestE5Embedder_Dimensions_FromMock(t *testing.T) {
	e := newMockE5(t, mockE5Script)
	if e.Dimensions() != 4 {
		t.Errorf("expected dimensions 4, got %d", e.Dimensions())
	}
}
