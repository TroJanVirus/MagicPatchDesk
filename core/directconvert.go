package core

import (
	"bytes"
	"errors"
	"fmt"
	"hash/crc32"
	"path/filepath"
	"strings"
)

// ConversionPlan reports whether a source XBE is essential to preserve the
// semantics of a conversion. It is intentionally conservative: a binary delta
// cannot be validated merely because its filename or magic is recognizable.
type ConversionPlan struct {
	RequiresSource    bool
	SourceRecommended bool
	Reason            string
}

// PlanPatchConversion examines the actual records, not just the extension.
// Formats: 0=JMP, 1=IPS, 2=Xdelta, 3=encrypted IPS, 4=BPS.
func PlanPatchConversion(name string, data []byte, outputFormat int, password string) (ConversionPlan, error) {
	if outputFormat < 0 || outputFormat > 4 {
		return ConversionPlan{}, errors.New("unknown output patch format")
	}
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".jmp":
		p, err := ParseJMP(name, data)
		if err != nil {
			return ConversionPlan{}, err
		}
		switch outputFormat {
		case 0:
			return ConversionPlan{SourceRecommended: true, Reason: "A source XBE is optional for JMP-to-JMP. Supply one to verify Title ID, CRC32 and resulting bytes."}, nil
		case 1, 3:
			for _, r := range p.Records {
				if !r.Enabled {
					continue
				}
				switch r.Kind {
				case Offset:
					if r.Offset >= 0x454F46 && r.Offset < 0x454F47 { // IPS reserved EOF offset
						return ConversionPlan{RequiresSource: true, Reason: "An offset-only JMP writes at IPS's reserved EOF marker offset; original bytes are required to work around the reserved value."}, nil
					}
					if r.Offset < 0 || r.Offset+int64(len(r.Data)) > 0xFFFFFF {
						return ConversionPlan{}, errors.New("JMP writes beyond IPS's 24-bit range; choose BPS or Xdelta with the original XBE")
					}
				case SearchReplace:
					return ConversionPlan{RequiresSource: true, Reason: "This JMP searches for bytes. The original XBE is needed to discover the replacement offsets."}, nil
				case Append:
					return ConversionPlan{RequiresSource: true, Reason: "This JMP appends bytes. The original XBE is needed to determine the EOF offset."}, nil
				default:
					return ConversionPlan{}, fmt.Errorf("unknown JMP record type %q", r.Kind)
				}
			}
			return ConversionPlan{SourceRecommended: true, Reason: "Offset-only JMP can be converted directly. Supply the original XBE to verify its CRC32/Title ID; standard IPS cannot retain those JMP checks."}, nil
		default:
			return ConversionPlan{RequiresSource: true, Reason: "BPS and Xdelta need a source XBE to construct source-dependent patches and checksums."}, nil
		}
	case ".ips", ".eips":
		contents := data
		if ext == ".eips" {
			var err error
			contents, err = DecryptIPS(data, password)
			if err != nil {
				return ConversionPlan{}, fmt.Errorf("encrypted IPS requires the correct password: %w", err)
			}
		}
		ips, err := DecodeIPS(contents)
		if err != nil {
			return ConversionPlan{}, err
		}
		if outputFormat == 1 || outputFormat == 3 {
			return ConversionPlan{SourceRecommended: false, Reason: "IPS and encrypted IPS share the same records and optional final size; they can be re-encoded directly, without the original XBE."}, nil
		}
		if outputFormat == 0 && !ips.HasFinalSize {
			return ConversionPlan{SourceRecommended: true, Reason: "IPS records can become JMP offset records without a source. A source XBE is recommended to check bounds and add CRC32 validation; JMP cannot extend a file with offset records."}, nil
		}
		if outputFormat == 0 {
			return ConversionPlan{RequiresSource: true, Reason: "IPS specifies a final file size; the original XBE is needed to reproduce its resize/append/truncation semantics as JMP."}, nil
		}
		return ConversionPlan{RequiresSource: true, Reason: "Xdelta/BPS output requires the exact original XBE to calculate the source-dependent delta and checksums."}, nil
	case ".bps":
		if len(data) < 19 || string(data[:4]) != "BPS1" {
			return ConversionPlan{}, errors.New("invalid BPS patch")
		}
		return ConversionPlan{RequiresSource: true, Reason: "BPS requires the original file. Its source CRC32 must match before the target can be reconstructed."}, nil
	case ".xdelta", ".vcdiff":
		if len(data) < 5 || !bytes.Equal(data[:3], []byte{0xD6, 0xC3, 0xC4}) {
			return ConversionPlan{}, errors.New("invalid VCDIFF header")
		}
		return ConversionPlan{RequiresSource: true, Reason: "Xdelta/VCDIFF must be decoded against the original XBE; this decoder supports uncompressed default-table VCDIFF."}, nil
	default:
		return ConversionPlan{}, fmt.Errorf("unsupported source patch extension %q", ext)
	}
}

// EncodeIPSPatch reserializes IPS records without needing the bytes they patch.
// It preserves ordering, overlapping writes, optional final-size field and
// RLE semantics, instead of first applying the patch to a fabricated source.
func EncodeIPSPatch(p IPSPatch, useRLE bool) ([]byte, error) {
	var out bytes.Buffer
	out.WriteString("PATCH")
	emit := func(offset int, raw []byte) error {
		if offset < 0 || offset > 0xFFFFFF {
			return errors.New("IPS record offset outside 24-bit range")
		}
		if offset == 0x454F46 {
			return errors.New("IPS record starts at reserved EOF marker; supply a source XBE")
		}
		if len(raw) == 0 {
			return errors.New("empty IPS record")
		}
		if offset+len(raw) > 0xFFFFFF {
			return errors.New("IPS write exceeds 24-bit size limit")
		}
		for i := 0; i < len(raw); {
			off := offset + i
			if off == 0x454F46 {
				return errors.New("IPS record split at reserved EOF marker; supply a source XBE")
			}
			n := 1
			if useRLE {
				for i+n < len(raw) && n < 0xFFFF && raw[i+n] == raw[i] {
					n++
				}
				if n >= 4 {
					out.Write([]byte{byte(off >> 16), byte(off >> 8), byte(off), 0, 0, byte(n >> 8), byte(n), raw[i]})
					i += n
					continue
				}
			}
			start := i
			i++
			for i < len(raw) && i-start < 0xFFFF {
				if useRLE {
					run := 1
					for i+run < len(raw) && run < 4 && raw[i+run] == raw[i] {
						run++
					}
					if run >= 4 {
						break
					}
				}
				i++
			}
			n = i - start
			out.Write([]byte{byte(off >> 16), byte(off >> 8), byte(off), byte(n >> 8), byte(n)})
			out.Write(raw[start:i])
		}
		return nil
	}
	for _, r := range p.Records {
		if err := emit(r.Offset, r.Data); err != nil {
			return nil, err
		}
	}
	out.WriteString("EOF")
	if p.HasFinalSize {
		if p.FinalSize < 0 || p.FinalSize > 0xFFFFFF {
			return nil, errors.New("IPS final size outside 24-bit range")
		}
		out.Write([]byte{byte(p.FinalSize >> 16), byte(p.FinalSize >> 8), byte(p.FinalSize)})
	}
	return out.Bytes(), nil
}

// SerializeJMP preserves JMP headers, comments, grouped records and enabled states.
func SerializeJMP(p *Patch) []byte {
	var b strings.Builder
	b.WriteString("#Jay's Magic Patcher (www.jayxbox.com)\n")
	for _, h := range []string{"system", "game-title", "region", "version", "author", "notes"} {
		b.WriteString(h + "=" + strings.ReplaceAll(strings.ReplaceAll(p.Headers[h], "\n", " "), "\r", " ") + "\n")
	}
	for _, r := range p.Records {
		b.WriteString("#" + r.Comment)
		if !r.Enabled {
			b.WriteString(" [-]")
		}
		b.WriteByte('\n')
		switch r.Kind {
		case SearchReplace:
			b.WriteString(fmt.Sprintf("%X\n%X\n", r.Find, r.Replace))
		case Offset:
			b.WriteString(fmt.Sprintf("0x%X:%X\n", r.Offset, r.Data))
		case Append:
			b.WriteString(fmt.Sprintf("APPEND:%X\n", r.Data))
		}
	}
	return []byte(b.String())
}

// ConvertPatchBytes converts the actual patch, requiring the original file only
// where necessary. If supplied, the original is always checked by applying the
// source and generated patches and comparing their complete outputs.
func ConvertPatchBytes(sourceName string, patchBytes []byte, sourceXBE []byte, outputFormat int, password string, useRLE bool, title string) ([]byte, ConversionPlan, error) {
	plan, err := PlanPatchConversion(sourceName, patchBytes, outputFormat, password)
	if err != nil {
		return nil, plan, err
	}
	if sourceXBE == nil && plan.RequiresSource {
		return nil, plan, errors.New("original XBE required: " + plan.Reason)
	}
	if sourceXBE != nil {
		var after []byte
		ext := strings.ToLower(filepath.Ext(sourceName))
		switch ext {
		case ".jmp":
			p, e := ParseJMP(sourceName, patchBytes)
			err = e
			if err == nil {
				after, err = p.Apply(sourceXBE, false)
			}
		case ".ips", ".eips":
			raw := patchBytes
			if ext == ".eips" {
				raw, err = DecryptIPS(patchBytes, password)
			}
			if err == nil {
				var p IPSPatch
				p, err = DecodeIPS(raw)
				if err == nil {
					after, err = p.Apply(sourceXBE)
				}
			}
		case ".bps":
			after, err = DecodeBPS(sourceXBE, patchBytes)
		case ".xdelta", ".vcdiff":
			after, err = DecodeXdelta(sourceXBE, patchBytes)
		}
		if err != nil {
			return nil, plan, fmt.Errorf("source patch cannot be applied to this original: %w", err)
		}
		var converted []byte
		if !plan.RequiresSource {
			// Preserve the exact original records and their write order. A stock
			// XBE here is for verification, not a reason to silently strip
			// writes that happen to be no-ops on that particular version.
			converted, _, err = ConvertPatchBytes(sourceName, patchBytes, nil, outputFormat, password, useRLE, title)
			if err == nil && outputFormat == 0 && (ext == ".ips" || ext == ".eips") {
				// IPS contains no identity checks. When a source is available,
				// attach its verified CRC32 to the generated JMP while leaving
				// the offset records unchanged.
				var p *Patch
				p, err = ParseJMP("converted.jmp", converted)
				if err == nil {
					info, inspectErr := InspectXBE(sourceXBE)
					if inspectErr == nil {
						p.Headers["version"] = fmt.Sprintf("%s {%08X}", info.TitleID, crc32.ChecksumIEEE(sourceXBE))
					} else {
						p.Headers["version"] = fmt.Sprintf("{%08X}", crc32.ChecksumIEEE(sourceXBE))
					}
					converted = SerializeJMP(p)
				}
			}
		} else {
			converted, err = BuildPatchWithOptions(sourceXBE, after, outputFormat, password, title, useRLE)
		}
		if err != nil {
			return nil, plan, err
		}
		// An identical result with incompatible output semantics is not a
		// successful conversion; verify the bytes before writing the file.
		var round []byte
		switch outputFormat {
		case 0:
			var p *Patch
			p, err = ParseJMP("converted.jmp", converted)
			if err == nil {
				round, err = p.Apply(sourceXBE, false)
			}
		case 1, 3:
			raw := converted
			if outputFormat == 3 {
				raw, err = DecryptIPS(converted, password)
			}
			if err == nil {
				var p IPSPatch
				p, err = DecodeIPS(raw)
				if err == nil {
					round, err = p.Apply(sourceXBE)
				}
			}
		case 2:
			round, err = DecodeXdelta(sourceXBE, converted)
		case 4:
			round, err = DecodeBPS(sourceXBE, converted)
		}
		if err != nil {
			return nil, plan, fmt.Errorf("converted patch failed verification: %w", err)
		}
		if !bytes.Equal(after, round) {
			return nil, plan, errors.New("converted patch output differs from source patch output")
		}
		return converted, plan, nil
	}
	ext := strings.ToLower(filepath.Ext(sourceName))
	var ips IPSPatch
	switch ext {
	case ".jmp":
		p, err := ParseJMP(sourceName, patchBytes)
		if err != nil {
			return nil, plan, err
		}
		if outputFormat == 0 {
			return SerializeJMP(p), plan, nil
		}
		for _, r := range p.Records {
			if !r.Enabled {
				continue
			}
			if r.Kind != Offset {
				return nil, plan, errors.New("source XBE required for non-offset JMP records")
			}
			ips.Records = append(ips.Records, IPSRecord{Offset: int(r.Offset), Data: r.Data})
		}
	case ".ips", ".eips":
		data := patchBytes
		if ext == ".eips" {
			data, err = DecryptIPS(data, password)
			if err != nil {
				return nil, plan, err
			}
		}
		ips, err = DecodeIPS(data)
		if err != nil {
			return nil, plan, err
		}
		if outputFormat == 0 {
			if ips.HasFinalSize {
				return nil, plan, errors.New("source XBE required for IPS final-size handling")
			}
			var s strings.Builder
			s.WriteString("#Jay's Magic Patcher (www.jayxbox.com)\nsystem=Xbox\ngame-title=" + strings.ReplaceAll(title, "\n", " ") + "\nregion=\nversion=\nauthor=\nnotes=Converted from IPS; no source CRC32 verification\n")
			for i, r := range ips.Records {
				if r.Offset < 0 || r.Offset+len(r.Data) > 512<<20 {
					return nil, plan, errors.New("offset outside XBE limit")
				}
				s.WriteString(fmt.Sprintf("#IPS record %d\n0x%X:%X\n", i+1, r.Offset, r.Data))
			}
			return []byte(s.String()), plan, nil
		}
	default:
		return nil, plan, errors.New("source required for this conversion")
	}
	encoded, err := EncodeIPSPatch(ips, useRLE)
	if err != nil {
		return nil, plan, err
	}
	if outputFormat == 3 {
		encoded, err = EncryptIPS(encoded, password)
	}
	return encoded, plan, err
}

// BPS and VCDIFF use source/target-dependent encodings and must never be
// generated without a trusted original, even if a delta contains literals.