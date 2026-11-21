package core

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"path"
	"strings"
)

// A Magic Patch Desk pack is a ZIP archive containing only patch files and a
// JSON manifest. It contains no copyrighted game assets or original game data.
// Each member addresses exactly ONE explicitly named game file.
const PatchPackExtension = ".mpdpack"

type PackEntry struct {
	Target         string `json:"target"`
	File           string `json:"patch"`
	Format         string `json:"format"`
	Records        int    `json:"records"`
	RequiresSource bool   `json:"requires_source"`
	PatchSHA256    string `json:"patch_sha256"`
	SourceSHA256   string `json:"source_sha256,omitempty"`
	ResultSHA256   string `json:"result_sha256,omitempty"`
}
type PackOmission struct {
	Target string `json:"target"`
	Format string `json:"format"`
	Reason string `json:"reason"`
}
type PackManifest struct {
	OriginalJMP       string         `json:"original_jmp,omitempty"`
	OriginalJMPSHA256 string         `json:"original_jmp_sha256,omitempty"`
	Omitted           []PackOmission `json:"omitted_formats,omitempty"`
	Schema            string         `json:"schema"`
	Title             string         `json:"title"`
	Author            string         `json:"author,omitempty"`
	Version           string         `json:"game_version,omitempty"`
	TitleID           string         `json:"title_id,omitempty"`
	ExpectedCRCs      []string       `json:"expected_crc32,omitempty"`
	Origin            string         `json:"origin"`
	Entries           []PackEntry    `json:"entries"`
}
type PackTargetPlan struct {
	Target            string
	RecordCount       int
	RequiresSource    bool
	SourceRecommended bool
	Reason            string
}
type PatchPackPlan struct{ Targets []PackTargetPlan }

func (p PatchPackPlan) MissingSources(src map[string][]byte) []string {
	missing := []string{}
	for _, item := range p.Targets {
		if item.RequiresSource && src[item.Target] == nil {
			missing = append(missing, item.Target)
		}
	}
	return missing
}

func formatExtension(n int) (string, error) {
	switch n {
	case 1:
		return ".ips", nil
	case 2:
		return ".xdelta", nil
	case 3:
		return ".eips", nil
	case 4:
		return ".bps", nil
	}
	return "", errors.New("patch packs support IPS, Xdelta, EIPS or BPS; JMP is always exported as one JMP")
}

// PlanJMPPack describes requirements for EACH target (unlike the single-file
// PlanPatchConversion API). It allows source-free offset JMP -> IPS packs, but
// only with source-dependent data when required for search/append/BPS/Xdelta.
func PlanJMPPack(name string, data []byte, outputFormat int) (PatchPackPlan, error) {
	var plan PatchPackPlan
	if _, err := formatExtension(outputFormat); err != nil {
		return plan, err
	}
	p, err := ParseJMP(name, data)
	if err != nil {
		return plan, err
	}
	targets, err := p.TargetFiles()
	if err != nil {
		return plan, err
	}
	if len(targets) == 0 {
		return plan, errors.New("no enabled JMP records")
	}
	for _, target := range targets {
		tp := PackTargetPlan{Target: target, SourceRecommended: true}
		for _, r := range p.Records {
			if !r.Enabled {
				continue
			}
			rt, e := p.RecordTarget(r)
			if e != nil {
				return plan, e
			}
			if rt != target {
				continue
			}
			tp.RecordCount++
			if outputFormat == 2 || outputFormat == 4 {
				tp.RequiresSource = true
				tp.Reason = "BPS and Xdelta are source-dependent formats"
			} else {
				switch r.Kind {
				case SearchReplace:
					tp.RequiresSource = true
					tp.Reason = "search-and-replace needs the original file to locate matching bytes"
				case Append:
					tp.RequiresSource = true
					tp.Reason = "APPEND needs the original file to calculate its final offset"
				case Offset:
					if r.Offset < 0 || r.Offset+int64(len(r.Data)) > 0xFFFFFF {
						return plan, fmt.Errorf("%s: IPS offset outside 24-bit range; select BPS/Xdelta and supply the source file", target)
					}
					if r.Offset == 0x454F46 {
						tp.RequiresSource = true
						tp.Reason = "IPS uses this offset as its EOF marker"
					}
				default:
					return plan, fmt.Errorf("unknown record kind %s", r.Kind)
				}
			}
		}
		tp.SourceRecommended = !tp.RequiresSource
		if tp.Reason == "" {
			tp.Reason = "offset-only JMP can be encoded without game files; an original verifies CRC and bounds"
		}
		plan.Targets = append(plan.Targets, tp)
	}
	return plan, nil
}

func shaString(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }

func encodeForPack(p *Patch, target string, original []byte, fmtN int, password string, rle bool) ([]byte, error) {
	scoped, err := p.ScopedPatch(target)
	if err != nil {
		return nil, err
	}
	if len(scoped.Records) == 0 {
		return nil, fmt.Errorf("no records for %s", target)
	}
	if fmtN == 1 || fmtN == 3 {
		offsetOnly := true
		for _, r := range scoped.Records {
			if r.Kind != Offset {
				offsetOnly = false
				break
			}
		}
		if offsetOnly {
			ips := IPSPatch{}
			for _, r := range scoped.Records {
				ips.Records = append(ips.Records, IPSRecord{Offset: int(r.Offset), Data: append([]byte(nil), r.Data...)})
			}
			encoded, e := EncodeIPSPatch(ips, rle)
			if e != nil {
				return nil, e
			}
			if fmtN == 3 {
				return EncryptIPS(encoded, password)
			}
			return encoded, nil
		}
	}
	if original == nil {
		return nil, fmt.Errorf("original file %s required to resolve patch records", target)
	}
	out, err := p.ApplyToTarget(original, target, false)
	if err != nil {
		return nil, err
	}
	return BuildPatchWithOptions(original, out, fmtN, password, p.Headers["game-title"], rle)
}

func applyPackedBytes(src, patch []byte, fmtN int, password string) ([]byte, error) {
	switch fmtN {
	case 1:
		p, e := DecodeIPS(patch)
		if e != nil {
			return nil, e
		}
		return p.Apply(src)
	case 2:
		return DecodeXdelta(src, patch)
	case 3:
		raw, e := DecryptIPS(patch, password)
		if e != nil {
			return nil, e
		}
		p, e := DecodeIPS(raw)
		if e != nil {
			return nil, e
		}
		return p.Apply(src)
	case 4:
		return DecodeBPS(src, patch)
	}
	return nil, errors.New("unknown packed format")
}

// BuildJMPPatchPack outputs a ZIP-compatible .mpdpack. All enabled record
// targets are included; it NEVER silently omits records or sources. Passwords
// are not stored in the manifest. Original bytes are not bundled.
func BuildJMPPatchPack(name string, data []byte, originalFiles map[string][]byte, outputFormat int, password string, rle bool) ([]byte, PatchPackPlan, error) {
	plan, err := PlanJMPPack(name, data, outputFormat)
	if err != nil {
		return nil, plan, err
	}
	if missing := plan.MissingSources(originalFiles); len(missing) > 0 {
		return nil, plan, fmt.Errorf("conversion requires original file(s): %s", strings.Join(missing, ", "))
	}
	ext, _ := formatExtension(outputFormat)
	p, err := ParseJMP(name, data)
	if err != nil {
		return nil, plan, err
	}
	var result bytes.Buffer
	zw := zip.NewWriter(&result)
	manifest := PackManifest{Schema: "magicpatchdesk-pack-v1", Title: p.Headers["game-title"], Author: p.Headers["author"], Origin: name, Version: p.Headers["version"], TitleID: TitleIDFromVersion(p.Headers["version"])}
	for _, c := range p.CRCs {
		manifest.ExpectedCRCs = append(manifest.ExpectedCRCs, fmt.Sprintf("%08X", c))
	}
	for _, t := range plan.Targets {
		original := originalFiles[t.Target]
		encoded, e := encodeForPack(p, t.Target, original, outputFormat, password, rle)
		if e != nil {
			zw.Close()
			return nil, plan, fmt.Errorf("%s: %w", t.Target, e)
		}
		entry := PackEntry{Target: t.Target, File: path.Join("patches", t.Target+ext), Format: strings.TrimPrefix(ext, "."), Records: t.RecordCount, RequiresSource: t.RequiresSource, PatchSHA256: shaString(encoded)}
		if original != nil {
			expected, e := p.ApplyToTarget(original, t.Target, false)
			if e != nil {
				zw.Close()
				return nil, plan, fmt.Errorf("%s: %w", t.Target, e)
			}
			got, e := applyPackedBytes(original, encoded, outputFormat, password)
			if e != nil || !bytes.Equal(expected, got) {
				zw.Close()
				return nil, plan, fmt.Errorf("%s: encoded patch fails exact round-trip: %v", t.Target, e)
			}
			entry.SourceSHA256 = shaString(original)
			entry.ResultSHA256 = shaString(expected)
		}
		f, e := zw.Create(entry.File)
		if e != nil {
			zw.Close()
			return nil, plan, e
		}
		if _, e = f.Write(encoded); e != nil {
			zw.Close()
			return nil, plan, e
		}
		manifest.Entries = append(manifest.Entries, entry)
	}
	f, e := zw.Create("manifest.json")
	if e != nil {
		zw.Close()
		return nil, plan, e
	}
	metadata, e := json.MarshalIndent(manifest, "", "  ")
	if e == nil {
		_, e = f.Write(append(metadata, '\n'))
	}
	if e != nil {
		zw.Close()
		return nil, plan, e
	}
	if e = zw.Close(); e != nil {
		return nil, plan, e
	}
	return result.Bytes(), plan, nil
}

// ReadPatchPack validates structure and member names before consuming data.
func ReadPatchPack(data []byte) (PackManifest, map[string][]byte, error) {
	var manifest PackManifest
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return manifest, nil, err
	}
	if len(zr.File) > 405 {
		return manifest, nil, errors.New("patch pack has too many files")
	}
	content := map[string][]byte{}
	var size uint64
	for _, f := range zr.File {
		if f.UncompressedSize64 > 64<<20 {
			return manifest, nil, fmt.Errorf("pack member %q is too large", f.Name)
		}
		size += f.UncompressedSize64
		if size > 512<<20 {
			return manifest, nil, errors.New("patch pack decompression limit exceeded")
		}
		if f.Name == "" || strings.HasPrefix(f.Name, "/") || strings.Contains(f.Name, "\\") || path.Clean(f.Name) != f.Name || strings.HasPrefix(f.Name, "../") {
			return manifest, nil, errors.New("unsafe ZIP path")
		}
		if _, exists := content[f.Name]; exists {
			return manifest, nil, errors.New("duplicate pack file")
		}
		in, e := f.Open()
		if e != nil {
			return manifest, nil, e
		}
		b, e := io.ReadAll(io.LimitReader(in, int64(f.UncompressedSize64)+1))
		in.Close()
		if e != nil {
			return manifest, nil, e
		}
		if uint64(len(b)) != f.UncompressedSize64 {
			return manifest, nil, errors.New("invalid pack member size")
		}
		content[f.Name] = b
	}
	b, ok := content["manifest.json"]
	if !ok {
		return manifest, nil, errors.New("patch pack has no manifest.json")
	}
	if err = json.Unmarshal(b, &manifest); err != nil {
		return manifest, nil, err
	}
	if (manifest.Schema != "magicpatchdesk-pack-v1" && manifest.Schema != "magicpatchdesk-pack-v2") || len(manifest.Entries) == 0 || len(manifest.Entries) > 400 {
		return manifest, nil, errors.New("unsupported/invalid patch pack manifest")
	}
	expectedMembers := len(manifest.Entries) + 1
	if manifest.Schema == "magicpatchdesk-pack-v2" && manifest.OriginalJMP != "" {
		if manifest.OriginalJMP != "source/original.jmp" {
			return manifest, nil, errors.New("invalid source JMP path")
		}
		if jmp, ok := content[manifest.OriginalJMP]; !ok {
			return manifest, nil, errors.New("missing original JMP")
		} else if manifest.OriginalJMPSHA256 == "" || shaString(jmp) != manifest.OriginalJMPSHA256 {
			return manifest, nil, errors.New("original JMP checksum mismatch")
		}
		expectedMembers++
	}
	if len(content) != expectedMembers {
		return manifest, nil, errors.New("unlisted or missing pack member")
	}
	targets := map[string]bool{}
	formats := map[string]map[string]bool{}
	for _, e := range manifest.Entries {
		normalized, ee := NormaliseTarget(e.Target)
		if ee != nil || normalized != e.Target {
			return manifest, nil, fmt.Errorf("unsafe target %q", e.Target)
		}
		if targets[e.Target] && manifest.Schema == "magicpatchdesk-pack-v1" {
			return manifest, nil, fmt.Errorf("duplicate target %s", e.Target)
		}
		targets[e.Target] = true
		if formats[e.Target] == nil {
			formats[e.Target] = make(map[string]bool)
		}
		if formats[e.Target][e.Format] {
			return manifest, nil, fmt.Errorf("duplicate format %s for %s", e.Format, e.Target)
		}
		formats[e.Target][e.Format] = true
		if e.File != "patches/"+e.Target+"."+e.Format {
			return manifest, nil, fmt.Errorf("unexpected patch path %q", e.File)
		}
		if e.Format != "ips" && e.Format != "eips" && e.Format != "bps" && e.Format != "xdelta" {
			return manifest, nil, errors.New("unsupported pack member format")
		}
		patch, ok := content[e.File]
		if !ok {
			return manifest, nil, fmt.Errorf("missing %s", e.File)
		}
		if shaString(patch) != e.PatchSHA256 {
			return manifest, nil, fmt.Errorf("pack member %s has invalid SHA256", e.File)
		}
	}
	return manifest, content, nil
}

// ApplyPatchPack works on copies of named source files; neither sources nor
// outputs are ever altered. Missing source files fail instead of being skipped.
func ApplyPatchPack(data []byte, sourceFiles map[string][]byte, password string) (map[string][]byte, error) {
	manifest, files, err := ReadPatchPack(data)
	if err != nil {
		return nil, err
	}
	results := map[string][]byte{}
	selected := manifest.Entries
	if manifest.Schema == "magicpatchdesk-pack-v2" {
		// Strongest source/target checks win; do not fall back on a failing BPS patch.
		byTarget := map[string]PackEntry{}
		rank := map[string]int{"bps": 0, "xdelta": 1, "ips": 2, "eips": 3}
		for _, entry := range manifest.Entries {
			old, ok := byTarget[entry.Target]
			if !ok || rank[entry.Format] < rank[old.Format] {
				byTarget[entry.Target] = entry
			}
		}
		selected = make([]PackEntry, 0, len(byTarget))
		for _, e := range manifest.Entries {
			if e == byTarget[e.Target] {
				selected = append(selected, e)
			}
		}
	}
	for _, e := range selected {
		source, found := sourceFiles[e.Target]
		if !found || source == nil {
			return nil, fmt.Errorf("missing source %s", e.Target)
		}
		if e.Target == "default.xbe" {
			if len(manifest.ExpectedCRCs) > 0 {
				actual := fmt.Sprintf("%08X", crc32.ChecksumIEEE(source))
				valid := false
				for _, expected := range manifest.ExpectedCRCs {
					if strings.EqualFold(actual, expected) {
						valid = true
						break
					}
				}
				if !valid {
					return nil, fmt.Errorf("default.xbe CRC32 %s does not match JMP expected CRC32", actual)
				}
			}
			if manifest.TitleID != "" {
				if x, inspectErr := InspectXBE(source); inspectErr == nil && !strings.EqualFold(x.TitleID, manifest.TitleID) {
					return nil, fmt.Errorf("default.xbe Title ID does not match JMP patch")
				}
			}
		}
		if e.SourceSHA256 != "" && shaString(source) != e.SourceSHA256 {
			return nil, fmt.Errorf("source checksum mismatch for %s", e.Target)
		}
		fmtN := map[string]int{"ips": 1, "xdelta": 2, "eips": 3, "bps": 4}[e.Format]
		output, err := applyPackedBytes(source, files[e.File], fmtN, password)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", e.Target, err)
		}
		if e.ResultSHA256 != "" && shaString(output) != e.ResultSHA256 {
			return nil, fmt.Errorf("result checksum mismatch for %s", e.Target)
		}
		results[e.Target] = output
	}
	return results, nil
}
