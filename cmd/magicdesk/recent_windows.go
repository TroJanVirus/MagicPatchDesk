//go:build windows

package main

import (
	"fmt"
	"magicpatchdesk/core"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unsafe"
)

var recentStoreMu sync.Mutex

func refreshRecentFiles() {
	recentStoreMu.Lock()
	defer recentStoreMu.Unlock()
	app.mu.Lock()
	root := app.root
	app.mu.Unlock()
	if root == "" {
		return
	}
	files, err := core.LoadRecentFiles(root)
	if err != nil {
		notify("Cannot read recent file history: " + err.Error())
		return
	}
	app.mu.Lock()
	app.recentFiles = files
	if len(files) == 0 {
		app.recentSelected = 0
		app.recentOffset = 0
	}
	if app.recentSelected >= len(files) {
		app.recentSelected = len(files) - 1
	}
	if app.recentSelected < 0 {
		app.recentSelected = 0
	}
	app.mu.Unlock()
}

func rememberRecent(kind, path string) {
	if path == "" {
		return
	}
	recentStoreMu.Lock()
	defer recentStoreMu.Unlock()
	app.mu.Lock()
	root := app.root
	previous := append([]core.RecentFile(nil), app.recentFiles...)
	app.mu.Unlock()
	if root == "" {
		return
	}
	files, err := core.AddRecentFile(previous, kind, path, time.Now())
	if err != nil {
		return
	}
	if err = core.SaveRecentFiles(root, files); err != nil {
		notify("Recent file history could not be saved: " + err.Error())
		return
	}
	app.mu.Lock()
	app.recentFiles = files
	app.recentSelected = 0
	app.recentOffset = 0
	app.mu.Unlock()
	redraw()
}

func recentChosen() (core.RecentFile, bool) {
	app.mu.Lock()
	defer app.mu.Unlock()
	if app.recentSelected < 0 || app.recentSelected >= len(app.recentFiles) {
		return core.RecentFile{}, false
	}
	return app.recentFiles[app.recentSelected], true
}

func recentFolder(path string) {
	if path == "" {
		return
	}
	info, err := os.Stat(path)
	folder := filepath.Dir(path)
	if err == nil && info.IsDir() {
		folder = path
	}
	if _, e := os.Stat(folder); e != nil {
		notify("Folder is no longer available.")
		return
	}
	sysCall(shellExecute, app.hwnd, uintptr(unsafe.Pointer(ptr("open"))), uintptr(unsafe.Pointer(ptr(folder))), 0, 0, SW_SHOW)
}

func recentOpen(item core.RecentFile) {
	info, err := os.Stat(item.Path)
	if err != nil {
		notify("Recent file is missing or inaccessible. Use Remove to forget it.")
		return
	}
	if item.Kind != "output" && !info.Mode().IsRegular() {
		notify("This recent input is not a regular file.")
		return
	}
	app.mu.Lock()
	busy := app.busy
	app.mu.Unlock()
	if busy {
		notify("Finish the current operation before opening another file.")
		return
	}
	switch item.Kind {
	case "input":
		app.tab = 0
		openInput(item.Path)
	case "patch":
		app.tab = 0
		app.directFile = item.Path
		app.parsed = nil
		app.packSources = nil
		app.status = "Reopened patch: " + filepath.Base(item.Path)
	case "convert_original":
		app.tab = 1
		app.aFile = item.Path
		refreshConversionHint()
	case "convert_patch":
		app.tab = 1
		app.convertPatchFile = item.Path
		app.bFile = ""
		app.packSources = nil
		if strings.EqualFold(filepath.Ext(item.Path), ".jmp") {
			app.format = core.AllPatchFormats
		}
		if strings.EqualFold(filepath.Ext(item.Path), ".pf") {
			app.format = 0
		}
		refreshConversionHint()
	case "convert_modified":
		app.tab = 1
		app.bFile = item.Path
		app.convertPatchFile = ""
		refreshConversionHint()
	case "output":
		recentFolder(item.Path)
	default:
		return
	}
	// Reloading a recent source also moves it to the top, like a file picker.
	rememberRecent(item.Kind, item.Path)
	redraw()
}

func recentAction(action string, index int) {
	switch action {
	case "recent-select":
		app.mu.Lock()
		if index >= 0 && index < len(app.recentFiles) {
			app.recentSelected = index
		}
		app.mu.Unlock()
	case "recent-prev", "recent-next":
		step := recentVisibleRows(app.h)
		app.mu.Lock()
		if action == "recent-prev" {
			app.recentOffset -= step
		} else {
			app.recentOffset += step
		}
		if app.recentOffset < 0 {
			app.recentOffset = 0
		}
		if app.recentOffset >= len(app.recentFiles) && len(app.recentFiles) > 0 {
			app.recentOffset = ((len(app.recentFiles) - 1) / step) * step
		}
		app.mu.Unlock()
	case "recent-open", "recent-folder", "recent-copy", "recent-remove":
		item, ok := recentChosen()
		if !ok {
			notify("Select a recent file first.")
			return
		}
		switch action {
		case "recent-open":
			recentOpen(item)
		case "recent-folder":
			recentFolder(item.Path)
		case "recent-copy":
			if e := copyDiagnosticText(item.Path); e != nil {
				notify("Clipboard unavailable: " + e.Error())
			} else {
				notify("Selected local file path copied.")
			}
		case "recent-remove":
			recentStoreMu.Lock()
			app.mu.Lock()
			root := app.root
			items := append([]core.RecentFile(nil), app.recentFiles...)
			app.mu.Unlock()
			out := items[:0]
			for _, v := range items {
				if v.Path != item.Path {
					out = append(out, v)
				}
			}
			e := core.SaveRecentFiles(root, out)
			if e == nil {
				app.mu.Lock()
				app.recentFiles = append([]core.RecentFile(nil), out...)
				app.recentSelected = 0
				app.recentOffset = 0
				app.mu.Unlock()
			}
			recentStoreMu.Unlock()
			if e != nil {
				notify("Could not update recent files: " + e.Error())
			}
		}
	case "recent-clear":
		answer := sysCall(targetedMessageBox, app.hwnd, uintptr(unsafe.Pointer(ptr("Clear local recent-file history?\n\nThis will not delete any patch files, game files or reports."))), uintptr(unsafe.Pointer(ptr("Clear recent history"))), 0x24|0x100)
		if answer != 6 {
			return
		}
		recentStoreMu.Lock()
		app.mu.Lock()
		root := app.root
		app.mu.Unlock()
		err := core.ClearRecentFiles(root)
		if err == nil {
			app.mu.Lock()
			app.recentFiles = nil
			app.recentSelected = 0
			app.recentOffset = 0
			app.mu.Unlock()
		}
		recentStoreMu.Unlock()
		if err != nil {
			notify("Could not clear history: " + err.Error())
		} else {
			notify("Recent-file history cleared; no game files deleted.")
		}
	}
	redraw()
}

func recentVisibleRows(h int32) int {
	count := int((h - 350) / 62)
	if count < 2 {
		count = 2
	}
	if count > 9 {
		count = 9
	}
	return count
}
func recentLabel(kind string) string {
	switch kind {
	case "input":
		return "GAME SOURCE"
	case "patch":
		return "PATCH FILE"
	case "convert_original":
		return "CONVERT SOURCE"
	case "convert_patch":
		return "CONVERT PATCH"
	case "convert_modified":
		return "MODIFIED XBE"
	case "output":
		return "SAVED OUTPUT"
	}
	return "FILE"
}

func drawRecent(hdc uintptr, w, h int32) {
	drawPageHeader(hdc, "Recent files", "Quickly reopen your last patches, game sources and saved outputs. Everything is stored locally.", w)
	x := int32(265)
	width := w - 295
	panel(hdc, ir(x, 130, width, h-215), surface, 18)
	textAt(hdc, "RECENT WORK", x+25, 144, width-230, 29, app.fonts.Title, white)
	textAt(hdc, "Ctrl+O  Open source     Ctrl+P  Load patch     Ctrl+F  Search     Ctrl+R  Recent", x+26, 176, width-48, 24, app.fonts.Small, muted)
	app.mu.Lock()
	items := append([]core.RecentFile(nil), app.recentFiles...)
	selected := app.recentSelected
	offset := app.recentOffset
	app.mu.Unlock()
	textAt(hdc, fmt.Sprintf("%d saved", len(items)), x+width-120, 151, 95, 22, app.fonts.Small, muted)
	visible := recentVisibleRows(h)
	if offset < 0 {
		offset = 0
	}
	if len(items) > 0 && offset >= len(items) {
		offset = ((len(items) - 1) / visible) * visible
	}
	end := offset + visible
	if end > len(items) {
		end = len(items)
	}
	y := int32(215)
	if len(items) == 0 {
		textAt(hdc, "No recent files yet. Load an original XBE/ISO or a patch to begin.", x+26, y+35, width-58, 58, app.fonts.Regular, muted)
	}
	for i := offset; i < end; i++ {
		item := items[i]
		selectedRow := selected == i
		color := uint32(0x202F47)
		if selectedRow {
			color = 0x2D4C71
		} else if app.hoverAction == "recent-select" && app.hoverIndex == i {
			color = 0x293E5A
		}
		rowY := y + int32(i-offset)*62
		panel(hdc, ir(x+25, rowY, width-52, 56), color, 9)
		if selectedRow {
			fill(hdc, ir(x+25, rowY+7, 3, 42), blue)
		}
		name := filepath.Base(item.Path)
		textAt(hdc, strLimit(name, 74), x+38, rowY+5, width-245, 24, app.fonts.Medium, white)
		textAt(hdc, strLimit(filepath.Dir(item.Path), 90), x+38, rowY+28, width-245, 21, app.fonts.Small, muted)
		textAt(hdc, recentLabel(item.Kind), x+width-188, rowY+13, 145, 22, app.fonts.Micro, mint)
		addHit(ir(x+24, rowY, width-50, 56), "recent-select", i)
	}
	topButtons := h - 133
	if topButtons < y+62 {
		topButtons = y + 62
	}
	const buttonGap int32 = 10
	buttonWidth := (width - 72) / 4
	if buttonWidth < 74 {
		buttonWidth = 74
	}
	for i, a := range []struct{ label, action string }{{"Open selected", "recent-open"}, {"Open folder", "recent-folder"}, {"Copy path", "recent-copy"}, {"Forget", "recent-remove"}} {
		button(hdc, a.label, ir(x+25+int32(i)*(buttonWidth+buttonGap), topButtons, buttonWidth, 38), a.action, 0, false)
	}
	button(hdc, "Clear history", ir(x+25, topButtons+47, 145, 33), "recent-clear", 0, false)
	textAt(hdc, fmt.Sprintf("Page %d/%d", offset/visible+1, max(1, (len(items)+visible-1)/visible)), x+width-338, topButtons+50, 115, 24, app.fonts.Small, muted)
	button(hdc, "Previous", ir(x+width-214, topButtons+47, 94, 33), "recent-prev", 0, false)
	button(hdc, "Next", ir(x+width-108, topButtons+47, 81, 33), "recent-next", 0, false)
}

func handleQuickShortcut(key uintptr) bool {
	// Ctrl+letter shortcuts are ignored during a live patch operation.
	ctrl := int16(sysCall(user32.NewProc("GetKeyState"), 0x11)) < 0
	if !ctrl {
		if key == 0x74 {
			go reloadCatalog()
			notify("Refreshing local patch library…")
			return true
		} // F5
		return false
	}
	app.mu.Lock()
	busy := app.busy
	app.mu.Unlock()
	if busy {
		return false
	}
	switch key {
	case 'O':
		handle("open", 0)
	case 'P':
		handle("direct", 0)
	case 'F':
		app.tab = 0
		app.searchFocus = true
		app.passFocus = false
		redraw()
	case 'R':
		app.tab = 5
		app.searchFocus = false
		app.passFocus = false
		refreshRecentFiles()
		redraw()
	case '1', '2', '3', '4', '5', '6', '7':
		handle("tab", int(key-'1'))
	default:
		return false
	}
	return true
}
