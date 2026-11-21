//go:build windows

package main

import (
	"errors"
	"magicpatchdesk/core"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

const diagnosticAppVersion = core.AppVersion

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
		refreshDiagnosticGroups()
		refreshDeveloperDashboardLocal()
		if maintenancePref.AutoReports {go syncStoredReports()}
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

// refreshDiagnosticGroups rebuilds a local index; it never reads GitHub or
// sends telemetry. Called at startup and after a new report is saved.
func refreshDiagnosticGroups() {
	app.mu.Lock()
	root := app.root
	app.mu.Unlock()
	if root == "" {
		return
	}
	groups, err := core.RecentDiagnosticGroups(root)
	if err != nil {
		return
	}
	app.mu.Lock()
	app.diagnosticGroups = groups
	app.diagnosticSelected = 0
	app.mu.Unlock()
}

func selectedDiagnosticGroup() (core.DiagnosticGroup, bool) {
	app.mu.Lock()
	defer app.mu.Unlock()
	i := app.diagnosticSelected
	if len(app.diagnosticGroups) == 0 || i < 0 || i >= len(app.diagnosticGroups) {
		return core.DiagnosticGroup{}, false
	}
	return app.diagnosticGroups[i], true
}

func openDiagnosticURL(address string) {
	sysCall(shellExecute, app.hwnd, uintptr(unsafe.Pointer(ptr("open"))), uintptr(unsafe.Pointer(ptr(address))), 0, 0, SW_SHOW)
}

func findMatchingDiagnosticIssues(g core.DiagnosticGroup) {
	prompt := "Search GitHub for this diagnostic group?\n\nThe category, operation and Title ID are represented by a group ID in the search URL. No game files, patches, local paths or passwords are sent.\n\nContinue?"
	title := "Find matching issue reports"
	if sysCall(diagnosticMessageBox, app.hwnd, uintptr(unsafe.Pointer(ptr(prompt))), uintptr(unsafe.Pointer(ptr(title))), 0x24|0x100) != 6 {
		return
	}
	query := url.Values{}
	query.Set("q", "is:issue in:title "+g.ID)
	openDiagnosticURL("https://github.com/TroJanVirus/MagicPatchDesk/issues?" + query.Encode())
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
	if action == "diagnostics-prev" || action == "diagnostics-next" {
		app.mu.Lock()
		size := len(app.diagnosticGroups)
		if size > 0 {
			if action == "diagnostics-next" {
				app.diagnosticSelected = (app.diagnosticSelected + 1) % size
			} else {
				app.diagnosticSelected = (app.diagnosticSelected + size - 1) % size
			}
		}
		app.mu.Unlock()
		redraw()
		return
	}
	switch action {
	case "diagnostics-folder":
		path, err := diagnosticFolder()
		if err != nil {
			notify(err.Error())
			return
		}
		openDiagnosticURL(path)
	case "diagnostics-view":
		group, ok := selectedDiagnosticGroup()
		if !ok {
			notify("No failures have been recorded yet.")
			return
		}
		body := group.Summary() + "\n\nAll original reports remain in the local Reports folder. Review before sharing."
		title := "Diagnostic group · " + group.ID
		sysCall(diagnosticMessageBox, app.hwnd, uintptr(unsafe.Pointer(ptr(body))), uintptr(unsafe.Pointer(ptr(title))), 0x40)
	case "diagnostics-copy":
		group, ok := selectedDiagnosticGroup()
		if !ok {
			notify("No diagnostic group to copy.")
			return
		}
		if err := copyDiagnosticText(group.Summary()); err != nil {
			notify("Clipboard unavailable: " + err.Error())
			return
		}
		notify("Group summary copied. Nothing transmitted.")
	case "diagnostics-find":
		group, ok := selectedDiagnosticGroup()
		if !ok {
			notify("No diagnostic group to search.")
			return
		}
		findMatchingDiagnosticIssues(group)
	case "diagnostics-github":
		group, ok := selectedDiagnosticGroup()
		if !ok {
			notify("No diagnostic group to share.")
			return
		}
		prompt := "This group may already have a GitHub issue.\n\nYES = Search matching issues first (recommended)\nNO = Compose a new issue (review before submitting)\nCANCEL = Do nothing\n\nNo files, passwords or full local paths are automatically shared. GitHub issues may be public."
		title := "Diagnostic report · " + group.ID
		switch sysCall(diagnosticMessageBox, app.hwnd, uintptr(unsafe.Pointer(ptr(prompt))), uintptr(unsafe.Pointer(ptr(title))), 0x03|0x30|0x100) {
		case 6:
			// Copy so the person can add the group summary as a comment to an
			// existing issue. A failed clipboard operation must not block search.
			_ = copyDiagnosticText(group.Summary())
			findMatchingDiagnosticIssues(group)
		case 7:
			query := url.Values{}
			query.Set("title", group.IssueTitle())
			query.Set("body", group.Summary()+"\n\nSteps to reproduce:\n1. \n2. \n\nExpected behaviour:\n\nActual behaviour:\n\nIf an issue with this group ID exists, add a comment to it instead.\n\nOptional: attach a REVIEWED JSON from the Reports folder. Do not attach game binaries, patches, credentials or private paths.")
			openDiagnosticURL("https://github.com/TroJanVirus/MagicPatchDesk/issues/new?" + query.Encode())
		}
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
