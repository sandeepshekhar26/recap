// Package retrieval implements recap's hybrid recall: FTS5 keyword search and
// vector cosine fused by reciprocal-rank fusion, trimmed to a token budget, with
// active rejected-approaches always surfaced first (decision.md §6, §10).
package retrieval

import (
	"context"

	"github.com/sandeepshekhar26/recap/internal/embed"
	"github.com/sandeepshekhar26/recap/internal/store"
)

// Defaults for Recall when the caller passes zero values.
const (
	DefaultLimit       = 10
	DefaultTokenBudget = 1200
	candidatePoolSize  = 500 // memories pulled for vector scan / keyword depth
)

// Query describes what to recall.
type Query struct {
	ClientID  string
	ProjectID string
	Text      string // free text; empty means "most relevant recent"
	Limit     int    // max memories returned after fusion; default DefaultLimit
}

// Result is what SessionStart injection and memory_recall return: the project's
// active rejections (always included, high-signal) plus fused memories.
type Result struct {
	Rejections []store.Rejection
	Memories   []ScoredMemory
}

// Retriever recalls memories from a store, optionally using an Embedder for the
// vector half of hybrid search.
type Retriever struct {
	store    *store.DB
	embedder embed.Embedder
}

// New builds a Retriever. A nil embedder falls back to embed.Nop (keyword-only).
func New(s *store.DB, e embed.Embedder) *Retriever {
	if e == nil {
		e = embed.Nop{}
	}
	return &Retriever{store: s, embedder: e}
}

// Recall returns the rejections and fused memories most relevant to q, trimmed
// to tokenBudget. Rejections are always included (they are the highest-signal
// records); memories fill the remaining budget by fused rank.
func (r *Retriever) Recall(ctx context.Context, q Query, tokenBudget int) (Result, error) {
	if q.Limit <= 0 {
		q.Limit = DefaultLimit
	}
	if tokenBudget <= 0 {
		tokenBudget = DefaultTokenBudget
	}

	rejections, err := r.store.ListRejections(ctx, q.ClientID, q.ProjectID)
	if err != nil {
		return Result{}, err
	}

	keyword, err := r.keywordHits(ctx, q)
	if err != nil {
		return Result{}, err
	}
	vector, err := r.vectorHits(ctx, q)
	if err != nil {
		return Result{}, err
	}

	fused := fuseRRF(keyword, vector)
	if len(fused) > q.Limit {
		fused = fused[:q.Limit]
	}

	return Result{
		Rejections: rejections,
		Memories:   trimToBudget(rejections, fused, tokenBudget),
	}, nil
}

func (r *Retriever) keywordHits(ctx context.Context, q Query) ([]store.Memory, error) {
	match := sanitizeFTS(q.Text)
	if match == "" {
		// No usable query terms: fall back to recent memories so an empty-text
		// recall (e.g. session start) still has candidates.
		return r.store.ListMemories(ctx, q.ClientID, q.ProjectID, q.Limit)
	}
	return r.store.SearchMemories(ctx, q.ClientID, q.ProjectID, match, candidatePoolSize)
}

func (r *Retriever) vectorHits(ctx context.Context, q Query) ([]store.Memory, error) {
	if q.Text == "" || r.embedder.Dims() == 0 {
		return nil, nil
	}
	vecs, err := r.embedder.Embed(ctx, []string{q.Text})
	if err != nil {
		return nil, err
	}
	if len(vecs) != 1 || len(vecs[0]) == 0 {
		return nil, nil
	}
	candidates, err := r.store.ListMemories(ctx, q.ClientID, q.ProjectID, candidatePoolSize)
	if err != nil {
		return nil, err
	}
	return rankByCosine(vecs[0], candidates), nil
}

// trimToBudget keeps fused memories in rank order until the token budget is
// exhausted. Active rejections are charged against the budget first (they are
// always injected), but at least one memory is kept if any exist.
func trimToBudget(rejections []store.Rejection, fused []ScoredMemory, budget int) []ScoredMemory {
	used := 0
	for _, rj := range rejections {
		used += estimateTokens(rj.Approach) + estimateTokens(rj.ReasonRejected) + 8
	}
	var kept []ScoredMemory
	for _, sm := range fused {
		cost := estimateTokens(sm.Content) + estimateTokens(sm.Rationale) + 8
		if used+cost > budget && len(kept) > 0 {
			break
		}
		used += cost
		kept = append(kept, sm)
	}
	return kept
}
