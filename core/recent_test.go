package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRecentFilesOrderPersistenceAndPrivacyBoundary(t *testing.T) {
	root := t.TempDir()
	a := filepath.Join(root, "game", "default.xbe")
	b := filepath.Join(root, "fix.jmp")
	now := time.Date(2026, 11, 21, 7, 0, 0, 0, time.UTC)
	items, e := AddRecentFile(nil, "input", a, now)
	if e != nil {
		t.Fatal(e)
	}
	items, e = AddRecentFile(items, "patch", b, now.Add(time.Minute))
	if e != nil {
		t.Fatal(e)
	}
	items, e = AddRecentFile(items, "convert_original", a, now.Add(2*time.Minute))
	if e != nil {
		t.Fatal(e)
	}
	if len(items) != 2 || items[0].Kind != "convert_original" || items[1].Kind != "patch" {
		t.Fatalf("bad recents %+v", items)
	}
	if e = SaveRecentFiles(root, items); e != nil {
		t.Fatal(e)
	}
	got, e := LoadRecentFiles(root)
	if e != nil || len(got) != 2 || got[0] != items[0] {
		t.Fatalf("read %+v: %v", got, e)
	}
	raw, e := os.ReadFile(filepath.Join(root, "recent-files.json"))
	if e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(string(raw), "default.xbe") {
		t.Fatal("recent history should contain local paths so they can reopen")
	}
	if e = ClearRecentFiles(root); e != nil {
		t.Fatal(e)
	}
	got, e = LoadRecentFiles(root)
	if e != nil || len(got) != 0 {
		t.Fatalf("clear failed: %+v %v", got, e)
	}
}
func TestRecentFilesCapAndInvalidEntries(t *testing.T) {
	items := []RecentFile{}
	now := time.Now().UTC()
	for i := 0; i < 25; i++ {
		var e error
		items, e = AddRecentFile(items, "input", filepath.Join(t.TempDir(), "x.xbe"), now.Add(time.Duration(i)*time.Second))
		if e != nil {
			t.Fatal(e)
		}
	}
	if len(items) != RecentLimit {
		t.Fatalf("expected %d, got %d", RecentLimit, len(items))
	}
	if _, e := AddRecentFile(nil, "unknown", "x.xbe", now); e == nil {
		t.Fatal("invalid kind accepted")
	}
	if _, e := AddRecentFile(nil, "patch", string([]byte{'x', 0, 'y'}), now); e == nil {
		t.Fatal("NUL accepted")
	}
	root := t.TempDir()
	if e := os.WriteFile(filepath.Join(root, "recent-files.json"), []byte(`{"schema":"unrecognized","items":[]}`), 0600); e != nil {
		t.Fatal(e)
	}
	if _, e := LoadRecentFiles(root); e == nil {
		t.Fatal("invalid history schema accepted")
	}
	if e := ClearRecentFiles(root); e != nil {
		t.Fatal(e)
	}
	if e := SaveRecentFiles(root, append(items, items[0])); e == nil {
		t.Fatal("overflow accepted")
	}
}
func TestRecentFilenamesNotUsedForNetworkOrDiagnostics(t *testing.T) {
	now := time.Now().UTC()
	path := filepath.Join(t.TempDir(), "private-user-secret", "default.xbe")
	history, e := AddRecentFile(nil, "input", path, now)
	if e != nil {
		t.Fatal(e)
	}
	report, e := NewFailureReport(FailureContext{SourcePath: history[0].Path, Operation: "patch.apply"}, os.ErrNotExist)
	if e != nil {
		t.Fatal(e)
	}
	if strings.Contains(report.Summary(), "private-user-secret") {
		t.Fatal("recent path leaked to diagnostics")
	}
}
