package core

import (
	"bytes"
	"fmt"
	"hash/crc32"
	"strings"
)

type RecordPreview struct {
	Number       int
	Group        string
	Kind         RecordKind
	Offset       int64
	Size         int
	ChangedBytes int
}

type PatchPreview struct {
	OriginalSize    int64
	PatchedSize     int64
	OriginalCRC32   string
	PatchedCRC32    string
	ModifiedBytes   int64
	AppendedBytes   int64
	SelectedRecords int
	Records         []RecordPreview
}

// Preview applies a patch exclusively to an in-memory copy and reports its
// intended writes. It does not modify any source file.
func (p *Patch) Preview(input []byte, overrideCRC bool) (PatchPreview, error) {
	_, report, err := p.applyWithTrace(input, overrideCRC)
	return report, err
}

// PreviewDifference is for non-JMP patch formats, where individual records
// cannot be meaningfully reconstructed after decoding the result.
func PreviewDifference(before, after []byte) PatchPreview {
	report := PatchPreview{OriginalSize: int64(len(before)), PatchedSize: int64(len(after)), OriginalCRC32: fmt.Sprintf("%08X", crc32.ChecksumIEEE(before)), PatchedCRC32: fmt.Sprintf("%08X", crc32.ChecksumIEEE(after))}
	size := len(before)
	if len(after) < size {
		size = len(after)
	}
	for i := 0; i < size; i++ {
		if before[i] != after[i] {
			report.ModifiedBytes++
		}
	}
	if len(after) > len(before) {
		report.AppendedBytes = int64(len(after) - len(before))
	}
	return report
}

func (p *Patch) applyWithTrace(input []byte, overrideCRC bool) ([]byte, PatchPreview, error) {
	var empty PatchPreview
	if !overrideCRC {
		if titleID := TitleIDFromVersion(p.Headers["version"]); titleID != "" {
			if info, err := InspectXBE(input); err == nil && !strings.EqualFold(info.TitleID, titleID) {
				return nil, empty, fmt.Errorf("Title ID mismatch: XBE is %s but patch expects %s", info.TitleID, titleID)
			}
		}
		if err := p.VerifyCRC(input); err != nil {
			return nil, empty, err
		}
	}
	out := append([]byte(nil), input...)
	changes := []RecordPreview{}
	for i, r := range p.Records {
		if !r.Enabled {
			continue
		}
		entry := RecordPreview{Number: i + 1, Group: r.Group, Kind: r.Kind}
		switch r.Kind {
		case Offset:
			if r.Offset < 0 || r.Offset+int64(len(r.Data)) > int64(len(out)) {
				return nil, empty, fmt.Errorf("record %d (%s): offset 0x%X outside XBE", i+1, r.Group, r.Offset)
			}
			entry.Offset = r.Offset
			entry.Size = len(r.Data)
			for k, b := range r.Data {
				if out[int(r.Offset)+k] != b {
					entry.ChangedBytes++
				}
			}
			copy(out[r.Offset:], r.Data)
		case SearchReplace:
			pos := bytes.Index(out, r.Find)
			if pos < 0 {
				return nil, empty, fmt.Errorf("record %d (%s): search pattern not found", i+1, r.Group)
			}
			entry.Offset = int64(pos)
			entry.Size = len(r.Replace)
			for k, b := range r.Replace {
				if out[pos+k] != b {
					entry.ChangedBytes++
				}
			}
			copy(out[pos:], r.Replace)
		case Append:
			if len(out)+len(r.Data) > 512<<20 {
				return nil, empty, fmt.Errorf("record %d: XBE exceeds 512 MiB safety cap", i+1)
			}
			entry.Offset = int64(len(out))
			entry.Size = len(r.Data)
			entry.ChangedBytes = len(r.Data)
			out = append(out, r.Data...)
		default:
			return nil, empty, fmt.Errorf("unknown record kind %s", r.Kind)
		}
		changes = append(changes, entry)
	}
	result := PreviewDifference(input, out)
	result.SelectedRecords = len(changes)
	result.Records = changes
	return out, result, nil
}
