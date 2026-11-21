package core

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBPSRoundtripAndIntegrity(t *testing.T) {
	source := []byte(strings.Repeat("abcdefgh0123456789", 150))
	cases := [][]byte{
		[]byte(""),
		[]byte("123"),
		append(append([]byte{}, source...), []byte(" append ")...),
		append([]byte("prefix"), source[:48]...),
		[]byte(strings.Repeat("xx", 800)),
		bytes.Clone(source),
	}
	for n, target := range cases {
		patch, err := EncodeBPS(source, target)
		if err != nil {
			t.Fatalf("case %d encode: %v", n, err)
		}
		got, err := DecodeBPS(source, patch)
		if err != nil {
			t.Fatalf("case %d decode: %v", n, err)
		}
		if !bytes.Equal(got, target) {
			t.Fatalf("case %d target mismatch", n)
		}
		badSource := bytes.Clone(source)
		badSource[0] ^= 0xff
		if _, err := DecodeBPS(badSource, patch); err == nil {
			t.Fatalf("case %d accepted wrong source", n)
		}
		corrupted := bytes.Clone(patch)
		corrupted[len(corrupted)-5] ^= 0x11
		if _, err := DecodeBPS(source, corrupted); err == nil {
			t.Fatalf("case %d accepted corrupt patch", n)
		}
	}
}

func bpsRawPatch(source, target []byte, actions []byte) []byte {
	b := append([]byte("BPS1"), appendBPSNumber(nil, uint64(len(source)))...)
	b = appendBPSNumber(b, uint64(len(target)))
	b = appendBPSNumber(b, 0)
	b = append(b, actions...)
	var c [4]byte
	for _, v := range []uint32{crc32.ChecksumIEEE(source), crc32.ChecksumIEEE(target)} {
		binary.LittleEndian.PutUint32(c[:], v)
		b = append(b, c[:]...)
	}
	binary.LittleEndian.PutUint32(c[:], crc32.ChecksumIEEE(b))
	b = append(b, c[:]...)
	return b
}

func TestBPSAllCopyActions(t *testing.T) {
	source := []byte("0123456789")
	// SourceCopy: seek forward by 2, copy 3 bytes, then seek back by 1 and copy 2.
	a := appendBPSNumber(nil, uint64((3-1)<<2|2))
	a = appendBPSNumber(a, 2<<1)
	a = appendBPSNumber(a, uint64((2-1)<<2|2))
	a = appendBPSNumber(a, 1<<1|1)
	patch := bpsRawPatch(source, []byte("23445"), a)
	got, err := DecodeBPS(source, patch)
	if err != nil || !bytes.Equal(got, []byte("23445")) {
		t.Fatalf("SourceCopy: %q, %v", got, err)
	}

	// TargetRead one A, then TargetCopy six overlapping bytes from output offset 0.
	a = appendBPSNumber(nil, uint64((1-1)<<2|1))
	a = append(a, 'A')
	a = appendBPSNumber(a, uint64((6-1)<<2|3))
	a = appendBPSNumber(a, 0)
	patch = bpsRawPatch(source, []byte("AAAAAAA"), a)
	got, err = DecodeBPS(source, patch)
	if err != nil || !bytes.Equal(got, []byte("AAAAAAA")) {
		t.Fatalf("TargetCopy: %q, %v", got, err)
	}
}

func TestBPSMalformed(t *testing.T) {
	s := []byte("abc")
	p, _ := EncodeBPS(s, []byte("axc"))
	for i := 0; i < len(p); i++ {
		if i%3 != 0 {
			continue
		}
		b := bytes.Clone(p[:i])
		if _, err := DecodeBPS(s, b); err == nil {
			t.Fatalf("accepted truncated BPS len=%d", i)
		}
	}
	p = bpsRawPatch(s, []byte("CCCC"), appendBPSNumber(nil, uint64((4-1)<<2|0)))
	if _, err := DecodeBPS(s, p); err == nil {
		t.Fatal("accepted SourceRead beyond source")
	}
}

func TestIPSRLEEncodeAndReservedEOF(t *testing.T) {
	a := bytes.Repeat([]byte{'A'}, 1024)
	b := bytes.Clone(a)
	copy(b[100:700], bytes.Repeat([]byte{'B'}, 600))
	compact, err := EncodeIPSWithRLE(a, b, true)
	if err != nil {
		t.Fatal(err)
	}
	literal, err := EncodeIPSWithRLE(a, b, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(compact) >= len(literal) {
		t.Fatalf("RLE expected smaller: %d vs %d", len(compact), len(literal))
	}
	for _, raw := range [][]byte{compact, literal} {
		ip, err := DecodeIPS(raw)
		if err != nil {
			t.Fatal(err)
		}
		got, err := ip.Apply(a)
		if err != nil || !bytes.Equal(got, b) {
			t.Fatalf("IPS mismatch: %v", err)
		}
	}
	const reserved = 0x454F46
	a = make([]byte, reserved+20)
	b = bytes.Clone(a)
	b[reserved] = 'X'
	b[reserved+1] = 'Y'
	for _, rle := range []bool{false, true} {
		patch, err := EncodeIPSWithRLE(a, b, rle)
		if err != nil {
			t.Fatal(err)
		}
		ip, err := DecodeIPS(patch)
		if err != nil {
			t.Fatal(err)
		}
		got, err := ip.Apply(a)
		if err != nil || !bytes.Equal(got, b) {
			t.Fatalf("reserved offset rle=%v: %v", rle, err)
		}
	}
}

func TestPatchFormatCrossConversions(t *testing.T) {
	stock := []byte(strings.Repeat("magic-source-0000", 100))
	target := bytes.Clone(stock)
	copy(target[10:], []byte("DIFFERENT!!"))
	target = append(target, []byte("end")...)
	tmp := t.TempDir()
	ext := []string{".jmp", ".ips", ".xdelta", ".eips", ".bps"}
	for inputFmt := 0; inputFmt < 5; inputFmt++ {
		raw, err := BuildPatchWithOptions(stock, target, inputFmt, "password123", "Sample", true)
		if err != nil {
			t.Fatalf("input %d build: %v", inputFmt, err)
		}
		path := filepath.Join(tmp, "source"+ext[inputFmt])
		if err := os.WriteFile(path, raw, 0600); err != nil {
			t.Fatal(err)
		}
		decoded, err := ApplyPatchFile(stock, path, "password123", false)
		if err != nil {
			t.Fatalf("input %d apply: %v", inputFmt, err)
		}
		if !bytes.Equal(decoded, target) {
			t.Fatalf("input %d wrong target", inputFmt)
		}
		for outputFmt := 0; outputFmt < 5; outputFmt++ {
			converted, err := BuildPatchWithOptions(stock, decoded, outputFmt, "password123", "Sample", true)
			if err != nil {
				t.Fatalf("%d -> %d build: %v", inputFmt, outputFmt, err)
			}
			out := filepath.Join(tmp, "converted"+ext[outputFmt])
			if err := os.WriteFile(out, converted, 0600); err != nil {
				t.Fatal(err)
			}
			result, err := ApplyPatchFile(stock, out, "password123", false)
			if err != nil || !bytes.Equal(result, target) {
				t.Fatalf("%d -> %d apply: %v", inputFmt, outputFmt, err)
			}
		}
	}
}
