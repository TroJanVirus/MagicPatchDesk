package core

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"os"
	"path/filepath"
	"testing"
)

func fakeXBE() []byte {
	b := make([]byte, 0x1000)
	copy(b, "XBEH")
	binary.LittleEndian.PutUint32(b[0x104:], 0x10000)
	binary.LittleEndian.PutUint32(b[0x118:], 0x10180)
	binary.LittleEndian.PutUint32(b[0x188:], 0x4D530064)
	for i, r := range []rune("Halo 2") {
		binary.LittleEndian.PutUint16(b[0x18C+i*2:], uint16(r))
	}
	binary.LittleEndian.PutUint32(b[0x220:], 1)
	copy(b[0x390:], []byte{0xAA, 0xBB, 0xCC, 0xDD})
	return b
}
func fixtureJMP(src []byte) string {
	return fmt.Sprintf("#Jay's Magic Patcher\nsystem=Xbox\ngame-title=Halo 2\nregion=Global\nversion=4D530064 (MS-100) {%08X}\nauthor=Tester\nnotes=Sample\n#Widescreen\nAABBCCDD\nDDCCBBAA\n#Widescreen\n0x220:FF0100C0\n#Disabled [-]\n0x391:11\n#Add\nAPPEND:9988\n", crc32.ChecksumIEEE(src))
}
func TestJMPGroupingCRC(t *testing.T) {
	b := fakeXBE()
	p, err := ParseJMP("test.jmp", []byte(fixtureJMP(b)))
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Groups) != 3 || p.Groups[0].Count != 2 {
		t.Fatalf("grouping broken: %+v", p.Groups)
	}
	v, err := p.Apply(b, false)
	if err != nil {
		t.Fatal(err)
	}
	if got := v[0x390:0x394]; !bytes.Equal(got, []byte{0xDD, 0xCC, 0xBB, 0xAA}) {
		t.Fatalf("search bad %X", got)
	}
	if v[0x391] != 0xCC {
		t.Fatal("disabled record applied")
	}
	if !bytes.Equal(v[len(v)-2:], []byte{0x99, 0x88}) {
		t.Fatal("append failed")
	}
	p.EnableGroup("Widescreen", false)
	v, err = p.Apply(b, false)
	if err != nil {
		t.Fatal(err)
	}
	if v[0x390] != 0xAA {
		t.Fatal("group wasn't disabled")
	}
	p.CRCs = []uint32{0x12345678}
	if _, err = p.Apply(b, false); err == nil {
		t.Fatal("bad CRC was accepted")
	}
	if _, err = p.Apply(b, true); err != nil {
		t.Fatal(err)
	}
}
func TestIPS(t *testing.T) {
	a := make([]byte, 512)
	b := append([]byte{}, a...)
	b[29], b[30] = 0xAA, 0xBB
	encoded, err := EncodeIPS(a, b)
	if err != nil {
		t.Fatal(err)
	}
	p, err := DecodeIPS(encoded)
	if err != nil {
		t.Fatal(err)
	}
	out, err := p.Apply(a)
	if err != nil || !bytes.Equal(out, b) {
		t.Fatalf("IPS roundtrip %v", err)
	}
	raw := append([]byte("PATCH"), []byte{0, 0, 5, 0, 0, 0, 3, 0x7F}...)
	raw = append(raw, []byte("EOF")...)
	p, err = DecodeIPS(raw)
	if err != nil {
		t.Fatal(err)
	}
	out, err = p.Apply(make([]byte, 16))
	if err != nil || !bytes.Equal(out[5:8], []byte{0x7F, 0x7F, 0x7F}) {
		t.Fatalf("RLE failed: %v", err)
	}
}
func makeISO(t *testing.T, src []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "source.iso")
	b := make([]byte, 0x20000)
	base := 0x10000
	copy(b[base:], volumeMagic)
	copy(b[base+0x7EC:], volumeMagic)
	binary.LittleEndian.PutUint32(b[base+0x14:], 34)
	binary.LittleEndian.PutUint32(b[base+0x18:], 2048)
	tab := 34 * 2048
	binary.LittleEndian.PutUint32(b[tab+4:], 40)
	binary.LittleEndian.PutUint32(b[tab+8:], uint32(len(src)))
	b[tab+12] = 0x20
	b[tab+13] = 11
	copy(b[tab+14:], "default.xbe")
	copy(b[40*2048:], src)
	if err := os.WriteFile(path, b, 0644); err != nil {
		t.Fatal(err)
	}
	return path
}
func TestXISO(t *testing.T) {
	src := fakeXBE()
	path := makeISO(t, src)
	im, err := OpenXISO(path)
	if err != nil {
		t.Fatal(err)
	}
	entry, err := im.DefaultXBE()
	if err != nil {
		t.Fatal(err)
	}
	b, err := im.ReadFile(entry)
	if err != nil || !bytes.Equal(b, src) {
		t.Fatalf("bad image data %v", err)
	}
	p, err := ParseJMP("sample.jmp", []byte(fixtureJMP(src)))
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "patched.iso")
	if err := PatchImage(path, out, p, false, 3, nil); err != nil {
		t.Fatal(err)
	}
	im, err = OpenXISO(out)
	if err != nil {
		t.Fatal(err)
	}
	entry, _ = im.DefaultXBE()
	after, _ := im.ReadFile(entry)
	if len(after) != len(src)+2 || after[0x390] != 0xDD {
		t.Fatal("iso patch incorrect")
	}
	if entry.StartSector == 40 {
		t.Fatal("growing XBE was not relocated")
	}
}
func TestConvertJMP(t *testing.T) {
	a := fakeXBE()
	b := append([]byte{}, a...)
	b[0x390] = 0x42
	j, err := DiffToJMP(a, b, "Halo")
	if err != nil {
		t.Fatal(err)
	}
	p, err := ParseJMP("out.jmp", []byte(j))
	if err != nil {
		t.Fatal(err)
	}
	v, err := p.Apply(a, false)
	if err != nil || !bytes.Equal(b, v) {
		t.Fatalf("roundtrip failed: %v", err)
	}
}
func TestXdeltaRoundTrip(t *testing.T) {
	a := bytes.Repeat([]byte{0xA3, 0x10, 0x55, 0x00, 0x23}, 220000)
	b := append([]byte{}, a...)
	for i := 230; i < 200000; i += 13 {
		b[i] = 0x44
	}
	b = append(b, 0x01, 0x02, 0x03)
	delta, err := EncodeXdelta(a, b)
	if err != nil {
		t.Fatal(err)
	}
	out, err := DecodeXdelta(a, delta)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out, b) {
		t.Fatal("VCDIFF roundtrip wrong")
	}
}
func TestEncryptedIPSRoundTrip(t *testing.T) {
	a := make([]byte, 128)
	b := append([]byte{}, a...)
	b[20] = 0xFF
	p, err := EncodeIPS(a, b)
	if err != nil {
		t.Fatal(err)
	}
	enc, err := EncryptIPS(p, "correct-horse")
	if err != nil {
		t.Fatal(err)
	}
	dec, err := DecryptIPS(enc, "correct-horse")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(dec, p) {
		t.Fatal("EIPS content changed")
	}
	if _, err := DecryptIPS(enc, "wrong-password"); err == nil {
		t.Fatal("EIPS accepted wrong password")
	}
}

func TestPatchHubFolderCatalog(t *testing.T) {
	root := t.TempDir()
	folder := filepath.Join(root, "Custom", "PatchHub", "titles", "Halo 2", "NTSC_100_4D530064", "720p")
	if err := os.MkdirAll(folder, 0755); err != nil {
		t.Fatal(err)
	}
	patch := fixtureJMP(fakeXBE())
	if err := os.WriteFile(filepath.Join(folder, "Halo 2 {720p} (NTSC) [Tester].JMP"), []byte(patch), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(folder, "RAM.txt"), []byte("yes\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Custom", "PatchHub", "titles", "Halo 2", "cover.jpg"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}
	got, err := LoadCatalog(root, 4)
	if err != nil {
		t.Fatal(err)
	}
	if got.GameCount != 1 || got.PatchCount != 1 {
		t.Fatalf("unexpected catalogue %+v", got)
	}
	p := got.Games[0].Patches[0]
	if got.Games[0].Cover == "" || p.Game != "Halo 2" || p.TitleID != "4D530064" || p.Region != "NTSC" || p.RAM != "64MB compatible" {
		t.Fatalf("PatchHub metadata missing: %+v", p)
	}
}

func TestXISOGrowthSafetyForFullDisc(t *testing.T) {
	src := fakeXBE()
	plain := makeISO(t, src)
	body, err := os.ReadFile(plain)
	if err != nil {
		t.Fatal(err)
	}
	const base = 0x02080000
	full := filepath.Join(t.TempDir(), "redump.iso")
	f, err := os.Create(full)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate(base + int64(len(body))); err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteAt(body, base); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	p, err := ParseJMP("sample.jmp", []byte(fixtureJMP(src)))
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "patched.iso")
	if err := PatchImage(full, out, p, false, 2, nil); err == nil {
		t.Fatal("growth of full-disc ISO was not prevented")
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatal("failed patch left an output ISO")
	}
	for i := range p.Records {
		if p.Records[i].Kind == Append {
			p.Records[i].Enabled = false
		}
	}
	if err := PatchImage(full, out, p, false, 2, nil); err != nil {
		t.Fatal(err)
	}
	im, err := OpenXISO(out)
	if err != nil {
		t.Fatal(err)
	}
	ent, err := im.DefaultXBE()
	if err != nil {
		t.Fatal(err)
	}
	x, err := im.ReadFile(ent)
	if err != nil || x[0x390] != 0xDD {
		t.Fatalf("patch in full-disc iso failed: %v", err)
	}
}

func TestParsePatchHubMarkup(t *testing.T) {
	var b bytes.Buffer
	for i := 0; i < 55; i++ {
		fmt.Fprintf(&b, `<div class="game-card" data-title="game %d"><img data-src="/All-Patches/titles/Game%%20%d/cover.jpg" alt="Game %d"><div class="game-title">Game %d</div><div class="sub-list"><div class="sub-item">Title ID: 4D530064 | Region: NTSC | Version: 1</div></div></div>`, i, i, i, i)
	}
	idx, err := ParsePatchHubHTML(b.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if len(idx.Games) != 55 || idx.Games[0].Versions[0].TitleID != "4D530064" || idx.Games[54].Title != "Game 54" || idx.Games[12].Image == "" {
		t.Fatalf("PatchHub index parse incorrect: %+v", idx.Games[12])
	}
}
