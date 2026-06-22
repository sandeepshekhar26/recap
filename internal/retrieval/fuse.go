package retrieval

import (
	"math"
	"sort"
	"strings"
	"unicode"

	"github.com/sandeepshekhar26/recap/internal/store"
)

// rrfK is the reciprocal-rank-fusion constant. 60 is the value from the original
// RRF paper and what Zep/Hindsight use (docs/STUDY.md §3).
const rrfK = 60

// ScoredMemory is a memory with its fused relevance score.
type ScoredMemory struct {
	store.Memory
	Score float64
}

// fuseRRF combines several ranked memory lists with reciprocal-rank fusion:
// score(d) = Σ 1/(k + rank_i(d)). Memories are deduped by id; results are sorted
// by score descending (newest id breaks ties for determinism).
func fuseRRF(lists ...[]store.Memory) []ScoredMemory {
	score := map[int64]float64{}
	mem := map[int64]store.Memory{}
	for _, list := range lists {
		for rank, m := range list {
			score[m.ID] += 1.0 / float64(rrfK+rank+1)
			mem[m.ID] = m
		}
	}
	out := make([]ScoredMemory, 0, len(score))
	for id, s := range score {
		out = append(out, ScoredMemory{Memory: mem[id], Score: s})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		return out[i].ID > out[j].ID
	})
	return out
}

// cosine returns the cosine similarity of two equal-length vectors, or 0 if they
// are empty, mismatched, or zero-norm.
func cosine(a, b []float32) float64 {
	if len(a) == 0 || len(a) != len(b) {
		return 0
	}
	var dot, na, nb float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		na += float64(a[i]) * float64(a[i])
		nb += float64(b[i]) * float64(b[i])
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}

// rankByCosine returns the candidates that have embeddings, ordered by cosine
// similarity to query (descending). Candidates without embeddings are skipped.
func rankByCosine(query []float32, candidates []store.Memory) []store.Memory {
	type scored struct {
		m store.Memory
		s float64
	}
	ranked := make([]scored, 0, len(candidates))
	for _, m := range candidates {
		if len(m.Embedding) == 0 {
			continue
		}
		ranked = append(ranked, scored{m, cosine(query, m.Embedding)})
	}
	sort.SliceStable(ranked, func(i, j int) bool { return ranked[i].s > ranked[j].s })
	out := make([]store.Memory, len(ranked))
	for i := range ranked {
		out[i] = ranked[i].m
	}
	return out
}

// estimateTokens is a cheap, model-agnostic token estimate (~4 chars/token).
func estimateTokens(s string) int {
	return (len(s) + 3) / 4
}

// sanitizeFTS turns free text into a safe FTS5 MATCH expression: alphanumeric
// tokens, each quoted, OR-joined. Returns "" when there are no usable tokens
// (caller should then skip keyword search). This avoids FTS5 syntax errors on
// punctuation in user/agent queries.
func sanitizeFTS(text string) string {
	fields := strings.FieldsFunc(text, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
	if len(fields) == 0 {
		return ""
	}
	quoted := make([]string, len(fields))
	for i, f := range fields {
		quoted[i] = `"` + f + `"`
	}
	return strings.Join(quoted, " OR ")
}
