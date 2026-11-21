package core

import (
	"bytes"
	"fmt"
	"hash/crc32"
	"strings"
)

// NeedsOriginalForJMP reports when PF's bytes cannot be translated to JMP
// without resolving their effects against actual original file contents.
func (p *PF) NeedsOriginalForJMP() bool {
	for _, op := range p.Operations {
		switch op.Kind {
		case PFMedia, PFRegion:
			return true
		case PFHex:
			for _, pair := range op.Pairs {
				if pair.Count != 1 || len(pair.Find) != len(pair.Replace) {
					return true
				}
			}
		}
	}
	return false
}

// PFToJMP converts non-destructive PF byte operations to one multi-file JMP.
// CP/MV/RN/RM have no JMP equivalents, and are refused. The optional originals
// permit resolving HR occurrence counts, certificate flags and file extension.
// A converted JMP is always a SINGLE file even when it targets several files.
func (p *PF) ToJMP(originals map[string][]byte) ([]byte, error) {
	if p.HasFilesystemOps() {
		return nil, fmt.Errorf("PF includes copy/move/rename/remove commands not representable in JMP; keep the PF format to preserve file operations")
	}
	needSource := p.NeedsOriginalForJMP()
	if needSource && len(originals) == 0 {
		return nil, fmt.Errorf("this PF conversion requires the original game files for its search context and certificate offsets")
	}
	title := strings.ReplaceAll(p.Header["title"], "\n", " ")
	id := strings.ToUpper(strings.TrimSpace(p.Header["titleid"]))
	if id == "MULTI" {
		id = ""
	}
	var b strings.Builder
	b.WriteString("#Jay's Magic Patcher (www.jayxbox.com)\nsystem=Xbox\ngame-title=" + title + "\nregion=" + p.Header["region"] + "\nversion=" + id)
	if x := originals["default.xbe"]; x != nil {
		b.WriteString(fmt.Sprintf(" {%08X}", crc32.ChecksumIEEE(x)))
	}
	b.WriteString("\nauthor=" + p.Header["author"] + "\nnotes=Converted from XBMC4Gamers PF; original PF may carry additional metadata\n")
	if !needSource && !pfAllSourcesAvailable(p, originals) {
		for _, op := range p.Operations {
			comment := fmt.Sprintf("#PF %s (%s)\n", op.Kind, op.Target)
			for _, pair := range op.Pairs {
				b.WriteString(comment)
				switch op.Kind {
				case PFHex:
					b.WriteString(fmt.Sprintf("%X\n%X\n", pair.Find, pair.Replace))
				case PFOffset:
					b.WriteString(fmt.Sprintf("0x%X:%X\n", pair.Offset, pair.Bytes))
				default:
					return nil, fmt.Errorf("PF operation %s requires an original", op.Kind)
				}
			}
		}
	} else {
		// With originals, resolve the actual effects and write exact byte-offset
		// replacements per target. This preserves the result even for multiple HR
		// instances, as long as JMP can represent the resulting file length.
		result, e := p.ApplyPF(originals)
		if e != nil {
			return nil, e
		}
		for _, name := range p.TargetNames() {
			orig, ok := originals[name]
			if !ok {
				return nil, fmt.Errorf("original %s missing for PF conversion", name)
			}
			target, ok := result.Files[name]
			if !ok {
				return nil, fmt.Errorf("target %s removed during PF conversion", name)
			}
			if bytes.Equal(orig, target) {
				continue
			}
			if len(target) < len(orig) {
				return nil, fmt.Errorf("target %s shrinks: JMP does not support truncation", name)
			}
			for at := 0; at < len(orig); {
				if orig[at] == target[at] {
					at++
					continue
				}
				first := at
				for at < len(orig) && orig[at] != target[at] && at-first < 8192 {
					at++
				}
				b.WriteString(fmt.Sprintf("#PF diff (%s)\n0x%X:%X\n", name, first, target[first:at]))
			}
			for at := len(orig); at < len(target); at += 8192 {
				end := at + 8192
				if end > len(target) {
					end = len(target)
				}
				b.WriteString(fmt.Sprintf("#PF append (%s)\nAPPEND:%X\n", name, target[at:end]))
			}
		}
	}
	out := []byte(b.String())
	if _, e := ParseJMP(p.Filename+".jmp", out); e != nil {
		return nil, e
	}
	return out, nil
}

// PFFromJMP covers the compatible, context-free subset. Anything else needs
// an explicitly selected original to reconstruct accurate source-dependent HR
// data. JMP APPEND is not natively supported by Rocky5's PF interpreter.
func PFFromJMP(j *Patch) ([]byte, error) {
	var b strings.Builder
	b.WriteString("# Author: " + j.Headers["author"] + "\n# Info: Converted from JMP by Magic Patch Desk\n# Region: " + j.Headers["region"] + "\n# Title: " + j.Headers["game-title"] + "\n# TitleID: " + TitleIDFromVersion(j.Headers["version"]) + "\n# Type: Other Patch\n")
	for _, r := range j.Records {
		if !r.Enabled {
			continue
		}
		target, e := j.RecordTarget(r)
		if e != nil {
			return nil, e
		}
		switch r.Kind {
		case Offset:
			b.WriteString(fmt.Sprintf("OF|%s|%d|%X\n", target, r.Offset, r.Data))
		case SearchReplace:
			b.WriteString(fmt.Sprintf("HR|%s|1|%X|%X\n", target, r.Find, r.Replace))
		default:
			return nil, fmt.Errorf("JMP contains APPEND, which XBMC4Gamers PF cannot represent without rebuilding the target file")
		}
	}
	if _, e := ParsePF("converted.pf", []byte(b.String())); e != nil {
		return nil, e
	}
	return []byte(b.String()), nil
}

func pfAllSourcesAvailable(p *PF, orig map[string][]byte) bool {
	for _, name := range p.TargetNames() {
		if orig[name] == nil {
			return false
		}
	}
	return true
}
