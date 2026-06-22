package viewer

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/sandeepshekhar26/recap/internal/store"
)

func newTestServer(t *testing.T) (*httptest.Server, *store.DB) {
	t.Helper()
	db, err := store.Open(context.Background(), filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	ts := httptest.NewServer(New(db, "acme", "web").Handler())
	t.Cleanup(func() { ts.Close(); db.Close() })
	return ts, db
}

func TestViewerListAndDelete(t *testing.T) {
	ctx := context.Background()
	ts, db := newTestServer(t)

	id, err := db.SaveMemory(ctx, store.Memory{
		ClientID: "acme", ProjectID: "web", Type: store.TypeDecision, Content: "use postgres",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.SaveRejection(ctx, store.Rejection{
		ClientID: "acme", ProjectID: "web", Approach: "use mongo", ReasonRejected: "no txns",
	}); err != nil {
		t.Fatal(err)
	}

	// info reflects scope
	var info map[string]string
	getJSON(t, ts.URL+"/api/info", &info)
	if info["client_id"] != "acme" || info["project_id"] != "web" {
		t.Errorf("info = %v", info)
	}

	// memories list
	var mems []memoryDTO
	getJSON(t, ts.URL+"/api/memories", &mems)
	if len(mems) != 1 || mems[0].Content != "use postgres" {
		t.Fatalf("memories = %+v", mems)
	}

	// rejections list
	var rej []rejectionDTO
	getJSON(t, ts.URL+"/api/rejections", &rej)
	if len(rej) != 1 || rej[0].Approach != "use mongo" {
		t.Fatalf("rejections = %+v", rej)
	}

	// delete the memory
	req, _ := http.NewRequest(http.MethodDelete, ts.URL+"/api/memories/"+itoa(id), nil)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Errorf("delete status = %d, want 204", res.StatusCode)
	}

	getJSON(t, ts.URL+"/api/memories", &mems)
	if len(mems) != 0 {
		t.Errorf("after delete, memories = %+v", mems)
	}
}

func TestViewerServesIndex(t *testing.T) {
	ts, _ := newTestServer(t)
	res, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Errorf("index status = %d", res.StatusCode)
	}
	if ct := res.Header.Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Errorf("index content-type = %q", ct)
	}
}

func getJSON(t *testing.T, url string, v any) {
	t.Helper()
	res, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET %s -> %d", url, res.StatusCode)
	}
	if err := json.NewDecoder(res.Body).Decode(v); err != nil {
		t.Fatalf("decode %s: %v", url, err)
	}
}

func itoa(n int64) string {
	return strconv.FormatInt(n, 10)
}
