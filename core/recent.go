package core

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// RecentFile is private on-device UI history, not diagnostic metadata. It is
// never exported with a patch or included in a GitHub issue/report.
type RecentFile struct {
	Kind    string `json:"kind"`
	Path    string `json:"path"`
	UsedUTC string `json:"used_utc"`
}

type recentFileStore struct {
	Schema string       `json:"schema"`
	Items  []RecentFile `json:"items"`
}

const RecentLimit = 18
const recentSchema = "magicpatchdesk-recent-v1"

func validRecentKind(kind string) bool {
	switch kind {
	case "input", "patch", "convert_original", "convert_patch", "convert_modified", "output":
		return true
	}
	return false
}

func cleanRecentPath(path string) (string, error) {
	if path == "" || strings.IndexByte(path, 0) >= 0 {
		return "", errors.New("invalid recent file path")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	abs = filepath.Clean(abs)
	if len(abs) > 32767 || abs == "." {
		return "", errors.New("invalid recent file path")
	}
	return abs, nil
}

func sameRecentPath(a, b string) bool {
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}

// AddRecentFile keeps the newest entry first and deduplicates by *file path*,
// not workflow type: selecting a previously used file updates the kind.
func AddRecentFile(history []RecentFile, kind, path string, now time.Time) ([]RecentFile, error) {
	if !validRecentKind(kind) {
		return nil, errors.New("unknown recent file category")
	}
	clean, err := cleanRecentPath(path)
	if err != nil {
		return nil, err
	}
	out := []RecentFile{{Kind: kind, Path: clean, UsedUTC: now.UTC().Format(time.RFC3339)}}
	for _, item := range history {
		if len(out) >= RecentLimit {
			break
		}
		if !validRecentKind(item.Kind) || !filepath.IsAbs(item.Path) || sameRecentPath(filepath.Clean(item.Path), clean) {
			continue
		}
		found := false
		for _, existing := range out {
			if sameRecentPath(filepath.Clean(existing.Path), filepath.Clean(item.Path)) {
				found = true
				break
			}
		}
		if found {
			continue
		}
		if _, err := time.Parse(time.RFC3339, item.UsedUTC); err != nil {
			continue
		}
		out = append(out, item)
	}
	return out, nil
}

func recentFileLocation(root string) (string, error) {
	if root == "" {
		return "", errors.New("application data directory unavailable")
	}
	return filepath.Join(root, "recent-files.json"), nil
}

func LoadRecentFiles(root string) ([]RecentFile, error) {
	name, err := recentFileLocation(root)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(name)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, 64<<10+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > 64<<10 {
		return nil, errors.New("recent file history exceeds size limit")
	}
	var store recentFileStore
	if err = json.Unmarshal(raw, &store); err != nil {
		return nil, err
	}
	if store.Schema != recentSchema {
		return nil, errors.New("unsupported recent history schema")
	}
	clean := make([]RecentFile, 0, RecentLimit)
	for _, item := range store.Items {
		if !validRecentKind(item.Kind) || !filepath.IsAbs(item.Path) {
			continue
		}
		path, e := cleanRecentPath(item.Path)
		if e != nil {
			continue
		}
		when, e := time.Parse(time.RFC3339, item.UsedUTC)
		if e != nil {
			continue
		}
		// Do not allow an old malformed file to grow a history beyond its cap.
		exists := false
		for _, prior := range clean {
			if sameRecentPath(prior.Path, path) {
				exists = true
				break
			}
		}
		if exists {
			continue
		}
		clean = append(clean, RecentFile{Kind: item.Kind, Path: path, UsedUTC: when.UTC().Format(time.RFC3339)})
		if len(clean) == RecentLimit {
			break
		}
	}
	return clean, nil
}

// SaveRecentFiles writes to a private temporary file first. An interrupted
// update leaves the previous history readable. It never checks file contents.
func SaveRecentFiles(root string, items []RecentFile) error {
	name, err := recentFileLocation(root)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(root, 0700); err != nil {
		return err
	}
	if len(items) > RecentLimit {
		return errors.New("too many recent entries")
	}
	for _, item := range items {
		if !validRecentKind(item.Kind) || !filepath.IsAbs(item.Path) {
			return errors.New("invalid recent history entry")
		}
		if _, e := time.Parse(time.RFC3339, item.UsedUTC); e != nil {
			return errors.New("invalid recent timestamp")
		}
	}
	data, err := json.MarshalIndent(recentFileStore{Schema: recentSchema, Items: items}, "", "  ")
	if err != nil {
		return err
	}
	if len(data) > 64<<10 {
		return errors.New("recent history exceeds size limit")
	}
	file, err := os.CreateTemp(root, ".recent-*.json.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if err = file.Chmod(0600); err != nil {
		file.Close()
		return err
	}
	if _, err = file.Write(append(data, '\n')); err != nil {
		file.Close()
		return err
	}
	if err = file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), name)
}

func ClearRecentFiles(root string) error {
	name, err := recentFileLocation(root)
	if err != nil {
		return err
	}
	if err = os.Remove(name); errors.Is(err, os.ErrNotExist) {
		return nil
	} else {
		return err
	}
}
