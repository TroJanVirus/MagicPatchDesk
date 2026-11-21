//go:build windows

package main

import (
	"fmt"
	"magicpatchdesk/core"
	"path/filepath"
	"strings"
)

// A multi-file JMP can remain a SINGLE JMP. When converting to a binary
// patch format, create a .mpdpack instead of misrepresenting one patch as
// capable of modifying multiple files.
func refreshPackHint(name string, patch []byte) bool {
	if !strings.EqualFold(filepath.Ext(name), ".jmp") {
		app.packMode = false
		return false
	}
	p, err := core.ParseJMP(name, patch)
	if err != nil {
		app.packMode = false
		return false
	}
	special, err := p.ExternalTargets()
	if err != nil {
		app.packMode = false
		app.conversionBlocked = true
		app.conversionHint = "Ambiguous filename: " + err.Error()
		return true
	}
	if len(special) == 0 || app.format == 0 {
		app.packMode = false
		return false
	}
	app.packMode = true
	plan, err := core.PlanJMPPack(name, patch, app.format)
	if err != nil {
		app.conversionBlocked = true
		app.conversionHint = "Cannot create pack: " + err.Error()
		return true
	}
	app.packPlan = plan
	sources := currentPackPaths(plan)
	missing := []string{}
	for _, t := range plan.Targets {
		if t.RequiresSource && sources[t.Target] == "" {
			missing = append(missing, t.Target)
		}
	}
	if app.format == 3 && len([]rune(app.password)) < 8 {
		app.conversionBlocked = true
		app.conversionHint = "Encrypted IPS requires a password of at least eight characters"
		return true
	}
	if len(missing) > 0 {
		app.conversionBlocked = true
		app.conversionHint = "Source file required for: " + strings.Join(missing, ", ") + " · Click Locate target originals"
	} else {
		app.conversionBlocked = false
		app.conversionHint = fmt.Sprintf("Patch pack: %d targets → one .mpdpack archive (each file patched separately)", len(plan.Targets))
	}
	return true
}

func currentPackPaths(plan core.PatchPackPlan) map[string]string {
	paths := map[string]string{}
	for _, t := range plan.Targets {
		if app.packSources != nil && app.packSources[t.Target] != "" {
			paths[t.Target] = app.packSources[t.Target]
			continue
		}
		if app.aFile != "" {
			if t.Target == "default.xbe" || strings.EqualFold(filepath.Base(app.aFile), filepath.Base(filepath.FromSlash(t.Target))) {
				paths[t.Target] = app.aFile
			}
		}
	}
	return paths
}

// All required sources must be explicitly selected. Non-required sources may
// also be supplied to get exact round-trip / CRC validation for each file.
func choosePackSources() {
	if !app.packMode {
		notify("Choose a multi-file JMP and IPS, BPS, Xdelta or EIPS output first.")
		return
	}
	if app.packSources == nil {
		app.packSources = map[string]string{}
	}
	for _, t := range app.packPlan.Targets {
		existing := currentPackPaths(app.packPlan)[t.Target]
		if existing != "" {
			continue
		}
		hint := "Locate original " + t.Target
		if !t.RequiresSource {
			hint = "Optional verification source for " + t.Target
		}
		chosen := pickFile(false, "All files\x00*.*\x00", hint, "")
		if chosen == "" {
			if t.RequiresSource {
				notify("Cannot convert: missing original " + t.Target)
				return
			}
			continue
		}
		if !strings.EqualFold(filepath.Base(chosen), filepath.Base(filepath.FromSlash(t.Target))) {
			notify("Wrong file: expected " + filepath.Base(filepath.FromSlash(t.Target)))
			return
		}
		app.packSources[t.Target] = chosen
	}
	refreshConversionHint()
	redraw()
}

func runPackConversion() {
	if app.busy {
		return
	}
	name, patch, err := selectedConversionInput()
	if err != nil {
		notify("Cannot read JMP: " + err.Error())
		return
	}
	plan, err := core.PlanJMPPack(name, patch, app.format)
	if err != nil {
		notify(err.Error())
		return
	}
	paths := currentPackPaths(plan)
	for _, t := range plan.Targets {
		if t.RequiresSource && paths[t.Target] == "" {
			notify("Conversion needs original file: " + t.Target)
			return
		}
	}
	if app.format == 3 && len([]rune(app.password)) < 8 {
		notify("Encrypted IPS password too short")
		return
	}
	output := pickFile(true, "Magic Patch Desk packs\x00*.mpdpack\x00All files\x00*.*\x00", "Save multi-file patch pack", strings.TrimSuffix(name, filepath.Ext(name))+core.PatchPackExtension)
	if output == "" {
		return
	}
	if filepath.Ext(output) == "" {
		output += core.PatchPackExtension
	}
	if !strings.EqualFold(filepath.Ext(output), core.PatchPackExtension) {
		notify("Pack output must use .mpdpack")
		return
	}
	for _, in := range paths {
		if in != "" && strings.EqualFold(filepath.Clean(in), filepath.Clean(output)) {
			notify("Output cannot replace an original file")
			return
		}
	}
	if app.convertPatchFile != "" && strings.EqualFold(filepath.Clean(app.convertPatchFile), filepath.Clean(output)) {
		notify("Output cannot replace input JMP")
		return
	}
	fmtN, password, rle := app.format, app.password, app.ipsRLE
	app.mu.Lock()
	app.busy = true
	app.status = "Building multi-file patch pack…"
	app.mu.Unlock()
	redraw()
	go func() {
		originalFiles := map[string][]byte{}
		var jobErr error
		for t, p := range paths {
			var b []byte
			b, jobErr = readBoundedSource(p)
			if jobErr != nil {
				jobErr = fmt.Errorf("%s: %w", t, jobErr)
				break
			}
			originalFiles[t] = b
		}
		if jobErr == nil {
			var packed []byte
			packed, _, jobErr = core.BuildJMPPatchPack(name, patch, originalFiles, fmtN, password, rle)
			if jobErr == nil {
				jobErr = atomicWrite(output, packed)
			}
		}
		app.mu.Lock()
		app.busy = false
		if jobErr != nil {
			app.status = "Pack conversion failed: " + jobErr.Error()
		} else {
			app.status = "Saved patch pack: " + output
		}
		app.mu.Unlock()
		redraw()
	}()
}
