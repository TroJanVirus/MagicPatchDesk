package core

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func ApplyPatchFile(original []byte, patchPath, password string, overrideCRC bool) ([]byte, error) {
	b, err := os.ReadFile(patchPath)
	if err != nil {
		return nil, err
	}
	switch strings.ToLower(filepath.Ext(patchPath)) {
	case ".jmp":
		p, err := ParseJMP(filepath.Base(patchPath), b)
		if err != nil {
			return nil, err
		}
		return p.Apply(original, overrideCRC)
	case ".ips":
		p, err := DecodeIPS(b)
		if err != nil {
			return nil, err
		}
		return p.Apply(original)
	case ".eips":
		dec, err := DecryptIPS(b, password)
		if err != nil {
			return nil, err
		}
		p, err := DecodeIPS(dec)
		if err != nil {
			return nil, err
		}
		return p.Apply(original)
	case ".xdelta", ".vcdiff":
		return DecodeXdelta(original, b)
	case ".bps":
		return DecodeBPS(original, b)
	default:
		return nil, fmt.Errorf("unsupported patch file %q", filepath.Ext(patchPath))
	}
}
func BuildPatch(original, modified []byte, format int, password string, title string) ([]byte, error) {
	return BuildPatchWithOptions(original, modified, format, password, title, true)
}

// BuildPatchWithOptions explicitly controls IPS run-length encoding for both
// classic IPS and encrypted IPS payloads.
func BuildPatchWithOptions(original, modified []byte, format int, password string, title string, ipsRLE bool) ([]byte, error) {
	switch format {
	case 0:
		s, err := DiffToJMP(original, modified, title)
		return []byte(s), err
	case 1:
		return EncodeIPSWithRLE(original, modified, ipsRLE)
	case 2:
		return EncodeXdelta(original, modified)
	case 3:
		b, err := EncodeIPSWithRLE(original, modified, ipsRLE)
		if err != nil {
			return nil, err
		}
		return EncryptIPS(b, password)
	case 4:
		return EncodeBPS(original, modified)
	default:
		return nil, errors.New("unsupported output format")
	}
}
