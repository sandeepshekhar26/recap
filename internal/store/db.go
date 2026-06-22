// Package store is recap's persistence layer: one SQLite database file per
// client_id (the hard isolation boundary), with FTS5 keyword search and BLOB
// embeddings. It uses the pure-Go modernc.org/sqlite driver so the binary stays
// CGo-free and cross-compiles trivially (decision.md §11, docs/TECH.md).
package store

import (
	"context"
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// schemaVersion is bumped when migrate() changes. Stored in PRAGMA user_version.
const schemaVersion = 1

// DB is a handle to one client's SQLite database.
type DB struct {
	sql *sql.DB
}

// Open opens (creating if needed) the SQLite database at path and applies
// migrations. SQLite is single-writer; we cap connections at 1 to avoid
// "database is locked" under recap's low-concurrency local use.
func Open(ctx context.Context, path string) (*DB, error) {
	sqlDB, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite %q: %w", path, err)
	}
	sqlDB.SetMaxOpenConns(1)

	for _, pragma := range []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA busy_timeout=5000",
		"PRAGMA foreign_keys=ON",
	} {
		if _, err := sqlDB.ExecContext(ctx, pragma); err != nil {
			sqlDB.Close()
			return nil, fmt.Errorf("%s: %w", pragma, err)
		}
	}

	db := &DB{sql: sqlDB}
	if err := db.migrate(ctx); err != nil {
		sqlDB.Close()
		return nil, err
	}
	return db, nil
}

// Close closes the underlying database.
func (db *DB) Close() error { return db.sql.Close() }

func (db *DB) migrate(ctx context.Context) error {
	if _, err := db.sql.ExecContext(ctx, schema); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	if _, err := db.sql.ExecContext(ctx, fmt.Sprintf("PRAGMA user_version=%d", schemaVersion)); err != nil {
		return fmt.Errorf("set user_version: %w", err)
	}
	return nil
}

// schema is the v1 DDL. CREATE ... IF NOT EXISTS keeps Open idempotent. The
// memories_fts external-content table mirrors memories via triggers (the
// standard FTS5 pattern) so keyword search stays in sync without duplicate rows.
const schema = `
CREATE TABLE IF NOT EXISTS memories (
  id         INTEGER PRIMARY KEY,
  client_id  TEXT NOT NULL,
  project_id TEXT NOT NULL,
  type       TEXT NOT NULL,
  content    TEXT NOT NULL,
  rationale  TEXT NOT NULL DEFAULT '',
  created_at INTEGER NOT NULL,
  embedding  BLOB
);
CREATE INDEX IF NOT EXISTS idx_memories_project ON memories(client_id, project_id);

CREATE TABLE IF NOT EXISTS rejected_approaches (
  id              INTEGER PRIMARY KEY,
  client_id       TEXT NOT NULL,
  project_id      TEXT NOT NULL,
  approach        TEXT NOT NULL,
  reason_rejected TEXT NOT NULL,
  created_at      INTEGER NOT NULL,
  embedding       BLOB
);
CREATE INDEX IF NOT EXISTS idx_rejections_project ON rejected_approaches(client_id, project_id);

CREATE TABLE IF NOT EXISTS sessions (
  id         TEXT PRIMARY KEY,
  client_id  TEXT NOT NULL,
  project_id TEXT NOT NULL,
  summary    TEXT NOT NULL DEFAULT '',
  started_at INTEGER,
  ended_at   INTEGER
);

CREATE VIRTUAL TABLE IF NOT EXISTS memories_fts USING fts5(
  content, rationale,
  content='memories', content_rowid='id'
);

CREATE TRIGGER IF NOT EXISTS memories_ai AFTER INSERT ON memories BEGIN
  INSERT INTO memories_fts(rowid, content, rationale)
  VALUES (new.id, new.content, new.rationale);
END;
CREATE TRIGGER IF NOT EXISTS memories_ad AFTER DELETE ON memories BEGIN
  INSERT INTO memories_fts(memories_fts, rowid, content, rationale)
  VALUES ('delete', old.id, old.content, old.rationale);
END;
CREATE TRIGGER IF NOT EXISTS memories_au AFTER UPDATE ON memories BEGIN
  INSERT INTO memories_fts(memories_fts, rowid, content, rationale)
  VALUES ('delete', old.id, old.content, old.rationale);
  INSERT INTO memories_fts(rowid, content, rationale)
  VALUES (new.id, new.content, new.rationale);
END;
`
