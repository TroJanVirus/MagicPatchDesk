package core

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

type IPSRecord struct {
	Offset int
	Data   []byte
}
type IPSPatch struct {
	Records      []IPSRecord
	FinalSize    int
	HasFinalSize bool
}

func DecodeIPS(b []byte) (IPSPatch, error) {
	p := IPSPatch{}
	if len(b) < 8 || string(b[:5]) != "PATCH" {
		return p, errors.New("invalid IPS magic")
	}
	i := 5
	for {
		if i+3 > len(b) {
			return p, errors.New("missing IPS EOF")
		}
		if string(b[i:i+3]) == "EOF" {
			i += 3
			break
		}
		if i+5 > len(b) {
			return p, errors.New("truncated IPS record")
		}
		off := int(b[i])<<16 | int(b[i+1])<<8 | int(b[i+2])
		size := int(binary.BigEndian.Uint16(b[i+3:]))
		i += 5
		var value []byte
		if size == 0 {
			if i+3 > len(b) {
				return p, errors.New("truncated IPS RLE")
			}
			n := int(binary.BigEndian.Uint16(b[i:]))
			value = bytes.Repeat([]byte{b[i+2]}, n)
			i += 3
		} else {
			if i+size > len(b) {
				return p, errors.New("truncated IPS data")
			}
			value = append([]byte{}, b[i:i+size]...)
			i += size
		}
		p.Records = append(p.Records, IPSRecord{off, value})
	}
	if i+3 == len(b) {
		p.HasFinalSize = true
		p.FinalSize = int(b[i])<<16 | int(b[i+1])<<8 | int(b[i+2])
	} else if i != len(b) {
		return p, errors.New("unexpected trailing IPS bytes")
	}
	return p, nil
}
func (p IPSPatch) Apply(src []byte) ([]byte, error) {
	out := append([]byte{}, src...)
	for _, r := range p.Records {
		if r.Offset+len(r.Data) > 64<<20 {
			return nil, fmt.Errorf("IPS output exceeds 64MiB cap")
		}
		if need := r.Offset + len(r.Data); need > len(out) {
			out = append(out, make([]byte, need-len(out))...)
		}
		copy(out[r.Offset:], r.Data)
	}
	if p.HasFinalSize {
		if p.FinalSize < 0 || p.FinalSize > 64<<20 {
			return nil, errors.New("invalid IPS final size")
		}
		if p.FinalSize < len(out) {
			out = out[:p.FinalSize]
		} else {
			out = append(out, make([]byte, p.FinalSize-len(out))...)
		}
	}
	return out, nil
}

// EncodeIPS emits standard PATCH/EOF IPS. Repeated byte ranges are represented
// using IPS RLE records when they occupy fewer bytes than a literal record.
func EncodeIPS(a, b []byte) ([]byte, error) {
	return EncodeIPSWithRLE(a, b, true)
}

// EncodeIPSWithRLE allows users to choose regular IPS records or the standard
// zero-length IPS RLE record encoding. Both are ordinary .ips files.
func EncodeIPSWithRLE(a, b []byte, useRLE bool) ([]byte, error) {
	if len(b) > 0xFFFFFF {
		return nil, errors.New("IPS output exceeds 24-bit final-size limit (16 MiB); use BPS or Xdelta")
	}
	var out bytes.Buffer
	out.WriteString("PATCH")
	const eofOffset = 0x454F46
	writeHead := func(start, count int) {
		out.Write([]byte{byte(start >> 16), byte(start >> 8), byte(start), byte(count >> 8), byte(count)})
	}
	same := func(i int) bool { return i < len(a) && a[i] == b[i] }
	runLength := func(i int) int {
		n := 1
		for i+n < len(b) && n < 0xFFFF && b[i+n] == b[i] {
			n++
		}
		return n
	}
	for i := 0; i < len(b); {
		if same(i) {
			i++
			continue
		}
		// Reserved EOF marker cannot appear as a record offset. Add one byte
		// of unchanged target context to a literal record starting earlier.
		if i == eofOffset {
			start := i - 1
			end := i + 1
			for end < len(b) && end-start < 0xFFFF && !same(end) {
				end++
			}
			writeHead(start, end-start)
			out.Write(b[start:end])
			i = end
			continue
		}
		run := runLength(i)
		if useRLE && run >= 4 {
			out.Write([]byte{byte(i >> 16), byte(i >> 8), byte(i), 0, 0, byte(run >> 8), byte(run), b[i]})
			i += run
			continue
		}
		start := i
		i++
		for i < len(b) && i-start < 0xFFFF && !same(i) {
			if i == eofOffset || (useRLE && runLength(i) >= 4) {
				break
			}
			i++
		}
		writeHead(start, i-start)
		out.Write(b[start:i])
	}
	out.WriteString("EOF")
	if len(a) != len(b) {
		out.Write([]byte{byte(len(b) >> 16), byte(len(b) >> 8), byte(len(b))})
	}
	return out.Bytes(), nil
}
