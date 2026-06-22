package store

import (
	"os"
	"path/filepath"
	"strings"
)

// DefaultClientID is used when no rule matches the working directory.
const DefaultClientID = "default"

// ClientRule maps a directory path prefix to a client_id. This is how the
// agency/freelancer per-client boundary is configured: directories under
// PathPrefix belong to ClientID and get their own physically-separate DB file.
type ClientRule struct {
	PathPrefix string
	ClientID   string
}

// Config controls per-client database resolution.
type Config struct {
	BaseDir string       // directory holding the <client_id>.db files; default ~/.recap
	Rules   []ClientRule // longest matching PathPrefix wins
}

// Home returns recap's base directory: $RECAP_HOME if set, else ~/.recap.
func Home() (string, error) {
	if h := os.Getenv("RECAP_HOME"); h != "" {
		return h, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".recap"), nil
}

// ResolveClientID returns the client_id for a working directory, choosing the
// longest matching rule prefix and falling back to DefaultClientID.
func (c Config) ResolveClientID(cwd string) string {
	cwd = filepath.Clean(cwd)
	bestLen := -1
	id := DefaultClientID
	for _, r := range c.Rules {
		p := filepath.Clean(r.PathPrefix)
		if cwd == p || strings.HasPrefix(cwd, p+string(filepath.Separator)) {
			if len(p) > bestLen {
				bestLen = len(p)
				id = r.ClientID
			}
		}
	}
	return id
}

// DBPath returns the SQLite file path for a client_id, ensuring BaseDir exists.
func (c Config) DBPath(clientID string) (string, error) {
	base := c.BaseDir
	if base == "" {
		h, err := Home()
		if err != nil {
			return "", err
		}
		base = h
	}
	if err := os.MkdirAll(base, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(base, sanitizeClientID(clientID)+".db"), nil
}

// ResolveProjectID returns a stable project id for cwd: the base name of the
// nearest ancestor that contains a .git entry, else the base name of cwd.
func ResolveProjectID(cwd string) string {
	dir := filepath.Clean(cwd)
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return filepath.Base(dir)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return filepath.Base(filepath.Clean(cwd))
}

// sanitizeClientID keeps a client_id safe to use as a filename.
func sanitizeClientID(id string) string {
	if id == "" {
		return DefaultClientID
	}
	mapped := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			return r
		default:
			return '-'
		}
	}, id)
	if mapped == "" {
		return DefaultClientID
	}
	return mapped
}
