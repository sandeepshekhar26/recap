// Package config loads recap's optional on-disk configuration — chiefly the
// directory→client_id rules that drive per-client isolation — into a store.Config.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/sandeepshekhar26/recap/internal/store"
)

// file is the config filename inside recap's home directory.
const file = "config.json"

// fileShape mirrors config.json on disk.
type fileShape struct {
	BaseDir string `json:"base_dir,omitempty"`
	Clients []struct {
		PathPrefix string `json:"path_prefix"`
		ClientID   string `json:"client_id"`
	} `json:"clients,omitempty"`
}

// Load reads $RECAP_HOME/config.json (or ~/.recap/config.json). A missing file
// is not an error: it yields the zero Config (default client, default base dir).
func Load() (store.Config, error) {
	home, err := store.Home()
	if err != nil {
		return store.Config{}, err
	}
	return loadFrom(filepath.Join(home, file))
}

func loadFrom(path string) (store.Config, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return store.Config{}, nil
	}
	if err != nil {
		return store.Config{}, fmt.Errorf("read config %q: %w", path, err)
	}

	var fc fileShape
	if err := json.Unmarshal(data, &fc); err != nil {
		return store.Config{}, fmt.Errorf("parse config %q: %w", path, err)
	}

	cfg := store.Config{BaseDir: fc.BaseDir}
	for _, c := range fc.Clients {
		cfg.Rules = append(cfg.Rules, store.ClientRule{PathPrefix: c.PathPrefix, ClientID: c.ClientID})
	}
	return cfg, nil
}
