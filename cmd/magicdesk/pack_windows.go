//go:build windows

package main

import (
	"bytes"
	"fmt"
	"magicpatchdesk/core"
	"os"
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
	if (len(special) == 0 && app.format != core.AllPatchFormats) || app.format == 0 {
		app.packMode = false
		return false
	}
	app.packMode = true
	var plan core.PatchPackPlan
	if app.format == core.AllPatchFormats {
		plan, err = core.PlanJMPAllFormatsPack(name, patch)
	} else {
		plan, err = core.PlanJMPPack(name, patch, app.format)
	}
	if err != nil {
		app.conversionBlocked = true
		app.conversionHint = "Cannot create pack: " + err.Error()
		return true
	}
	app.packPlan = plan
	sources := currentPackPaths(plan)
	missing := []string{}
	for _, t := range plan.Targets {
        if t.RequiresSource&&sources[t.Target]==""&&!packFromISO(t.Target) {
            why:=t.Reason
            if why==""{why="original file required"}
            missing=append(missing,t.Target+": "+why)
        }
	}
	if app.format == 3 && len([]rune(app.password)) < 8 {
		app.conversionBlocked = true
		app.conversionHint = "Encrypted IPS requires a password of at least eight characters"
		return true
	}
	if len(missing) > 0 {
		app.conversionBlocked = true
        app.conversionHint="Source-free conversion blocked. "+strings.Join(missing,"; ")+". Use Locate target originals to continue."
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
			if strings.EqualFold(filepath.Ext(app.aFile), ".iso") {
				continue
			}
			if t.Target == "default.xbe" || strings.EqualFold(filepath.Base(app.aFile), filepath.Base(filepath.FromSlash(t.Target))) {
				paths[t.Target] = app.aFile
			} else {
				// If the user supplied the game's default.xbe, nearby game
				// files can be found relative to its containing directory.
				candidate := filepath.Join(filepath.Dir(app.aFile), filepath.FromSlash(t.Target))
				if info, e := os.Stat(candidate); e == nil && !info.IsDir() {
					paths[t.Target] = candidate
				}
			}
		}
	}
	return paths
}

// All required sources must be explicitly selected. Non-required sources may
// also be supplied to get exact round-trip / CRC validation for each file.
func choosePackSources() {
	if !app.packMode {
		notify("Choose a JMP with an output patch format or All Formats Pack.")
		return
	}
	if app.packSources == nil {
		app.packSources = map[string]string{}
	}
	for _, t := range app.packPlan.Targets {
		existing := currentPackPaths(app.packPlan)[t.Target]
		if existing != "" || packFromISO(t.Target) {
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
	var plan core.PatchPackPlan
	if app.format == core.AllPatchFormats {
		plan, err = core.PlanJMPAllFormatsPack(name, patch)
	} else {
		plan, err = core.PlanJMPPack(name, patch, app.format)
	}
	if err != nil {
		notify(err.Error())
		return
	}
	paths := currentPackPaths(plan)
	for _, t := range plan.Targets {
		if t.RequiresSource && paths[t.Target] == "" && !packFromISO(t.Target) {
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
	sourceISO := app.aFile
    expectedXBEPath:=app.bFile
	app.mu.Lock()
	app.busy = true
	app.status = "Building multi-file patch pack…"
	app.mu.Unlock()
	redraw()
	go func() {
        defer catchFatalPanic()
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
		if jobErr == nil && strings.EqualFold(filepath.Ext(sourceISO), ".iso") {
			var im *core.Image
			im, jobErr = core.OpenXISO(sourceISO)
			if jobErr == nil {
				for _, t := range plan.Targets {
					if originalFiles[t.Target] != nil {
						continue
					}
					var f core.ImageFile
					f, jobErr = im.ResolveTarget(t.Target)
					if jobErr != nil {
						break
					}
					originalFiles[t.Target], jobErr = im.ReadFile(f)
					if jobErr != nil {
						break
					}
				}
			}
		}
		if jobErr == nil {
			var packed []byte
			if fmtN == core.AllPatchFormats {
				packed, _, jobErr = core.BuildJMPAllFormatsPack(name, patch, originalFiles, password, rle)
			} else {
				packed, _, jobErr = core.BuildJMPPatchPack(name, patch, originalFiles, fmtN, password, rle)
			}
            if jobErr==nil&&expectedXBEPath!="" {
                // An expected modified XBE also verifies an all-formats pack:
                // decode/apply the generated archive and compare default.xbe
                // byte-for-byte, including all source/target hash checks.
                if originalFiles["default.xbe"]==nil{
                    jobErr=fmt.Errorf("modified XBE verification requires original default.xbe in the selected sources")
                }else{
                    var expected []byte
                    expected,jobErr=os.ReadFile(expectedXBEPath)
                    if jobErr==nil{
                        var checked map[string][]byte
                        checked,jobErr=core.ApplyPatchPack(packed,originalFiles,password)
                        if jobErr==nil {
                            got:=checked["default.xbe"]
                            if got==nil||!bytes.Equal(got,expected){
                                jobErr=fmt.Errorf("converted patch pack does not reproduce the modified XBE byte-for-byte")
                            }
                        }
                    }
                }
            }
            if jobErr==nil{jobErr=atomicWrite(output,packed)}
		}
		app.mu.Lock()
		app.busy = false
		if jobErr != nil {
			app.status = "Pack conversion failed: " + jobErr.Error()
		} else {
			app.status = "Saved patch pack: " + output
		}
		app.mu.Unlock()
		if jobErr == nil {rememberRecent("output",output)}
		if jobErr != nil {recordFailure("pack.create",jobErr,sourceISO,name,output)}
		redraw()
	}()
}

// Resolve game files from the chosen XISO's XDVDFS table, not by feeding
// a multi-GB ISO to a patch generator as if it were default.xbe.
func packFromISO(target string) bool {
	if app.aFile == "" || !strings.EqualFold(filepath.Ext(app.aFile), ".iso") {
		return false
	}
	im, e := core.OpenXISO(app.aFile)
	if e != nil {
		return false
	}
	_, e = im.ResolveTarget(target)
	return e == nil
}
