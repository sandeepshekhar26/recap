package store

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func openTemp(t *testing.T) *DB {
	t.Helper()
	db, err := Open(context.Background(), filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestSaveGetMemory(t *testing.T) {
	ctx := context.Background()
	db := openTemp(t)

	want := Memory{
		ClientID:  "acme",
		ProjectID: "web",
		Type:      TypeDecision,
		Content:   "use Postgres",
		Rationale: "team knows it",
		Embedding: []float32{0.1, -0.2, 0.3},
	}
	id, err := db.SaveMemory(ctx, want)
	if err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := db.GetMemory(ctx, id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Content != want.Content || got.Type != want.Type || got.Rationale != want.Rationale {
		t.Errorf("got %+v, want content/type/rationale of %+v", got, want)
	}
	if got.CreatedAt == 0 {
		t.Error("created_at should default to now")
	}
	if !reflect.DeepEqual(got.Embedding, want.Embedding) {
		t.Errorf("embedding roundtrip: got %v, want %v", got.Embedding, want.Embedding)
	}
}

func TestSaveMemoryValidation(t *testing.T) {
	ctx := context.Background()
	db := openTemp(t)

	if _, err := db.SaveMemory(ctx, Memory{ClientID: "a", ProjectID: "p", Type: "bogus", Content: "x"}); err == nil {
		t.Error("expected error for invalid type")
	}
	if _, err := db.SaveMemory(ctx, Memory{ClientID: "a", ProjectID: "p", Type: TypeDecision}); err == nil {
		t.Error("expected error for empty content")
	}
	if _, err := db.GetMemory(ctx, 999); err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestRejections(t *testing.T) {
	ctx := context.Background()
	db := openTemp(t)

	if _, err := db.SaveRejection(ctx, Rejection{
		ClientID: "acme", ProjectID: "web", Approach: "use Mongo",
		ReasonRejected: "no transactions", CreatedAt: 100,
	}); err != nil {
		t.Fatalf("save 1: %v", err)
	}
	if _, err := db.SaveRejection(ctx, Rejection{
		ClientID: "acme", ProjectID: "web", Approach: "use a global mutex",
		ReasonRejected: "contention", CreatedAt: 200,
	}); err != nil {
		t.Fatalf("save 2: %v", err)
	}
	// different project — must not show up
	if _, err := db.SaveRejection(ctx, Rejection{
		ClientID: "acme", ProjectID: "mobile", Approach: "x", ReasonRejected: "y",
	}); err != nil {
		t.Fatalf("save 3: %v", err)
	}

	got, err := db.ListRejections(ctx, "acme", "web")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d rejections, want 2", len(got))
	}
	if got[0].Approach != "use a global mutex" { // newest first
		t.Errorf("expected newest first, got %q", got[0].Approach)
	}
}

func TestFTS5Search(t *testing.T) {
	ctx := context.Background()
	db := openTemp(t)

	for _, m := range []Memory{
		{ClientID: "acme", ProjectID: "web", Type: TypeDecision, Content: "adopt the repository pattern for storage"},
		{ClientID: "acme", ProjectID: "web", Type: TypeConvention, Content: "all timestamps are unix seconds"},
		{ClientID: "acme", ProjectID: "web", Type: TypeDecision, Content: "use reciprocal rank fusion for retrieval"},
	} {
		if _, err := db.SaveMemory(ctx, m); err != nil {
			t.Fatalf("save: %v", err)
		}
	}

	hits, err := db.SearchMemories(ctx, "acme", "web", "retrieval", 10)
	if err != nil {
		t.Fatalf("search (FTS5 may be unavailable): %v", err)
	}
	if len(hits) != 1 || hits[0].Content != "use reciprocal rank fusion for retrieval" {
		t.Fatalf("got %+v, want the retrieval memory", hits)
	}
}

// TestIsolation is the core privacy guarantee: each client_id resolves to a
// separate DB file, so one client's data is physically unreachable from another.
func TestIsolation(t *testing.T) {
	ctx := context.Background()
	cfg := Config{BaseDir: t.TempDir()}

	pathA, _ := cfg.DBPath("acme")
	pathB, _ := cfg.DBPath("globex")
	if pathA == pathB {
		t.Fatal("different clients must map to different DB files")
	}

	dbA, err := Open(ctx, pathA)
	if err != nil {
		t.Fatal(err)
	}
	defer dbA.Close()
	dbB, err := Open(ctx, pathB)
	if err != nil {
		t.Fatal(err)
	}
	defer dbB.Close()

	if _, err := dbA.SaveRejection(ctx, Rejection{
		ClientID: "acme", ProjectID: "p", Approach: "secret-A", ReasonRejected: "r",
	}); err != nil {
		t.Fatal(err)
	}

	// Client B's database can never see Client A's rows.
	got, err := dbB.ListRejections(ctx, "acme", "p")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("client B leaked %d of client A's rejections", len(got))
	}
}

func TestResolveClientID(t *testing.T) {
	cfg := Config{Rules: []ClientRule{
		{PathPrefix: "/work", ClientID: "personal"},
		{PathPrefix: "/work/acme", ClientID: "acme"}, // longer prefix wins
	}}
	cases := map[string]string{
		"/work/acme/web":   "acme",
		"/work/acme":       "acme",
		"/work/side":       "personal",
		"/somewhere/else":  DefaultClientID,
		"/work/acmecorp/x": "personal", // not a path-segment match for /work/acme
	}
	for cwd, want := range cases {
		if got := cfg.ResolveClientID(cwd); got != want {
			t.Errorf("ResolveClientID(%q) = %q, want %q", cwd, got, want)
		}
	}
}

func TestResolveProjectID(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "service", "api")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got, want := ResolveProjectID(sub), filepath.Base(root); got != want {
		t.Errorf("ResolveProjectID walked to wrong root: got %q, want %q", got, want)
	}
}
