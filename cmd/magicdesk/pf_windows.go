//go:build windows

package main

import (
	"context"
	"fmt"
	"magicpatchdesk/core"
	"os"
	"path/filepath"
	"strings"
	"unsafe"
)

// PF is an Xbox GAME-DIRECTORY patch script, not a patch that can safely be
// applied to default.xbe alone. All commands run in Go against staged copies.
func maybeStartPF(previewOnly bool, direct, source string, workers int) (bool, error) {
	if !strings.EqualFold(filepath.Ext(direct), ".pf") {
		return false, nil
	}
	if source == "" {
		return true, fmt.Errorf("choose the game's default.xbe or an Xbox ISO before using this PF")
	}
	data, err := readBoundedSource(direct)
	if err != nil {
		return true, err
	}
	pf, err := core.ParsePF(filepath.Base(direct), data)
	if err != nil {
		return true, err
	}
	isISO := strings.EqualFold(filepath.Ext(source), ".iso")
	if isISO && pf.HasFilesystemOps() {
		return true, fmt.Errorf("this PF copies/moves/renames/removes files: select an extracted game's default.xbe instead; ISO directory rebuilding is not supported")
	}
	if !isISO && !strings.EqualFold(filepath.Ext(source), ".xbe") {
		return true, fmt.Errorf("select default.xbe in the extracted game folder, or an Xbox ISO")
	}
	var dest string
	if !previewOnly {
		if isISO {
			base := strings.TrimSuffix(filepath.Base(source), filepath.Ext(source))
			dest = pickFile(true, "Xbox disc image\x00*.iso\x00", "Save independently patched PF ISO", base+"_PF_patched.iso")
			if dest == "" {
				return true, nil
			}
			if !strings.EqualFold(filepath.Ext(dest), ".iso") {
				return true, fmt.Errorf("PF ISO output must be .iso")
			}
		} else {
			gameDir := filepath.Dir(source)
			dest = filepath.Join(filepath.Dir(gameDir), filepath.Base(gameDir)+"_PF_patched")
			if _, e := os.Stat(dest); e == nil {
				return true, fmt.Errorf("output folder already exists: %s", dest)
			}
		}
		if strings.EqualFold(filepath.Clean(dest), filepath.Clean(source)) {
			return true, fmt.Errorf("PF output cannot overwrite source")
		}
		prompt := "PF: " + pf.Description() + "\n\nAuthor: " + pf.Header["author"] + "\nOperations: " + fmt.Sprint(len(pf.Operations)) + "\n\n"
		if isISO {
			prompt += "Save an independently patched XISO copy?"
		} else {
			prompt += "Copy the ENTIRE original game folder to:\n" + dest + "\n\nThe original game remains unchanged. This can require several GB of disk space. Continue?"
		}
		if sysCall(targetedMessageBox, app.hwnd, uintptr(unsafe.Pointer(ptr(prompt))), uintptr(unsafe.Pointer(ptr("XBMC4Gamers PF — confirm"))), 0x24|0x100) != 6 {
			return true, nil
		}
	}
	app.mu.Lock()
	if app.busy {
		app.mu.Unlock()
		return true, fmt.Errorf("another operation is in progress")
	}
	app.busy = true
	app.progress = 0
	app.status = "Checking Rocky5 PF patch operations…"
	app.mu.Unlock()
	redraw()
	go func() {
		var result core.PFOutcome
		var e error
		if isISO {
			if previewOnly {
				result, e = pf.PreviewPFISO(source)
			} else {
				result, e = pf.ApplyPFToISO(source, dest, workers, func(done, total int64) {
					if total > 0 {
						app.mu.Lock()
						app.progress = float64(done) / float64(total)
						app.mu.Unlock()
						redraw()
					}
				})
			}
		} else {
			root := filepath.Dir(source)
			if previewOnly {
				result, e = pf.PreviewDirectory(root)
			} else {
				result, e = pf.ApplyPFDirectory(context.Background(), root, dest, func(done, total int64) {
					if total > 0 {
						app.mu.Lock()
						app.progress = float64(done) / float64(total)
						app.mu.Unlock()
						redraw()
					}
				})
			}
		}
		if e == nil && len(result.Warnings)>0 {recordWarning("pf.apply",len(result.Warnings),source,direct)}
		if e == nil {
			message := fmt.Sprintf("PF: %s\n\nActions: %d\nModified: %s\nCreated: %s\nRemoved: %s", pf.Description(), result.Actions, strings.Join(result.Modified, ", "), strings.Join(result.Created, ", "), strings.Join(result.Removed, ", "))
			if len(result.Warnings) > 0 {
				message += "\n\nWarnings (review):\n" + strings.Join(result.Warnings, "\n")
			}
			if !previewOnly {
				message += "\n\nSaved in: " + dest
			} else {
				message += "\n\nPreview only. No files written."
			}
			targetedShowPreviewMessage(message, false)
		}
		app.mu.Lock()
		app.busy = false
		if e != nil {
			app.status = "PF error: " + e.Error()
			app.lastErr = e.Error()
		} else if previewOnly {
			app.status = "PF preview complete; no files modified"
		} else {
			app.status = "PF game copy saved: " + dest
			app.progress = 1
		}
		app.mu.Unlock()
		if e == nil && !previewOnly {rememberRecent("output",dest)}
		if e != nil {recordFailure("pf.apply",e,source,direct,dest)}
		redraw()
	}()
	return true, nil
}
