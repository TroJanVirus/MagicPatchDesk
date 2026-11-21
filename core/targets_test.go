package core

import (
	"bytes"
	"fmt"
	"hash/crc32"
	"strings"
	"testing"
)

func TestDetectCommentFilename(t *testing.T) {
	cases := []struct{ comment, want string }{
		{"Widescreen", "default.xbe"},
		{"Widescreen (settings.cfg)", "settings.cfg"},
		{"Widescreen [file=media/SETTINGS.CFG]", "media/settings.cfg"},
		{"Video: default.xbe", "default.xbe"},
		{"10.20 720p standard 1.2", "default.xbe"},
		{"credits www.jayxbox.com", "default.xbe"},
		{"Settings for Data\\CONFIG.ini", "data/config.ini"},
		{"Change settings.cfg: format", "settings.cfg"},
		{"web https://example.xyz/a and settings.cfg", "settings.cfg"},
	}
	for _, tc := range cases {
		got, err := TargetFromComment(tc.comment)
		if err != nil || got != tc.want {
			t.Errorf("%q = %q, %v; want %q", tc.comment, got, err, tc.want)
		}
	}
	if _, err := TargetFromComment("two files a.ini and b.cfg"); err == nil {
		t.Fatal("ambiguous comment accepted")
	}
	if _, err := TargetFromComment("a.ini,b.cfg"); err == nil {
		t.Fatal("comma-separated two targets ignored")
	}
	if _, err := TargetFromComment("../settings.cfg"); err == nil {
		t.Fatal("unsafe comment accepted")
	}
	for _, bad := range []string{"../other.ini", "/boot.ini", "C:\\boot.ini", "COM1.cfg", "assets/../../x.cfg"} {
		if _, err := NormaliseTarget(bad); err == nil {
			t.Errorf("unsafe path accepted: %s", bad)
		}
	}
}

func TestScopedJMPDoesNotCrossPatchFiles(t *testing.T) {
	xbe := fakeXBE()
	cfg := []byte("settingABC in the middle")
	j := fmt.Sprintf("#Jay's Magic Patcher\nsystem=Xbox\ngame-title=Test\nregion=NTSC\nversion={%08X}\nauthor=Test\nnotes=Demo\n#Main code\n0x10:AABBCC\n#Settings (settings.cfg)\n73657474696E67414243\n73657474696E6758595A\n#Another settings.cfg record\n0x0:58595A\n", crc32.ChecksumIEEE(xbe))
	p, err := ParseJMP("mixed.jmp", []byte(j))
	if err != nil {
		t.Fatal(err)
	}
	files, err := p.TargetFiles()
	if err != nil || strings.Join(files, ",") != "default.xbe,settings.cfg" {
		t.Fatalf("targets %v %v", files, err)
	}
	if err := p.CheckSingleFileOnly(); err == nil {
		t.Fatal("multi-target patch accepted as single XBE")
	}
	got, err := p.ApplyToTarget(xbe, "default.xbe", false)
	if err != nil {
		t.Fatal(err)
	}
	if got[0x10] != 0xAA || !bytes.Equal(got[:0x10], xbe[:0x10]) {
		t.Fatal("XBE wrong")
	}
	if _, err := p.ApplyToTarget(cfg, "settings.cfg", false); err != nil {
		t.Fatal(err)
	}
	if _, err := p.ApplyToTarget(cfg, "settings.cfg", true); err != nil {
		t.Fatal(err)
	}
	if _, err := p.ApplyToTarget(cfg, "default.xbe", false); err == nil {
		t.Fatal("CRC guard on default missing")
	}
	p.Records[1].Enabled = false
	if files, err := p.TargetFiles(); err != nil || len(files) != 2 {
		t.Fatalf("skipping one does not skip both %v %v", files, err)
	}
}

func TestSourceRequirementsAndMultiTargetConversion(t *testing.T) {
	header := "#Jay's Magic Patcher\nsystem=Xbox\ngame-title=X\nregion=\nversion=\nauthor=\nnotes=\n"
	search := []byte(header + "#find bytes\n414243\n444546\n")
	plan, err := PlanPatchConversion("find.jmp", search, 1, "")
	if err != nil || !plan.RequiresSource {
		t.Fatalf("find to IPS must require original: %+v %v", plan, err)
	}
	if _, _, err = ConvertPatchBytes("find.jmp", search, nil, 1, "", false, ""); err == nil {
		t.Fatal("find→IPS without original was permitted")
	}
	named := []byte(header + "#widescreen (settings.cfg)\n0x4:AABB\n")
	if _, err = PlanPatchConversion("named.jmp", named, 1, ""); err == nil {
		t.Fatal("named target illegally converted to single-file IPS")
	}
	if plan, err = PlanPatchConversion("named.jmp", named, 0, ""); err != nil || plan.RequiresSource {
		t.Fatalf("JMP copy should preserve named records %+v %v", plan, err)
	}
	plain := []byte(header + "#widescreen\n0x4:AABB\n")
	if plan, err = PlanPatchConversion("offsets.jmp", plain, 1, ""); err != nil || plan.RequiresSource {
		t.Fatalf("offset-only JMP must not require source %+v %v", plan, err)
	}
}
