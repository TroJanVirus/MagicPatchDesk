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

// TargetFromComment returns default.xbe when no target is indicated. An
// unambiguous filename with an extension in a record comment is considered a
// file target. URLs and obvious web domains are ignored to avoid accidental
// special handling of attribution comments.
func TargetFromComment(comment string) (string, error) {
	// Fail closed rather than strip an unsafe path prefix and accidentally
	// interpret its offset as a write to default.xbe.
	if strings.Contains(comment, "../") || strings.Contains(comment, `..\`) {
		return "", fmt.Errorf("unsafe parent path in JMP comment %q", comment)
	}
	seen := map[string]bool{}
	for _, span := range commentFileToken.FindAllStringIndex(comment, -1) {
		start, end := span[0], span[1]
		token := comment[start:end]
		preceding := comment[:start]
		// Attribution URLs are not patch targets. Only the URL word is
		// ignored; a separate settings.cfg later in the same comment counts.
		word := preceding
		if k := strings.LastIndexAny(word, " \t\r\n,;([{\"'"); k >= 0 {
			word = word[k+1:]
		}
		if strings.Contains(word, "://") || strings.HasPrefix(strings.ToLower(token), "www.") {
			continue
		}
		isEdge := func(c byte) bool { return strings.ContainsRune(" \t\n\r[](){}\"',;:=!?.", rune(c)) }
		if start > 0 && !isEdge(comment[start-1]) {
			if comment[start-1] == '/' || comment[start-1] == '\\' || comment[start-1] == ':' {
				return "", fmt.Errorf("unsafe or ambiguous path in JMP comment %q", comment)
			}
			continue
		}
		if end < len(comment) && !isEdge(comment[end]) {
			continue
		}
		ext := strings.ToLower(token[strings.LastIndexByte(token, '.')+1:])
		if ext == "com" || ext == "net" || ext == "org" || ext == "io" || ext == "nz" || ext == "uk" || ext == "edu" || ext == "gov" {
			continue
		}
		norm, err := NormaliseTarget(token)
		if err != nil {
			return "", err
		}
		seen[norm] = true
	}
	if len(seen) == 0 {
		return "default.xbe", nil
	}
	if len(seen) > 1 {
		names := make([]string, 0, len(seen))
		for k := range seen {
			names = append(names, k)
		}
		sort.Strings(names)
		return "", fmt.Errorf("JMP comment mentions multiple target files %s; specify one filename per record", strings.Join(names, ", "))
	}
	for k := range seen {
		return k, nil
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
		target, err := TargetFromComment(r.Comment)
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
		named, e := TargetFromComment(r.Comment)
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
