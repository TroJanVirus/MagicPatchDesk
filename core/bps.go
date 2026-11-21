package core

// BPS1 (Beat Patch System) uses the variable-length integer encoding documented
// in byuu's public-domain BPS specification. Encoder emits SourceRead/TargetRead
// actions. Decoder accepts all four standard commands, including relative copies.
import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"math"
)

const bpsMaxOutput = 512 << 20

func appendBPSNumber(out []byte, n uint64) []byte {
	for {
		b := byte(n & 0x7F)
		n >>= 7
		if n == 0 {
			return append(out, b|0x80)
		}
		out = append(out, b)
		n--
	}
}

func readBPSNumber(buf []byte, at *int, end int) (uint64, error) {
	var v, mul uint64 = 0, 1
	for {
		if *at >= end {
			return 0, errors.New("truncated BPS variable integer")
		}
		b := buf[*at]
		*at++
		digit := uint64(b & 0x7f)
		if digit != 0 && mul > (math.MaxUint64-v)/digit {
			return 0, errors.New("BPS integer overflow")
		}
		v += digit * mul
		if b&0x80 != 0 {
			return v, nil
		}
		if mul > (math.MaxUint64-v)/128 {
			return 0, errors.New("BPS integer overflow")
		}
		mul *= 128
		if v > math.MaxUint64-mul {
			return 0, errors.New("BPS integer overflow")
		}
		v += mul
	}
}

func bpsRelative(raw uint64) (int64, error) {
	// Absolute displacement is the upper bits; the low bit is a sign.
	if raw>>1 > math.MaxInt64 {
		return 0, errors.New("BPS relative offset overflow")
	}
	n := int64(raw >> 1)
	if raw&1 != 0 {
		return -n, nil
	}
	return n, nil
}

// DecodeBPS validates patch, original and output CRC32 before returning data.
func DecodeBPS(source, patch []byte) ([]byte, error) {
	if len(patch) < 19 || string(patch[:4]) != "BPS1" {
		return nil, errors.New("invalid BPS1 patch header")
	}
	end := len(patch) - 12
	wantSourceCRC := binary.LittleEndian.Uint32(patch[end:])
	wantTargetCRC := binary.LittleEndian.Uint32(patch[end+4:])
	wantPatchCRC := binary.LittleEndian.Uint32(patch[end+8:])
	if crc32.ChecksumIEEE(patch[:len(patch)-4]) != wantPatchCRC {
		return nil, errors.New("BPS patch CRC32 mismatch")
	}
	at := 4
	sourceLen, err := readBPSNumber(patch, &at, end)
	if err != nil {
		return nil, err
	}
	targetLen, err := readBPSNumber(patch, &at, end)
	if err != nil {
		return nil, err
	}
	metadataLen, err := readBPSNumber(patch, &at, end)
	if err != nil {
		return nil, err
	}
	if sourceLen != uint64(len(source)) {
		return nil, fmt.Errorf("BPS source size mismatch: got %d expected %d", len(source), sourceLen)
	}
	if crc32.ChecksumIEEE(source) != wantSourceCRC {
		return nil, errors.New("BPS source CRC32 mismatch: wrong original file")
	}
	if targetLen > bpsMaxOutput {
		return nil, errors.New("BPS output exceeds 512 MiB safety cap")
	}
	if metadataLen > uint64(end-at) {
		return nil, errors.New("truncated BPS metadata")
	}
	at += int(metadataLen)
	result := make([]byte, int(targetLen))
	out := 0
	var sourcePos, targetPos int64
	for out < len(result) {
		code, err := readBPSNumber(patch, &at, end)
		if err != nil {
			return nil, err
		}
		length := (code >> 2) + 1
		if length > uint64(len(result)-out) {
			return nil, errors.New("BPS action exceeds output length")
		}
		n := int(length)
		switch code & 3 {
		case 0: // SourceRead: same-offset copy
			if n > len(source)-out {
				return nil, errors.New("BPS SourceRead beyond source")
			}
			copy(result[out:out+n], source[out:out+n])
		case 1: // TargetRead: literal bytes in the patch
			if n > end-at {
				return nil, errors.New("truncated BPS TargetRead data")
			}
			copy(result[out:out+n], patch[at:at+n])
			at += n
		case 2: // SourceCopy: relative seek in the source
			delta, err := readBPSNumber(patch, &at, end)
			if err != nil {
				return nil, err
			}
			step, err := bpsRelative(delta)
			if err != nil {
				return nil, err
			}
			if (step > 0 && sourcePos > math.MaxInt64-step) || (step < 0 && sourcePos < math.MinInt64-step) {
				return nil, errors.New("BPS SourceCopy offset overflow")
			}
			sourcePos += step
			if sourcePos < 0 || sourcePos > int64(len(source)) || int64(n) > int64(len(source))-sourcePos {
				return nil, errors.New("BPS SourceCopy outside source")
			}
			copy(result[out:out+n], source[sourcePos:sourcePos+int64(n)])
			sourcePos += int64(n)
		case 3: // TargetCopy; bytewise for legal overlapping copies
			delta, err := readBPSNumber(patch, &at, end)
			if err != nil {
				return nil, err
			}
			step, err := bpsRelative(delta)
			if err != nil {
				return nil, err
			}
			if (step > 0 && targetPos > math.MaxInt64-step) || (step < 0 && targetPos < math.MinInt64-step) {
				return nil, errors.New("BPS TargetCopy offset overflow")
			}
			targetPos += step
			if targetPos < 0 || targetPos >= int64(out) {
				return nil, errors.New("BPS TargetCopy points to unwritten data")
			}
			for i := 0; i < n; i++ {
				if targetPos >= int64(out+i) {
					return nil, errors.New("BPS TargetCopy forward reference")
				}
				result[out+i] = result[targetPos]
				targetPos++
			}
		}
		out += n
	}
	if at != end {
		return nil, errors.New("BPS has trailing action bytes")
	}
	if crc32.ChecksumIEEE(result) != wantTargetCRC {
		return nil, errors.New("BPS target CRC32 mismatch")
	}
	return result, nil
}

// EncodeBPS builds standard BPS1 patches. It deliberately favours correctness
// and predictable memory use over an expensive full-file match optimiser.
func EncodeBPS(source, target []byte) ([]byte, error) {
	if len(source) > bpsMaxOutput || len(target) > bpsMaxOutput {
		return nil, errors.New("BPS input exceeds 512 MiB cap")
	}
	out := make([]byte, 0, min(len(target)+128, 1<<20))
	out = append(out, []byte("BPS1")...)
	out = appendBPSNumber(out, uint64(len(source)))
	out = appendBPSNumber(out, uint64(len(target)))
	out = appendBPSNumber(out, 0) // No metadata.
	for i := 0; i < len(target); {
		if i < len(source) && source[i] == target[i] {
			j := i + 1
			for j < len(target) && j < len(source) && source[j] == target[j] {
				j++
			}
			out = appendBPSNumber(out, (uint64(j-i-1)<<2)|0) // SourceRead
			i = j
		} else {
			j := i + 1
			for j < len(target) && (j >= len(source) || source[j] != target[j]) {
				j++
			}
			out = appendBPSNumber(out, (uint64(j-i-1)<<2)|1) // TargetRead
			out = append(out, target[i:j]...)
			i = j
		}
	}
	var footer [12]byte
	binary.LittleEndian.PutUint32(footer[0:4], crc32.ChecksumIEEE(source))
	binary.LittleEndian.PutUint32(footer[4:8], crc32.ChecksumIEEE(target))
	out = append(out, footer[:8]...)
	binary.LittleEndian.PutUint32(footer[8:12], crc32.ChecksumIEEE(out))
	out = append(out, footer[8:12]...)
	if !bytes.Equal(out[:4], []byte("BPS1")) {
		panic("unreachable")
	}
	return out, nil
}
