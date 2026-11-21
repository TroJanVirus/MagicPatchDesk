package core

import (
	"bytes"
	"context"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPFRealRocky5Syntax(t *testing.T) {
	// Text copied from the public Auto Modellista and generic patches.
	raw := []byte(`# Author: Nemesis2000
# Credits: Nemesis2000 - http://ps2wide.net/xbox.html
# Info: Patches in widescreen aspect ratio.
# Region: NTSC
# Title: Auto Modellista
# TitleID: 4343000E
# Type: Widescreen Patch

CP|default.xbe|defaultws.xbe
HR|defaultws.xbe|1|ABAAAA3F000000000000000040010000|388EE33F000000000000000040010000|1|164200ABAAAA3FA340174200890D4417|164200388EE33FA340174200890D4417`)
	p, err := ParsePF("auto modellista (n).pf", raw)
	if err != nil {
		t.Fatal(err)
	}
	if p.Header["title"] != "Auto Modellista" || len(p.Operations) != 2 || len(p.Operations[1].Pairs) != 2 {
		t.Fatalf("parsed %+v", p)
	}
	if !p.HasFilesystemOps() {
		t.Fatal("file copy not detected")
	}
	src := bytes.Repeat([]byte{0x00}, 128)
	old := p.Operations[1].Pairs[0].Find
	copy(src[17:], old)
	old2 := p.Operations[1].Pairs[1].Find
	copy(src[48:], old2)
	// Exact XBE validation requires authentic certificate, so use MULTI in this
	// independent format test; title-id validation is covered below.
	result, err := p.ApplyPF(map[string][]byte{"default.xbe": src})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(result.Files["default.xbe"], src) {
		t.Fatal("original XBE overwritten")
	}
	if bytes.Equal(result.Files["defaultws.xbe"], src) {
		t.Fatal("copy not patched")
	}
	if !bytes.Equal(result.Files["defaultws.xbe"][17:17+len(old)], p.Operations[1].Pairs[0].Replace) {
		t.Fatal("HR not applied")
	}
	if len(result.Created) != 1 || result.Created[0] != "defaultws.xbe" {
		t.Fatalf("created=%v", result.Created)
	}
}
func TestPFHexOccurrenceAndDecimalOffsets(t *testing.T) {
	p, err := ParsePF("test.pf", []byte("# TitleID: MULTI\nHR|settings.cfg|2|4142|5A5A|1|4F4B|4849\nOF|settings.cfg|10|11223344|0|FF\n"))
	if err != nil {
		t.Fatal(err)
	}
	out, err := p.ApplyPF(map[string][]byte{"settings.cfg": []byte("AB AB AB OK........")})
	if err != nil {
		t.Fatal(err)
	}
	got := out.Files["settings.cfg"]
	if string(got[1:3]) == "AB" || !bytes.Contains(got, []byte("AB")) {
		t.Fatalf("must patch first 2 not all: %q", got)
	}
	if got[0] != 0xFF || !bytes.Equal(got[10:14], []byte{0x11, 0x22, 0x33, 0x44}) {
		t.Fatalf("wrong decimal offset: %x", got)
	}
}
func TestPFMediaRegionCertificate(t *testing.T) {
	x := fakeXBE()
	p, err := ParsePF("generic.pf", []byte("# TitleID: MULTI\nSM|default.xbe|C00001FF\nSR|default.xbe|80000007\n"))
	if err != nil {
		t.Fatal(err)
	}
	res, err := p.ApplyPF(map[string][]byte{"default.xbe": x})
	if err != nil {
		t.Fatal(err)
	}
	base := binary.LittleEndian.Uint32(x[0x104:])
	cert := binary.LittleEndian.Uint32(x[0x118:])
	off := int(cert - base)
	b := res.Files["default.xbe"]
	if binary.LittleEndian.Uint32(b[off+0x9c:]) != 0xC00001FF || binary.LittleEndian.Uint32(b[off+0xa0:]) != 0x80000007 {
		t.Fatal("wrong certificate flags")
	}
	if bytes.Equal(x, b) {
		t.Fatal("original modified")
	}
}
func TestPFFileMoveRemoveAndAlreadyExists(t *testing.T) {
	p, e := ParsePF("file.pf", []byte("CP|default.xbe|backup.xbe\nRN|settings.cfg|settings.old\nMV|backup.xbe|alternate.xbe\nRM|temp.dat|unused.dat\n"))
	if e != nil {
		t.Fatal(e)
	}
	src := map[string][]byte{"default.xbe": []byte("ABC"), "settings.cfg": []byte("CFG"), "temp.dat": []byte("tmp")}
	out, e := p.ApplyPF(src)
	if e != nil {
		t.Fatal(e)
	}
	if _, ok := out.Files["settings.cfg"]; ok {
		t.Fatal("rename retained old")
	}
	if _, ok := out.Files["backup.xbe"]; ok {
		t.Fatal("move retained old")
	}
	if _, ok := out.Files["temp.dat"]; ok {
		t.Fatal("remove retained old")
	}
	if string(out.Files["alternate.xbe"]) != "ABC" || string(out.Files["settings.old"]) != "CFG" {
		t.Fatal("wrong file copies")
	}
	if _, e = p.ApplyPF(map[string][]byte{"default.xbe": []byte("ABC"), "backup.xbe": []byte("already")}); e == nil {
		t.Fatal("existing copy target overwritten")
	}
}
func TestPFMalformedAndUnsafeRejected(t *testing.T) {
	for _, src := range []string{"HR|a.cfg|1|XX|BB", "OF|settings.cfg|0x10|AA", "OF|../settings.cfg|10|FF", "BAD|default.xbe|FF", "HR|a.cfg|1|AA", "SR|default.xbe|FF", "CP|a.cfg|C:\\Windows\\system32\\x.cfg"} {
		if _, e := ParsePF("test.pf", []byte(src)); e == nil {
			t.Errorf("accepted invalid %q", src)
		}
	}
}
func TestPFGameFolderIsCopiedAndSourceUnchanged(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "game")
	dst := filepath.Join(root, "game-patched")
	if err := os.Mkdir(src, 0755); err != nil {
		t.Fatal(err)
	}
	for name, b := range map[string]string{"default.xbe": "XBEPayload", "config.cfg": "use=0", "README.txt": "unchanged"} {
		if err := os.WriteFile(filepath.Join(src, name), []byte(b), 0644); err != nil {
			t.Fatal(err)
		}
	}
	p, e := ParsePF("test.pf", []byte("CP|default.xbe|alt.xbe\nOF|alt.xbe|0|4142\nHR|config.cfg|1|7573653D30|7573653D31\nRN|README.txt|NOTES.txt\n"))
	if e != nil {
		t.Fatal(e)
	}
	result, e := p.ApplyPFDirectory(context.Background(), src, dst, nil)
	if e != nil {
		t.Fatal(e)
	}
	original, _ := os.ReadFile(filepath.Join(src, "config.cfg"))
	if string(original) != "use=0" {
		t.Fatal("source file modified")
	}
	patched, _ := os.ReadFile(filepath.Join(dst, "config.cfg"))
	if string(patched) != "use=1" {
		t.Fatalf("not patched: %q", patched)
	}
	alt, _ := os.ReadFile(filepath.Join(dst, "alt.xbe"))
	if !bytes.HasPrefix(alt, []byte("AB")) {
		t.Fatalf("alt XBE incorrect %q", alt)
	}
	if _, e = os.Stat(filepath.Join(dst, "README.txt")); !os.IsNotExist(e) {
		t.Fatal("rename retained original")
	}
	if _, e = os.Stat(filepath.Join(src, "README.txt")); e != nil {
		t.Fatal("source altered")
	}
	if len(result.Created) < 1 {
		t.Fatal("no created files reported")
	}
	if _, e = p.ApplyPFDirectory(context.Background(), src, dst, nil); e == nil || !strings.Contains(e.Error(), "exists") {
		t.Fatal("existing destination not protected")
	}
}
func TestPFNoFilesystemOpsCanPatchISO(t *testing.T) {
	iso := makeMultifileXISO(t, false)
	pf, e := ParsePF("test.pf", []byte("# TitleID: MULTI\nOF|settings.cfg|0|FE\n"))
	if e != nil {
		t.Fatal(e)
	}
	dest := filepath.Join(t.TempDir(), "updated.iso")
	_, e = pf.ApplyPFToISO(iso, dest, 1, nil)
	if e != nil {
		t.Fatal(e)
	}
	result, e := OpenXISO(dest)
	if e != nil {
		t.Fatal(e)
	}
	file, e := result.ResolveTarget("settings.cfg")
	if e != nil {
		t.Fatal(e)
	}
	b, e := result.ReadFile(file)
	if e != nil || b[0] != 0xFE {
		t.Fatal("PF ISO bytes incorrect", e)
	}
	copyOnly, e := ParsePF("copy.pf", []byte("CP|default.xbe|backup.xbe\n"))
	if e != nil {
		t.Fatal(e)
	}
	if _, e = copyOnly.ApplyPFToISO(iso, filepath.Join(t.TempDir(), "bad.iso"), 1, nil); e == nil {
		t.Fatal("copy operation incorrectly accepted in ISO")
	}
}
func TestPFToJMPAndBackForSimpleEdits(t *testing.T) {
	pf, e := ParsePF("test.pf", []byte("# Title: Test\n# TitleID: MULTI\nHR|settings.cfg|1|4142|4445\nOF|default.xbe|16|0102\n"))
	if e != nil {
		t.Fatal(e)
	}
	j, e := pf.ToJMP(nil)
	if e != nil {
		t.Fatal(e)
	}
	parsed, e := ParseJMP("converted.jmp", j)
	if e != nil {
		t.Fatal(e)
	}
	targets, e := parsed.TargetFiles()
	if e != nil || strings.Join(targets, ",") != "default.xbe,settings.cfg" {
		t.Fatalf("bad targets %v %v", targets, e)
	}
	source := map[string][]byte{"default.xbe": bytes.Repeat([]byte{0}, 64), "settings.cfg": []byte("AB AB")}
	for _, target := range targets {
		expected, e := pf.ApplyPF(source)
		if e != nil {
			t.Fatal(e)
		}
		got, e := parsed.ApplyToTarget(source[target], target, true)
		if e != nil {
			t.Fatal(e)
		}
		if !bytes.Equal(expected.Files[target], got) {
			t.Fatalf("wrong output %s", target)
		}
	}
	pf2, e := ParsePF("again.pf", mustBytes(t, func() ([]byte, error) { return PFFromJMP(parsed) }))
	if e != nil {
		t.Fatal(e)
	}
	if len(pf2.Operations) != 2 {
		t.Fatal("PF roundtrip records missing")
	}
}
func TestPFConversionRefusesFileOperationsAndContext(t *testing.T) {
	pf, _ := ParsePF("copy.pf", []byte("CP|default.xbe|alternate.xbe\n"))
	if _, e := pf.ToJMP(nil); e == nil {
		t.Fatal("lost CP")
	}
	pf, _ = ParsePF("count.pf", []byte("HR|settings.cfg|2|4142|4344\n"))
	if _, e := pf.ToJMP(nil); e == nil {
		t.Fatal("HR count=2 needs source")
	}
	j, e := pf.ToJMP(map[string][]byte{"settings.cfg": []byte("AB-AB")})
	if e != nil {
		t.Fatal(e)
	}
	jp, e := ParseJMP("count.jmp", j)
	if e != nil {
		t.Fatal(e)
	}
	actual, e := jp.ApplyToTarget([]byte("AB-AB"), "settings.cfg", true)
	if e != nil {
		t.Fatal(e)
	}
	if string(actual) != "CD-CD" {
		t.Fatalf("wrong offset result %q", actual)
	}
}
func mustBytes(t *testing.T, f func() ([]byte, error)) []byte {
	t.Helper()
	b, e := f()
	if e != nil {
		t.Fatal(e)
	}
	return b
}

func TestPFLibraryImportAndCatalog(t *testing.T) {
	root := t.TempDir()
	folder := t.TempDir()
	p := filepath.Join(folder, "sample.pf")
	raw := []byte("# Author: Rocky5\n# Title: Test Game\n# TitleID: 12345678\n# Region: NTSC\n# Type: Widescreen Patch\nOF|default.xbe|10|AABB\n")
	if err := os.WriteFile(p, raw, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := ImportJMP(root, p); err != nil {
		t.Fatal(err)
	}
	catalog, err := LoadCatalog(root, 2)
	if err != nil {
		t.Fatal(err)
	}
	if catalog.PatchCount != 1 || catalog.GameCount != 1 {
		t.Fatalf("catalog %+v", catalog)
	}
	pp := catalog.Games[0].Patches[0]
	if pp.TitleID != "12345678" || pp.Author != "Rocky5" || pp.Records != 1 || pp.Type != "Widescreen Patch" {
		t.Fatalf("PF metadata lost: %+v", pp)
	}
}
