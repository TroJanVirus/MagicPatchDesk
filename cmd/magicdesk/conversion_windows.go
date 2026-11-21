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

// In the converter, selecting a patch is an alternative to selecting a
// modified XBE. An original is optional only when the actual records allow it.
func selectedConversionInput() (name string, b []byte, err error) {
	if app.convertPatchFile != "" {
		return readConversionPatch(app.convertPatchFile)
	}
	if app.bFile != "" {
		return "", nil, nil
	}
	if app.directFile != "" {
		return readConversionPatch(app.directFile)
	}
	if app.parsed != nil {
		return app.parsed.Filename, core.SerializeJMP(app.parsed), nil
	}
	return "", nil, fmt.Errorf("choose a patch or modified XBE")
}

func readConversionPatch(path string) (string, []byte, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return "", nil, err
	}
	if fi.Size() > 512<<20 {
		return "", nil, fmt.Errorf("source patch exceeds 512 MiB limit")
	}
	b, err := os.ReadFile(path)
	return filepath.Base(path), b, err
}

func refreshConversionHint() {
	if app.bFile != "" {
		app.conversionHint = "ORIGINAL REQUIRED · Comparing stock and modified XBEs"
		return
	}
	name, b, err := selectedConversionInput()
	if err != nil {
		app.conversionHint = "Select a source patch to see whether the stock XBE is needed"
		return
	}
	plan, err := core.PlanPatchConversion(name, b, app.format, app.password)
	if err != nil {
		app.conversionHint = "Cannot inspect source patch: " + err.Error()
		return
	}
	if plan.RequiresSource {
		app.conversionHint = "ORIGINAL REQUIRED · " + plan.Reason
	} else {
		if plan.SourceRecommended {
			app.conversionHint = "ORIGINAL OPTIONAL (recommended for verification) · " + plan.Reason
		} else {
			app.conversionHint = "NO ORIGINAL NEEDED · " + plan.Reason
		}
	}
}

func confirmWithoutOriginal(plan core.ConversionPlan) bool {
	if !plan.SourceRecommended {
		return true
	}
	message := "This patch can be converted without the original XBE.\n\n" + plan.Reason + "\n\nCRC32/Title ID or target-byte verification cannot be performed without the original file. Continue?"
	proc := user32.NewProc("MessageBoxW")
	return sysCall(proc, app.hwnd, uintptr(unsafe.Pointer(ptr(message))), uintptr(unsafe.Pointer(ptr("Unverified conversion"))), 0x04|0x30) == 6
}

func runConversion() {
	if app.busy {
		return
	}
	if app.bFile == "" && app.convertPatchFile == "" && app.directFile == "" && app.parsed == nil {
		notify("Select a source patch or a modified XBE first.")
		return
	}
	outExt := []string{".jmp", ".ips", ".xdelta", ".eips", ".bps"}
	format := app.format
	if format < 0 || format >= len(outExt) {
		notify("Unknown output format")
		return
	}
	var inputName string
	var patch []byte
	var plan core.ConversionPlan
	var err error
	if app.bFile != "" {
		if app.aFile == "" {
			notify("Stock XBE required when converting a modified XBE.")
			return
		}
	} else {
		inputName, patch, err = selectedConversionInput()
		if err != nil {
			notify("Cannot read patch: " + err.Error())
			return
		}
		plan, err = core.PlanPatchConversion(inputName, patch, format, app.password)
		if err != nil {
			notify("Cannot analyze patch: " + err.Error())
			return
		}
		if plan.RequiresSource && app.aFile == "" {
			notify("Original XBE required: " + plan.Reason)
			// Also show a native dialog: truncated status-bar messages cannot
			// explain the reason to the user.
			proc := user32.NewProc("MessageBoxW")
			sysCall(proc, app.hwnd, uintptr(unsafe.Pointer(ptr(plan.Reason))), uintptr(unsafe.Pointer(ptr("Original XBE needed"))), 0x30)
			return
		}
		if app.aFile == "" && !confirmWithoutOriginal(plan) {
			return
		}
	}
	ext := outExt[format]
	output := pickFile(true, "Patch files\x00*.jmp;*.ips;*.eips;*.xdelta;*.bps\x00All files\x00*.*\x00", "Save converted patch", "ConvertedPatch"+ext)
	if output == "" {
		return
	}
	if filepath.Ext(output) == "" {
		output += ext
	}
	if !strings.EqualFold(filepath.Ext(output), ext) {
		notify("Output extension must be " + ext)
		return
	}
	for _, src := range []string{app.aFile, app.bFile, app.convertPatchFile, app.directFile} {
		if src != "" && strings.EqualFold(filepath.Clean(output), filepath.Clean(src)) {
			notify("Output may not overwrite an input file.")
			return
		}
	}
	sourcePath, modPath, password, useRLE := app.aFile, app.bFile, app.password, app.ipsRLE
	app.mu.Lock()
	app.busy = true
	app.status = "Converting patch…"
	app.mu.Unlock()
	redraw()
	go func() {
		var original []byte
		var converted []byte
		var jobErr error
		title := "Unknown"
		if sourcePath != "" {
			original, jobErr = os.ReadFile(sourcePath)
			title = strings.TrimSuffix(filepath.Base(sourcePath), filepath.Ext(sourcePath))
		}
		if jobErr == nil {
			if modPath != "" {
				modified, e := os.ReadFile(modPath)
				if e != nil {
					jobErr = e
				} else {
					converted, jobErr = core.BuildPatchWithOptions(original, modified, format, password, title, useRLE)
				}
			} else {
				converted, _, jobErr = core.ConvertPatchBytes(inputName, patch, original, format, password, useRLE, title)
			}
		}
		if jobErr == nil {
			jobErr = atomicWrite(output, converted)
		}
		app.mu.Lock()
		app.busy = false
		if jobErr != nil {
			app.status = "Conversion failed: " + jobErr.Error()
		} else {
			app.status = "Converted patch saved: " + output
		}
		app.mu.Unlock()
		redraw()
	}()
}
