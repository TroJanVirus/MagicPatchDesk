//go:build windows

package main

import (
	"errors"
	"fmt"
	"magicpatchdesk/core"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

const diagnosticAppVersion = "v0.8.0-test.1"

var (
	openClipboard        = user32.NewProc("OpenClipboard")
	emptyClipboard       = user32.NewProc("EmptyClipboard")
	setClipboardData     = user32.NewProc("SetClipboardData")
	closeClipboard       = user32.NewProc("CloseClipboard")
	globalAlloc          = k32.NewProc("GlobalAlloc")
	globalLock           = k32.NewProc("GlobalLock")
	globalUnlock         = k32.NewProc("GlobalUnlock")
	globalFree           = k32.NewProc("GlobalFree")
	diagnosticMessageBox = user32.NewProc("MessageBoxW")
)

func recordWarning(operation string, warningCount int, source, patch string) {
	if warningCount <= 0 {
		return
	}
	recordDiagnostic(operation+".warning", errors.New("patch operation warnings were reported"), source, patch, "", warningCount)
}

func recordFailure(operation string, cause error, source, patch, output string) {
	recordDiagnostic(operation, cause, source, patch, output, 0)
}

func recordDiagnostic(operation string, cause error, source, patch, output string, warningCount int) {
	if cause == nil {
		return
	}
	// A user pressing Cancel is not a problem report.
	low := strings.ToLower(cause.Error())
	if strings.Contains(low, "user cancelled") || strings.Contains(low, "user canceled") || strings.Contains(low, "dialog cancelled") {
		return
	}
	// Keep disk writes outside the Win32 message handler and outside app.mu.
	go func() {
		app.mu.Lock()
		root := app.root
		info := app.inputInfo
		if source == "" {
			source = app.input
		}
		if patch == "" {
			patch = app.directFile
		}
		app.mu.Unlock()
		if root == "" {
			return
		}
		report, err := core.NewFailureReport(core.FailureContext{
			AppVersion: diagnosticAppVersion, Operation: operation,
			SourcePath: source, PatchPath: patch, OutputFormat: output,
			TitleID: info.TitleID, SourceCRC32: info.CRC32, WarningCount: warningCount,
		}, cause)
		if err != nil {
			return
		}
		if _, err = core.SaveFailureReport(root, report); err != nil {
			return
		}
		app.mu.Lock()
		app.lastDiagnostic = report
		app.hasDiagnostic = true
		app.mu.Unlock()
		redraw()
	}()
}

// The existing UI uses notify() for nonfatal errors. Capture just recognisable
// errors; informational status updates and cancelled dialogs are not failures.
func reportNotifyError(s string) {
	lower := strings.ToLower(strings.TrimSpace(s))
	for _, prefix := range []string{"error", "failed", "cannot ", "could not ", "import error", "folder import error", "preview failed", "conversion failed", "patch failed", "unsupported "} {
		if strings.HasPrefix(lower, prefix) {
			recordFailure("ui.notification", errors.New(s), "", "", "")
			return
		}
	}
}

func latestDiagnostic() (core.DiagnosticReport, bool) {
	app.mu.Lock()
	r, has := app.lastDiagnostic, app.hasDiagnostic
	root := app.root
	app.mu.Unlock()
	if has {
		return r, true
	}
	reports, err := core.RecentFailureReports(root, 1)
	if err != nil || len(reports) == 0 {
		return core.DiagnosticReport{}, false
	}
	return reports[0], true
}

func diagnosticFolder() (string, error) {
	app.mu.Lock()
	root := app.root
	app.mu.Unlock()
	if root == "" {
		return "", errors.New("application data directory is unavailable")
	}
	p := filepath.Join(root, "Reports")
	if err := os.MkdirAll(p, 0700); err != nil {
		return "", err
	}
	return p, nil
}

func diagnosticsAction(action string) {
	switch action {
	case "diagnostics-folder":
		path, err := diagnosticFolder()
		if err != nil {
			notify(err.Error())
			return
		}
		sysCall(shellExecute, app.hwnd, uintptr(unsafe.Pointer(ptr("open"))), uintptr(unsafe.Pointer(ptr(path))), 0, 0, SW_SHOW)
	case "diagnostics-view":
		report, ok := latestDiagnostic()
		if !ok {
			notify("No failures have been recorded yet.")
			return
		}
		body := report.Summary() + "\n\nThe full local JSON report is in Settings > Open reports folder.\nReview it before sharing."
		title := "Diagnostic report · " + report.ID
		sysCall(diagnosticMessageBox, app.hwnd, uintptr(unsafe.Pointer(ptr(body))), uintptr(unsafe.Pointer(ptr(title))), 0x40)
	case "diagnostics-copy":
		report, ok := latestDiagnostic()
		if !ok {
			notify("No failure report to copy.")
			return
		}
		if err := copyDiagnosticText(report.Summary()); err != nil {
			notify("Clipboard unavailable: " + err.Error())
			return
		}
		notify("Safe diagnostic summary copied to clipboard. No data transmitted.")
	case "diagnostics-github":
		report, ok := latestDiagnostic()
		if !ok {
			notify("No failure report to share.")
			return
		}
		prompt := "This will open a NEW GitHub issue in your browser with a safe diagnostic summary prefilled.\n\nNothing is posted unless you submit the issue. GitHub issues may be public. Please review the text and any attachments before sharing.\n\nContinue?"
		title := "Review diagnostic sharing"
		if sysCall(diagnosticMessageBox, app.hwnd, uintptr(unsafe.Pointer(ptr(prompt))), uintptr(unsafe.Pointer(ptr(title))), 0x24|0x100) != 6 {
			return
		}
		query := url.Values{}
		query.Set("title", fmt.Sprintf("[Diagnostic] %s in %s", report.ErrorCode, report.Operation))
		query.Set("body", report.Summary()+"\n\nSteps to reproduce:\n1. \n2. \n\nExpected behaviour:\n\nActual behaviour:\n\nOptional: attach the reviewed JSON from the local Reports folder. Do not attach game binaries, source files, passwords, or private paths.")
		address := "https://github.com/TroJanVirus/MagicPatchDesk/issues/new?" + query.Encode()
		sysCall(shellExecute, app.hwnd, uintptr(unsafe.Pointer(ptr("open"))), uintptr(unsafe.Pointer(ptr(address))), 0, 0, SW_SHOW)
	}
}

func copyDiagnosticText(s string) error {
	utf := syscall.StringToUTF16(s)
	size := uintptr(len(utf) * 2)
	// GMEM_MOVEABLE and CF_UNICODETEXT, required by SetClipboardData.
	handle := sysCall(globalAlloc, 2, size)
	if handle == 0 {
		return errors.New("clipboard allocation failed")
	}
	defer func() {
		if handle != 0 {
			sysCall(globalFree, handle)
		}
	}()
	ptrHandle := sysCall(globalLock, handle)
	if ptrHandle == 0 {
		return errors.New("clipboard memory lock failed")
	}
	dst := unsafe.Slice((*uint16)(unsafe.Pointer(ptrHandle)), len(utf))
	copy(dst, utf)
	sysCall(globalUnlock, handle)
	if sysCall(openClipboard, app.hwnd) == 0 {
		return errors.New("clipboard is in use")
	}
	defer sysCall(closeClipboard)
	if sysCall(emptyClipboard) == 0 {
		return errors.New("cannot clear clipboard")
	}
	if sysCall(setClipboardData, 13, handle) == 0 {
		return errors.New("cannot set clipboard text")
	}
	handle = 0 // Clipboard now owns the global allocation.
	return nil
}
