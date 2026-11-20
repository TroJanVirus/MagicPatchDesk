package core

import (
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"strings"
	"unicode/utf16"
)

type XBEInfo struct {
	Title   string
	TitleID string
	Region  string
	CRC32   string
	Size    int64
	Valid   bool
}

func InspectXBE(data []byte) (XBEInfo, error) {
	v := XBEInfo{CRC32: fmt.Sprintf("%08X", crc32.ChecksumIEEE(data)), Size: int64(len(data))}
	if len(data) < 0x200 || string(data[:4]) != "XBEH" {
		return v, errors.New("not a valid XBE header (XBEH missing)")
	}
	base := binary.LittleEndian.Uint32(data[0x104:])
	cert := binary.LittleEndian.Uint32(data[0x118:])
	if cert < base {
		return v, errors.New("XBE certificate address below image base")
	}
	c := int64(cert - base)
	if c < 0 || c+0xA4 > int64(len(data)) {
		return v, errors.New("XBE certificate outside file")
	}
	v.TitleID = fmt.Sprintf("%08X", binary.LittleEndian.Uint32(data[c+8:]))
	u := []uint16{}
	for i := int64(0); i < 40; i++ {
		x := binary.LittleEndian.Uint16(data[c+12+i*2:])
		if x == 0 {
			break
		}
		u = append(u, x)
	}
	v.Title = strings.TrimSpace(string(utf16.Decode(u)))
	if v.Title == "" {
		v.Title = "Unknown Xbox title"
	}
	r := binary.LittleEndian.Uint32(data[c+0xA0:])
	bits := []string{}
	if r&1 != 0 {
		bits = append(bits, "NTSC-U")
	}
	if r&2 != 0 {
		bits = append(bits, "NTSC-J")
	}
	if r&4 != 0 {
		bits = append(bits, "PAL")
	}
	if r&0x80000000 != 0 {
		bits = append(bits, "Debug")
	}
	v.Region = strings.Join(bits, " / ")
	if v.Region == "" {
		v.Region = "Unknown"
	}
	v.Valid = true
	return v, nil
}
