package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func sampleDirectJMP(src []byte) []byte {
	return []byte(fmt.Sprintf("#Jay's Magic Patcher\nsystem=Xbox\ngame-title=Halo 2\nregion=Global\nversion=4D530064 {%08X}\nauthor=Tester\nnotes=Example\n#Widescreen\n0x390:DDCCBBAA\n#Widescreen\n0x600:EEEEEE\n#Disabled [-]\n0x601:77\n", crc32.ChecksumIEEE(src)))
}

func TestSourceFreeOffsetJMPToIPS(t *testing.T) {
	original := fakeXBE()
	raw := sampleDirectJMP(original)
	plan, err := PlanPatchConversion("offset.jmp", raw, 1, "")
	if err != nil || plan.RequiresSource || !plan.SourceRecommended {
		t.Fatalf("bad plan %+v: %v", plan, err)
	}
	out, _, err := ConvertPatchBytes("offset.jmp", raw, nil, 1, "", true, "Halo 2")
	if err != nil {
		t.Fatal(err)
	}
	ips, err := DecodeIPS(out)
	if err != nil {
		t.Fatal(err)
	}
	p, err := ParseJMP("offset.jmp", raw)
	if err != nil {
		t.Fatal(err)
	}
	a, err := p.Apply(original, false)
	if err != nil {
		t.Fatal(err)
	}
	b, err := ips.Apply(original)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a, b) {
		t.Fatal("direct IPS changed output")
	}
	if b[0x601] != 0xEE {
		t.Fatalf("disabled JMP record became enabled: %X", b[0x601])
	}
	if _, _, err := ConvertPatchBytes("offset.jmp", raw, original, 1, "", true, "Halo 2"); err != nil {
		t.Fatal("verified conversion: ", err)
	}
}

func TestDirectIPStoJMPWithoutOriginal(t *testing.T) {
	ip := IPSPatch{Records: []IPSRecord{{Offset: 23, Data: []byte{0x45, 0x45, 0x45, 0x45, 0x45, 0x45}}, {Offset: 0x120, Data: []byte{0xFF, 0x31}}}}
	encoded, err := EncodeIPSPatch(ip, true)
	if err != nil {
		t.Fatal(err)
	}
	out, plan, err := ConvertPatchBytes("sample.ips", encoded, nil, 0, "", true, "Halo")
	if err != nil || plan.RequiresSource {
		t.Fatalf("bad direct IPS to JMP %v %+v", err, plan)
	}
	orig := fakeXBE()
	j, err := ParseJMP("output.jmp", out)
	if err != nil {
		t.Fatal(err)
	}
	b, err := j.Apply(orig, false)
	if err != nil {
		t.Fatal(err)
	}
	a, err := ip.Apply(orig)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a, b) {
		t.Fatal("IPS -> JMP differs")
	}
	if !strings.Contains(string(out), "version=\n") {
		t.Fatal("invented CRC32 without original")
	}
}

func TestDirectIPSReencodingPreservesFinalSize(t *testing.T) {
	patch := IPSPatch{Records: []IPSRecord{{Offset: 12, Data: bytes.Repeat([]byte{0x32}, 70)}}, HasFinalSize: true, FinalSize: 140}
	encoded, err := EncodeIPSPatch(patch, true)
	if err != nil {
		t.Fatal(err)
	}
	for _, to := range []int{1, 3} {
		pass := "correct-horse-battery"
		output, plan, e := ConvertPatchBytes("foo.ips", encoded, nil, to, pass, true, "Halo")
		if e != nil || plan.RequiresSource {
			t.Fatalf("direct IPS target %d: %v %+v", to, e, plan)
		}
		if to == 3 {
			output, e = DecryptIPS(output, pass)
			if e != nil {
				t.Fatal(e)
			}
		}
		parsed, e := DecodeIPS(output)
		if e != nil {
			t.Fatal(e)
		}
		if !parsed.HasFinalSize || parsed.FinalSize != 140 {
			t.Fatal("lost final size")
		}
		a, e := patch.Apply(make([]byte, 200))
		if e != nil {
			t.Fatal(e)
		}
		b, e := parsed.Apply(make([]byte, 200))
		if e != nil {
			t.Fatal(e)
		}
		if !bytes.Equal(a, b) {
			t.Fatal("direct reencoding changed payload")
		}
	}
	_, p, e := ConvertPatchBytes("foo.ips", encoded, nil, 0, "", true, "Halo")
	if e == nil || !p.RequiresSource {
		t.Fatal("IPS truncation to JMP did not require source")
	}
}

func TestSearchAndAppendJMPRequireOriginal(t *testing.T) {
	original := fakeXBE()
	raw := []byte(fixtureJMP(original))
	plan, e := PlanPatchConversion("mixed.jmp", raw, 1, "")
	if e != nil || !plan.RequiresSource {
		t.Fatalf("bad plan %+v: %v", plan, e)
	}
	if _, _, e = ConvertPatchBytes("mixed.jmp", raw, nil, 1, "", true, "Halo"); e == nil {
		t.Fatal("allowed mixed JMP conversion without original")
	}
	if _, _, e = ConvertPatchBytes("mixed.jmp", raw, original, 1, "", true, "Halo"); e != nil {
		t.Fatal("source-based mixed JMP conversion: ", e)
	}
}

func TestBPSAndXdeltaRequireOriginal(t *testing.T) {
	original := fakeXBE()
	changed := append([]byte{}, original...)
	changed[100] = 0xA5
	for _, f := range []struct {
		ext  string
		make func([]byte, []byte) ([]byte, error)
	}{{".bps", EncodeBPS}, {".xdelta", EncodeXdelta}} {
		enc, err := f.make(original, changed)
		if err != nil {
			t.Fatal(err)
		}
		plan, e := PlanPatchConversion("test"+f.ext, enc, 1, "")
		if e != nil || !plan.RequiresSource {
			t.Fatalf("bad plan for %s: %v %+v", f.ext, e, plan)
		}
		if _, _, e := ConvertPatchBytes("test"+f.ext, enc, nil, 1, "", true, "Halo"); e == nil {
			t.Fatal("source was not required for " + f.ext)
		}
		ips, _, e := ConvertPatchBytes("test"+f.ext, enc, original, 1, "", true, "Halo")
		if e != nil {
			t.Fatal(e)
		}
		p, e := DecodeIPS(ips)
		if e != nil {
			t.Fatal(e)
		}
		got, e := p.Apply(original)
		if e != nil || !bytes.Equal(got, changed) {
			t.Fatal("source-based patch conversion differs", e)
		}
	}
}

func TestJMPCRCStillVerifiedWhenOriginalSupplied(t *testing.T) {
	src := fakeXBE()
	jmp := sampleDirectJMP(src)
	src[1] ^= 1
	if _, _, err := ConvertPatchBytes("foo.jmp", jmp, src, 1, "", true, "Halo"); err == nil {
		t.Fatal("CRC mismatch silently accepted")
	}
}

func TestReservedIPSStartDoesNotCreateBrokenIPS(t *testing.T) {
	patch := IPSPatch{Records: []IPSRecord{{Offset: 0x454F46, Data: []byte{7}}}}
	if _, err := EncodeIPSPatch(patch, true); err == nil {
		t.Fatal("reserved marker emitted")
	}
	patch = IPSPatch{Records: []IPSRecord{{Offset: 0x454F45, Data: []byte{1, 2, 3}}}}
	data, err := EncodeIPSPatch(patch, false)
	if err != nil {
		t.Fatal(err)
	}
	p, err := DecodeIPS(data)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(p.Records[0].Data, []byte{1, 2, 3}) {
		t.Fatal("crossing marker broke data")
	}
}

func TestCoverValidationAndTitleMatching(t *testing.T) {
	var im image.Image = image.NewRGBA(image.Rect(0, 0, 24, 32))
	im.(*image.RGBA).Set(0, 0, color.RGBA{R: 255, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, im); err != nil {
		t.Fatal(err)
	}
	if err := validateCoverBytes(buf.Bytes()); err != nil {
		t.Fatal(err)
	}
	if err := validateCoverBytes([]byte("<!DOCTYPE html><html><body>Error 404 Not found</body></html>")); err == nil {
		t.Fatal("cached fake cover accepted")
	}
	root := t.TempDir()
	path := filepath.Join(root, "bad.jpg")
	if err := os.WriteFile(path, []byte("<html>bogus</html>"), 0644); err != nil {
		t.Fatal(err)
	}
	if ValidCoverFile(path) {
		t.Fatal("HTML cache accepted")
	}
	if err := os.WriteFile(path, buf.Bytes(), 0644); err != nil {
		t.Fatal(err)
	}
	if !ValidCoverFile(path) {
		t.Fatal("valid PNG not recognized")
	}
	if CoverMatchKey("Halo: Combat Evolved") != CoverMatchKey("HALO - Combat Evolved") {
		t.Fatal("punctuation prevented title matching")
	}
}

func TestPatchHubNormalizedTitleLinksCover(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "Custom")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	txt := strings.Replace(fixtureJMP(fakeXBE()), "game-title=Halo 2", "game-title=Halo: Combat Evolved", 1)
	if err := os.WriteFile(filepath.Join(dir, "Halo - Combat Evolved {720p}.jmp"), []byte(txt), 0644); err != nil {
		t.Fatal(err)
	}
	idx := HubIndex{Updated: time.Now(), Games: []HubGame{{Title: "Halo - Combat Evolved", Image: "https://jayxbox.com/All-Patches/Halo/cover.jpg"}}}
	data, err := json.Marshal(idx)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "patchhub-index.json"), data, 0644); err != nil {
		t.Fatal(err)
	}
	cat, err := LoadCatalog(root, 2)
	if err != nil {
		t.Fatal(err)
	}
	if cat.GameCount != 1 || cat.Games[0].CoverURL == "" {
		t.Fatalf("cover not merged: %+v", cat)
	}
}

func TestPatchHubTemplateImageParsing(t *testing.T) {
	// Generated markup as found in the supplied PatchHub.php source.
	var html strings.Builder
	for i := 0; i < 60; i++ {
		fmt.Fprintf(&html, "<div class=\"game-card\"><img data-src=\"/All-Patches/Game%%20%d/cover.jpg\" alt=\"Game\"><div class=\"game-title\">Game %d</div><div>Title ID: 4D530064 | Region: USA | Version: 1</div></div>", i, i)
	}
	got, err := ParsePatchHubHTML([]byte(html.String()))
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Games) != 60 || !strings.Contains(got.Games[3].Image, "/All-Patches/") {
		t.Fatalf("image parsing failed %+v", got.Games[3])
	}
}

func TestOriginalForVerificationDoesNotStripNoOpWrites(t *testing.T) {
	original := fakeXBE()
	payload := []byte(fmt.Sprintf("#Jay's Magic Patcher\nsystem=Xbox\ngame-title=Halo 2\nregion=Global\nversion=4D530064 {%08X}\nauthor=Test\nnotes=Explicit no-op must survive\n#Explicit record\n0x390:AABBCCDD\n", crc32.ChecksumIEEE(original)))
	without, _, err := ConvertPatchBytes("noop.jmp", payload, nil, 1, "", true, "Halo")
	if err != nil {
		t.Fatal(err)
	}
	with, _, err := ConvertPatchBytes("noop.jmp", payload, original, 1, "", true, "Halo")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(without, with) {
		t.Fatal("supplying source stripped explicit JMP offset record")
	}
	parsed, err := DecodeIPS(with)
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed.Records) != 1 || parsed.Records[0].Offset != 0x390 {
		t.Fatalf("lost record: %+v", parsed.Records)
	}
}

func TestVerifiedIPStoJMPAddsOriginalChecksum(t *testing.T) {
	original := fakeXBE()
	ips := IPSPatch{Records: []IPSRecord{{Offset: 300, Data: []byte{0x42}}}}
	data, err := EncodeIPSPatch(ips, false)
	if err != nil {
		t.Fatal(err)
	}
	output, _, err := ConvertPatchBytes("one.ips", data, original, 0, "", true, "Halo")
	if err != nil {
		t.Fatal(err)
	}
	p, err := ParseJMP("out.jmp", output)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.CRCs) != 1 || p.CRCs[0] != crc32.ChecksumIEEE(original) {
		t.Fatal("verified original checksum not saved")
	}
	if _, err = p.Apply(original, false); err != nil {
		t.Fatal(err)
	}
}