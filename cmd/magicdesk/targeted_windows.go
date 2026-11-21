//go:build windows

package main

import (
	"fmt"
	"magicpatchdesk/core"
	"os"
	"path/filepath"
	"strings"
	"unsafe"
)

// resolvedJMP selects a library JMP with the user's record toggles intact,
// or loads a directly selected JMP. Other patch formats remain single-target.
func resolvedJMP(selected *core.Patch, direct string) (*core.Patch, error) {
	if direct == "" {
		return selected, nil
	}
	if !strings.EqualFold(filepath.Ext(direct), ".jmp") {
		return nil, nil
	}
	b, err := os.ReadFile(direct)
	if err != nil {
		return nil, err
	}
	return core.ParseJMP(filepath.Base(direct), b)
}

type targetChoice struct {
	name     string
	diskFile string
	inISO    bool
}

func chooseSpecialTargets(p *core.Patch, source string) ([]targetChoice, error) {
	targets, err := p.ExternalTargets()
	if err != nil {
		return nil, err
	}
	if len(targets) == 0 {
		return nil, nil
	}
	var im *core.Image
	if !strings.EqualFold(filepath.Ext(source), ".xbe") {
		im, err = core.OpenXISO(source)
		if err != nil {
			return nil, err
		}
	}
	choices := make([]targetChoice, 0, len(targets))
	for _, name := range targets {
		found := ""
		if im != nil {
			member, err := im.ResolveTarget(name)
			if err == nil {
				found = member.Path
			}
		}
		var prompt string
		if found != "" {
			prompt = fmt.Sprintf("This JMP contains records for %q.\n\nFound in the ISO: %s\n\nYes = Patch this file inside the ISO\nNo = Skip all records for this file\nCancel = Cancel the entire patch", name, found)
		} else {
			prompt = fmt.Sprintf("This JMP contains records targeting %q, not the default XBE.\n\nYes = Locate that file on your computer\nNo = Skip all records for this file\nCancel = Cancel the entire patch\n\nOriginal files are NEVER overwritten.", name)
		}
		result := sysCall(targetedMessageBox, app.hwnd, uintptr(unsafe.Pointer(ptr(prompt))), uintptr(unsafe.Pointer(ptr("Special file patch — attention required"))), 0x03|0x30|0x100) // YES/NO/CANCEL, warning, default No
		switch result {
		case 7:
			continue // No: skip this target
		case 6:
			if found != "" {
				choices = append(choices, targetChoice{name: name, inISO: true})
				continue
			}
			picked := pickFile(false, "All files\x00*.*\x00", "Locate original "+name, "")
			if picked == "" {
				return nil, fmt.Errorf("no file selected for %s; patch cancelled", name)
			}
			if !strings.EqualFold(filepath.Base(picked), filepath.Base(filepath.FromSlash(name))) {
				return nil, fmt.Errorf("selected %q, but JMP requests %q; choose the correct filename", filepath.Base(picked), name)
			}
			choices = append(choices, targetChoice{name: name, diskFile: picked})
		default:
			return nil, fmt.Errorf("patch cancelled")
		}
	}
	return choices, nil
}

// Multi-file patches are handled with their own transaction; never pass them
// to the regular default.xbe-only preview/apply pipeline.
func maybeStartTargeted(previewOnly bool, selected *core.Patch, direct, source string, workers int, override bool) (bool, error) {
	p, err := resolvedJMP(selected, direct)
	if err != nil {
		return true, err
	}
	if p == nil {
		return false, nil
	}
	targets, err := p.ExternalTargets()
	if err != nil {
		return true, err
	}
	if len(targets) == 0 {
		return false, nil
	}
	choices, err := chooseSpecialTargets(p, source)
	if err != nil {
		return true, err
	}
	hasDefault := false
	files, err := p.TargetFiles()
	if err != nil {
		return true, err
	}
	for _, n := range files {
		if n == "default.xbe" {
			hasDefault = true
		}
	}
	if len(choices) == 0 && !hasDefault {
		return true, fmt.Errorf("all special-file records skipped; no enabled patch records remain")
	}
	var dest string
	if !previewOnly {
		ext := strings.ToLower(filepath.Ext(source))
		name := strings.TrimSuffix(filepath.Base(source), filepath.Ext(source))
		dest = pickFile(true, "Xbox executable or disc image\x00*.xbe;*.iso\x00All files\x00*.*\x00", "Save patched copy", name+"_patched"+ext)
		if dest == "" {
			return true, nil
		}
		if strings.EqualFold(filepath.Clean(source), filepath.Clean(dest)) {
			return true, fmt.Errorf("output must not overwrite the original %s", source)
		}
		if _, statErr := os.Stat(dest); statErr == nil {
			return true, fmt.Errorf("output already exists; select a new filename to preserve the existing output")
		} else if !os.IsNotExist(statErr) {
			return true, statErr
		}
	}
	app.mu.Lock()
	if app.busy {
		app.mu.Unlock()
		return true, nil
	}
	app.busy = true
	app.status = "Checking targeted file patches…"
	app.mu.Unlock()
	redraw()
	go func() {
		errorResult := performTargetedPatch(p, source, dest, choices, hasDefault, workers, override, previewOnly)
		app.mu.Lock()
		app.busy = false
		if errorResult != nil {
			app.status = "Targeted patch failed: " + errorResult.Error()
			app.lastErr = errorResult.Error()
		} else if previewOnly {
			app.status = "Targeted preview completed; no files written"
		} else {
			app.status = "SUCCESS · Patched files saved; originals unchanged"
			app.progress = 1
		}
		app.mu.Unlock()
		if errorResult == nil && !previewOnly {rememberRecent("output",dest)}
		if errorResult != nil {recordFailure("jmp.targeted",errorResult,source,direct,dest)}
		redraw()
	}()
	return true, nil
}

func readBoundedSource(p string) ([]byte, error) {
	stat, err := os.Stat(p)
	if err != nil {
		return nil, err
	}
	if stat.Size() > 512<<20 {
		return nil, fmt.Errorf("file %s exceeds 512 MiB limit", filepath.Base(p))
	}
	return os.ReadFile(p)
}

func performTargetedPatch(p *core.Patch, source, dest string, choices []targetChoice, hasDefault bool, workers int, override, previewOnly bool) error {
	isLoose := strings.EqualFold(filepath.Ext(source), ".xbe")
	var im *core.Image
	var err error
	var xbe []byte
	var xbeEntry core.ImageFile
	if isLoose {
		xbe, err = readBoundedSource(source)
	} else {
		im, err = core.OpenXISO(source)
		if err == nil {
			xbeEntry, err = im.DefaultXBE()
		}
		if err == nil {
			xbe, err = im.ReadFile(xbeEntry)
		}
	}
	if err != nil {
		return err
	}
	if !override {
		if info, e := core.InspectXBE(xbe); e == nil {
			if expected := core.TitleIDFromVersion(p.Headers["version"]); expected != "" && !strings.EqualFold(info.TitleID, expected) {
				return fmt.Errorf("XBE Title ID mismatch: %s vs expected %s", info.TitleID, expected)
			}
		}
		if err = p.VerifyCRC(xbe); err != nil {
			return err
		}
	}
	isoOutputs := make(map[string][]byte)
	diskOutputs := make(map[string][]byte)
	lines := []string{fmt.Sprintf("JMP targets several files (%s).", filepath.Base(source)), ""}
	if hasDefault {
		modified, e := p.ApplyToTarget(xbe, "default.xbe", override)
		if e != nil {
			return e
		}
		if im != nil {
			isoOutputs["default.xbe"] = modified
		} else {
			diskOutputs["default.xbe"] = modified
		}
		lines = append(lines, fmt.Sprintf("default.xbe: %d → %d bytes", len(xbe), len(modified)))
	} else if isLoose && !previewOnly {
		diskOutputs["default.xbe"] = xbe
	} // preserve a loose XBE copy for complete output
	for _, choice := range choices {
		var input []byte
		if choice.inISO {
			f, e := im.ResolveTarget(choice.name)
			if e != nil {
				return e
			}
			input, err = im.ReadFile(f)
		} else {
			input, err = readBoundedSource(choice.diskFile)
		}
		if err != nil {
			return err
		}
		modified, e := p.ApplyToTarget(input, choice.name, override)
		if e != nil {
			return fmt.Errorf("%s: %w", choice.name, e)
		}
		lines = append(lines, fmt.Sprintf("%s: %d → %d bytes", choice.name, len(input), len(modified)))
		if choice.inISO {
			f, _ := im.ResolveTarget(choice.name)
			if im.Base != 0 && (int64(len(modified))+2047)/2048 > (int64(f.Size)+2047)/2048 {
				return fmt.Errorf("%s grows beyond allocated full-disc ISO sectors; repack to a game-partition XISO", choice.name)
			}
			isoOutputs[choice.name] = modified
		} else {
			diskOutputs[choice.name] = modified
		}
	}
	lines = append(lines, "", "Chosen files are patched independently. Originals are not changed.")
	if previewOnly {
		targetedShowPreviewMessage(strings.Join(lines, "\n"), false)
		return nil
	}
	if !targetedShowPreviewMessage(strings.Join(lines, "\n"), true) {
		return fmt.Errorf("patch cancelled at confirmation")
	}
	// Stage sidecar files privately, then publish them in a newly named
	// directory. Never write a patched sidecar over its original path.
	sidecarCount := 0
	for name := range diskOutputs {
		if name != "default.xbe" {
			sidecarCount++
		}
	}
	var outDir string
	var tempDir string
	if sidecarCount > 0 {
		stem := strings.TrimSuffix(dest, filepath.Ext(dest))
		outDir = stem + "_patched_files"
		if _, e := os.Stat(outDir); e == nil {
			return fmt.Errorf("output folder already exists: %s; choose a different destination", outDir)
		} else if !os.IsNotExist(e) {
			return e
		}
		tempDir, err = os.MkdirTemp(filepath.Dir(dest), ".magicdesk-targets-*")
		if err != nil {
			return err
		}
		defer os.RemoveAll(tempDir)
		for name, data := range diskOutputs {
			if name == "default.xbe" {
				continue
			}
			norm, e := core.NormaliseTarget(name)
			if e != nil {
				return e
			}
			final := filepath.Join(tempDir, filepath.FromSlash(norm))
			if e = os.MkdirAll(filepath.Dir(final), 0755); e != nil {
				return e
			}
			if e = os.WriteFile(final, data, 0644); e != nil {
				return e
			}
		}
	}
	var tempISO string
	if im != nil {
		f, e := os.CreateTemp(filepath.Dir(dest), ".magicdesk-*.iso")
		if e != nil {
			return e
		}
		tempISO = f.Name()
		f.Close()
		os.Remove(tempISO)
		defer os.Remove(tempISO)
		// When all selected targets were external to the image, still preserve
		// the original image in the independent output copy.
		if len(isoOutputs) > 0 {
			err = core.PatchImageFiles(source, tempISO, isoOutputs, workers, func(done, total int64) {
				if total > 0 {
					app.mu.Lock()
					app.progress = float64(done) / float64(total)
					app.mu.Unlock()
				}
			})
		} else {
			err = core.CopyImage(source, tempISO, workers, nil)
		}
		if err != nil {
			return err
		}
	}
	var tempXBE string
	if isLoose {
		f, e := os.CreateTemp(filepath.Dir(dest), ".magicdesk-*.xbe")
		if e != nil {
			return e
		}
		tempXBE = f.Name()
		f.Close()
		defer os.Remove(tempXBE)
		if err = os.WriteFile(tempXBE, diskOutputs["default.xbe"], 0644); err != nil {
			return err
		}
	}
	if _, e := os.Stat(dest); e == nil {
		return fmt.Errorf("output file already exists: %s; choose a new destination", dest)
	} else if !os.IsNotExist(e) {
		return e
	}
	if sidecarCount > 0 {
		if err = os.Rename(tempDir, outDir); err != nil {
			return err
		}
		tempDir = ""
	}
	mainTemp := tempISO
	if isLoose {
		mainTemp = tempXBE
	}
	if err = os.Rename(mainTemp, dest); err != nil {
		if outDir != "" {
			os.RemoveAll(outDir)
		}
		return err
	}
	return nil
}

var targetedMessageBox = user32.NewProc("MessageBoxW")

func targetedShowPreviewMessage(body string, confirm bool) bool {
	flags := uintptr(0x40)
	caption := "Special-file patch preview"
	if confirm {
		flags = 0x24 | 0x100
		caption = "Confirm multi-file patch"
		body += "\n\nApply to the separate output copy and save all selected files?"
	}
	result := sysCall(targetedMessageBox, app.hwnd, uintptr(unsafe.Pointer(ptr(body))), uintptr(unsafe.Pointer(ptr(caption))), flags)
	return !confirm || result == 6
}
