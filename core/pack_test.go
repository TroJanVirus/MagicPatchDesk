package core

import (
	"archive/zip"
	"bytes"
	"fmt"
	"hash/crc32"
	"io"
	"strings"
	"testing"
)

func jmpForPack(notes, records string) []byte {
	return []byte("#Jay's Magic Patcher\nsystem=Xbox\ngame-title=Pack Test\nregion=\nversion=\nauthor=Jay\nnotes=" + notes + "\n" + records)
}

func TestFileMentionsInNotesAndComments(t *testing.T) {
	cases := []struct{ notes, records, want string }{
		{"Patch driving.xbe", "#Video\n0x2:9988\n", "driving.xbe"},
		{"Visit https://www.jayxbox.com for fixes", "#Widescreen\n0x2:9988\n", "default.xbe"},
		{"patch default.xbe and media/settings.cfg", "#Values (media/settings.cfg)\n0x2:9988\n", "media/settings.cfg"},
		{"Game patch needs settings.cfg", "#Header\n0x2:9988\n", "settings.cfg"},
	}
	for _, c := range cases {
		p, e := ParseJMP("test.jmp", jmpForPack(c.notes, c.records))
		if e != nil {
			t.Fatal(e)
		}
		targets, e := p.TargetFiles()
		if e != nil || len(targets) != 1 || targets[0] != c.want {
			t.Fatalf("notes=%q targets=%v err=%v", c.notes, targets, e)
		}
	}
	p, e := ParseJMP("ambiguous.jmp", jmpForPack("Patch a.cfg and b.cfg", "#Unspecified\n0x2:CAFE\n"))
	if e != nil {
		t.Fatal(e)
	}
	if _, e = p.TargetFiles(); e == nil {
		t.Fatal("ambiguous notes accepted without record targets")
	}
	p, e = ParseJMP("explicit.jmp", jmpForPack("Patch a.cfg and b.cfg", "#Entry (a.cfg)\n0x2:CAFE\n#Entry (b.cfg)\n0x3:CAFE\n"))
	if e != nil {
		t.Fatal(e)
	}
	targets, e := p.TargetFiles()
	if e != nil || strings.Join(targets, ",") != "a.cfg,b.cfg" {
		t.Fatalf("explicit notes targets %v %v", targets, e)
	}
}

func TestPackedMultiFileIPSSourceFree(t *testing.T) {
	patch := jmpForPack("Edit driving.xbe and settings.cfg", "#Driving (driving.xbe)\n0x2:AABBCC\n#Config (settings.cfg)\n0x0:6677\n")
	plan, e := PlanJMPPack("combo.jmp", patch, 1)
	if e != nil {
		t.Fatal(e)
	}
	if len(plan.Targets) != 2 || len(plan.MissingSources(nil)) != 0 {
		t.Fatalf("unexpected pack plan %+v", plan)
	}
	packed, _, e := BuildJMPPatchPack("combo.jmp", patch, nil, 1, "", true)
	if e != nil {
		t.Fatal(e)
	}
	man, files, e := ReadPatchPack(packed)
	if e != nil {
		t.Fatal(e)
	}
	if len(man.Entries) != 2 || len(files) != 3 {
		t.Fatalf("bad manifest %+v", man)
	}
	orig := map[string][]byte{"driving.xbe": []byte("000000000000"), "settings.cfg": []byte("abcdefgh")}
	results, e := ApplyPatchPack(packed, orig, "")
	if e != nil {
		t.Fatal(e)
	}
	if !bytes.Equal(results["driving.xbe"][2:5], []byte{0xAA, 0xBB, 0xCC}) || !bytes.Equal(results["settings.cfg"][:2], []byte{0x66, 0x77}) {
		t.Fatal("incorrect pack patches")
	}
	if !bytes.Equal(orig["driving.xbe"], []byte("000000000000")) {
		t.Fatal("source mutated")
	}
	// The files in the archive have distinct names and do not contain XBE data.
	zr, e := zip.NewReader(bytes.NewReader(packed), int64(len(packed)))
	if e != nil {
		t.Fatal(e)
	}
	for _, f := range zr.File {
		if strings.Contains(f.Name, "../") {
			t.Fatal("unsafe entry")
		}
		rc, _ := f.Open()
		b, _ := io.ReadAll(rc)
		rc.Close()
		if bytes.Contains(b, []byte("000000000000")) {
			t.Fatal("source data leaked into archive")
		}
	}
}

func TestPackedSourceRequiredAndVerified(t *testing.T) {
	cfg := []byte("First hello second")
	xbe := fakeXBE()
	xbe[100] = 0xEE
	patch := []byte(fmt.Sprintf("#Jay's Magic Patcher\nsystem=Xbox\ngame-title=Pack Test\nregion=\nversion={%08X}\nauthor=Jay\nnotes=Additional file settings.cfg\n#Main (default.xbe)\n0x64:AA\n#Config (settings.cfg)\n68656C6C6F\n574F524C44\n", crc32.ChecksumIEEE(xbe)))
	plan, e := PlanJMPPack("combo.jmp", patch, 1)
	if e != nil {
		t.Fatal(e)
	}
	missing := plan.MissingSources(nil)
	if len(missing) != 1 || missing[0] != "settings.cfg" {
		t.Fatalf("expected settings.cfg missing, got %v", missing)
	}
	if _, _, e = BuildJMPPatchPack("combo.jmp", patch, nil, 1, "", false); e == nil {
		t.Fatal("accepted missing original")
	}
	originals := map[string][]byte{"default.xbe": xbe, "settings.cfg": cfg}
	packed, _, e := BuildJMPPatchPack("combo.jmp", patch, originals, 1, "", true)
	if e != nil {
		t.Fatal(e)
	}
	results, e := ApplyPatchPack(packed, originals, "")
	if e != nil {
		t.Fatal(e)
	}
	if results["default.xbe"][100] != 0xAA || !bytes.Contains(results["settings.cfg"], []byte("WORLD")) {
		t.Fatal("wrong patch results")
	}
	originals["settings.cfg"][0] = 'X'
	if _, e = ApplyPatchPack(packed, originals, ""); e == nil {
		t.Fatal("source checksum mismatch allowed")
	}
}

func TestPackedBPSRequiresEverySource(t *testing.T) {
	patch := jmpForPack("", "#Main (default.xbe)\n0x3:7A\n#Special (driving.xbe)\n0x4:7B\n")
	plan, e := PlanJMPPack("combo.jmp", patch, 4)
	if e != nil {
		t.Fatal(e)
	}
	if len(plan.MissingSources(nil)) != 2 {
		t.Fatal("missing source detection failed")
	}
	files := map[string][]byte{"default.xbe": []byte("01234567"), "driving.xbe": []byte("01234567")}
	packed, _, e := BuildJMPPatchPack("combo.jmp", patch, files, 4, "", false)
	if e != nil {
		t.Fatal(e)
	}
	outputs, e := ApplyPatchPack(packed, files, "")
	if e != nil {
		t.Fatal(e)
	}
	if outputs["default.xbe"][3] != 'z' || outputs["driving.xbe"][4] != '{' {
		t.Fatalf("BPS output not applied: %q %q", outputs["default.xbe"], outputs["driving.xbe"])
	}
}

func TestPackTamperingRejected(t *testing.T) {
	patch := jmpForPack("settings.cfg", "#One\n0x0:FE\n")
	data, _, err := BuildJMPPatchPack("x.jmp", patch, nil, 1, "", false)
	if err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	zw := zip.NewWriter(&out)
	for _, member := range zr.File {
		reader, err := member.Open()
		if err != nil {
			t.Fatal(err)
		}
		value, err := io.ReadAll(reader)
		reader.Close()
		if err != nil {
			t.Fatal(err)
		}
		if member.Name != "manifest.json" {
			value[0] ^= 0xFF
		}
		writer, err := zw.Create(member.Name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := writer.Write(value); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if _, _, err = ReadPatchPack(out.Bytes()); err == nil {
		t.Fatal("tampered patch SHA was accepted")
	}
}

func TestPackPreservesDefaultJMPChecksumWithoutOriginalAtExport(t *testing.T) {
	xbe := fakeXBE()
	j := []byte(fmt.Sprintf("#Jay's Magic Patcher\nsystem=Xbox\ngame-title=Checksum\nregion=\nversion={%08X}\nauthor=Jay\nnotes=Settings and executable\n#Main (default.xbe)\n0x100:AABB\n#Config (settings.cfg)\n0x0:11\n", crc32.ChecksumIEEE(xbe)))
	pack, _, err := BuildJMPPatchPack("mixed.jmp", j, nil, 1, "", false)
	if err != nil {
		t.Fatal(err)
	}
	manifest, _, err := ReadPatchPack(pack)
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.ExpectedCRCs) != 1 {
		t.Fatal("original JMP CRC32 was dropped")
	}
	originals := map[string][]byte{"default.xbe": append([]byte(nil), xbe...), "settings.cfg": []byte("ABCDEF")}
	if _, err = ApplyPatchPack(pack, originals, ""); err != nil {
		t.Fatal(err)
	}
	originals["default.xbe"][500] ^= 0x01
	if _, err = ApplyPatchPack(pack, originals, ""); err == nil {
		t.Fatal("source-free export failed to preserve original checksum constraint")
	}
}

func TestJMPPackSearchReplaceReasonDoesNotGetHiddenByBPSRequirement(t *testing.T){
    patch:=jmpForPack("Edit config.cfg",
        "#Find and replace (config.cfg)\n68656C6C6F\n574F524C44\n")
    for _,fmtN:=range []int{1,4}{
        plan,err:=PlanJMPPack("mixed.jmp",patch,fmtN)
        if err!=nil||len(plan.Targets)!=1{
            t.Fatalf("format %d did not plan target: %+v %v",fmtN,plan,err)
        }
        item:=plan.Targets[0]
        if !item.RequiresSource||!strings.Contains(strings.ToLower(item.Reason),"search-and-replace"){
            t.Fatalf("format %d must block without original and explain search-and-replace: %+v",fmtN,item)
        }
        if len(plan.MissingSources(nil))!=1{t.Fatalf("format %d allowed missing original",fmtN)}
    }
}
