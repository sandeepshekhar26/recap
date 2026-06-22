package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// ErrNotFound is returned when a lookup matches no row.
var ErrNotFound = errors.New("store: not found")

// SaveMemory inserts a memory and returns its new id. CreatedAt defaults to now
// when zero. Type must be valid.
func (db *DB) SaveMemory(ctx context.Context, m Memory) (int64, error) {
	if !m.Type.Valid() {
		return 0, fmt.Errorf("save memory: invalid type %q", m.Type)
	}
	if m.ClientID == "" || m.ProjectID == "" || m.Content == "" {
		return 0, fmt.Errorf("save memory: client_id, project_id and content are required")
	}
	if m.CreatedAt == 0 {
		m.CreatedAt = time.Now().Unix()
	}
	res, err := db.sql.ExecContext(ctx,
		`INSERT INTO memories (client_id, project_id, type, content, rationale, created_at, embedding)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		m.ClientID, m.ProjectID, string(m.Type), m.Content, m.Rationale, m.CreatedAt, encodeEmbedding(m.Embedding))
	if err != nil {
		return 0, fmt.Errorf("save memory: %w", err)
	}
	return res.LastInsertId()
}

// GetMemory fetches a memory by id. Returns ErrNotFound if absent.
func (db *DB) GetMemory(ctx context.Context, id int64) (Memory, error) {
	var (
		m   Memory
		emb []byte
	)
	err := db.sql.QueryRowContext(ctx,
		`SELECT id, client_id, project_id, type, content, rationale, created_at, embedding
		 FROM memories WHERE id = ?`, id).
		Scan(&m.ID, &m.ClientID, &m.ProjectID, &m.Type, &m.Content, &m.Rationale, &m.CreatedAt, &emb)
	if errors.Is(err, sql.ErrNoRows) {
		return Memory{}, ErrNotFound
	}
	if err != nil {
		return Memory{}, fmt.Errorf("get memory: %w", err)
	}
	m.Embedding = decodeEmbedding(emb)
	return m, nil
}

// SaveRejection inserts a rejected approach and returns its new id.
func (db *DB) SaveRejection(ctx context.Context, r Rejection) (int64, error) {
	if r.ClientID == "" || r.ProjectID == "" || r.Approach == "" || r.ReasonRejected == "" {
		return 0, fmt.Errorf("save rejection: client_id, project_id, approach and reason_rejected are required")
	}
	if r.CreatedAt == 0 {
		r.CreatedAt = time.Now().Unix()
	}
	res, err := db.sql.ExecContext(ctx,
		`INSERT INTO rejected_approaches (client_id, project_id, approach, reason_rejected, created_at, embedding)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		r.ClientID, r.ProjectID, r.Approach, r.ReasonRejected, r.CreatedAt, encodeEmbedding(r.Embedding))
	if err != nil {
		return 0, fmt.Errorf("save rejection: %w", err)
	}
	return res.LastInsertId()
}

// ListRejections returns a project's rejected approaches, newest first.
func (db *DB) ListRejections(ctx context.Context, clientID, projectID string) ([]Rejection, error) {
	rows, err := db.sql.QueryContext(ctx,
		`SELECT id, client_id, project_id, approach, reason_rejected, created_at, embedding
		 FROM rejected_approaches
		 WHERE client_id = ? AND project_id = ?
		 ORDER BY created_at DESC, id DESC`, clientID, projectID)
	if err != nil {
		return nil, fmt.Errorf("list rejections: %w", err)
	}
	defer rows.Close()

	var out []Rejection
	for rows.Next() {
		var (
			r   Rejection
			emb []byte
		)
		if err := rows.Scan(&r.ID, &r.ClientID, &r.ProjectID, &r.Approach, &r.ReasonRejected, &r.CreatedAt, &emb); err != nil {
			return nil, fmt.Errorf("list rejections: scan: %w", err)
		}
		r.Embedding = decodeEmbedding(emb)
		out = append(out, r)
	}
	return out, rows.Err()
}

// AllMemories returns every memory in this (per-client) database, newest first.
// Used by the local web viewer.
func (db *DB) AllMemories(ctx context.Context) ([]Memory, error) {
	rows, err := db.sql.QueryContext(ctx,
		`SELECT id, client_id, project_id, type, content, rationale, created_at, embedding
		 FROM memories ORDER BY created_at DESC, id DESC`)
	if err != nil {
		return nil, fmt.Errorf("all memories: %w", err)
	}
	defer rows.Close()

	var out []Memory
	for rows.Next() {
		var (
			m   Memory
			emb []byte
		)
		if err := rows.Scan(&m.ID, &m.ClientID, &m.ProjectID, &m.Type, &m.Content, &m.Rationale, &m.CreatedAt, &emb); err != nil {
			return nil, fmt.Errorf("all memories: scan: %w", err)
		}
		m.Embedding = decodeEmbedding(emb)
		out = append(out, m)
	}
	return out, rows.Err()
}

// AllRejections returns every rejected approach in this database, newest first.
func (db *DB) AllRejections(ctx context.Context) ([]Rejection, error) {
	rows, err := db.sql.QueryContext(ctx,
		`SELECT id, client_id, project_id, approach, reason_rejected, created_at, embedding
		 FROM rejected_approaches ORDER BY created_at DESC, id DESC`)
	if err != nil {
		return nil, fmt.Errorf("all rejections: %w", err)
	}
	defer rows.Close()

	var out []Rejection
	for rows.Next() {
		var (
			r   Rejection
			emb []byte
		)
		if err := rows.Scan(&r.ID, &r.ClientID, &r.ProjectID, &r.Approach, &r.ReasonRejected, &r.CreatedAt, &emb); err != nil {
			return nil, fmt.Errorf("all rejections: scan: %w", err)
		}
		r.Embedding = decodeEmbedding(emb)
		out = append(out, r)
	}
	return out, rows.Err()
}

// DeleteMemory removes a memory by id (FTS index is kept in sync by triggers).
// Deleting a missing id is not an error.
func (db *DB) DeleteMemory(ctx context.Context, id int64) error {
	if _, err := db.sql.ExecContext(ctx, `DELETE FROM memories WHERE id = ?`, id); err != nil {
		return fmt.Errorf("delete memory: %w", err)
	}
	return nil
}

// DeleteRejection removes a rejected approach by id.
func (db *DB) DeleteRejection(ctx context.Context, id int64) error {
	if _, err := db.sql.ExecContext(ctx, `DELETE FROM rejected_approaches WHERE id = ?`, id); err != nil {
		return fmt.Errorf("delete rejection: %w", err)
	}
	return nil
}

// UpsertSession inserts a session row or updates its summary/ended_at if the id
// already exists. Used by the session-end hook for lightweight bookkeeping.
func (db *DB) UpsertSession(ctx context.Context, s Session) error {
	if s.ID == "" {
		return fmt.Errorf("upsert session: id is required")
	}
	_, err := db.sql.ExecContext(ctx,
		`INSERT INTO sessions (id, client_id, project_id, summary, started_at, ended_at)
		 VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		   summary    = excluded.summary,
		   ended_at   = excluded.ended_at`,
		s.ID, s.ClientID, s.ProjectID, s.Summary, s.StartedAt, s.EndedAt)
	if err != nil {
		return fmt.Errorf("upsert session: %w", err)
	}
	return nil
}

// ListMemories returns a project's memories, newest first, up to limit. The
// retrieval layer uses this to gather candidates for vector scoring.
func (db *DB) ListMemories(ctx context.Context, clientID, projectID string, limit int) ([]Memory, error) {
	if limit <= 0 {
		limit = 500
	}
	rows, err := db.sql.QueryContext(ctx,
		`SELECT id, client_id, project_id, type, content, rationale, created_at, embedding
		 FROM memories
		 WHERE client_id = ? AND project_id = ?
		 ORDER BY created_at DESC, id DESC
		 LIMIT ?`, clientID, projectID, limit)
	if err != nil {
		return nil, fmt.Errorf("list memories: %w", err)
	}
	defer rows.Close()

	var out []Memory
	for rows.Next() {
		var (
			m   Memory
			emb []byte
		)
		if err := rows.Scan(&m.ID, &m.ClientID, &m.ProjectID, &m.Type, &m.Content, &m.Rationale, &m.CreatedAt, &emb); err != nil {
			return nil, fmt.Errorf("list memories: scan: %w", err)
		}
		m.Embedding = decodeEmbedding(emb)
		out = append(out, m)
	}
	return out, rows.Err()
}

// SearchMemories runs an FTS5 keyword query within one project, best match
// first (BM25). This is the keyword half of the hybrid retrieval built in §3;
// query is raw FTS5 match syntax.
func (db *DB) SearchMemories(ctx context.Context, clientID, projectID, query string, limit int) ([]Memory, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := db.sql.QueryContext(ctx,
		`SELECT m.id, m.client_id, m.project_id, m.type, m.content, m.rationale, m.created_at, m.embedding
		 FROM memories_fts f
		 JOIN memories m ON m.id = f.rowid
		 WHERE f.memories_fts MATCH ? AND m.client_id = ? AND m.project_id = ?
		 ORDER BY bm25(f.memories_fts)
		 LIMIT ?`, query, clientID, projectID, limit)
	if err != nil {
		return nil, fmt.Errorf("search memories: %w", err)
	}
	defer rows.Close()

	var out []Memory
	for rows.Next() {
		var (
			m   Memory
			emb []byte
		)
		if err := rows.Scan(&m.ID, &m.ClientID, &m.ProjectID, &m.Type, &m.Content, &m.Rationale, &m.CreatedAt, &emb); err != nil {
			return nil, fmt.Errorf("search memories: scan: %w", err)
		}
		m.Embedding = decodeEmbedding(emb)
		out = append(out, m)
	}
	return out, rows.Err()
}
