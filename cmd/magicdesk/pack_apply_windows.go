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

// A .mpdpack is a group of independently verified single-file patches.
// No member is silently omitted. The user must provide each source and must
// explicitly choose ISO members or external originals for named targets.
func maybeStartPack(previewOnly bool, direct, source string, workers int, password string) (bool, error) {
	if !strings.EqualFold(filepath.Ext(direct), core.PatchPackExtension) {
		return false, nil
	}
	raw, err := readBoundedSource(direct)
	if err != nil {
		return true, err
	}
	manifest, _, err := core.ReadPatchPack(raw)
	if err != nil {
		return true, err
	}
	isLoose := strings.EqualFold(filepath.Ext(source), ".xbe")
	var im *core.Image
	if !isLoose {
		im, err = core.OpenXISO(source)
		if err != nil {
			return true, err
		}
	}
	choices := []targetChoice{}
    already:=map[string]bool{}
	for _, entry := range manifest.Entries {
        if already[entry.Target] {continue}
        already[entry.Target]=true
		if entry.Target == "default.xbe" {
			choices = append(choices, targetChoice{name: entry.Target, inISO: im != nil, diskFile: source})
			continue
		}
		if isLoose && strings.EqualFold(filepath.Base(source), filepath.Base(filepath.FromSlash(entry.Target))) {
			choices = append(choices, targetChoice{name: entry.Target, diskFile: source})
			continue
		}
		found := ""
		if im != nil {
			if f, e := im.ResolveTarget(entry.Target); e == nil {
				found = f.Path
			}
		}
		if found != "" {
			prompt := fmt.Sprintf("Patch pack targets %q.\n\nFound in ISO: %s\n\nYES: Patch inside the image\nNO: Locate an external original\nCANCEL: Abort the entire pack", entry.Target, found)
			result := sysCall(targetedMessageBox, app.hwnd, uintptr(unsafe.Pointer(ptr(prompt))), uintptr(unsafe.Pointer(ptr("Patch pack target"))), 0x03|0x20|0x100)
			if result == 2 {
				return true, fmt.Errorf("cancelled")
			}
			if result == 6 {
				choices = append(choices, targetChoice{name: entry.Target, inISO: true})
				continue
			}
		} else {
			prompt := fmt.Sprintf("This patch pack contains a patch for %q.\n\nLocate the matching ORIGINAL game file?\nCancel will stop the complete pack; files will not be partially patched.", entry.Target)
			if sysCall(targetedMessageBox, app.hwnd, uintptr(unsafe.Pointer(ptr(prompt))), uintptr(unsafe.Pointer(ptr("Original file required"))), 0x31) != 1 {
				return true, fmt.Errorf("pack cancelled")
			}
		}
		picked := pickFile(false, "All files\x00*.*\x00", "Locate original "+entry.Target, "")
		if picked == "" {
			return true, fmt.Errorf("original %s not provided", entry.Target)
		}
		if !strings.EqualFold(filepath.Base(picked), filepath.Base(filepath.FromSlash(entry.Target))) {
			return true, fmt.Errorf("expected file named %s; got %s", entry.Target, filepath.Base(picked))
		}
		choices = append(choices, targetChoice{name: entry.Target, diskFile: picked})
	}
	var dest string
	if !previewOnly {
		ext := filepath.Ext(source)
		dest = pickFile(true, "Xbox executable or disc image\x00*.xbe;*.iso\x00All files\x00*.*\x00", "Save patched output", strings.TrimSuffix(filepath.Base(source), ext)+"_patched"+ext)
		if dest == "" {
			return true, nil
		}
		for _, v := range append([]targetChoice{{name: "source", diskFile: source}}, choices...) {
			if v.diskFile != "" && strings.EqualFold(filepath.Clean(dest), filepath.Clean(v.diskFile)) {
				return true, fmt.Errorf("output may not overwrite an original file")
			}
		}
		if _, e := os.Stat(dest); e == nil {
			return true, fmt.Errorf("output already exists; choose a new filename")
		} else if !os.IsNotExist(e) {
			return true, e
		}
	}
	app.mu.Lock()
	if app.busy {
		app.mu.Unlock()
		return true, fmt.Errorf("another operation is running")
	}
	app.busy = true
	app.status = "Verifying packed patches…"
	app.mu.Unlock()
	redraw()
	go func() {
		err := executePack(raw, source, dest, choices, workers, password, previewOnly)
		app.mu.Lock()
		app.busy = false
		if err != nil {
			app.status = "Patch pack failed: " + err.Error()
		} else if previewOnly {
			app.status = "Patch pack preview completed; no file written"
		} else {
			app.status = "Patch pack completed; originals unchanged"
			app.progress = 1
		}
		app.mu.Unlock()
		redraw()
	}()
	return true, nil
}

func executePack(raw []byte, source, dest string, choices []targetChoice, workers int, password string, previewOnly bool) error {
	imLoose := strings.EqualFold(filepath.Ext(source), ".xbe")
	var im *core.Image
	var err error
	if !imLoose {
		im, err = core.OpenXISO(source)
		if err != nil {
			return err
		}
	}
	originals := map[string][]byte{}
	for _, ch := range choices {
		var data []byte
		if ch.inISO {
			f, e := im.ResolveTarget(ch.name)
			if e != nil {
				return e
			}
			data, err = im.ReadFile(f)
		} else {
			data, err = readBoundedSource(ch.diskFile)
		}
		if err != nil {
			return err
		}
		originals[ch.name] = data
	}
	outputs, err := core.ApplyPatchPack(raw, originals, password)
	if err != nil {
		return err
	}
	description := []string{"All members will be patched together:", ""}
	for _, ch := range choices {
		description = append(description, fmt.Sprintf("%s: %d → %d bytes", ch.name, len(originals[ch.name]), len(outputs[ch.name])))
	}
	description = append(description, "", "Originals remain unchanged; any external files go into a separate _patched_files folder.")
	if previewOnly {
		targetedShowPreviewMessage(strings.Join(description, "\n"), false)
		return nil
	}
	if !targetedShowPreviewMessage(strings.Join(description, "\n"), true) {
		return fmt.Errorf("user cancelled patch pack")
	}
	isoOutputs := map[string][]byte{}
	diskOutputs := map[string][]byte{}
	var primary []byte
	for _, ch := range choices {
		if ch.inISO {
			isoOutputs[ch.name] = outputs[ch.name]
		} else {
			if imLoose && filepath.Clean(ch.diskFile) == filepath.Clean(source) {
				primary = outputs[ch.name]
			} else {
				diskOutputs[ch.name] = outputs[ch.name]
			}
		}
	}
	if imLoose && primary == nil {
		primary, err = readBoundedSource(source)
		if err != nil {
			return err
		}
	}
	outDir := ""
	tempDir := ""
	if len(diskOutputs) > 0 {
		outDir = strings.TrimSuffix(dest, filepath.Ext(dest)) + "_patched_files"
		if _, e := os.Stat(outDir); e == nil {
			return fmt.Errorf("output sidecar folder already exists")
		} else if !os.IsNotExist(e) {
			return e
		}
		tempDir, err = os.MkdirTemp(filepath.Dir(dest), ".magicdesk-packed-*")
		if err != nil {
			return err
		}
		defer os.RemoveAll(tempDir)
		for name, value := range diskOutputs {
			norm, e := core.NormaliseTarget(name)
			if e != nil {
				return e
			}
			p := filepath.Join(tempDir, filepath.FromSlash(norm))
			if e = os.MkdirAll(filepath.Dir(p), 0755); e != nil {
				return e
			}
			if e = os.WriteFile(p, value, 0644); e != nil {
				return e
			}
		}
	}
	var mainTemp string
	if imLoose {
		f, e := os.CreateTemp(filepath.Dir(dest), ".magicdesk-packed-*.xbe")
		if e != nil {
			return e
		}
		mainTemp = f.Name()
		f.Close()
		defer os.Remove(mainTemp)
		if e = os.WriteFile(mainTemp, primary, 0644); e != nil {
			return e
		}
	} else {
		f, e := os.CreateTemp(filepath.Dir(dest), ".magicdesk-packed-*.iso")
		if e != nil {
			return e
		}
		mainTemp = f.Name()
		f.Close()
		os.Remove(mainTemp)
		defer os.Remove(mainTemp)
		if len(isoOutputs) > 0 {
			err = core.PatchImageFiles(source, mainTemp, isoOutputs, workers, nil)
		} else {
			err = core.CopyImage(source, mainTemp, workers, nil)
		}
		if err != nil {
			return err
		}
	}
	if _, e := os.Stat(dest); e == nil {
		return fmt.Errorf("output file exists")
	} else if !os.IsNotExist(e) {
		return e
	}
	if outDir != "" {
		if e := os.Rename(tempDir, outDir); e != nil {
			return e
		}
		tempDir = ""
	}
	if e := os.Rename(mainTemp, dest); e != nil {
		if outDir != "" {
			os.RemoveAll(outDir)
		}
		return e
	}
	return nil
}
