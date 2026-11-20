package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPreviewIsReadOnlyAndMatchesApply(t *testing.T) {
	orig := fakeXBE()
	originalCopy := bytes.Clone(orig)
	p, err := ParseJMP("preview.jmp", []byte(fixtureJMP(orig)))
	if err != nil {
		t.Fatal(err)
	}
	report, err := p.Preview(orig, false)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(orig, originalCopy) {
		t.Fatal("preview changed original data")
	}
	if report.SelectedRecords != 3 || report.OriginalSize != int64(len(orig)) || report.PatchedSize != int64(len(orig)+2) || report.AppendedBytes != 2 {
		t.Fatalf("preview stats %+v", report)
	}
	if report.Records[0].Offset != 0x390 || report.Records[1].Offset != 0x220 || report.Records[2].Kind != Append {
		t.Fatalf("preview record offsets wrong %+v", report.Records)
	}
	result, err := p.Apply(orig, false)
	if err != nil {
		t.Fatal(err)
	}
	if report.PatchedCRC32 != fmt.Sprintf("%08X", crc32.ChecksumIEEE(result)) || report.ModifiedBytes != PreviewDifference(orig, result).ModifiedBytes {
		t.Fatalf("preview disagrees with application: %+v", report)
	}
	if !bytes.Equal(orig, originalCopy) {
		t.Fatal("Apply modified source data")
	}
}

func TestPreviewRejectsInvalidInputsBeforeWrite(t *testing.T) {
	orig := fakeXBE()
	p, err := ParseJMP("preview.jmp", []byte(fixtureJMP(orig)))
	if err != nil {
		t.Fatal(err)
	}
	mutated := bytes.Clone(orig)
	mutated[0x390] = 0x13
	if _, err = p.Preview(mutated, false); err == nil || !strings.Contains(err.Error(), "CRC32") {
		t.Fatalf("bad CRC should fail preflight: %v", err)
	}
	p.CRCs = nil
	if _, err = p.Preview(mutated, false); err == nil || !strings.Contains(err.Error(), "search pattern not found") {
		t.Fatalf("missing search should fail preflight: %v", err)
	}
	if _, err = p.Preview(mutated, true); err == nil {
		t.Fatal("override bypassed a missing search match")
	}
}

func TestPerRecordToggleAndMixedGroups(t *testing.T) {
	orig := fakeXBE()
	p, err := ParseJMP("groups.jmp", []byte(fixtureJMP(orig)))
	if err != nil {
		t.Fatal(err)
	}
	if p.Groups[0].Mixed {
		t.Fatal("initial group should be fully enabled")
	}
	if err = p.SetRecordEnabled(0, false); err != nil {
		t.Fatal(err)
	}
	if p.Groups[0].Enabled || !p.Groups[0].Mixed {
		t.Fatalf("expected mixed group %+v", p.Groups[0])
	}
	preview, err := p.Preview(orig, false)
	if err != nil {
		t.Fatal(err)
	}
	if preview.SelectedRecords != 2 {
		t.Fatalf("wrong selected count: %d", preview.SelectedRecords)
	}
	if err = p.SetRecordEnabled(1, false); err != nil {
		t.Fatal(err)
	}
	if p.Groups[0].Enabled || p.Groups[0].Mixed {
		t.Fatal("all disabled group still mixed")
	}
	p.EnableGroup("WIDEscreen", true)
	if !p.Groups[0].Enabled || p.Groups[0].Mixed || !p.Records[0].Enabled || !p.Records[1].Enabled {
		t.Fatal("group toggle did not enable both records")
	}
	if err = p.SetRecordEnabled(-1, true); err == nil {
		t.Fatal("negative record index accepted")
	}
	if err = p.SetRecordEnabled(len(p.Records), true); err == nil {
		t.Fatal("out of bounds record index accepted")
	}
	if p.Groups[1].Enabled {
		t.Fatal("[-] record must still be disabled by default")
	}
}

func TestPreferencesRoundTripAndCorruptFallback(t *testing.T) {
	root := t.TempDir()
	defaults, err := LoadPreferences(root)
	if err != nil || defaults.Workers != 4 || defaults.Filter != 0 {
		t.Fatalf("bad defaults: %+v, %v", defaults, err)
	}
	if err = SavePreferences(root, Preferences{Workers: 7, Filter: 3}); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadPreferences(root)
	if err != nil || loaded.Workers != 7 || loaded.Filter != 3 {
		t.Fatalf("bad roundtrip: %+v, %v", loaded, err)
	}
	saved, err := os.ReadFile(filepath.Join(root, "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	var data map[string]interface{}
	if err = json.Unmarshal(saved, &data); err != nil {
		t.Fatal(err)
	}
	if _, ok := data["override"]; ok {
		t.Fatal("unsafe bypass must never persist")
	}
	if err = SavePreferences(root, Preferences{Workers: 999, Filter: -9}); err != nil {
		t.Fatal(err)
	}
	loaded, err = LoadPreferences(root)
	if err != nil || loaded.Workers != 4 || loaded.Filter != 0 {
		t.Fatalf("invalid values did not fall back: %+v, %v", loaded, err)
	}
	if err = os.WriteFile(filepath.Join(root, "settings.json"), []byte("not json"), 0600); err != nil {
		t.Fatal(err)
	}
	loaded, err = LoadPreferences(root)
	if err == nil || loaded.Workers != 4 {
		t.Fatalf("corrupt settings must return defaults + error: %+v, %v", loaded, err)
	}
}
