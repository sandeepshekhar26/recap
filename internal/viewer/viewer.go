// Package viewer serves recap's local web UI: a JSON API plus a single embedded
// HTML page to browse and delete the current client's memories and rejected
// approaches (decision.md §5 — claude-mem proved users want this).
package viewer

import (
	"context"
	"embed"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/sandeepshekhar26/recap/internal/store"
)

//go:embed index.html
var assets embed.FS

// Server is the viewer's HTTP handler over one client's database.
type Server struct {
	store     *store.DB
	clientID  string
	projectID string
}

// New builds a viewer Server for an open per-client store.
func New(s *store.DB, clientID, projectID string) *Server {
	return &Server{store: s, clientID: clientID, projectID: projectID}
}

// Handler returns the viewer's routes.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/info", s.info)
	mux.HandleFunc("GET /api/memories", s.listMemories)
	mux.HandleFunc("GET /api/rejections", s.listRejections)
	mux.HandleFunc("DELETE /api/memories/{id}", s.deleteMemory)
	mux.HandleFunc("DELETE /api/rejections/{id}", s.deleteRejection)
	mux.HandleFunc("GET /", s.index)
	return mux
}

type memoryDTO struct {
	ID        int64  `json:"id"`
	ProjectID string `json:"project_id"`
	Type      string `json:"type"`
	Content   string `json:"content"`
	Rationale string `json:"rationale"`
	CreatedAt int64  `json:"created_at"`
}

type rejectionDTO struct {
	ID             int64  `json:"id"`
	ProjectID      string `json:"project_id"`
	Approach       string `json:"approach"`
	ReasonRejected string `json:"reason_rejected"`
	CreatedAt      int64  `json:"created_at"`
}

func (s *Server) info(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"client_id":  s.clientID,
		"project_id": s.projectID,
	})
}

func (s *Server) listMemories(w http.ResponseWriter, r *http.Request) {
	ms, err := s.store.AllMemories(r.Context())
	if err != nil {
		httpError(w, err)
		return
	}
	out := make([]memoryDTO, 0, len(ms))
	for _, m := range ms {
		out = append(out, memoryDTO{m.ID, m.ProjectID, string(m.Type), m.Content, m.Rationale, m.CreatedAt})
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) listRejections(w http.ResponseWriter, r *http.Request) {
	rs, err := s.store.AllRejections(r.Context())
	if err != nil {
		httpError(w, err)
		return
	}
	out := make([]rejectionDTO, 0, len(rs))
	for _, r := range rs {
		out = append(out, rejectionDTO{r.ID, r.ProjectID, r.Approach, r.ReasonRejected, r.CreatedAt})
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) deleteMemory(w http.ResponseWriter, r *http.Request) {
	s.delete(w, r, s.store.DeleteMemory)
}

func (s *Server) deleteRejection(w http.ResponseWriter, r *http.Request) {
	s.delete(w, r, s.store.DeleteRejection)
}

func (s *Server) delete(w http.ResponseWriter, r *http.Request, fn func(context.Context, int64) error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := fn(r.Context(), id); err != nil {
		httpError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) index(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	page, err := assets.ReadFile("index.html")
	if err != nil {
		httpError(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(page)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func httpError(w http.ResponseWriter, err error) {
	http.Error(w, err.Error(), http.StatusInternalServerError)
}
