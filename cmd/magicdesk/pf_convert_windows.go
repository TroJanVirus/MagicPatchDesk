//go:build windows

package main

import (
	"fmt"
	"magicpatchdesk/core"
	"os"
	"path/filepath"
	"strings"
)

func pfSourcePaths(p *core.PF, chosen string) (map[string][]byte, error) {
	files := map[string][]byte{}
	if chosen == "" {
		return files, nil
	}
	if strings.EqualFold(filepath.Ext(chosen), ".iso") {
		im, e := core.OpenXISO(chosen)
		if e != nil {
			return nil, e
		}
		for _, target := range p.DeclaredNames() {
			f, e := im.ResolveTarget(target)
			if e != nil {
				return nil, e
			}
			b, e := im.ReadFile(f)
			if e != nil {
				return nil, e
			}
			files[target] = b
		}
	} else {
		var e error
		files, e = p.ReadPFDirectoryInputs(filepath.Dir(chosen))
		if e != nil {
			return nil, e
		}
	}
	if e := p.CheckTitleID(files); e != nil {
		return nil, e
	}
	return files, nil
}
func pfOriginalsReady(p *core.PF) bool {
	if app.aFile == "" {
		return false
	}
	if strings.EqualFold(filepath.Ext(app.aFile), ".iso") {
		im, e := core.OpenXISO(app.aFile)
		if e != nil {
			return false
		}
		for _, t := range p.TargetNames() {
			if _, e := im.ResolveTarget(t); e != nil {
				return false
			}
		}
		return true
	}
	for _, t := range p.TargetNames() {
		f := filepath.Join(filepath.Dir(app.aFile), filepath.FromSlash(t))
		stat, e := os.Stat(f)
		if e != nil || stat.IsDir() {
			return false
		}
	}
	return true
}

// Convert controls stay disabled until source-dependent PF records have their
// complete original files. Unsupported file-system effects are never discarded.
func refreshPFConversionHint(name string, raw []byte) bool {
	if !strings.EqualFold(filepath.Ext(name), ".pf") {
		return false
	}
	app.packMode = false
	pf, e := core.ParsePF(name, raw)
	if e != nil {
		app.conversionBlocked = true
		app.conversionHint = "Invalid PF: " + e.Error()
		return true
	}
	if pf.HasFilesystemOps() {
		app.conversionBlocked = true
		app.conversionHint = "This PF copies, moves, renames or removes files. Apply it as PF; conversion would lose file operations."
		return true
	}
	if app.format == 0 {
		if pf.NeedsOriginalForJMP() && !pfOriginalsReady(pf) {
			app.conversionBlocked = true
			app.conversionHint = "This PF→JMP conversion requires the original files for search context or XBE certificate offsets."
		} else {
			app.conversionBlocked = false
			app.conversionHint = "PF byte operations can be converted into one multi-file JMP."
		}
		return true
	}
	if !pfOriginalsReady(pf) {
		app.conversionBlocked = true
		app.conversionHint = "This PF conversion requires the original game files. Choose a stock XBE or XISO."
		return true
	}
	app.conversionBlocked = false
	if app.format == core.AllPatchFormats {
		app.conversionHint = "Create an all-format pack from a PF's byte-only changes (PF filesystem commands are not supported)."
	} else {
		app.conversionHint = "Original files verified. This PF can be converted without losing byte edits."
	}
	return true
}

func runPFConversion(name string, raw []byte) {
	pf, err := core.ParsePF(name, raw)
	if err != nil {
		notify(err.Error())
		return
	}
	if pf.HasFilesystemOps() {
		notify("PF has CP/MV/RN/RM file operations; cannot export a lossless JMP or binary patch pack.")
		return
	}
	originals, err := pfSourcePaths(pf, app.aFile)
	if err != nil {
		notify(err.Error())
		return
	}
	if pf.NeedsOriginalForJMP() && len(originals) == 0 {
		notify("Original game file required for this PF conversion")
		return
	}
	fmtN := app.format
	ext := ".jmp"
	targets := pf.TargetNames()
	if fmtN != 0 {
		if len(targets) > 1 || fmtN == core.AllPatchFormats {
			ext = core.PatchPackExtension
		} else {
			ext = map[int]string{1: ".ips", 2: ".xdelta", 3: ".eips", 4: ".bps"}[fmtN]
		}
	}
	dest := pickFile(true, "Patch file\x00*.jmp;*.mpdpack;*.ips;*.eips;*.bps;*.xdelta\x00", "Save converted PF patch", strings.TrimSuffix(name, filepath.Ext(name))+ext)
	if dest == "" {
		return
	}
	if filepath.Ext(dest) == "" {
		dest += ext
	}
	if !strings.EqualFold(filepath.Ext(dest), ext) {
		notify("Output extension must be " + ext)
		return
	}
	for _, input := range []string{app.convertPatchFile, app.aFile, app.directFile} {
		if input != "" && strings.EqualFold(filepath.Clean(input), filepath.Clean(dest)) {
			notify("Output must not overwrite input")
			return
		}
	}
	password, rle := app.password, app.ipsRLE
	app.mu.Lock()
	if app.busy {
		app.mu.Unlock()
		return
	}
	app.busy = true
	app.status = "Converting XBMC4Gamers PF…"
	app.mu.Unlock()
	redraw()
	go func() {
		var output []byte
		converted, e := pf.ToJMP(originals)
		if e == nil && fmtN == 0 {
			output = converted
		}
		if e == nil && fmtN == core.AllPatchFormats {
			var manifest core.PackManifest
			output, manifest, e = core.BuildJMPAllFormatsPack(name+".jmp", converted, originals, password, rle)
			_ = manifest
		} else if e == nil && fmtN != 0 {
			patch, e2 := core.ParseJMP(name+".jmp", converted)
			if e2 != nil {
				e = e2
			} else {
				var targets []string
				targets, e = patch.TargetFiles()
				if e == nil && len(targets) > 1 {
					output, _, e = core.BuildJMPPatchPack(name+".jmp", converted, originals, fmtN, password, rle)
				}
				if e == nil && len(targets) == 1 {
					var src []byte
					src = originals[targets[0]]
					if src == nil {
						e = fmt.Errorf("original %s required", targets[0])
					}
					if e == nil {
						scoped, er := patch.ScopedPatch(targets[0])
						if er != nil {
							e = er
						} else {
							after, er := scoped.ApplyToTarget(src, targets[0], true)
							if er != nil {
								e = er
							} else {
								output, e = core.BuildPatchWithOptions(src, after, fmtN, password, pf.Header["title"], rle)
							}
						}
					}
				}
			}
		}
		if e == nil {
			e = atomicWrite(dest, output)
		}
		app.mu.Lock()
		app.busy = false
		if e != nil {
			app.status = "PF conversion failed: " + e.Error()
		} else {
			app.status = "Converted PF saved: " + dest
		}
		app.mu.Unlock()
		redraw()
	}()
}
