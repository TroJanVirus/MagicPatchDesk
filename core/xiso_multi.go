package core

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// ResolveTarget finds the unique XDVDFS member named by a JMP comment.
// A relative path is matched exactly; a bare filename is only accepted when
// it is unique in the image. Ambiguous names must not be guessed.
func (im *Image) ResolveTarget(target string) (ImageFile, error) {
	normalized, err := NormaliseTarget(target)
	if err != nil {
		return ImageFile{}, err
	}
	if normalized == "default.xbe" {
		return im.DefaultXBE()
	}
	matches := make([]ImageFile, 0, 1)
	for _, f := range im.Files {
		name := strings.ToLower(strings.ReplaceAll(f.Path, "\\", "/"))
		if name == normalized || (!strings.Contains(normalized, "/") && strings.EqualFold(path.Base(name), normalized)) {
			matches = append(matches, f)
		}
	}
	if len(matches) == 0 {
		return ImageFile{}, fmt.Errorf("target file %q is not in this XISO", target)
	}
	if len(matches) > 1 {
		return ImageFile{}, fmt.Errorf("target file %q is ambiguous in this XISO (%d matches); specify its relative path in the JMP comment", target, len(matches))
	}
	return matches[0], nil
}

// PatchImageFiles replaces multiple EXISTING files within an XDVDFS image.
// A game-partition XISO can relocate expanding files; a full-disc ISO refuses
// growth beyond the original file's sector allocation. The destination must
// be a separate path (callers should stage it for atomic publication).
func PatchImageFiles(input, output string, replacements map[string][]byte, workers int, onProgress func(int64, int64)) error {
	if len(replacements) == 0 {
		return errors.New("no XISO files selected for patching")
	}
	if strings.EqualFold(filepath.Clean(input), filepath.Clean(output)) {
		return errors.New("output must differ from source")
	}
	im, err := OpenXISO(input)
	if err != nil {
		return err
	}
	type pending struct {
		name  string
		entry ImageFile
		data  []byte
	}
	jobs := make([]pending, 0, len(replacements))
	seen := map[int64]bool{}
	for name, data := range replacements {
		f, e := im.ResolveTarget(name)
		if e != nil {
			return e
		}
		if seen[f.EntryAt] {
			return fmt.Errorf("same XISO member selected twice: %s", f.Path)
		}
		seen[f.EntryAt] = true
		if int64(len(data)) > int64(^uint32(0)) {
			return fmt.Errorf("file %s larger than XDVDFS 32-bit length", f.Path)
		}
		newSectors := (int64(len(data)) + sectorSize - 1) / sectorSize
		oldSectors := (int64(f.Size) + sectorSize - 1) / sectorSize
		if newSectors > oldSectors && im.Base != 0 {
			return fmt.Errorf("full-disc ISO cannot safely grow %q beyond its allocated sectors; use a game-partition XISO", f.Path)
		}
		jobs = append(jobs, pending{name: name, entry: f, data: data})
	}
	sort.Slice(jobs, func(i, j int) bool { return jobs[i].entry.EntryAt < jobs[j].entry.EntryAt })
	if err := CopyImage(input, output, workers, onProgress); err != nil {
		return err
	}
	success := false
	defer func() {
		if !success {
			os.Remove(output)
		}
	}()
	f, err := os.OpenFile(output, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil {
		return err
	}
	appendAt := (stat.Size() + sectorSize - 1) &^ (sectorSize - 1)
	for _, job := range jobs {
		e := job.entry
		oldSectors := (int64(e.Size) + sectorSize - 1) / sectorSize
		newSectors := (int64(len(job.data)) + sectorSize - 1) / sectorSize
		offset := im.Base + int64(e.StartSector)*sectorSize
		if newSectors > oldSectors {
			offset = appendAt
			next := offset + newSectors*sectorSize
			if next < offset {
				return errors.New("ISO sector allocation overflow")
			}
			relative := (offset - im.Base) / sectorSize
			if relative < 0 || relative > int64(^uint32(0)) {
				return errors.New("ISO exceeds 32-bit sector allocation")
			}
			e.StartSector = uint32(relative)
			appendAt = next
		}
		if _, err = f.WriteAt(job.data, offset); err != nil {
			return err
		}
		if padding := newSectors*sectorSize - int64(len(job.data)); padding > 0 {
			if _, err = f.WriteAt(bytes.Repeat([]byte{0xFF}, int(padding)), offset+int64(len(job.data))); err != nil {
				return err
			}
		}
		payload := make([]byte, 8)
		binary.LittleEndian.PutUint32(payload, e.StartSector)
		binary.LittleEndian.PutUint32(payload[4:], uint32(len(job.data)))
		if _, err = f.WriteAt(payload, e.EntryAt+4); err != nil {
			return err
		}
	}
	if err = f.Sync(); err != nil {
		return err
	}
	success = true
	return nil
}
