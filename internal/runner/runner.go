// Package runner loads runner definitions from embedded JSON.
package runner

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/icco/bugsim/runners"
)

// Definition is a declarative runner (runners/<id>.json).
type Definition struct {
	ID       string              `json:"id"`
	Image    string              `json:"image"`
	Network  string              `json:"network,omitempty"` // "none" (default) or "bridge"
	Env      map[string]string   `json:"env,omitempty"`
	Commands map[string][]string `json:"commands"`
}

// Load returns the runner definition for id.
func Load(id string) (*Definition, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, errors.New("runner id is empty")
	}
	name := fmt.Sprintf("%s.json", id)
	raw, err := runners.Files.ReadFile(name)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("unknown runner %q (missing %s)", id, filepath.Join("runners", name))
		}
		return nil, err
	}
	var d Definition
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&d); err != nil {
		return nil, fmt.Errorf("parse runner %s: %w", name, err)
	}
	if d.ID != id {
		return nil, fmt.Errorf("runner id mismatch: file %q declares id %q", name, d.ID)
	}
	if strings.TrimSpace(d.Image) == "" {
		return nil, fmt.Errorf("runner %q must define image", id)
	}
	if len(d.Commands["test"]) == 0 {
		return nil, fmt.Errorf("runner %q must define commands.test", id)
	}
	if d.Network == "" {
		d.Network = "none"
	}
	switch d.Network {
	case "none", "bridge":
	default:
		return nil, fmt.Errorf("runner %q: network must be \"none\" or \"bridge\"", id)
	}
	return &d, nil
}
