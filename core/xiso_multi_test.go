package core

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func makeMultifileXISO(t *testing.T, full bool) string {
	t.Helper()
	base := int64(0)
	if full {
		base = 0x02080000
	}
	orig := fakeXBE()
	cfg := []byte("video=480p\n")
	file := filepath.Join(t.TempDir(), "game.iso")
	b := make([]byte, base+0x22000)
	bi := int(base)
	copy(b[bi+0x10000:], volumeMagic)
	copy(b[bi+0x107EC:], volumeMagic)
	binary.LittleEndian.PutUint32(b[bi+0x10014:], 34)
	binary.LittleEndian.PutUint32(b[bi+0x10018:], 2048)
	tab := bi + 34*2048
	// At tree offset 0, settings.cfg is the left child at entry offset 64.
	binary.LittleEndian.PutUint16(b[tab:tab+2], 16)
	binary.LittleEndian.PutUint32(b[tab+4:tab+8], 40)
	binary.LittleEndian.PutUint32(b[tab+8:tab+12], uint32(len(orig)))
	b[tab+12] = 0x20
	b[tab+13] = 11
	copy(b[tab+14:], "default.xbe")
	cfgEntry := tab + 64
	binary.LittleEndian.PutUint32(b[cfgEntry+4:cfgEntry+8], 41)
	binary.LittleEndian.PutUint32(b[cfgEntry+8:cfgEntry+12], uint32(len(cfg)))
	b[cfgEntry+12] = 0x20
	b[cfgEntry+13] = 12
	copy(b[cfgEntry+14:], "settings.cfg")
	copy(b[bi+40*2048:], orig)
	copy(b[bi+41*2048:], cfg)
	if err := os.WriteFile(file, b, 0644); err != nil {
		t.Fatal(err)
	}
	return file
}
func TestPatchImageFilesTwoTargetsAndGrowth(t *testing.T) {
	src := makeMultifileXISO(t, false)
	original, err := OpenXISO(src)
	if err != nil {
		t.Fatal(err)
	}
	xbe, _ := original.DefaultXBE()
	origXBE, _ := original.ReadFile(xbe)
	out := filepath.Join(t.TempDir(), "patched.iso")
	bigger := bytes.Repeat([]byte("config"), 450)
	if err := PatchImageFiles(src, out, map[string][]byte{"settings.cfg": bigger, "default.xbe": append(origXBE, 0x4f)}, 2, nil); err != nil {
		t.Fatal(err)
	}
	changed, err := OpenXISO(out)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := changed.ResolveTarget("settings.cfg")
	if err != nil {
		t.Fatal(err)
	}
	data, err := changed.ReadFile(cfg)
	if err != nil || !bytes.Equal(data, bigger) {
		t.Fatalf("cfg not replaced %v", err)
	}
	newXBE, err := changed.DefaultXBE()
	if err != nil {
		t.Fatal(err)
	}
	b, err := changed.ReadFile(newXBE)
	if err != nil || !bytes.Equal(b, append(origXBE, 0x4f)) {
		t.Fatalf("XBE not replaced %v", err)
	}
	still, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	if len(still) < len(origXBE) || len(b) == len(origXBE) {
		t.Fatal("source not preserved or dest not updated")
	}
}
func TestPatchImageFilesFullDiscGrowthDenied(t *testing.T) {
	src := makeMultifileXISO(t, true)
	out := filepath.Join(t.TempDir(), "unwritten.iso")
	err := PatchImageFiles(src, out, map[string][]byte{"settings.cfg": bytes.Repeat([]byte{1}, 5000)}, 1, nil)
	if err == nil || !strings.Contains(err.Error(), "full-disc") {
		t.Fatalf("expected error, got %v", err)
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatalf("unexpected output present %v", err)
	}
}
