// Package embed defines the Embedder interface and its backends. The Go core
// stays CGo-free: real ONNX inference lives in a sidecar (Rust fastembed-rs) or
// Ollama, added in v1 (decision.md §11). For v0 the Nop embedder makes retrieval
// degrade gracefully to FTS5 keyword-only.
package embed

import "context"

// Embedder turns text into vectors. Backends: Nop (FTS5-only), Ollama (http),
// Sidecar (Rust, v1).
type Embedder interface {
	// Embed returns one vector per input text, each of length Dims().
	Embed(ctx context.Context, texts []string) ([][]float32, error)
	// Dims is the vector dimensionality; 0 means "no embeddings" (Nop).
	Dims() int
	// Name identifies the backend (for diagnostics).
	Name() string
}

// Nop produces no vectors, so retrieval falls back to keyword search. It is the
// v0 default until an embedding backend is wired up.
type Nop struct{}

// Embed returns a slice of nil vectors, one per input.
func (Nop) Embed(_ context.Context, texts []string) ([][]float32, error) {
	return make([][]float32, len(texts)), nil
}

// Dims reports 0: no embeddings.
func (Nop) Dims() int { return 0 }

// Name returns "nop".
func (Nop) Name() string { return "nop" }
