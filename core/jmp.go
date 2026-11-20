package core

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"hash/crc32"
	"regexp"
	"strconv"
	"strings"
)

type RecordKind string

const (
	SearchReplace RecordKind = "search"
	Offset        RecordKind = "offset"
	Append        RecordKind = "append"
)

type Record struct {
	Kind    RecordKind
	Comment string
	Group   string
	Enabled bool
	Offset  int64
	Find    []byte
	Replace []byte
	Data    []byte
	Line    int
}
type Patch struct {
	Filename string
	Headers  map[string]string
	Records  []Record
	Groups   []Group
	CRCs     []uint32
}
type Group struct {
	Name    string
	Count   int
	Enabled bool
}

var crcRx = regexp.MustCompile(`(?i)\{([0-9a-f]{8})\}`)
var hexRx = regexp.MustCompile(`^[0-9a-fA-F]+$`)

func parseHex(s string) ([]byte, error) {
	s = strings.Join(strings.Fields(s), "")
	if s == "" || len(s)%2 != 0 || !hexRx.MatchString(s) {
		return nil, fmt.Errorf("invalid even-length hexadecimal string: %q", s)
	}
	return hex.DecodeString(s)
}
func defaultGroup(s string) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "[-]", ""))
	if s == "" {
		return "Untitled record"
	}
	return s
}
func ParseJMP(name string, data []byte) (*Patch, error) {
	p := &Patch{Filename: name, Headers: map[string]string{}}
	lines := []string{}
	scan := bufio.NewScanner(bytes.NewReader(data))
	scan.Buffer(make([]byte, 4096), 32<<20)
	for scan.Scan() {
		lines = append(lines, strings.TrimSpace(strings.TrimPrefix(scan.Text(), "\ufeff")))
	}
	if err := scan.Err(); err != nil {
		return nil, err
	}
	i := 0
	for i < len(lines) && len(p.Headers) < 6 {
		line := lines[i]
		i++
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("bad JMP header on line %d", i)
		}
		key := strings.ToLower(strings.TrimSpace(parts[0]))
		switch key {
		case "system", "game-title", "region", "version", "author", "notes":
			p.Headers[key] = strings.TrimSpace(parts[1])
		default:
			return nil, fmt.Errorf("unexpected header %s", key)
		}
	}
	for _, k := range []string{"system", "game-title", "region", "version", "author", "notes"} {
		if _, ok := p.Headers[k]; !ok {
			return nil, fmt.Errorf("missing JMP header %s", k)
		}
	}
	for _, m := range crcRx.FindAllStringSubmatch(p.Headers["version"], -1) {
		n, _ := strconv.ParseUint(m[1], 16, 32)
		p.CRCs = append(p.CRCs, uint32(n))
	}
	groupIndex := map[string]int{}
	for i < len(lines) {
		if lines[i] == "" {
			i++
			continue
		}
		if !strings.HasPrefix(lines[i], "#") {
			return nil, fmt.Errorf("patch record on line %d requires a # comment", i+1)
		}
		comment := strings.TrimSpace(strings.TrimPrefix(lines[i], "#"))
		enabled := !strings.Contains(comment, "[-]")
		group := defaultGroup(comment)
		i++
		for i < len(lines) && lines[i] == "" {
			i++
		}
		if i >= len(lines) {
			break
		}
		if strings.HasPrefix(lines[i], "#") {
			continue
		}
		start := i + 1
		raw := lines[i]
		i++
		rec := Record{Comment: group, Group: group, Enabled: enabled, Line: start}
		if strings.HasPrefix(strings.ToUpper(raw), "APPEND:") {
			v, err := parseHex(raw[len("APPEND:"):])
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", start, err)
			}
			rec.Kind = Append
			rec.Data = v
		} else if strings.Contains(raw, ":") {
			parts := strings.SplitN(raw, ":", 2)
			s := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(parts[0])), "0x")
			n, err := strconv.ParseUint(s, 16, 63)
			if err != nil {
				return nil, fmt.Errorf("line %d: invalid hex offset %q", start, parts[0])
			}
			v, err := parseHex(parts[1])
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", start, err)
			}
			rec.Kind = Offset
			rec.Offset = int64(n)
			rec.Data = v
		} else {
			v, err := parseHex(raw)
			if err != nil {
				return nil, fmt.Errorf("line %d search: %w", start, err)
			}
			for i < len(lines) && lines[i] == "" {
				i++
			}
			if i >= len(lines) || strings.HasPrefix(lines[i], "#") {
				return nil, fmt.Errorf("line %d missing replacement", start)
			}
			to, err := parseHex(lines[i])
			if err != nil {
				return nil, fmt.Errorf("line %d replace: %w", i+1, err)
			}
			i++
			if len(v) != len(to) {
				return nil, fmt.Errorf("line %d: search/replace lengths differ", start)
			}
			rec.Kind = SearchReplace
			rec.Find = v
			rec.Replace = to
		}
		p.Records = append(p.Records, rec)
		key := strings.ToLower(group)
		idx, ok := groupIndex[key]
		if !ok {
			idx = len(p.Groups)
			groupIndex[key] = idx
			p.Groups = append(p.Groups, Group{Name: group, Enabled: true})
		}
		p.Groups[idx].Count++
		if !enabled {
			p.Groups[idx].Enabled = false
		}
	}
	if len(p.Records) == 0 {
		return nil, errors.New("no patch records found")
	}
	return p, nil
}
func (p *Patch) EnableGroup(name string, enable bool) {
	for i := range p.Records {
		if strings.EqualFold(p.Records[i].Group, name) {
			p.Records[i].Enabled = enable
		}
	}
	for i := range p.Groups {
		if strings.EqualFold(p.Groups[i].Name, name) {
			p.Groups[i].Enabled = enable
		}
	}
}
func (p *Patch) ExpectedCRC() string {
	if len(p.CRCs) == 0 {
		return "No CRC32 required"
	}
	s := []string{}
	for _, c := range p.CRCs {
		s = append(s, fmt.Sprintf("%08X", c))
	}
	return strings.Join(s, " / ")
}
func (p *Patch) VerifyCRC(input []byte) error {
	if len(p.CRCs) == 0 {
		return nil
	}
	got := crc32.ChecksumIEEE(input)
	for _, v := range p.CRCs {
		if v == got {
			return nil
		}
	}
	return fmt.Errorf("CRC32 mismatch: file %08X; patch expects %s", got, p.ExpectedCRC())
}
func (p *Patch) Apply(input []byte, overrideCRC bool) ([]byte, error) {
	if !overrideCRC {
		if titleId := TitleIDFromVersion(p.Headers["version"]); titleId != "" {
			if info, err := InspectXBE(input); err == nil && !strings.EqualFold(info.TitleID, titleId) {
				return nil, fmt.Errorf("Title ID mismatch: XBE is %s but patch expects %s", info.TitleID, titleId)
			}
		}
	}
	if !overrideCRC {
		if err := p.VerifyCRC(input); err != nil {
			return nil, err
		}
	}
	out := append([]byte{}, input...)
	for i, r := range p.Records {
		if !r.Enabled {
			continue
		}
		switch r.Kind {
		case Offset:
			if r.Offset < 0 || r.Offset+int64(len(r.Data)) > int64(len(out)) {
				return nil, fmt.Errorf("record %d (%s): offset 0x%X outside XBE", i+1, r.Group, r.Offset)
			}
			copy(out[r.Offset:], r.Data)
		case SearchReplace:
			pos := bytes.Index(out, r.Find)
			if pos < 0 {
				return nil, fmt.Errorf("record %d (%s): search pattern not found", i+1, r.Group)
			}
			copy(out[pos:], r.Replace)
		case Append:
			if len(out)+len(r.Data) > 512<<20 {
				return nil, fmt.Errorf("record %d: XBE exceeds 512 MiB safety cap", i+1)
			}
			out = append(out, r.Data...)
		default:
			return nil, fmt.Errorf("unknown record kind %s", r.Kind)
		}
	}
	return out, nil
}
