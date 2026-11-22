package core

import (
    "context"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Cerbios CCIM is an independent, per-2048-byte-sector LZ4 container. This
// implementation follows Team-Resurgent/libCCI's published format. A split set
// consists of independent, contiguous .1.cci, .2.cci, ... parts.
const cciHeaderSize = 32
const cciSectorSize = 2048
const cciMaxSectorSpan = 1 << 20
const cciMaxIndexSize = 64 << 20
const cciDefaultSplitLimit int64 = 0xFFBF6000

type CCIPart struct {
	path        string
	f           *os.File
	startSector int64
	sectors     int64
	index       []uint32
	indexAt     int64
}
type CCIReader struct {
	Parts   []CCIPart
	Sectors int64
}

func CCISourcePath(path string) bool { return strings.EqualFold(filepath.Ext(path), ".cci") }

// discoverCCIParts accepts either a standalone .cci or a complete numbered
// set. A missing middle part must never silently truncate the original disc.
func discoverCCIParts(path string) ([]string, error) {
    if !CCISourcePath(path) {return nil,errors.New("expected .cci input")}
    if _,err:=os.Stat(path);err!=nil{return nil,err}
    ext:=filepath.Ext(path)
    stem:=path[:len(path)-len(ext)]
    base:=stem
    selectedPart:=0
    if dot:=strings.LastIndex(stem,".");dot>=0 {
        if n,e:=strconv.Atoi(stem[dot+1:]);e==nil && n>=1 && n<=9999{
            selectedPart=n
            base=stem[:dot]
        }
    }
    first:=base+".1.cci"
    if _,err:=os.Stat(first);err!=nil{
        if selectedPart>0{return nil,fmt.Errorf("split CCI first part is missing: %s",filepath.Base(first))}
        if !os.IsNotExist(err){return nil,err}
        return []string{path},nil
    }
    // When both a standalone image and a numbered set exist, selecting the
    // unsuffixed name is ambiguous: make the user select part 1 explicitly.
    if selectedPart==0{return nil,errors.New("ambiguous CCI source: standalone .cci and .1.cci both exist; select .1.cci explicitly")}
    if selectedPart>64{return nil,errors.New("CCI split part number exceeds the supported 64-part limit")}
    entries,err:=os.ReadDir(filepath.Dir(base))
    if err!=nil{return nil,err}
    prefix:=strings.ToLower(filepath.Base(base))+"."
    parts:=map[int]string{}
    maxPart:=0
    for _,entry:=range entries {
        name:=entry.Name()
        lower:=strings.ToLower(name)
        // "game.cci" is shorter than "game.<part>.cci". Do not
        // slice it as though a numeric part exists (would panic).
        if len(lower)<=len(prefix)+4 || !strings.HasPrefix(lower,prefix) || !strings.HasSuffix(lower,".cci"){continue}
        between:=lower[len(prefix):len(lower)-4]
        n,e:=strconv.Atoi(between)
        if e!=nil||n<1{continue}
        if n>64{return nil,fmt.Errorf("CCI split set exceeds 64 parts: %s",name)}
        if _,exists:=parts[n];exists{return nil,fmt.Errorf("ambiguous duplicate CCI split part %d",n)}
        if !entry.Type().IsRegular(){return nil,fmt.Errorf("non-regular CCI split part: %s",name)}
        parts[n]=filepath.Join(filepath.Dir(base),name)
        if n>maxPart{maxPart=n}
    }
    if maxPart==0{return nil,errors.New("missing CCI split first part")}
    out:=make([]string,0,maxPart)
    for n:=1;n<=maxPart;n++{
        part,exists:=parts[n]
        if !exists{return nil,fmt.Errorf("CCI split set has missing part %d",n)}
        out=append(out,part)
    }
    if selectedPart>len(out){return nil,errors.New("requested CCI split part is missing")}
    return out,nil
}
func (r *CCIReader) Close() error {
	var first error
	for i := range r.Parts {
		if r.Parts[i].f != nil {
			if e := r.Parts[i].f.Close(); first == nil {
				first = e
			}
		}
	}
	r.Parts = nil
	return first
}
func OpenCCI(path string) (*CCIReader, error) {
    return OpenCCIContext(context.Background(),path)
}
func OpenCCIContext(ctx context.Context,path string)(*CCIReader,error){
    if err:=ctx.Err();err!=nil{return nil,err}
	paths, err := discoverCCIParts(path)
	if err != nil {
		return nil, err
	}
	r := &CCIReader{}
	ok := false
	defer func() {
		if !ok {
			r.Close()
		}
	}()
	for _, p := range paths {
        if e:=ctx.Err();e!=nil{return nil,e}
		f, e := os.Open(p)
		if e != nil {
			return nil, e
		}
		part := CCIPart{path: p, f: f, startSector: r.Sectors}
		r.Parts = append(r.Parts, part)
		current := &r.Parts[len(r.Parts)-1]
		info, e := f.Stat()
		if e != nil {
			return nil, e
		}
		if info.Size() < cciHeaderSize+8 {
			return nil, errors.New("truncated CCI")
		}
		var h [cciHeaderSize]byte
		if _, e = f.ReadAt(h[:], 0); e != nil {
			return nil, e
		}
		if string(h[:4]) != "CCIM" || binary.LittleEndian.Uint32(h[4:8]) != 32 || binary.LittleEndian.Uint32(h[24:28]) != 2048 || h[28] != 1 || h[29] != 2 {
			return nil, errors.New("unsupported CCI header")
		}
		size := binary.LittleEndian.Uint64(h[8:16])
		indexAt := binary.LittleEndian.Uint64(h[16:24])
		if size == 0 || size%2048 != 0 || size > 1<<41 {
			return nil, errors.New("invalid CCI uncompressed length")
		}
		sectors := int64(size / 2048)
		indexBytes := (sectors + 1) * 4
		if indexBytes > cciMaxIndexSize || indexAt > uint64(info.Size()) || uint64(indexBytes) > uint64(info.Size())-indexAt {
			return nil, errors.New("CCI index outside image")
		}
        // CCI index allocation can be megabytes per part. Release it on
        // cancellation and stop before queuing additional split parts.
        if e:=ctx.Err();e!=nil{return nil,e}
		buf := make([]byte, indexBytes)
		if _, e = f.ReadAt(buf, int64(indexAt)); e != nil {
			return nil, e
		}
        if e:=ctx.Err();e!=nil{return nil,e}
		idx := make([]uint32, sectors+1)
		var prev int64
		for i := range idx {
            if i&4095==0 {
                if e:=ctx.Err();e!=nil{return nil,e}
            }
			idx[i] = binary.LittleEndian.Uint32(buf[i*4:])
			pos := int64(idx[i]&0x7FFFFFFF) << 2
			if i == 0 {
				if pos < 32 {
					return nil, errors.New("CCI index starts inside header")
				}
			} else {
				span := pos - prev
				if span <= 0 || span > cciMaxSectorSpan || ((idx[i-1]&0x80000000) == 0 && span != 2048) {
					return nil, fmt.Errorf("corrupt CCI sector index at %d", i-1)
				}
			}
			if pos > int64(indexAt) {
				return nil, errors.New("CCI sector index past index table")
			}
			prev = pos
		}
		if int64(idx[len(idx)-1]&0x7FFFFFFF)<<2 != int64(indexAt) {
			return nil, errors.New("CCI sentinel index does not match header")
		}
		current.index = idx
		current.sectors = sectors
		current.indexAt = int64(indexAt)
		r.Sectors += sectors
	}
	ok = true
	return r, nil
}
func (r *CCIReader) ReadSector(sector int64, b []byte) error {
	if len(b) < 2048 {
		return errors.New("CCI sector buffer too small")
	}
	if sector < 0 || sector >= r.Sectors {
		return io.EOF
	}
	for _, p := range r.Parts {
		if sector < p.startSector || sector >= p.startSector+p.sectors {
			continue
		}
		i := sector - p.startSector
		start := int64(p.index[i]&0x7FFFFFFF) << 2
		end := int64(p.index[i+1]&0x7FFFFFFF) << 2
		length := end - start
		if length < 1 || length > cciMaxSectorSpan {
			return errors.New("invalid CCI sector span")
		}
		compressed := p.index[i]&0x80000000 != 0
		if !compressed {
			if length != 2048 {
				return errors.New("invalid CCI raw sector length")
			}
			_, err := p.f.ReadAt(b[:2048], start)
			return err
		}
		blob := make([]byte, length)
		if _, err := p.f.ReadAt(blob, start); err != nil {
			return err
		}
		pad := int(blob[0])
		if pad > 3 || len(blob) <= 1+pad {
			return errors.New("invalid CCI compressed padding")
		}
		for _, v := range blob[len(blob)-pad:] {
			if v != 0 {
				return errors.New("nonzero CCI padding")
			}
		}
		return lz4DecodeSector(blob[1:len(blob)-pad], b[:2048])
	}
	return io.EOF
}
func lz4DecodeSector(src, dst []byte) error {
	in, out := 0, 0
	for in < len(src) {
		token := int(src[in])
		in++
		literals := token >> 4
		if literals == 15 {
			for {
				if in >= len(src) {
					return io.ErrUnexpectedEOF
				}
				x := int(src[in])
				in++
				literals += x
				if x != 255 {
					break
				}
			}
		}
		if literals < 0 || in+literals > len(src) || out+literals > len(dst) {
			return errors.New("corrupt CCI LZ4 literal")
		}
		copy(dst[out:], src[in:in+literals])
		in += literals
		out += literals
		if in == len(src) {
			break
		}
		if in+2 > len(src) {
			return io.ErrUnexpectedEOF
		}
		offset := int(binary.LittleEndian.Uint16(src[in:]))
		in += 2
		if offset == 0 || offset > out {
			return errors.New("corrupt CCI LZ4 match offset")
		}
		match := (token & 15) + 4
		if token&15 == 15 {
			for {
				if in >= len(src) {
					return io.ErrUnexpectedEOF
				}
				x := int(src[in])
				in++
				match += x
				if x != 255 {
					break
				}
			}
		}
		if match > len(dst)-out {
			return errors.New("CCI LZ4 output overflow")
		}
		for j := 0; j < match; j++ {
			dst[out] = dst[out-offset]
			out++
		}
	}
	if out != len(dst) {
		return fmt.Errorf("CCI LZ4 sector decoded %d bytes instead of %d", out, len(dst))
	}
	return nil
}
func lz4AppendLength(dst []byte, n int) []byte {
	for n >= 255 {
		dst = append(dst, 255)
		n -= 255
	}
	return append(dst, byte(n))
}
func lz4CompressSector(src []byte) []byte {
	const hashBits = 12
	var hash [1 << hashBits]int
	for i := range hash {
		hash[i] = -1
	}
	out := make([]byte, 0, len(src))
	anchor, i := 0, 0
	for i+12 < len(src) {
		value := binary.LittleEndian.Uint32(src[i:])
		h := int((value * 2654435761) >> (32 - hashBits))
		prev := hash[h]
		hash[h] = i
		if prev < 0 || i-prev > 65535 || !bytes.Equal(src[prev:prev+4], src[i:i+4]) {
			i++
			continue
		}
		run := 4
		for i+run < len(src) && src[prev+run] == src[i+run] {
			run++
		}
		lit := i - anchor
		match := run - 4
		token := byte(0)
		if lit < 15 {
			token = byte(lit << 4)
		} else {
			token = 15 << 4
		}
		if match < 15 {
			token |= byte(match)
		} else {
			token |= 15
		}
		out = append(out, token)
		if lit >= 15 {
			out = lz4AppendLength(out, lit-15)
		}
		out = append(out, src[anchor:i]...)
		out = append(out, byte(i-prev), byte((i-prev)>>8))
		if match >= 15 {
			out = lz4AppendLength(out, match-15)
		}
		i += run
		anchor = i
	}
	lit := len(src) - anchor
	token := byte(15 << 4)
	if lit < 15 {
		token = byte(lit << 4)
	}
	out = append(out, token)
	if lit >= 15 {
		out = lz4AppendLength(out, lit-15)
	}
	out = append(out, src[anchor:]...)
	return out
}

// ExtractCCI writes decoded sectors to an independent ISO file, preserving
// exact decoded bytes. It never writes to, truncates, or deletes a source part.
func ExtractCCI(input, output string, progress func(int64, int64)) error {
	if strings.EqualFold(filepath.Clean(input), filepath.Clean(output)) {
		return errors.New("source and output must differ")
	}
	r, err := OpenCCI(input)
	if err != nil {
		return err
	}
	defer r.Close()
	f, err := os.OpenFile(output, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	ok := false
	defer func() {
		f.Close()
		if !ok {
			os.Remove(output)
		}
	}()
	block := make([]byte, 2048)
	for i := int64(0); i < r.Sectors; i++ {
		if err = r.ReadSector(i, block); err != nil {
			return err
		}
		if _, err = f.Write(block); err != nil {
			return err
		}
		if progress != nil && i%4096 == 0 {
			progress(i, r.Sectors)
		}
	}
	if err = f.Sync(); err != nil {
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	ok = true
	if progress != nil {
		progress(r.Sectors, r.Sectors)
	}
	return nil
}

// CompressCCI creates a Cerbios-compatible LZ4 CCI with raw fallback for
// incompressible sectors. Multi-part output is split below the FATX file cap.
func CompressCCI(input, output string, progress func(int64, int64)) error {
	return compressCCIWithLimit(input, output, cciDefaultSplitLimit, progress)
}
func compressCCIWithLimit(input, output string, splitLimit int64, progress func(int64, int64)) error {
    if IsNumberedRawImagePart(input) {
        return errors.New("cannot compress only one numbered ISO part; select the complete split game in Patch Library and choose CCI output")
    }
	if splitLimit < 8192 || splitLimit > cciDefaultSplitLimit {
		return errors.New("invalid CCI split limit")
	}
	if !CCISourcePath(output) {
		return errors.New("CCI output must have .cci extension")
	}
	if strings.EqualFold(filepath.Clean(input), filepath.Clean(output)) {
		return errors.New("source and output must differ")
	}
	src, err := os.Open(input)
	if err != nil {
		return err
	}
	defer src.Close()
	stat, err := src.Stat()
	if err != nil {
		return err
	}
	if stat.Size() <= 0 || stat.Size()%2048 != 0 {
		return errors.New("input ISO must be sector-aligned and nonempty")
	}
	totalSectors := stat.Size() / 2048
	if _, err = os.Stat(output); err == nil {
		return errors.New("CCI output already exists")
	}
	base := output[:len(output)-4]
	created := []string{}
	ok := false
	defer func() {
		if !ok {
			for _, p := range created {
				os.Remove(p)
			}
		}
	}()
	var pos int64
	var count int64
	var part int
	var entries []uint32
	var f *os.File
	openPart := func() error {
		part++
		tmp := output
		if part > 1 {
			tmp = fmt.Sprintf("%s.%d.cci", base, part)
		}
		if part > 1 && part == 2 {
			if _, e := os.Stat(base + ".1.cci"); e == nil {
				return errors.New("CCI first split part already exists")
			}
			if e := os.Rename(output, base+".1.cci"); e != nil {
				return e
			}
			created[0] = base + ".1.cci"
		}
		var e error
		f, e = os.OpenFile(tmp, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0644)
		if e != nil {
			return e
		}
		created = append(created, tmp)
		if _, e = f.Write(make([]byte, 32)); e != nil {
			return e
		}
		pos = 32
		count = 0
		entries = nil
		return nil
	}
	closePart := func() error {
		if f == nil {
			return nil
		}
		if count == 0 {
			return errors.New("empty CCI part")
		}
		indexAt := pos
		entries = append(entries, uint32(indexAt>>2))
		for _, v := range entries {
			var four [4]byte
			binary.LittleEndian.PutUint32(four[:], v)
			if _, e := f.Write(four[:]); e != nil {
				return e
			}
		}
		var h [32]byte
		copy(h[:4], "CCIM")
		binary.LittleEndian.PutUint32(h[4:8], 32)
		binary.LittleEndian.PutUint64(h[8:16], uint64(count*2048))
		binary.LittleEndian.PutUint64(h[16:24], uint64(indexAt))
		binary.LittleEndian.PutUint32(h[24:28], 2048)
		h[28] = 1
		h[29] = 2
		if _, e := f.WriteAt(h[:], 0); e != nil {
			return e
		}
		if e := f.Sync(); e != nil {
			return e
		}
		e := f.Close()
		f = nil
		return e
	}
	if err = openPart(); err != nil {
		return err
	}
	raw := make([]byte, 2048)
	for sector := int64(0); sector < totalSectors; sector++ {
		if _, err = io.ReadFull(src, raw); err != nil {
			return err
		}
		compressed := lz4CompressSector(raw)
		pad := (4 - ((len(compressed) + 1) % 4)) % 4
		useCompressed := len(compressed)+1+pad < 2048-4
		block := raw
		if useCompressed {
			block = make([]byte, 1+len(compressed)+pad)
			block[0] = byte(pad)
			copy(block[1:], compressed)
		}
		if pos+int64(len(block))+int64((count+2)*4) > splitLimit {
			if err = closePart(); err != nil {
				return err
			}
			if err = openPart(); err != nil {
				return err
			}
		}
		if pos&3 != 0 {
			return errors.New("internal CCI index misalignment")
		}
		index := uint32(pos >> 2)
		if useCompressed {
			index |= 0x80000000
		}
		entries = append(entries, index)
		if _, err = f.Write(block); err != nil {
			return err
		}
		pos += int64(len(block))
		count++
		if progress != nil && sector%4096 == 0 {
			progress(sector, totalSectors)
		}
	}
	if err = closePart(); err != nil {
		return err
	}
	ok = true
	if progress != nil {
		progress(totalSectors, totalSectors)
	}
	return nil
}
