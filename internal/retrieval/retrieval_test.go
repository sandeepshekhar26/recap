package retrieval

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/sandeepshekhar26/recap/internal/embed"
	"github.com/sandeepshekhar26/recap/internal/store"
)

func openTemp(t *testing.T) *store.DB {
	t.Helper()
	db, err := store.Open(context.Background(), filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// fakeEmbedder returns a fixed vector for any input — enough to drive the vector
// half of fusion deterministically in tests.
type fakeEmbedder struct{ vec []float32 }

func (f fakeEmbedder) Embed(_ context.Context, texts []string) ([][]float32, error) {
	out := make([][]float32, len(texts))
	for i := range out {
		out[i] = f.vec
	}
	return out, nil
}
func (f fakeEmbedder) Dims() int    { return len(f.vec) }
func (f fakeEmbedder) Name() string { return "fake" }

func TestFuseRRF(t *testing.T) {
	a := store.Memory{ID: 1}
	b := store.Memory{ID: 2}
	c := store.Memory{ID: 3}

	// a is rank 0 in both lists -> should win. b appears in both too; c once.
	fused := fuseRRF(
		[]store.Memory{a, b, c},
		[]store.Memory{a, c, b},
	)
	if len(fused) != 3 {
		t.Fatalf("got %d fused, want 3 (deduped)", len(fused))
	}
	if fused[0].ID != 1 {
		t.Errorf("expected id 1 ranked first, got %d", fused[0].ID)
	}
}

func TestCosine(t *testing.T) {
	if got := cosine([]float32{1, 0}, []float32{1, 0}); got < 0.999 {
		t.Errorf("identical vectors cosine = %v, want ~1", got)
	}
	if got := cosine([]float32{1, 0}, []float32{0, 1}); got != 0 {
		t.Errorf("orthogonal cosine = %v, want 0", got)
	}
	if got := cosine([]float32{1, 0}, []float32{1}); got != 0 {
		t.Errorf("mismatched lengths cosine = %v, want 0", got)
	}
}

func TestSanitizeFTS(t *testing.T) {
	cases := map[string]string{
		"postgres":            `"postgres"`,
		"use Postgres, again": `"use" OR "Postgres" OR "again"`,
		"!!!":                 "",
		"":                    "",
	}
	for in, want := range cases {
		if got := sanitizeFTS(in); got != want {
			t.Errorf("sanitizeFTS(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRecallKeyword(t *testing.T) {
	ctx := context.Background()
	db := openTemp(t)
	r := New(db, embed.Nop{}) // keyword-only

	mustSave(t, db, store.Memory{ClientID: "c", ProjectID: "p", Type: store.TypeDecision, Content: "use postgres for storage"})
	mustSave(t, db, store.Memory{ClientID: "c", ProjectID: "p", Type: store.TypeDecision, Content: "use redis for caching"})
	if _, err := db.SaveRejection(ctx, store.Rejection{ClientID: "c", ProjectID: "p", Approach: "use Mongo", ReasonRejected: "no txns"}); err != nil {
		t.Fatal(err)
	}

	res, err := r.Recall(ctx, Query{ClientID: "c", ProjectID: "p", Text: "postgres"}, 0)
	if err != nil {
		t.Fatalf("recall: %v", err)
	}
	if len(res.Rejections) != 1 {
		t.Errorf("want 1 rejection always surfaced, got %d", len(res.Rejections))
	}
	if len(res.Memories) == 0 || res.Memories[0].Content != "use postgres for storage" {
		t.Errorf("want postgres memory first, got %+v", res.Memories)
	}
}

func TestRecallVectorFusion(t *testing.T) {
	ctx := context.Background()
	db := openTemp(t)
	// query vector points at mem "alpha"
	r := New(db, fakeEmbedder{vec: []float32{1, 0, 0}})

	mustSave(t, db, store.Memory{ClientID: "c", ProjectID: "p", Type: store.TypeDecision, Content: "alpha note", Embedding: []float32{1, 0, 0}})
	mustSave(t, db, store.Memory{ClientID: "c", ProjectID: "p", Type: store.TypeDecision, Content: "beta note", Embedding: []float32{0, 1, 0}})

	// Text has no keyword overlap with content, so ranking is vector-driven.
	res, err := r.Recall(ctx, Query{ClientID: "c", ProjectID: "p", Text: "gamma"}, 0)
	if err != nil {
		t.Fatalf("recall: %v", err)
	}
	if len(res.Memories) == 0 || res.Memories[0].Content != "alpha note" {
		t.Errorf("want alpha note ranked first by cosine, got %+v", res.Memories)
	}
}

func TestRecallBudget(t *testing.T) {
	ctx := context.Background()
	db := openTemp(t)
	r := New(db, embed.Nop{})

	for _, c := range []string{"alpha alpha", "beta beta", "gamma gamma"} {
		mustSave(t, db, store.Memory{ClientID: "c", ProjectID: "p", Type: store.TypeDecision, Content: c})
	}
	// Budget large enough for ~1 short memory only.
	res, err := r.Recall(ctx, Query{ClientID: "c", ProjectID: "p", Text: "alpha beta gamma"}, 10)
	if err != nil {
		t.Fatalf("recall: %v", err)
	}
	if len(res.Memories) != 1 {
		t.Errorf("tiny budget should keep exactly 1 memory, got %d", len(res.Memories))
	}
}

func mustSave(t *testing.T, db *store.DB, m store.Memory) {
	t.Helper()
	if _, err := db.SaveMemory(context.Background(), m); err != nil {
		t.Fatalf("save memory: %v", err)
	}
}
