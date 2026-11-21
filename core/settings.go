package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Preferences contains only harmless, portable UI preferences.
// Deliberately do not persist the unsafe CRC32/Title-ID bypass setting.
type Preferences struct {
	Workers int `json:"workers"`
	Filter  int `json:"catalog_filter"`
	GameOutputFormats map[string]uint32 `json:"game_output_formats,omitempty"`
}

func DefaultPreferences() Preferences { return Preferences{Workers: 4, Filter: 0} }

func (p Preferences) Validate() Preferences {
	if p.Workers < 1 || p.Workers > 8 {
		p.Workers = 4
	}
	if p.Filter < 0 || p.Filter > 4 {
		p.Filter = 0
	}
	for kind,mask := range p.GameOutputFormats {
        switch kind {
        case "xbe","folder","xiso","fulliso","cci":
            if mask==0 || mask&^GameOutputAll!=0 {delete(p.GameOutputFormats,kind)}
        default:delete(p.GameOutputFormats,kind)
        }
    }
	return p
}

func LoadPreferences(root string) (Preferences, error) {
	defaults := DefaultPreferences()
	data, err := os.ReadFile(filepath.Join(root, "settings.json"))
	if errors.Is(err, os.ErrNotExist) {
		return defaults, nil
	}
	if err != nil {
		return defaults, err
	}
	var p Preferences
	if err := json.Unmarshal(data, &p); err != nil {
		return defaults, fmt.Errorf("settings.json: %w", err)
	}
	return p.Validate(), nil
}

func SavePreferences(root string, p Preferences) error {
	if root == "" {
		return errors.New("settings folder unavailable")
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(p.Validate(), "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	f, err := os.CreateTemp(root, ".settings-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), filepath.Join(root, "settings.json"))
}
