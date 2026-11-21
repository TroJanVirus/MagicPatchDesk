package core

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCCISectorsRoundTrip(t *testing.T) {
	root := t.TempDir()
	in := filepath.Join(root, "test.iso")
	out := filepath.Join(root, "test.cci")
	restored := filepath.Join(root, "restore.iso")
	payload := make([]byte, 2048*64)
	for i := 0; i < 2048; i++ {
		payload[2048*30+i] = byte(i % 256)
	}
	for i := 0; i < 2048; i++ {
		payload[2048*40+i] = byte(i*17 + i>>5)
	}
	for i := 0; i < 2048; i++ {
		payload[2048*42+i] = byte(3 + i%5)
	}
	if err := os.WriteFile(in, payload, 0600); err != nil {
		t.Fatal(err)
	}
	if err := CompressCCI(in, out, nil); err != nil {
		t.Fatal(err)
	}
	r, err := OpenCCI(out)
	if err != nil {
		t.Fatal(err)
	}
	if r.Sectors != 64 {
		t.Fatalf("wrong sectors %d", r.Sectors)
	}
	b := make([]byte, 2048)
	for i := int64(0); i < 64; i++ {
		if err := r.ReadSector(i, b); err != nil {
			t.Fatal(i, err)
		}
		if !bytes.Equal(b, payload[i*2048:i*2048+2048]) {
			t.Fatalf("wrong sector %d", i)
		}
	}
	r.Close()
	if err := ExtractCCI(out, restored, nil); err != nil {
		t.Fatal(err)
	}
	full, err := os.ReadFile(restored)
	if err != nil || !bytes.Equal(full, payload) {
		t.Fatal("roundtrip failed", err)
	}
	if err := CompressCCI(in, out, nil); err == nil {
		t.Fatal("existing output overwritten")
	}
}
func TestCCIBadHeadersAndBlocks(t *testing.T) {
	root := t.TempDir()
	p := filepath.Join(root, "bad.cci")
	if err := os.WriteFile(p, []byte("garbage"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenCCI(p); err == nil {
		t.Fatal("bad header accepted")
	}
	data := make([]byte, 2048)
	for i := range data {
		data[i] = 0x7f
	}
	encoded := lz4CompressSector(data)
	decoded := make([]byte, 2048)
	if err := lz4DecodeSector(encoded, decoded); err != nil || !bytes.Equal(data, decoded) {
		t.Fatal(err)
	}
	if err := lz4DecodeSector([]byte{0, 0, 0}, decoded); err == nil {
		t.Fatal("zero offset accepted")
	}
	if err := lz4DecodeSector(encoded[:2], decoded); err == nil {
		t.Fatal("truncated stream accepted")
	}
}
func TestCCICompressedIntegrityChecks(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "a.iso")
	out := filepath.Join(dir, "a.cci")
	if err := os.WriteFile(src, bytes.Repeat([]byte{0x42}, 2048*4), 0600); err != nil {
		t.Fatal(err)
	}
	if err := CompressCCI(src, out, nil); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(out)
	idx := int(binary.LittleEndian.Uint64(b[16:24]))
	if idx == 0 {
		t.Fatal("no index")
	}
	b[idx] |= 0x80 // invalid low index byte, offset exceeds bounds
	if err := os.WriteFile(out, b, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenCCI(out); err == nil {
		t.Fatal("corrupt index accepted")
	}
	if !strings.HasSuffix(filepath.Ext(out), "cci") {
		t.Fatal("unexpected extension")
	}
}

func TestCCISplitPartsAndCorruptSet(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "split.iso")
	out := filepath.Join(root, "split.cci")
	data := make([]byte, 2048*99)
	for i := range data {
		data[i] = byte((i*i + 17*i + i/27) & 255)
	} // mostly not compressible
	if e := os.WriteFile(src, data, 0600); e != nil {
		t.Fatal(e)
	}
	if e := compressCCIWithLimit(src, out, 20480, nil); e != nil {
		t.Fatal(e)
	}
	first := filepath.Join(root, "split.1.cci")
	if _, e := os.Stat(first); e != nil {
		t.Fatalf("missing split first part: %v", e)
	}
	r, e := OpenCCI(first)
	if e != nil {
		t.Fatal(e)
	}
	if len(r.Parts) < 2 || r.Sectors != 99 {
		t.Fatalf("wrong split parts: %d sectors %d", len(r.Parts), r.Sectors)
	}
	for i := int64(0); i < r.Sectors; i++ {
		b := make([]byte, 2048)
		if e := r.ReadSector(i, b); e != nil {
			t.Fatal(i, e)
		}
		if !bytes.Equal(b, data[i*2048:i*2048+2048]) {
			t.Fatalf("split sector %d mismatch", i)
		}
	}
	r.Close()
	if e = os.Remove(first); e != nil {
		t.Fatal(e)
	}
	if _, e = OpenCCI(filepath.Join(root, "split.2.cci")); e == nil {
		t.Fatal("missing first part accepted")
	}
}
