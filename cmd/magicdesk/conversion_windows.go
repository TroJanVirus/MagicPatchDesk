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

// The converter accepts a loose XBE or an ISO, but its patch engines must
// receive the actual default.xbe bytes, never raw 2 KiB disc sectors.
func readConverterOriginal(path string)([]byte,error){
    if strings.EqualFold(filepath.Ext(path),".iso"){
        im,err:=core.OpenXISO(path)
        if err!=nil{return nil,err}
        entry,err:=im.DefaultXBE()
        if err!=nil{return nil,err}
        return im.ReadFile(entry)
    }
    if !strings.EqualFold(filepath.Ext(path),".xbe"){
        return nil,fmt.Errorf("choose an original XBE or supported Xbox ISO")
    }
    return os.ReadFile(path)
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
	app.conversionBlocked = true
	if app.bFile != "" && app.convertPatchFile == "" {
        if app.format == core.AllPatchFormats {app.conversionHint="All Formats Pack requires a JMP or PF script; choose a source patch";return}
		if app.aFile == "" {
			app.conversionHint = "This patch conversion requires the source file for the context to create it · Choose stock/original XBE"
		} else {
			app.conversionHint = "Source ready · Comparing stock and modified XBEs"
			app.conversionBlocked = false
		}
		return
	}
	name, b, err := selectedConversionInput()
	if err != nil {
		app.conversionHint = "Choose a source patch or modified XBE"
		return
	}
    if app.bFile!="" && app.convertPatchFile!="" && app.aFile=="" {
        app.conversionHint="Original XBE required to compare source patch output with pre-patched XBE"
        return
    }
	ext := strings.ToLower(filepath.Ext(name))
	// Encrypted IPS costs 210k PBKDF2 rounds. Never derive keys on each
	// keystroke or on every Win32 UI repaint.
	if ext == ".eips" {
		if app.password == "" {
			app.conversionHint = "Enter the encrypted IPS password before converting"
			return
		}
		if app.format == 2 || app.format == 4 || app.format == 0 {
			if app.aFile == "" {
				app.conversionHint = "This patch conversion requires the source file for the context to create it"
			} else {
				app.conversionHint = "Source selected; encrypted IPS structure will be verified on Convert"
				app.conversionBlocked = false
			}
			return
		}
		app.conversionHint = "IPS and encrypted IPS can be re-encoded without the original"
		app.conversionBlocked = app.format == 3 && len([]rune(app.password)) < 8
		if app.conversionBlocked {
			app.conversionHint = "Encryption password must be at least 8 characters"
		}
		return
	}

	if refreshPFConversionHint(name,b) {return}
	if refreshPackHint(name, b) {
		return
	}
	plan, err := core.PlanPatchConversion(name, b, app.format, app.password)
	if err != nil {
		app.conversionHint = "Cannot convert this patch: " + err.Error()
		return
	}
	if plan.RequiresSource && app.aFile == "" {
		app.conversionHint = "This patch conversion requires the source file for the context to create it"
		return
	}
	if app.format == 3 && len([]rune(app.password)) < 8 {
		app.conversionHint = "Encryption password must be at least 8 characters"
		return
	}
	app.conversionBlocked = false
	if plan.RequiresSource {
		app.conversionHint = "Original source selected · " + plan.Reason
	} else if plan.SourceRecommended {
		app.conversionHint = "Original optional — recommended for verification · " + plan.Reason
	} else {
		app.conversionHint = "No original needed · " + plan.Reason
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
	refreshConversionHint()
	if app.conversionBlocked {
		notify(app.conversionHint)
		return
	}
	if !app.packMode {
        name,raw,e:=selectedConversionInput()
        if e==nil && strings.EqualFold(filepath.Ext(name),".pf") {runPFConversion(name,raw);return}
    }
	if app.packMode {
		runPackConversion()
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
	if app.bFile != "" && app.convertPatchFile == "" {
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
    // Capture everything the conversion worker uses. The user may change
    // the selected patch before a long conversion finishes.
    patch = append([]byte(nil),patch...)
	app.mu.Lock()
	app.busy = true
	app.status = "Converting patch…"
	app.mu.Unlock()
	redraw()
    go func() {
        defer catchFatalPanic()
        var original []byte
        var converted []byte
        var modifiedReference []byte
        var jobErr error
        title:="Unknown"
        if sourcePath!="" {
            original,jobErr=readConverterOriginal(sourcePath)
            title=strings.TrimSuffix(filepath.Base(sourcePath),filepath.Ext(sourcePath))
        }
        if jobErr==nil && modPath!="" {
            modifiedReference,jobErr=os.ReadFile(modPath)
        }
        if jobErr==nil {
            if modPath!=""&&inputName=="" {
                // Create a new patch from two XBEs. The encoded patch still
                // has to round-trip back to the exact expected modified XBE.
                converted,jobErr=core.BuildPatchWithOptions(original,modifiedReference,format,password,title,useRLE)
            }else{
                if modPath!="" {
                    // First check the existing patch really makes the user's
                    // expected modified XBE, not just overlapping edit ranges.
                    jobErr=core.VerifyPatchAgainstModified(original,modifiedReference,patch,inputName,password)
                }
                if jobErr==nil {
                    converted,_,jobErr=core.ConvertPatchBytes(inputName,patch,original,format,password,useRLE,title)
                }
            }
        }
        if jobErr==nil&&modPath!="" {
            // Then independently apply the converted output in memory.
            // Unlike the old two-XBE path, it must reproduce the reference
            // byte-for-byte BEFORE a new patch file is written.
            jobErr=core.VerifyPatchAgainstModified(original,modifiedReference,converted,"ConvertedPatch"+ext,password)
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
		if jobErr == nil {rememberRecent("output",output)}
		if jobErr != nil {recordFailure("patch.convert",jobErr,sourcePath,inputName,output)}
		redraw()
	}()
}
