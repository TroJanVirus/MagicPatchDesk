package core

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

// PF is Rocky5 XBMC4Gamers' File Patcher format. Unlike JMP, it is an
// ordered operation program against a *game directory*, and can copy, move,
// remove and rename files. Nothing in this core executes Python or shells.
type PF struct {
	Filename   string
	Header     map[string]string
	Operations []PFOperation
}

type PFOp string

const (
	PFHex    PFOp = "HR"
	PFOffset PFOp = "OF"
	PFMedia  PFOp = "SM"
	PFRegion PFOp = "SR"
	PFCopy   PFOp = "CP"
	PFMove   PFOp = "MV"
	PFRename PFOp = "RN"
	PFRemove PFOp = "RM"
)

type PFPair struct {
	Count                int
	Offset               int64
	Find, Replace, Bytes []byte
}
type PFOperation struct {
	Kind    PFOp
	Line    int
	Comment string
	Target  string
	To      string
	Names   []string
	Pairs   []PFPair
}
type PFOutcome struct {
	Files    map[string][]byte
	Modified []string
	Removed  []string
	Created  []string
	Warnings []string
	Actions  int
}

func (p *PF) HasFilesystemOps() bool {
	for _, op := range p.Operations {
		switch op.Kind {
		case PFCopy, PFMove, PFRename, PFRemove:
			return true
		}
	}
	return false
}
func (p *PF) TargetNames() []string {
	seen := map[string]bool{}
	for _, op := range p.Operations {
		switch op.Kind {
		case PFRemove:
			for _, n := range op.Names {
				seen[n] = true
			}
		default:
			seen[op.Target] = true
		}
	}
	names := make([]string, 0, len(seen))
	for n := range seen {
		names = append(names, n)
	}
	// Keep names deterministic for UI prompts and byte-level testing.
	sortStrings(names)
	return names
}

// PFRequiredInputs accounts for files created by CP / MV / RN earlier in the
// patch, so it does not ask the user to locate a newly created alternate XBE.
func (p *PF) RequiredInputs() []string {
	created := map[string]bool{}
	required := map[string]bool{}
	for _, op := range p.Operations {
		names := []string{op.Target}
		if op.Kind == PFRemove {
			names = op.Names
		}
		for _, n := range names {
			if !created[n] {
				required[n] = true
			}
		}
		switch op.Kind {
		case PFCopy:
			created[op.To] = true
		case PFMove, PFRename:
			created[op.To] = true
			delete(created, op.Target)
		case PFRemove:
			for _, n := range op.Names {
				delete(created, n)
			}
		}
	}
	// Move / remove absent files are no-op in the Rocky5 engine, but cannot
	// be inferred to exist: a user can elect to skip those with a warning.
	out := make([]string, 0, len(required))
	for n := range required {
		out = append(out, n)
	}
	sortStrings(out)
	return out
}
func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

func ParsePF(filename string, data []byte) (*PF, error) {
	if len(data) > 32<<20 {
		return nil, errors.New("PF larger than 32 MiB")
	}
	p := &PF{Filename: filename, Header: map[string]string{}}
	var comment string
	for i, line := range strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(line, "\ufeff"))
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") {
			comment = strings.TrimSpace(strings.TrimPrefix(line, "#"))
			if k, v, yes := strings.Cut(comment, ":"); yes {
				key := strings.ToLower(strings.TrimSpace(k))
				switch key {
				case "author", "credits", "info", "region", "title", "titleid", "title id", "type":
					if key == "title id" {
						key = "titleid"
					}
					p.Header[key] = strings.TrimSpace(v)
				}
			}
			continue
		}
		fields := strings.Split(line, "|")
		if len(fields) < 2 || len(strings.TrimSpace(fields[0])) != 2 {
			return nil, fmt.Errorf("PF line %d: invalid command", i+1)
		}
		kind := PFOp(strings.ToUpper(strings.TrimSpace(fields[0])))
		op := PFOperation{Kind: kind, Line: i + 1, Comment: comment}
		file := func(j int) (string, error) {
			if j >= len(fields) {
				return "", errors.New("missing filename")
			}
			return NormaliseTarget(fields[j])
		}
		n, e := file(1)
		if e != nil {
			return nil, fmt.Errorf("PF line %d: %w", i+1, e)
		}
		op.Target = n
		switch kind {
		case PFCopy, PFMove, PFRename:
			if len(fields) != 3 {
				return nil, fmt.Errorf("PF line %d: %s requires source and destination", i+1, kind)
			}
			op.To, e = file(2)
			if e != nil || op.To == op.Target {
				return nil, fmt.Errorf("PF line %d: invalid destination: %v", i+1, e)
			}
		case PFRemove:
			for j := 1; j < len(fields); j++ {
				n, e = file(j)
				if e != nil {
					return nil, fmt.Errorf("PF line %d: %w", i+1, e)
				}
				op.Names = append(op.Names, n)
			}
		case PFHex:
			if len(fields) < 5 || (len(fields)-2)%3 != 0 {
				return nil, fmt.Errorf("PF line %d: HR requires count|find|replace triplets", i+1)
			}
			for j := 2; j < len(fields); j += 3 {
				count, e := strconv.Atoi(strings.TrimSpace(fields[j]))
				if e != nil || count < 1 || count > 1000000 {
					return nil, fmt.Errorf("PF line %d: invalid HR occurrence count", i+1)
				}
				find, e := pfHex(fields[j+1])
				if e != nil {
					return nil, fmt.Errorf("PF line %d HR find: %w", i+1, e)
				}
				replacement, e := pfHex(fields[j+2])
				if e != nil {
					return nil, fmt.Errorf("PF line %d HR replacement: %w", i+1, e)
				}
				op.Pairs = append(op.Pairs, PFPair{Count: count, Find: find, Replace: replacement})
			}
		case PFOffset:
			if len(fields) < 4 || (len(fields)-2)%2 != 0 {
				return nil, fmt.Errorf("PF line %d: OF requires decimal offset|hex triplets", i+1)
			}
			for j := 2; j < len(fields); j += 2 {
				off, e := strconv.ParseInt(strings.TrimSpace(fields[j]), 10, 64)
				if e != nil || off < 0 || off > 512<<20 {
					return nil, fmt.Errorf("PF line %d: invalid decimal OF offset", i+1)
				}
				b, e := pfHex(fields[j+1])
				if e != nil {
					return nil, fmt.Errorf("PF line %d OF: %w", i+1, e)
				}
				if off+int64(len(b)) > 512<<20 {
					return nil, fmt.Errorf("PF line %d: offset exceeds safe limit", i+1)
				}
				op.Pairs = append(op.Pairs, PFPair{Offset: off, Bytes: b})
			}
		case PFMedia, PFRegion:
			if len(fields) < 3 {
				return nil, fmt.Errorf("PF line %d: %s requires one hexadecimal dword", i+1, kind)
			}
			for _, f := range fields[2:] {
				b, e := pfHex(f)
				if e != nil || len(b) != 4 {
					return nil, fmt.Errorf("PF line %d: %s requires eight hexadecimal digits", i+1, kind)
				}
				op.Pairs = append(op.Pairs, PFPair{Bytes: b})
			}
		default:
			return nil, fmt.Errorf("PF line %d: unsupported operation %q", i+1, kind)
		}
		p.Operations = append(p.Operations, op)
	}
	if len(p.Operations) == 0 {
		return nil, errors.New("PF has no operations")
	}
	return p, nil
}
func pfHex(s string) ([]byte, error) {
	s = strings.Join(strings.Fields(s), "")
	if s == "" || len(s)%2 != 0 {
		return nil, errors.New("hex must be nonempty and have an even length")
	}
	b, e := hex.DecodeString(s)
	if e != nil {
		return nil, e
	}
	return b, nil
}

// ApplyPF operates entirely on independent byte copies. For input game
// folders, callers can stage the output directory first, then publish only
// after every operation succeeds. Missing search patterns are warnings to
// match Rocky5's optional-alternative HR behaviour.
func (p *PF) ApplyPF(sources map[string][]byte) (PFOutcome, error) {
	result := PFOutcome{Files: map[string][]byte{}}
	if len(sources) == 0 {
		return result, errors.New("no game source files provided")
	}
	originals := map[string][]byte{}
	for name, b := range sources {
		norm, e := NormaliseTarget(name)
		if e != nil {
			return result, e
		}
		if _, yes := result.Files[norm]; yes {
			return result, fmt.Errorf("ambiguous duplicate %s", norm)
		}
		result.Files[norm] = bytes.Clone(b)
		originals[norm] = bytes.Clone(b)
	}
	touched := map[string]bool{}
	created := map[string]bool{}
	removed := map[string]bool{}
	applyOne := func(op PFOperation) error {
		switch op.Kind {
		case PFCopy:
			src, exists := result.Files[op.Target]
			if !exists {
				return fmt.Errorf("PF line %d: copy source %s missing", op.Line, op.Target)
			}
			if _, exists = result.Files[op.To]; exists {
				return fmt.Errorf("PF line %d: copy target %s already exists (possibly already patched)", op.Line, op.To)
			}
			result.Files[op.To] = bytes.Clone(src)
			created[op.To] = true
			result.Actions++
		case PFMove, PFRename:
			src, exists := result.Files[op.Target]
			if !exists {
				result.Warnings = append(result.Warnings, fmt.Sprintf("line %d: %s not found, move skipped", op.Line, op.Target))
				return nil
			}
			if _, exists = result.Files[op.To]; exists {
				return fmt.Errorf("PF line %d: destination %s already exists", op.Line, op.To)
			}
			result.Files[op.To] = src
			delete(result.Files, op.Target)
			removed[op.Target] = true
			created[op.To] = true
			result.Actions++
		case PFRemove:
			for _, n := range op.Names {
				if _, exists := result.Files[n]; exists {
					delete(result.Files, n)
					removed[n] = true
					result.Actions++
				}
			}
		case PFHex, PFOffset, PFMedia, PFRegion:
			b, ok := result.Files[op.Target]
			if !ok {
				return fmt.Errorf("PF line %d: target %s missing", op.Line, op.Target)
			}
			switch op.Kind {
			case PFHex:
				for _, pair := range op.Pairs {
					at := 0
					hits := 0
					for hits < pair.Count && at <= len(b)-len(pair.Find) {
						k := bytes.Index(b[at:], pair.Find)
						if k < 0 {
							break
						}
						k += at
						next := make([]byte, 0, len(b)-len(pair.Find)+len(pair.Replace))
						next = append(next, b[:k]...)
						next = append(next, pair.Replace...)
						next = append(next, b[k+len(pair.Find):]...)
						if len(next) > 512<<20 {
							return fmt.Errorf("PF line %d: resulting file too large", op.Line)
						}
						b = next
						at = k + len(pair.Replace)
						hits++
					}
					if hits == 0 {
						result.Warnings = append(result.Warnings, fmt.Sprintf("line %d: HR sequence not found in %s", op.Line, op.Target))
					} else if hits < pair.Count {
						result.Warnings = append(result.Warnings, fmt.Sprintf("line %d: HR found %d of %d instances", op.Line, hits, pair.Count))
					}
					result.Actions += hits
				}
			case PFOffset:
				for _, pair := range op.Pairs {
					end := int(pair.Offset) + len(pair.Bytes)
					if end > len(b) {
						b = append(b, make([]byte, end-len(b))...)
					}
					copy(b[int(pair.Offset):], pair.Bytes)
					result.Actions++
				}
			case PFMedia, PFRegion:
				info, e := InspectXBE(b)
				if e != nil {
					return fmt.Errorf("PF line %d: %s: %w", op.Line, op.Target, e)
				}
				_ = info
				load := binary.LittleEndian.Uint32(b[0x104:])
				cert := binary.LittleEndian.Uint32(b[0x118:])
				off := int(cert-load) + 0x9c
				if op.Kind == PFRegion {
					off += 4
				}
				if off < 0 || off+4 > len(b) {
					return fmt.Errorf("PF line %d: XBE certificate field outside file", op.Line)
				}
				for _, pair := range op.Pairs {
					n := binary.BigEndian.Uint32(pair.Bytes)
					binary.LittleEndian.PutUint32(b[off:], n)
					result.Actions++
				}
			}
			result.Files[op.Target] = b
			touched[op.Target] = true
		}
		return nil
	}
	// Copies run first in Rocky5's patcher, even if the CP lines occur after
	// HR lines. Remaining operations run in input order for reproducibility.
	for _, op := range p.Operations {
		if op.Kind == PFCopy {
			if e := applyOne(op); e != nil {
				return PFOutcome{}, e
			}
		}
	}
	for _, op := range p.Operations {
		if op.Kind != PFCopy {
			if e := applyOne(op); e != nil {
				return PFOutcome{}, e
			}
		}
	}
	for name, b := range result.Files {
		old, ok := originals[name]
		if !ok {
			created[name] = true
		}
		if touched[name] && (!ok || !bytes.Equal(old, b)) {
			result.Modified = append(result.Modified, name)
		}
	}
	for name := range removed {
		if _, ok := result.Files[name]; !ok {
			result.Removed = append(result.Removed, name)
		}
	}
	for name := range created {
		if _, ok := result.Files[name]; ok {
			result.Created = append(result.Created, name)
		}
	}
	sortStrings(result.Modified)
	sortStrings(result.Removed)
	sortStrings(result.Created)
	return result, nil
}

func (p *PF) CheckTitleID(source map[string][]byte) error {
	wanted := strings.ToUpper(strings.TrimSpace(p.Header["titleid"]))
	if wanted == "" || wanted == "MULTI" {
		return nil
	}
	if len(wanted) != 8 {
		return fmt.Errorf("PF TitleID %q is not eight hex digits", wanted)
	}
	if _, e := hex.DecodeString(wanted); e != nil {
		return fmt.Errorf("invalid PF TitleID: %w", e)
	}
	// The original script validates the first XBE it processes. Validate every
	// relevant original executable, excluding files created by CP.
	for _, op := range p.Operations {
		if !strings.HasSuffix(strings.ToLower(op.Target), ".xbe") {
			continue
		}
		b, ok := source[op.Target]
		if !ok {
			continue
		}
		x, e := InspectXBE(b)
		if e != nil {
			return fmt.Errorf("%s: %w", op.Target, e)
		}
		if !strings.EqualFold(x.TitleID, wanted) {
			return fmt.Errorf("PF TitleID mismatch: %s is %s, expected %s", op.Target, x.TitleID, wanted)
		}
	}
	return nil
}

func (p *PF) Description() string {
	return fmt.Sprintf("%s | %s | %s | %d operations", p.Header["title"], p.Header["region"], p.Header["type"], len(p.Operations))
}

func PFName(path string) string { return strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)) }
