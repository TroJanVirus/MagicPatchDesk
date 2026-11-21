//go:build windows

package main

import (
	"fmt"
	"magicpatchdesk/core"
	"os"
	"path/filepath"
)

func drawMaintenance(hdc uintptr, w, h int32) {
	drawPageHeader(hdc, "Updates & maintenance", "Install, update, recover and manage reports or Cerbios CCI images.", w)
	x := int32(266)
	width := w - 298
	compact := h < 800
	heightA := int32(185)
	heightB := int32(90)
	heightC := int32(140)
	heightD := int32(105)
	gap := int32(11)
	if compact {
		heightA = 148
		heightB = 72
		heightC = 115
		heightD = 90
		gap = 8
	}
	y := int32(130)
	panel(hdc, ir(x, y, width, heightA), surface, 16)
	textAt(hdc, "SELF-UPDATE", x+24, y+8, width-45, 28, app.fonts.Title, white)
	current := core.AppVersion
	status := maintenanceUpdate.Status
	if status == "" {
		status = "Check for new releases to see available updates"
	}
	textAt(hdc, "Installed version: "+current+"  ·  "+strLimit(status, 85), x+24, y+40, width-50, 25, app.fonts.Small, muted)
	exe, _ := os.Executable()
	where := "Portable EXE (not installed)"
	if core.IsInstalledPath(exe) {
		where = "Installed: " + filepath.Dir(exe)
	}
	if !compact {
		textAt(hdc, strLimit(where, 85), x+24, y+67, width-50, 22, app.fonts.Small, muted)
	}
	by := y + heightA - 49
	bw := (width - 75) / 3
	button(hdc, "Install app", ir(x+20, by, bw, 37), "maint-install", 0, false)
	button(hdc, "Check updates", ir(x+29+bw, by, bw, 37), "maint-check", 0, false)
	button(hdc, "Update now", ir(x+38+bw*2, by, bw, 37), "maint-update", 0, maintenanceUpdate.Found)
	y += heightA + gap
	panel(hdc, ir(x, y, width, heightB), surface, 16)
	textAt(hdc, "CHANNEL & STORAGE", x+24, y+7, width-45, 24, app.fonts.Title, white)
	button(hdc, func() string {
		if maintenancePref.Beta {
			return "PREVIEWS ON"
		}
		return "STABLE ONLY"
	}(), ir(x+22, y+heightB-43, 142, 33), "maint-beta", 0, maintenancePref.Beta)
	button(hdc, func() string {
		if maintenancePref.NoAutoCheck {
			return "CHECK MANUALLY"
		}
		return "CHECK AT START"
	}(), ir(x+174, y+heightB-43, 162, 33), "maint-autocheck", 0, !maintenancePref.NoAutoCheck)
	if width > 530 {
		button(hdc, "App folder", ir(x+width-202, y+heightB-43, 81, 33), "maint-program", 0, false)
		button(hdc, "Data folder", ir(x+width-114, y+heightB-43, 91, 33), "maint-data", 0, false)
	}
	y += heightB + gap
	panel(hdc, ir(x, y, width, heightC), surface, 16)
	textAt(hdc, "AUTOMATED DIAGNOSTIC REPORTS", x+24, y+8, width-45, 25, app.fonts.Title, white)
	info := "Reports never upload game files. GitHub Issues permission is required."
	if maintenancePref.AutoReports {
		info = "Opted in: privacy-filtered reports submit on error; queued offline on failure."
	}
	textAt(hdc, strLimit(info, 95), x+24, y+39, width-48, 22, app.fonts.Small, muted)
	bw = (width - 65) / 4
	if bw < 90 {
		bw = 90
	}
	by = y + heightC - 42
	for i, a := range []struct {
		label, action string
		emphasis      bool
	}{{"Connect token", "maint-connect", false}, {"Disconnect", "maint-disconnect", false}, {"Auto-report", "maint-reports", maintenancePref.AutoReports}, {"Sync now", "maint-sync", false}} {
		button(hdc, a.label, ir(x+18+int32(i)*(bw+7), by, bw, 34), a.action, 0, a.emphasis)
	}
	y += heightC + gap
	panel(hdc, ir(x, y, width, heightD), surface, 16)
	textAt(hdc, "CERBIOS CCI TOOLS", x+24, y+8, width-45, 26, app.fonts.Title, white)
	if !compact {
		textAt(hdc, "Decode CCIM/LZ4 to XISO or compress a plain ISO back to CCI (source preserved).", x+24, y+34, width-47, 22, app.fonts.Small, muted)
	}
	by = y + heightD - 42
	bw = (width - 62) / 2
	button(hdc, "CCI → ISO", ir(x+21, by, bw, 33), "maint-cci-decompress", 0, false)
	button(hdc, "ISO → CCI", ir(x+32+bw, by, bw, 33), "maint-cci-compress", 0, false)
	if h > 850 {
		textAt(hdc, fmt.Sprintf("Update protocol %d · No administrator rights needed · User data stays in %%APPDATA%%", core.UpdateProtocol), x+22, y+heightD+7, width-50, 20, app.fonts.Small, muted)
	}
}
