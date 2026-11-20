package core

import (
	"bytes"
	"errors"
	"fmt"
	"hash/adler32"
)

// VCDIFF / Xdelta subset without secondary compression or custom instruction tables.
// Compatible with the uncompressed default-code-table streams emitted by the HTML workbench.
var vcdMagic = []byte{0xD6, 0xC3, 0xC4, 0x00, 0x00}

func vint(n int) []byte {
	if n < 0 {
		return nil
	}
	var b []byte
	b = append(b, byte(n&0x7f))
	for n >>= 7; n > 0; n >>= 7 {
		b = append([]byte{byte(n&0x7f) | 0x80}, b...)
	}
	return b
}
func getVint(b []byte, i *int) (int, error) {
	n := int64(0)
	for k := 0; k < 10; k++ {
		if *i >= len(b) {
			return 0, errors.New("truncated VCDIFF varint")
		}
		x := b[*i]
		*i++
		n = (n << 7) | int64(x&127)
		if n > 1<<31 {
			return 0, errors.New("VCDIFF varint overflow")
		}
		if x&128 == 0 {
			return int(n), nil
		}
	}
	return 0, errors.New("VCDIFF varint too long")
}
func EncodeXdelta(source, target []byte) ([]byte, error) {
	if len(target) > 512<<20 {
		return nil, errors.New("Xdelta source limited to 512 MiB")
	}
	var out bytes.Buffer
	out.Write(vcdMagic)
	const block = 1 << 20
	for start := 0; start < len(target); start += block {
		end := start + block
		if end > len(target) {
			end = len(target)
		}
		t := target[start:end]
		src := []byte{}
		if start < len(source) {
			endSrc := end
			if endSrc > len(source) {
				endSrc = len(source)
			}
			src = source[start:endSrc]
		}
		var d, ins, addrs bytes.Buffer
		for p := 0; p < len(t); {
			run := 0
			for p+run < len(t) && p+run < len(src) && t[p+run] == src[p+run] {
				run++
			}
			if run >= 4 {
				ins.WriteByte(19)
				ins.Write(vint(run))
				addrs.Write(vint(p))
				p += run
				continue
			}
			from := p
			p++
			for p < len(t) {
				n := 0
				for p+n < len(t) && p+n < len(src) && n < 4 && t[p+n] == src[p+n] {
					n++
				}
				if n >= 4 {
					break
				}
				p++
			}
			ins.WriteByte(1)
			ins.Write(vint(p - from))
			d.Write(t[from:p])
		}
		var window bytes.Buffer
		flag := byte(0)
		if len(src) > 0 {
			flag = 1
		}
		window.WriteByte(flag)
		if flag == 1 {
			window.Write(vint(len(src)))
			window.Write(vint(start))
		}
		var section bytes.Buffer
		section.Write(vint(len(t)))
		section.WriteByte(0)
		section.Write(vint(d.Len()))
		section.Write(vint(ins.Len()))
		section.Write(vint(addrs.Len()))
		section.Write(d.Bytes())
		section.Write(ins.Bytes())
		section.Write(addrs.Bytes())
		window.Write(vint(section.Len()))
		window.Write(section.Bytes())
		out.Write(window.Bytes())
	}
	return out.Bytes(), nil
}
func DecodeXdelta(src, patch []byte) ([]byte, error) {
	if len(patch) < 5 || !bytes.Equal(patch[:4], vcdMagic[:4]) || patch[4]&3 != 0 {
		return nil, errors.New("unsupported VCDIFF header")
	}
	i := 5
	hdr := patch[4]
	if hdr&4 != 0 {
		n, err := getVint(patch, &i)
		if err != nil {
			return nil, err
		}
		if n > len(patch)-i {
			return nil, errors.New("VCDIFF app header truncated")
		}
		i += n
	}
	var output bytes.Buffer
	for i < len(patch) {
		win := patch[i]
		i++
		if win&^byte(7) != 0 || win&3 == 3 {
			return nil, errors.New("unsupported VCDIFF window flags")
		}
		seg := []byte{}
		if win&3 != 0 {
			sz, err := getVint(patch, &i)
			if err != nil {
				return nil, err
			}
			pos, err := getVint(patch, &i)
			if err != nil {
				return nil, err
			}
			parent := src
			if win&2 != 0 {
				parent = output.Bytes()
			}
			if pos+sz > len(parent) {
				return nil, errors.New("VCDIFF source range out of bounds")
			}
			seg = parent[pos : pos+sz]
		}
		length, err := getVint(patch, &i)
		if err != nil {
			return nil, err
		}
		end := i + length
		if end > len(patch) {
			return nil, errors.New("VCDIFF window truncated")
		}
		targetLen, err := getVint(patch, &i)
		if err != nil {
			return nil, err
		}
		if targetLen > 512<<20-output.Len() {
			return nil, errors.New("VCDIFF output size exceeds cap")
		}
		if i >= end || patch[i] != 0 {
			return nil, errors.New("VCDIFF secondary compressor unsupported")
		}
		i++
		dlen, err := getVint(patch, &i)
		if err != nil {
			return nil, err
		}
		ilen, err := getVint(patch, &i)
		if err != nil {
			return nil, err
		}
		alen, err := getVint(patch, &i)
		if err != nil {
			return nil, err
		}
		var checksum uint32
		if win&4 != 0 {
			if i+4 > end {
				return nil, errors.New("VCDIFF checksum truncated")
			}
			checksum = uint32(patch[i])<<24 | uint32(patch[i+1])<<16 | uint32(patch[i+2])<<8 | uint32(patch[i+3])
			i += 4
		}
		dataPos := i
		instPos := dataPos + dlen
		addrPos := instPos + ilen
		if addrPos+alen != end {
			return nil, errors.New("VCDIFF section mismatch")
		}
		di, ii, ai := dataPos, instPos, addrPos
		buff := make([]byte, targetLen)
		written := 0
		var near [4]int
		var same [768]int
		nearPtr := 0
		do := func(typ, size, mode int) error {
			if typ == 0 {
				return nil
			}
			if size == 0 {
				n, e := getVint(patch, &ii)
				if e != nil {
					return e
				}
				size = n
			}
			if size <= 0 || size > targetLen-written {
				return errors.New("VCDIFF instruction exceeds output")
			}
			switch typ {
			case 1:
				if di+size > instPos {
					return errors.New("VCDIFF ADD truncated")
				}
				copy(buff[written:], patch[di:di+size])
				di += size
			case 2:
				if di >= instPos {
					return errors.New("VCDIFF RUN truncated")
				}
				v := patch[di]
				di++
				for j := 0; j < size; j++ {
					buff[written+j] = v
				}
			case 3:
				here := len(seg) + written
				var off int
				if mode == 0 {
					n, e := getVint(patch, &ai)
					if e != nil {
						return e
					}
					off = n
				} else if mode == 1 {
					n, e := getVint(patch, &ai)
					if e != nil {
						return e
					}
					off = here - n
				} else if mode <= 5 {
					n, e := getVint(patch, &ai)
					if e != nil {
						return e
					}
					off = near[mode-2] + n
				} else {
					if ai >= end {
						return errors.New("VCDIFF SAME address missing")
					}
					off = same[(mode-6)*256+int(patch[ai])]
					ai++
				}
				if off < 0 || off >= here {
					return errors.New("VCDIFF COPY offset out of bounds")
				}
				near[nearPtr] = off
				nearPtr = (nearPtr + 1) % 4
				same[off%768] = off
				if off < len(seg) {
					if off+size > len(seg) {
						return errors.New("VCDIFF copy crosses source range")
					}
					copy(buff[written:], seg[off:off+size])
				} else {
					offset := off - len(seg)
					if offset >= written {
						return errors.New("VCDIFF COPY refers to unwritten output")
					}
					for j := 0; j < size; j++ {
						buff[written+j] = buff[offset+j]
					}
				}
			default:
				return errors.New("unknown VCDIFF instruction")
			}
			written += size
			return nil
		}
		for ii < addrPos {
			code := int(patch[ii])
			ii++
			type1, size1, mode1, type2, size2, mode2 := 0, 0, 0, 0, 0, 0
			switch {
			case code == 0:
				type1 = 2
			case code <= 18:
				type1 = 1
				if code > 1 {
					size1 = code - 1
				}
			case code <= 162:
				type1 = 3
				mode1 = (code - 19) / 16
				sz := (code - 19) % 16
				if sz > 0 {
					size1 = sz + 3
				}
			case code <= 234:
				type1 = 1
				type2 = 3
				mode2 = (code - 163) / 12
				j := (code - 163) % 12
				size1 = j/3 + 1
				size2 = j%3 + 4
			case code <= 246:
				type1 = 1
				type2 = 3
				mode2 = 6 + (code-235)/4
				size1 = (code-235)%4 + 1
				size2 = 4
			default:
				type1 = 3
				mode1 = code - 247
				size1 = 4
				type2 = 1
				size2 = 1
			}
			if e := do(type1, size1, mode1); e != nil {
				return nil, fmt.Errorf("VCDIFF code %d: %w", code, e)
			}
			if e := do(type2, size2, mode2); e != nil {
				return nil, fmt.Errorf("VCDIFF code %d: %w", code, e)
			}
		}
		if written != targetLen || di != instPos || ai != end {
			return nil, errors.New("VCDIFF decoded lengths mismatch")
		}
		if win&4 != 0 && adler32.Checksum(buff) != checksum {
			return nil, errors.New("VCDIFF checksum mismatch")
		}
		output.Write(buff)
		i = end
	}
	return output.Bytes(), nil
}
