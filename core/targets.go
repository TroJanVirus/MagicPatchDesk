package core

import (
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"
)

// JMP comments can name files other than default.xbe, for example:
//
//	#Widescreen (settings.cfg)
//	#Video patch [file=Media\\settings.cfg]
//
// A comment is not an instruction to read arbitrary paths: target names are
// always relative and validated, and actual files must be resolved explicitly.
var commentFileToken = regexp.MustCompile(`(?i)[a-z0-9_+.-]+(?:[\\/][a-z0-9_+.-]+)*\.[a-z][a-z0-9]{1,11}`)

// NormaliseTarget rejects absolute paths, traversal, drive names, special
// Windows device names and unsafe path characters. Case-folding is used only
// for matching; paths are never concatenated into disk output locations.
func NormaliseTarget(s string) (string, error) {
	s = strings.TrimSpace(strings.ReplaceAll(s, `\`, "/"))
	if s == "" || strings.HasPrefix(s, "/") || strings.ContainsAny(s, ":*?\"<>|\x00") {
		return "", fmt.Errorf("unsafe target filename %q", s)
	}
	for _, part := range strings.Split(s, "/") {
		if part == "" || part == "." || part == ".." || strings.HasSuffix(part, ".") || strings.HasSuffix(part, " ") {
			return "", fmt.Errorf("unsafe target filename %q", s)
		}
		stem := strings.ToUpper(strings.SplitN(part, ".", 2)[0])
		if stem == "CON" || stem == "NUL" || stem == "AUX" || stem == "PRN" || strings.HasPrefix(stem, "COM") && len(stem) == 4 && stem[3] >= '1' && stem[3] <= '9' || strings.HasPrefix(stem, "LPT") && len(stem) == 4 && stem[3] >= '1' && stem[3] <= '9' {
			return "", fmt.Errorf("reserved device filename %q", s)
		}
	}
	return strings.ToLower(path.Clean(s)), nil
}

// FileTargetsInText finds file-like tokens in JMP comments or the patch notes.
// Website names and URLs are ignored. Every returned path is safe, relative,
// unique and normalized. Detection is a heuristic, not automatic permission to
// modify a file: the GUI must show the target and ask before patching.
func FileTargetsInText(text string) ([]string, error) {
	if strings.Contains(text, "../") || strings.Contains(text, `..\`) {
		return nil, fmt.Errorf("unsafe parent path in text %q", text)
	}
	seen := map[string]bool{}
	for _, span := range commentFileToken.FindAllStringIndex(text, -1) {
		start, end := span[0], span[1]
		token := text[start:end]
		preceding := text[:start]
		word := preceding
		if k := strings.LastIndexAny(word, " \t\r\n,;([{\"'"); k >= 0 {
			word = word[k+1:]
		}
		if strings.Contains(word, "://") || strings.HasPrefix(strings.ToLower(token), "www.") {
			continue
		}
		isEdge := func(c byte) bool { return strings.ContainsRune(" \t\n\r[](){}\"',;:=!?.", rune(c)) }
		if start > 0 && !isEdge(text[start-1]) {
			if text[start-1] == '/' || text[start-1] == '\\' || text[start-1] == ':' {
				return nil, fmt.Errorf("unsafe or ambiguous filename in text %q", text)
			}
			continue
		}
		if end < len(text) && !isEdge(text[end]) {
			continue
		}
		ext := strings.ToLower(token[strings.LastIndexByte(token, '.')+1:])
		if ext == "com" || ext == "net" || ext == "org" || ext == "io" || ext == "nz" || ext == "uk" || ext == "edu" || ext == "gov" {
			continue
		}
		norm, err := NormaliseTarget(token)
		if err != nil {
			return nil, err
		}
		seen[norm] = true
	}
	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	return names, nil
}

func TargetFromComment(comment string) (string, error) {
	files, err := FileTargetsInText(comment)
	if err != nil {
		return "", err
	}
	if len(files) == 0 {
		return "default.xbe", nil
	}
	if len(files) > 1 {
		return "", fmt.Errorf("JMP comment mentions multiple target files %s; specify one filename per record", strings.Join(files, ", "))
	}
	return files[0], nil
}

// RecordTarget resolves a record-specific comment first. A single filename in
// notes applies to records without a filename of their own; notes naming more
// than one file require per-record comments to disambiguate. This prevents a
// note like "driving.xbe" from being silently ignored.
func (p *Patch) RecordTarget(r Record) (string, error) {
	named, err := FileTargetsInText(r.Comment)
	if err != nil {
		return "", err
	}
	if len(named) > 1 {
		return "", fmt.Errorf("record comment mentions multiple files: %s", strings.Join(named, ", "))
	}
	if len(named) == 1 {
		return named[0], nil
	}
	notes, err := FileTargetsInText(p.Headers["notes"])
	if err != nil {
		return "", err
	}
	if len(notes) > 1 {
		return "", fmt.Errorf("patch notes mention multiple files (%s): specify the filename in each record's comment", strings.Join(notes, ", "))
	}
	if len(notes) == 1 {
		return notes[0], nil
	}
	return "default.xbe", nil
}

// TargetFiles lists enabled file targets in deterministic order. It does not
// include disabled records, and may return only default.xbe.
func (p *Patch) TargetFiles() ([]string, error) {
	set := map[string]bool{}
	for _, r := range p.Records {
		if !r.Enabled {
			continue
		}
		target, err := p.RecordTarget(r)
		if err != nil {
			return nil, fmt.Errorf("JMP line %d: %w", r.Line, err)
		}
		set[target] = true
	}
	keys := make([]string, 0, len(set))
	for k := range set {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys, nil
}

func (p *Patch) ExternalTargets() ([]string, error) {
	files, err := p.TargetFiles()
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(files))
	for _, name := range files {
		if name != "default.xbe" {
			out = append(out, name)
		}
	}
	return out, nil
}

// ScopedPatch returns a copy with only records meant for target. For secondary
// files the JMP's version/CRC headers identify the game's default.xbe, not the
// sidecar: they must not be erroneously checked against e.g. settings.cfg.
func (p *Patch) ScopedPatch(target string) (*Patch, error) {
	target, err := NormaliseTarget(target)
	if err != nil {
		return nil, err
	}
	q := &Patch{Filename: p.Filename, Headers: map[string]string{}, CRCs: append([]uint32(nil), p.CRCs...)}
	for k, v := range p.Headers {
		q.Headers[k] = v
	}
	for _, r := range p.Records {
		if !r.Enabled {
			continue
		}
		named, e := p.RecordTarget(r)
		if e != nil {
			return nil, fmt.Errorf("JMP line %d: %w", r.Line, e)
		}
		if named == target {
			q.Records = append(q.Records, r)
		}
	}
	if target != "default.xbe" {
		q.Headers["version"] = ""
		q.CRCs = nil
	}
	return q, nil
}

func (p *Patch) ApplyToTarget(input []byte, target string, override bool) ([]byte, error) {
	q, err := p.ScopedPatch(target)
	if err != nil {
		return nil, err
	}
	if len(q.Records) == 0 {
		return nil, fmt.Errorf("no enabled patch records target %q", target)
	}
	// applyWithTrace is the same engine used by ordinary JMP Apply.
	result, _, err := q.applyWithTrace(input, override)
	return result, err
}

func (p *Patch) PreviewTarget(input []byte, target string, override bool) (PatchPreview, error) {
	q, err := p.ScopedPatch(target)
	if err != nil {
		return PatchPreview{}, err
	}
	if len(q.Records) == 0 {
		return PatchPreview{}, fmt.Errorf("no enabled patch records target %q", target)
	}
	_, report, err := q.applyWithTrace(input, override)
	return report, err
}

// CheckSingleFileOnly stops library/direct JMP application from accidentally
// applying settings.cfg record offsets against default.xbe. Callers patching
// multiple files must resolve the extra targets and use ApplyToTarget instead.
func (p *Patch) CheckSingleFileOnly() error {
	targets, err := p.ExternalTargets()
	if err != nil {
		return err
	}
	if len(targets) > 0 {
		return fmt.Errorf("JMP targets additional file(s): %s; locate or skip these records before applying", strings.Join(targets, ", "))
	}
	return nil
}
