package store

import (
	"encoding/binary"
	"math"
)

// MemoryType enumerates the kinds of memory record (decision.md §5).
type MemoryType string

const (
	TypeDecision       MemoryType = "decision"
	TypeConvention     MemoryType = "convention"
	TypeSessionSummary MemoryType = "session_summary"
)

// Valid reports whether t is a known memory type.
func (t MemoryType) Valid() bool {
	switch t {
	case TypeDecision, TypeConvention, TypeSessionSummary:
		return true
	default:
		return false
	}
}

// Memory is a stored decision / convention / session summary.
type Memory struct {
	ID        int64
	ClientID  string
	ProjectID string
	Type      MemoryType
	Content   string
	Rationale string // the "why"
	CreatedAt int64  // unix seconds
	Embedding []float32
}

// Rejection is a rejected approach with its rationale — recap's differentiator
// (decision.md §4). Surfaced at session start so the agent never re-suggests it.
type Rejection struct {
	ID             int64
	ClientID       string
	ProjectID      string
	Approach       string
	ReasonRejected string
	CreatedAt      int64 // unix seconds
	Embedding      []float32
}

// Session is a coding session's lifecycle record.
type Session struct {
	ID        string
	ClientID  string
	ProjectID string
	Summary   string
	StartedAt int64
	EndedAt   int64
}

// encodeEmbedding serializes a float32 vector to a little-endian BLOB. A nil or
// empty vector encodes to nil (stored as SQL NULL).
func encodeEmbedding(v []float32) []byte {
	if len(v) == 0 {
		return nil
	}
	b := make([]byte, 4*len(v))
	for i, f := range v {
		binary.LittleEndian.PutUint32(b[i*4:], math.Float32bits(f))
	}
	return b
}

// decodeEmbedding is the inverse of encodeEmbedding.
func decodeEmbedding(b []byte) []float32 {
	if len(b) < 4 {
		return nil
	}
	v := make([]float32, len(b)/4)
	for i := range v {
		v[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[i*4:]))
	}
	return v
}
