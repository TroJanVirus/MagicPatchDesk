//go:build windows

package main

import (
	"errors"
	"fmt"
	"magicpatchdesk/core"
	"os"
	"path/filepath"
	"strings"
)

// maybeStartCCIPatch supports the safe, single-default-XBE pipeline directly
// from a Cerbios CCI. Other multi-file workflows use the CCI extractor and
// ISO patcher explicitly; no unsupported PF file operations are discarded.
func maybeStartCCIPatch(previewOnly bool, selected *core.Patch, direct, source, password string, workers int, override bool) (bool, error) {
	if !core.CCISourcePath(source) {
		return false, nil
	}
	if selected == nil && direct == "" {
		return true, errors.New("select an XBE patch before patching CCI")
	}
	if selected != nil {
		ext, e := selected.ExternalTargets()
		if e != nil {
			return true, e
		}
		if len(ext) > 0 {
			return true, errors.New("multi-file CCI patches: extract to XISO, patch the XISO, then recompress; all records must be preserved")
		}
	}
	if direct != "" {
		switch strings.ToLower(filepath.Ext(direct)) {
		case ".pf", ".mpdpack":
			return true, errors.New("PF or pack CCI targets require extracting to XISO first; use CCI Tools to avoid losing file operations")
		case ".jmp":
			p, e := resolvedJMP(selected, direct)
			if e != nil {
				return true, e
			}
			ext, e := p.ExternalTargets()
			if e != nil {
				return true, e
			}
			if len(ext) > 0 {
				return true, errors.New("multi-file JMP CCI targets require an extracted XISO for source selection")
			}
		}
	}
	output := ""
	if !previewOnly {
		suggested := strings.TrimSuffix(filepath.Base(source), filepath.Ext(source)) + "_patched.cci"
		output = pickFile(true, "Cerbios CCI\x00*.cci\x00", "Save patched CCI (source unchanged)", suggested)
		if output == "" {
			return true, nil
		}
		if !core.CCISourcePath(output) {
			return true, errors.New("patched CCI must end in .cci")
		}
		if strings.EqualFold(filepath.Clean(output), filepath.Clean(source)) {
			return true, errors.New("cannot overwrite input CCI")
		}
		if _, e := os.Stat(output); e == nil {
			return true, errors.New("patched CCI output already exists")
		}
	}
	app.mu.Lock()
	if app.busy {
		app.mu.Unlock()
		return true, errors.New("a patch operation is already running")
	}
	app.busy = true
	app.progress = 0
	app.status = "Decoding CCI image for safe patch preview…"
	app.mu.Unlock()
	redraw()
	go func() {
		scratch, e := os.MkdirTemp("", "mpd-cci-patch-*")
		if e != nil {
			finishCCI(previewOnly, e, output)
			return
		}
		defer os.RemoveAll(scratch)
		srcISO := filepath.Join(scratch, "original.iso")
		e = core.ExtractCCI(source, srcISO, nil)
		if e == nil {
			im, openErr := core.OpenXISO(srcISO)
			e = openErr
			if e == nil {
				entry, xbeErr := im.DefaultXBE()
				e = xbeErr
				if e == nil {
					original, readErr := im.ReadFile(entry)
					e = readErr
					if e == nil {
						var patched []byte
						if direct != "" {
							patched, e = core.ApplyPatchFile(original, direct, password, override)
						} else {
							patched, e = selected.Apply(original, override)
						}
						if e == nil {
							msg := fmt.Sprintf("CCI patch preview\n\nFile: %s\nOriginal XBE: %d bytes\nPatched XBE: %d bytes\n\nThe source CCI stays unchanged.", entry.Path, len(original), len(patched))
							if !targetedShowPreviewMessage(msg, !previewOnly) {
								e = errors.New("user cancelled CCI patch after preview")
							}
						}
					}
				}
			}
		}
		if e == nil && !previewOnly {
			outISO := filepath.Join(scratch, "patched.iso")
			var transform func([]byte) ([]byte, error)
			if direct != "" {
				transform = func(src []byte) ([]byte, error) { return core.ApplyPatchFile(src, direct, password, override) }
			} else {
				transform = func(src []byte) ([]byte, error) { return selected.Apply(src, override) }
			}
			e = core.PatchImageWithTransform(srcISO, outISO, transform, workers, nil)
			if e == nil {
				e = core.CompressCCI(outISO, output, func(done, total int64) {
					app.mu.Lock()
					if total > 0 {
						app.progress = float64(done) / float64(total)
					}
					app.mu.Unlock()
					redraw()
				})
			}
			if e == nil { // Verify the newly produced CCI can be decoded back into valid XDVDFS.
				checkISO := filepath.Join(scratch, "verify.iso")
				actualCCI := output
				if _, checkErr := os.Stat(actualCCI); os.IsNotExist(checkErr) {
					actualCCI = strings.TrimSuffix(output, ".cci") + ".1.cci"
				}
				e = core.ExtractCCI(actualCCI, checkISO, nil)
				if e == nil {
					_, e = core.OpenXISO(checkISO)
					if e == nil {
						var oldHash, newHash string
						oldHash, e = sourceIntegrity(outISO)
						if e == nil {
							newHash, e = sourceIntegrity(checkISO)
						}
						if e == nil && oldHash != newHash {
							e = errors.New("CCI re-encoded image failed exact verification")
						}
					}
				}
				if e != nil {
					os.Remove(output)
					stem := strings.TrimSuffix(output, ".cci")
					for n := 1; n <= 64; n++ {
						name := fmt.Sprintf("%s.%d.cci", stem, n)
						if _, x := os.Stat(name); x != nil {
							break
						}
						os.Remove(name)
					}
				}
			}
		}
		finishCCI(previewOnly, e, output)
	}()
	return true, nil
}
func finishCCI(previewOnly bool, err error, dest string) {
	app.mu.Lock()
	app.busy = false
	if err != nil {
		app.status = "CCI operation failed: " + err.Error()
	} else if previewOnly {
		app.status = "CCI preview complete; no files modified"
	} else {
		if _, check := os.Stat(dest); os.IsNotExist(check) {
			dest = strings.TrimSuffix(dest, ".cci") + ".1.cci"
		}
		app.status = "Patched CCI saved: " + dest
		app.progress = 1
	}
	app.mu.Unlock()
	if err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "user cancelled") {
			recordFailure("cci.apply", err, app.input, app.directFile, dest)
		}
	} else if !previewOnly {
		rememberRecent("output", dest)
	}
	redraw()
}
