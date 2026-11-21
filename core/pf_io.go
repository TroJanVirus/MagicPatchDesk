package core

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// DeclaredNames contains potential existing source AND destination files.
// Checking existing copy/rename destinations prevents accidental overwrites.
func (p *PF) DeclaredNames() []string {
	seen := map[string]bool{}
	for _, op := range p.Operations {
		if op.Kind == PFRemove {
			for _, n := range op.Names {
				seen[n] = true
			}
		} else {
			seen[op.Target] = true
		}
		if op.To != "" {
			seen[op.To] = true
		}
	}
	out := make([]string, 0, len(seen))
	for n := range seen {
		out = append(out, n)
	}
	sortStrings(out)
	return out
}

// ReadPFDirectoryInputs loads only files explicitly referenced in the PF,
// without reading every large game asset into memory. Physical symlinks are
// rejected for selected files; source paths are always relative to gameRoot.
func pfDirectoryNames(gameRoot string) (map[string]string, error) {
	names := make(map[string]string)
	err := filepath.WalkDir(gameRoot, func(disk string, entry os.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink in game directory: %s", disk)
		}
		if entry.IsDir() {
			return nil
		}
		relative, e := filepath.Rel(gameRoot, disk)
		if e != nil {
			return e
		}
		normalized, e := NormaliseTarget(filepath.ToSlash(relative))
		if e != nil {
			return e
		}
		if prior, ok := names[normalized]; ok && prior != relative {
			return fmt.Errorf("case-insensitive filename collision: %s and %s", prior, relative)
		}
		names[normalized] = relative
		return nil
	})
	return names, err
}

func (p *PF) ReadPFDirectoryInputs(gameRoot string) (map[string][]byte, error) {
	names, e := pfDirectoryNames(gameRoot)
	if e != nil {
		return nil, e
	}
	files := map[string][]byte{}
	for _, name := range p.DeclaredNames() {
		relative, ok := names[name]
		if !ok {
			continue
		}
		disk := filepath.Join(gameRoot, relative)
		f, e := os.Lstat(disk)
		if e != nil {
			return nil, e
		}
		if !f.Mode().IsRegular() {
			return nil, fmt.Errorf("PF expects regular file: %s", name)
		}
		if f.Size() > 512<<20 {
			return nil, fmt.Errorf("PF file %s exceeds 512MiB safety limit", name)
		}
		b, e := os.ReadFile(disk)
		if e != nil {
			return nil, e
		}
		files[name] = b
	}
	return files, nil
}

func (p *PF) PreviewDirectory(gameRoot string) (PFOutcome, error) {
	originals, e := p.ReadPFDirectoryInputs(gameRoot)
	if e != nil {
		return PFOutcome{}, e
	}
	if e = p.CheckTitleID(originals); e != nil {
		return PFOutcome{}, e
	}
	return p.ApplyPF(originals)
}

// ApplyPFDirectory creates a complete independent copy of the game directory
// and applies all PF file and byte operations to that staged copy. A successful
// rename publishes the new game directory. On errors, staging is removed and
// the original is never mutated. Deliberately refuses source symlinks.
func (p *PF) ApplyPFDirectory(ctx context.Context, srcDir, destDir string, progress func(int64, int64)) (PFOutcome, error) {
	var empty PFOutcome
	src, e := filepath.Abs(srcDir)
	if e != nil {
		return empty, e
	}
	dest, e := filepath.Abs(destDir)
	if e != nil {
		return empty, e
	}
	if strings.EqualFold(src, dest) || strings.HasPrefix(strings.ToLower(dest), strings.ToLower(src)+string(filepath.Separator)) {
		return empty, errors.New("destination must not be the source or inside the source game folder")
	}
	info, e := os.Stat(src)
	if e != nil {
		return empty, e
	}
	if !info.IsDir() {
		return empty, errors.New("source game is not a directory")
	}
	if _, e = os.Stat(dest); e == nil {
		return empty, errors.New("destination game folder already exists")
	}
	if !os.IsNotExist(e) {
		return empty, e
	}
	physicalNames, e := pfDirectoryNames(src)
	if e != nil {
		return empty, e
	}
	originals, e := p.ReadPFDirectoryInputs(src)
	if e != nil {
		return empty, e
	}
	if e = p.CheckTitleID(originals); e != nil {
		return empty, e
	}
	outcome, e := p.ApplyPF(originals)
	if e != nil {
		return empty, e
	}
	// Stage in destination's parent for an atomic same-volume final rename.
	parent := filepath.Dir(dest)
	if e = os.MkdirAll(parent, 0755); e != nil {
		return empty, e
	}
	stage, e := os.MkdirTemp(parent, ".magicdesk-pf-*")
	if e != nil {
		return empty, e
	}
	defer os.RemoveAll(stage)
	var total int64
	e = filepath.WalkDir(src, func(p string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if ctx != nil {
			if err = ctx.Err(); err != nil {
				return err
			}
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink in game folder: %s", p)
		}
		if !entry.IsDir() {
			s, err := entry.Info()
			if err != nil {
				return err
			}
			total += s.Size()
		}
		return nil
	})
	if e != nil {
		return empty, e
	}
	var copied int64
	buf := make([]byte, 1024*1024)
	e = filepath.WalkDir(src, func(p string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if ctx != nil {
			if err = ctx.Err(); err != nil {
				return err
			}
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink in game folder: %s", p)
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		dst := filepath.Join(stage, rel)
		if entry.IsDir() {
			return os.MkdirAll(dst, 0755)
		}
		fi, err := entry.Info()
		if err != nil {
			return err
		}
		inf, err := os.Open(p)
		if err != nil {
			return err
		}
		outf, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, fi.Mode().Perm())
		if err != nil {
			inf.Close()
			return err
		}
		n, copyErr := io.CopyBuffer(outf, inf, buf)
		closeErr := outf.Close()
		inf.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		copied += n
		if progress != nil {
			progress(copied, total)
		}
		return nil
	})
	if e != nil {
		return empty, e
	}
	// Every output name originates from validated NormaliseTarget.
	for _, name := range p.DeclaredNames() {
		targetName := filepath.FromSlash(name)
		if prior, ok := physicalNames[name]; ok {
			targetName = prior
		}
		target := filepath.Join(stage, targetName)
		updated, exists := outcome.Files[name]
		if !exists {
			if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
				return empty, err
			}
			continue
		}
		if e = os.MkdirAll(filepath.Dir(target), 0755); e != nil {
			return empty, e
		}
		if e = os.WriteFile(target, updated, 0644); e != nil {
			return empty, e
		}
	}
	if _, e = os.Stat(dest); e == nil {
		return empty, errors.New("destination appeared while patching")
	}
	if e = os.Rename(stage, dest); e != nil {
		return empty, e
	}
	return outcome, nil
}

// ApplyPFToISO patches only already-existing XDVDFS files. PF file-system
// operations (CP/MV/RN/RM) fundamentally require repacking the directory
// tree, so these are refused rather than silently omitted or misapplied.
func (p *PF) ApplyPFToISO(inputISO, outputISO string, workers int, progress func(int64, int64)) (PFOutcome, error) {
	var empty PFOutcome
	if p.HasFilesystemOps() {
		return empty, errors.New("PF copy/move/rename/delete needs an extracted game directory; this XISO writer can only replace existing file contents")
	}
	im, e := OpenXISO(inputISO)
	if e != nil {
		return empty, e
	}
	files := map[string][]byte{}
	for _, name := range p.DeclaredNames() {
		f, err := im.ResolveTarget(name)
		if err != nil {
			return empty, err
		}
		data, err := im.ReadFile(f)
		if err != nil {
			return empty, err
		}
		files[name] = data
	}
	if e = p.CheckTitleID(files); e != nil {
		return empty, e
	}
	result, e := p.ApplyPF(files)
	if e != nil {
		return empty, e
	}
	replacements := map[string][]byte{}
	for _, name := range result.Modified {
		replacements[name] = result.Files[name]
	}
	if len(replacements) == 0 {
		return result, errors.New("PF made no byte changes in XISO")
	}
	if e = PatchImageFiles(inputISO, outputISO, replacements, workers, progress); e != nil {
		return empty, e
	}
	return result, nil
}

// PreviewPFISO performs the same preflight as ApplyPFToISO but never writes
// an image. A PF containing filesystem operations must use a game directory.
func (p *PF) PreviewPFISO(inputISO string) (PFOutcome, error) {
	var empty PFOutcome
	if p.HasFilesystemOps() {
		return empty, errors.New("PF copies/renames/deletes files; extract the game first")
	}
	im, e := OpenXISO(inputISO)
	if e != nil {
		return empty, e
	}
	files := map[string][]byte{}
	for _, name := range p.DeclaredNames() {
		f, err := im.ResolveTarget(name)
		if err != nil {
			return empty, err
		}
		b, err := im.ReadFile(f)
		if err != nil {
			return empty, err
		}
		files[name] = b
	}
	if e = p.CheckTitleID(files); e != nil {
		return empty, e
	}
	return p.ApplyPF(files)
}
