package core

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"path"
	"strings"
)

// AllPatchFormats is the output mode used when a JMP is converted to a
// multi-format pack. A patch never embeds copyrighted original game bytes.
const AllPatchFormats = 5

// PlanJMPAllFormatsPack requires the original of every enabled target: BPS
// and Xdelta cannot be generated without reconstructing the result.
func PlanJMPAllFormatsPack(name string, data []byte) (PatchPackPlan, error) {
	var plan PatchPackPlan
	p, err := ParseJMP(name, data)
	if err != nil {
		return plan, err
	}
	targets, err := p.TargetFiles()
	if err != nil {
		return plan, err
	}
	for _, target := range targets {
		n := 0
		for _, r := range p.Records {
			if !r.Enabled {
				continue
			}
			t, e := p.RecordTarget(r)
			if e != nil {
				return plan, e
			}
			if t == target {
				n++
			}
		}
		plan.Targets = append(plan.Targets, PackTargetPlan{Target: target, RecordCount: n, RequiresSource: true, Reason: "Complete IPS/BPS/Xdelta pack needs the original file"})
	}
	if len(plan.Targets) == 0 {
		return plan, errors.New("no enabled JMP records")
	}
	return plan, nil
}

// BuildJMPAllFormatsPack emits every compatible unencrypted binary patch
// format for every enabled target. EIPS is added only when a user has provided
// a strong encryption password. It preserves the original single JMP, plus a
// v2 manifest with independently verifiable per-target alternatives.
// All originals are mandatory because BPS and Xdelta require source bytes.
func BuildJMPAllFormatsPack(name string, data []byte, originals map[string][]byte, password string, rle bool) ([]byte, PackManifest, error) {
	manifest := PackManifest{Schema: "magicpatchdesk-pack-v2", Origin: name, OriginalJMP: "source/original.jmp", OriginalJMPSHA256: shaString(data)}
	p, err := ParseJMP(name, data)
	if err != nil {
		return nil, manifest, err
	}
	targets, err := p.TargetFiles()
	if err != nil {
		return nil, manifest, err
	}
	if len(targets) == 0 {
		return nil, manifest, errors.New("no enabled JMP records")
	}
	manifest.Title = p.Headers["game-title"]
	manifest.Author = p.Headers["author"]
	manifest.Version = p.Headers["version"]
	manifest.TitleID = TitleIDFromVersion(p.Headers["version"])
	for _, c := range p.CRCs {
		manifest.ExpectedCRCs = append(manifest.ExpectedCRCs, fmt.Sprintf("%08X", c))
	}
	for _, target := range targets {
		if originals[target] == nil {
			return nil, manifest, fmt.Errorf("all-formats pack requires original file %s to create BPS and Xdelta", target)
		}
	}
	var buffer bytes.Buffer
	zw := zip.NewWriter(&buffer)
	fail := func(e error) ([]byte, PackManifest, error) { zw.Close(); return nil, manifest, e }
	source, err := zw.Create(manifest.OriginalJMP)
	if err != nil {
		return fail(err)
	}
	if _, err = source.Write(data); err != nil {
		return fail(err)
	}
	for _, target := range targets {
		orig := originals[target]
		scoped, err := p.ScopedPatch(target)
		if err != nil {
			return fail(err)
		}
		result, err := p.ApplyToTarget(orig, target, false)
		if err != nil {
			return fail(fmt.Errorf("%s: %w", target, err))
		}
		if target == "default.xbe" && len(p.CRCs) > 0 {
			if err = p.VerifyCRC(orig); err != nil {
				return fail(err)
			}
		}
		sourceHash, resultHash := shaString(orig), shaString(result)
		for _, form := range []int{1, 2, 4, 3} {
			ext, _ := formatExtension(form)
			if form == 3 && len([]rune(password)) < 8 {
				manifest.Omitted = append(manifest.Omitted, PackOmission{Target: target, Format: "eips", Reason: "encryption password (at least 8 characters) not provided"})
				continue
			}
			encoded, e := BuildPatchWithOptions(orig, result, form, password, manifest.Title, rle)
			if e != nil {
				// IPS cannot represent every file length/offset. Preserve a
				// concrete omission instead of falsely advertising support.
				manifest.Omitted = append(manifest.Omitted, PackOmission{Target: target, Format: strings.TrimPrefix(ext, "."), Reason: e.Error()})
				continue
			}
			got, e := applyPackedBytes(orig, encoded, form, password)
			if e != nil || !bytes.Equal(got, result) {
				return fail(fmt.Errorf("%s%s: format roundtrip mismatch: %v", target, ext, e))
			}
			entry := PackEntry{Target: target, File: path.Join("patches", target+ext), Format: strings.TrimPrefix(ext, "."), Records: len(scoped.Records), RequiresSource: true, PatchSHA256: shaString(encoded), SourceSHA256: sourceHash, ResultSHA256: resultHash}
			f, e := zw.Create(entry.File)
			if e != nil {
				return fail(e)
			}
			if _, e = f.Write(encoded); e != nil {
				return fail(e)
			}
			manifest.Entries = append(manifest.Entries, entry)
		}
	}
	found := map[string]bool{}
	for _, e := range manifest.Entries {
		found[e.Target] = true
	}
	for _, target := range targets {
		if !found[target] {
			return fail(fmt.Errorf("no supported binary patch format for %s", target))
		}
	}
	meta, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fail(err)
	}
	f, err := zw.Create("manifest.json")
	if err != nil {
		return fail(err)
	}
	if _, err = f.Write(append(meta, '\n')); err != nil {
		return fail(err)
	}
	if err = zw.Close(); err != nil {
		return nil, manifest, err
	}
	// Ensure we didn't produce a pack that our own reader rejects.
	if _, _, err = ReadPatchPack(buffer.Bytes()); err != nil {
		return nil, manifest, err
	}
	return buffer.Bytes(), manifest, nil
}

// ReadAllPackVariants returns the manifest, and creates a map keyed by target
// and format. Callers can present explicit choices without assuming one patch
// per game file. Every archive member is checksum verified before returning.
func ReadAllPackVariants(b []byte) (PackManifest, map[string]map[string][]byte, error) {
	m, files, err := ReadPatchPack(b)
	if err != nil {
		return m, nil, err
	}
	variants := make(map[string]map[string][]byte)
	for _, e := range m.Entries {
		if variants[e.Target] == nil {
			variants[e.Target] = make(map[string][]byte)
		}
		variants[e.Target][e.Format] = files[e.File]
	}
	return m, variants, nil
}

func PackCRC32(b []byte) string { return fmt.Sprintf("%08X", crc32.ChecksumIEEE(b)) }
