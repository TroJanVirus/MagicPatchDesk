package core

import (
	"bytes"
	"strings"
	"testing"
)

func TestAllFormatsPackIncludesAllVariantsAndOneJMP(t *testing.T) {
	jmp := jmpForPack("Patch config settings.cfg and executable default.xbe", "#Main (default.xbe)\n0x4:AA\n#Config (settings.cfg)\n0x5:BB\n")
	originals := map[string][]byte{"default.xbe": []byte("0123456789"), "settings.cfg": []byte("abcdefghij")}
	if _, _, e := BuildJMPAllFormatsPack("combo.jmp", jmp, nil, "", true); e == nil {
		t.Fatal("built complete BPS/Xdelta pack without source")
	}
	output, manifest, err := BuildJMPAllFormatsPack("combo.jmp", jmp, originals, "", true)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Schema != "magicpatchdesk-pack-v2" || manifest.OriginalJMP == "" {
		t.Fatal("bad v2 manifest")
	}
	m, variants, err := ReadAllPackVariants(output)
	if err != nil {
		t.Fatal(err)
	}
	if len(variants) != 2 {
		t.Fatalf("want 2 targets, got %d", len(variants))
	}
	for name, v := range variants {
		if len(v) != 3 {
			t.Fatalf("want IPS, BPS, Xdelta, got %v for %s", v, name)
		}
		for _, format := range []string{"ips", "bps", "xdelta"} {
			if len(v[format]) == 0 {
				t.Fatalf("missing %s on %s", format, name)
			}
		}
	}
	if len(m.Omitted) != 2 {
		t.Fatalf("want eips omitted without password, got %+v", m.Omitted)
	}
	patched, err := ApplyPatchPack(output, originals, "")
	if err != nil {
		t.Fatal(err)
	}
	if patched["default.xbe"][4] != 0xAA || patched["settings.cfg"][5] != 0xBB {
		t.Fatal("wrong outputs")
	}
	if !bytes.Equal(originals["default.xbe"], []byte("0123456789")) {
		t.Fatal("source modified")
	}
}
func TestAllFormatsPackWithEIPSAndBackwardsCompatibility(t *testing.T) {
	jmp := jmpForPack("", "#Edit\n0x1:FF\n")
	original := map[string][]byte{"default.xbe": []byte("abcdefgh")}
	b, m, e := BuildJMPAllFormatsPack("x.jmp", jmp, original, "abcdefgh", false)
	if e != nil {
		t.Fatal(e)
	}
	if len(m.Entries) != 4 || len(m.Omitted) != 0 {
		t.Fatalf("formats %+v, omitted %+v", m.Entries, m.Omitted)
	}
	_, variants, e := ReadAllPackVariants(b)
	if e != nil {
		t.Fatal(e)
	}
	if len(variants["default.xbe"]["eips"]) == 0 {
		t.Fatal("no EIPS")
	}
	out, e := ApplyPatchPack(b, original, "")
	if e != nil {
		t.Fatal("BPS should work without EIPS password: ", e)
	}
	if out["default.xbe"][1] != 0xFF {
		t.Fatal("bad result")
	}
	// Source checksum must not fall back to unverified IPS.
	bad := map[string][]byte{"default.xbe": []byte("zbcdefgh")}
	if _, e = ApplyPatchPack(b, bad, ""); e == nil {
		t.Fatal("accepted wrong original")
	}
	legacy, _, e := BuildJMPPatchPack("x.jmp", jmp, nil, 1, "", false)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = ApplyPatchPack(legacy, original, ""); e != nil {
		t.Fatal("v1 pack broken: ", e)
	}
}
func TestAllFormatsPackWithSourceDependentRecord(t *testing.T) {
	jmp := jmpForPack("", "#Search\n414243\n444546\n")
	b, _, err := BuildJMPAllFormatsPack("search.jmp", jmp, map[string][]byte{"default.xbe": []byte("123ABC456")}, "", false)
	if err != nil {
		t.Fatal(err)
	}
	output, e := ApplyPatchPack(b, map[string][]byte{"default.xbe": []byte("123ABC456")}, "")
	if e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(string(output["default.xbe"]), "DEF") {
		t.Fatal("search replacement missing")
	}
}
