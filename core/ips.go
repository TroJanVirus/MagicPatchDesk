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
func EncodeIPS(a, b []byte) ([]byte, error) {
	if len(b) > 0xFFFFFF {
		return nil, errors.New("IPS max output length is 16MiB; use JMP or Xdelta")
	}
	var out bytes.Buffer
	out.WriteString("PATCH")
	for i := 0; i < len(b); {
		if i < len(a) && a[i] == b[i] {
			i++
			continue
		}
		st := i
		for i < len(b) && (i >= len(a) || a[i] != b[i]) && i-st < 0xFFFF {
			i++
		}
		if st == 0x454F46 {
			return nil, errors.New("IPS cannot encode a record at reserved EOF offset; use Xdelta")
		}
		out.Write([]byte{byte(st >> 16), byte(st >> 8), byte(st), byte((i - st) >> 8), byte(i - st)})
		out.Write(b[st:i])
	}
	out.WriteString("EOF")
	if len(a) != len(b) {
		out.Write([]byte{byte(len(b) >> 16), byte(len(b) >> 8), byte(len(b))})
	}
	return out.Bytes(), nil
}
