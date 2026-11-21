//go:build windows

package main

import (
	"context"
	"fmt"
	"magicpatchdesk/core"
	"net/http"
	"net/url"
	"time"
	"unsafe"
)

// This workspace is read-only with respect to GitHub. Only an explicit Refresh
// button initiates a GET request; writes are prepared for manual browser review.
func refreshDeveloperDashboardLocal() {
	app.mu.Lock()
	root := app.root
	issues := append([]core.DashboardIssue(nil), app.dashboardIssues...)
	sortMode := app.dashboardSort
	previousKey := ""
	if app.dashboardSelected >= 0 && app.dashboardSelected < len(app.dashboardCases) {
		previousKey = app.dashboardCases[app.dashboardSelected].Key
	}
	app.mu.Unlock()
	if root == "" {
		return
	}
	reports, err := core.RecentFailureReports(root, 1000)
	if err != nil {
		return
	}
	cases := core.BuildDeveloperCases(reports, issues, sortMode)
	selected := 0
	for i, c := range cases {
		if c.Key == previousKey {
			selected = i
			break
		}
	}
	app.mu.Lock()
	app.dashboardCases = cases
	app.dashboardSelected = selected
	if len(cases) == 0 {
		app.dashboardScroll = 0
	}
	app.mu.Unlock()
	redraw()
}

func refreshDeveloperDashboardGitHub() {
	app.mu.Lock()
	if app.dashboardBusy {
		app.mu.Unlock()
		return
	}
	app.dashboardBusy = true
	app.dashboardStatus = "Fetching public GitHub issues (read-only)…"
	app.mu.Unlock()
	redraw()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 55*time.Second)
		defer cancel()
		client := &http.Client{Timeout: 12 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return fmt.Errorf("unexpected GitHub API redirect blocked")
		}}
		snapshot, err := core.FetchDeveloperIssues(ctx, client)
		app.mu.Lock()
		app.dashboardBusy = false
		if err != nil {
			app.dashboardStatus = "GitHub unavailable: " + err.Error() + " · Local reports still available"
		} else {
			app.dashboardIssues = snapshot.Issues
			app.dashboardFetchedAt = snapshot.FetchedAt
			app.dashboardIncomplete = snapshot.Incomplete
			suffix := ""
			if snapshot.Incomplete {
				suffix = " · first 500 issues only"
			}
			app.dashboardStatus = fmt.Sprintf("Fetched %d public issue/PR records%s", len(snapshot.Issues), suffix)
		}
		app.mu.Unlock()
		if err == nil {
			refreshDeveloperDashboardLocal()
		} else {
			redraw()
		}
	}()
}

func developerSelectedCase() (core.DeveloperCase, bool) {
	app.mu.Lock()
	defer app.mu.Unlock()
	if app.dashboardSelected < 0 || app.dashboardSelected >= len(app.dashboardCases) {
		return core.DeveloperCase{}, false
	}
	return app.dashboardCases[app.dashboardSelected], true
}

func developerAction(action string, idx int) {
	switch action {
	case "dev-refresh":
		refreshDeveloperDashboardGitHub()
		return
	case "dev-sort":
		app.mu.Lock()
		app.dashboardSort = core.DeveloperSort(idx)
		app.mu.Unlock()
		refreshDeveloperDashboardLocal()
		return
	case "dev-select":
		app.mu.Lock()
		if idx >= 0 && idx < len(app.dashboardCases) {
			app.dashboardSelected = idx
		}
		app.mu.Unlock()
		redraw()
		return
	case "dev-prev", "dev-next":
		app.mu.Lock()
		if len(app.dashboardCases) > 0 {
			step := -1
			if action == "dev-next" {
				step = 1
			}
			app.dashboardSelected = (app.dashboardSelected + step + len(app.dashboardCases)) % len(app.dashboardCases)
			app.dashboardScroll = app.dashboardSelected
		}
		app.mu.Unlock()
		redraw()
		return
	case "dev-github":
		openDiagnosticURL("https://github.com/TroJanVirus/MagicPatchDesk/issues")
		return
	}
	c, ok := developerSelectedCase()
	if !ok {
		notify("No diagnostic case is selected.")
		return
	}
	switch action {
	case "dev-view":
		body := c.Summary() + "\n\nGitHub report counts are issue counts, not confirmed user counts.\nThe local count covers only this computer."
		sysCall(diagnosticMessageBox, app.hwnd, uintptr(unsafe.Pointer(ptr(body))), uintptr(unsafe.Pointer(ptr("Developer case"))), 0x40)
	case "dev-copy":
		if e := copyDiagnosticText(c.Summary()); e != nil {
			notify("Unable to copy case summary: " + e.Error())
		} else {
			notify("Case summary copied; nothing uploaded")
		}
	case "dev-root":
		if u := core.DeveloperIssueURL(c.RootIssue); u != "" {
			openDiagnosticURL(u)
		} else {
			notify("No root-cause issue linked. Create one or copy related-group markers.")
		}
	case "dev-report":
		if len(c.LinkedIssueNumbers) == 0 {
			notify("No GitHub report issue linked yet.")
			return
		}
		n := c.LinkedIssueNumbers[0]
		for _, v := range c.LinkedIssueNumbers {
			if v != c.RootIssue {
				n = v
				break
			}
		}
		openDiagnosticURL(core.DeveloperIssueURL(n))
	case "dev-fixpr":
		if u := core.DeveloperPRURL(c.FixPR); u != "" {
			openDiagnosticURL(u)
		} else {
			notify("No fix PR is linked yet.")
		}
	case "dev-copygroups":
		if e := copyDiagnosticText(c.RelatedGroupsMarker()); e != nil {
			notify("Unable to copy markers: " + e.Error())
		} else {
			notify("Group markers copied. Add them to a root-cause issue in GitHub.")
		}
	case "dev-copyroot":
		if c.RootIssue == 0 {
			notify("No root issue linked yet. Create a tracking issue first.")
			return
		}
		if e := copyDiagnosticText(c.RootCauseMarker()); e != nil {
			notify("Clipboard failed: " + e.Error())
		} else {
			notify("Root cause marker copied. Paste into the relevant diagnostic issue body.")
		}
	case "dev-create":
		if c.RootIssue > 0 {
			notify("This case already has root issue #" + fmt.Sprint(c.RootIssue))
			return
		}
		title := "Create root-cause issue in GitHub?\n\nThis opens an editable, prefilled browser form. Nothing is posted until you submit it. Check the groups and remove any private information."
		if sysCall(diagnosticMessageBox, app.hwnd, uintptr(unsafe.Pointer(ptr(title))), uintptr(unsafe.Pointer(ptr("Developer triage"))), 0x24|0x100) != 6 {
			return
		}
		query := url.Values{}
		query.Set("title", fmt.Sprintf("[Root cause] %s / %s", c.Category, c.Operation))
		query.Set("body", c.RelatedGroupsMarker()+"\n\nCause and reproducible steps:\n\nFix PR: #\n\nVerification checklist:\n- [ ] Reproduced\n- [ ] Fix implemented\n- [ ] Windows checked\n- [ ] xemu / Xbox checked\n\nRelated diagnostic issue links:\n")
		openDiagnosticURL("https://github.com/TroJanVirus/MagicPatchDesk/issues/new?" + query.Encode())
	}
}

func drawDeveloperDashboard(hdc uintptr, w, h int32) {
	drawPageHeader(hdc, "Developer diagnostics", "Triage recurring failures from this PC and public GitHub Issues. Refresh is opt-in.", w)
	x := int32(255)
	width := w - 278
	app.mu.Lock()
	cases := append([]core.DeveloperCase(nil), app.dashboardCases...)
	selected := app.dashboardSelected
	scroll := app.dashboardScroll
	sortMode := app.dashboardSort
	status := app.dashboardStatus
	busy := app.dashboardBusy
	fetched := app.dashboardFetchedAt
	partial := app.dashboardIncomplete
	app.mu.Unlock()
	button(hdc, "Refresh GitHub", ir(w-205, 36, 174, 43), "dev-refresh", 0, true)
	panel(hdc, ir(x, 120, width, 132), surface, 16)
	local, online, openCount, tracked := 0, 0, 0, 0
	for _, c := range cases {
		local += c.LocalOccurrences
		online += c.GitHubReports
		openCount += c.OpenReports
		if c.RootIssue > 0 {
			tracked++
		}
	}
	statXs := []int32{x + 22, x + width/4 + 14, x + width/2 + 8, x + 3*width/4 + 2}
	for i, metric := range []struct {
		n     int
		label string
	}{{len(cases), "CASES"}, {online, "GITHUB REPORTS"}, {openCount, "OPEN REPORTS"}, {local, "LOCAL FAILURES"}} {
		textAt(hdc, fmt.Sprintf("%d", metric.n), statXs[i], 133, width/4-15, 42, app.fonts.Hero, white)
		textAt(hdc, metric.label, statXs[i], 179, width/4-15, 18, app.fonts.Micro, muted)
	}
	syncLine := "GitHub not loaded. Local reports are still listed."
	if !fetched.IsZero() {
		syncLine = "Last GitHub refresh (UTC): " + fetched.UTC().Format("2006-01-02 15:04")
	}
	if partial {
		syncLine += "  ·  PARTIAL (first 500 records)"
	}
	if status != "" {
		syncLine = status
	}
	if busy {
		syncLine = "Fetching GitHub issues on demand…"
	}
	textAt(hdc, strLimit(syncLine, 135), x+22, 215, width-44, 23, app.fonts.Small, muted)

	textAt(hdc, "SORT BY", x+10, 261, 65, 24, app.fonts.Micro, muted)
	buttonNames := []string{"Most reports", "Newest", "Open first"}
	for i, label := range buttonNames {
		bx := x + 78 + int32(i)*139
		if bx+132 > x+width-8 {
			break
		}
		if sortMode == core.DeveloperSort(i) {
			label = "● " + label
		}
		button(hdc, label, ir(bx, 257, 132, 33), "dev-sort", i, false)
	}
	bodyTop := int32(305)
	bodyBottom := h - 60
	bodyHeight := bodyBottom - bodyTop
	if bodyHeight < 140 {
		bodyHeight = 140
	}
	listWidth := width
	split := width >= 820 && h >= 780
	if split {
		listWidth = width * 53 / 100
	}
	panel(hdc, ir(x, bodyTop, listWidth, bodyHeight), surface, 16)
	textAt(hdc, fmt.Sprintf("TRACKED CASES  ·  %d LINKED ROOTS", tracked), x+18, bodyTop+10, listWidth-36, 22, app.fonts.Micro, muted)
	start := bodyTop + 44
	rows := int((bodyHeight - 65) / 61)
	if !split {
		rows = int((bodyHeight - 108) / 61)
	}
	if rows < 1 {
		rows = 1
	}
	if selected < scroll {
		scroll = selected
	}
	if selected >= scroll+rows {
		scroll = selected - rows + 1
	}
	if scroll < 0 {
		scroll = 0
	}
	if scroll > len(cases)-rows {
		scroll = len(cases) - rows
	}
	if scroll < 0 {
		scroll = 0
	}
	app.mu.Lock()
	app.dashboardScroll = scroll
	app.mu.Unlock()
	for i := 0; i < rows && i+scroll < len(cases); i++ {
		index := i + scroll
		c := cases[index]
		y := start + int32(i)*61
		back := uint32(0x202F45)
		if index == selected {
			back = 0x304B72
		} else if app.hoverAction == "dev-select" && app.hoverIndex == index {
			back = 0x29435E
		}
		panel(hdc, ir(x+11, y, listWidth-22, 55), back, 10)
		prefix := fmt.Sprintf("%d. %s", index+1, c.Label())
		textAt(hdc, strLimit(prefix, 45), x+23, y+5, listWidth-44, 23, app.fonts.Medium, white)
		detail := fmt.Sprintf("%d GitHub  ·  %d local  ·  %d open  ·  %d groups", c.GitHubReports, c.LocalOccurrences, c.OpenReports, len(c.GroupIDs))
		if c.RootConflict {
			detail = "CONFLICTING ROOT LINKS  ·  " + detail
		}
		textAt(hdc, strLimit(detail, 75), x+23, y+30, listWidth-43, 19, app.fonts.Small, map[bool]uint32{true: amber, false: muted}[c.RootConflict])
		addHit(ir(x+11, y, listWidth-22, 55), "dev-select", index)
	}
	if len(cases) == 0 {
		textAt(hdc, "No diagnostic cases yet", x+24, start+24, listWidth-45, 28, app.fonts.Title, white)
		textAt(hdc, "Local failures appear automatically. Press Refresh GitHub to load shared reports.", x+24, start+57, listWidth-46, 65, app.fonts.Small, muted)
	}
	if !split {
		button(hdc, "Open issue", ir(x+16, bodyBottom-53, (listWidth-52)/2, 37), "dev-report", 0, false)
		button(hdc, "Case details", ir(x+29+(listWidth-52)/2, bodyBottom-53, (listWidth-52)/2, 37), "dev-view", 0, false)
		return
	}
	dx := x + listWidth + 12
	dw := width - listWidth - 12
	panel(hdc, ir(dx, bodyTop, dw, bodyHeight), surface, 16)
	if selected < 0 || selected >= len(cases) {
		return
	}
	c := cases[selected]
	textAt(hdc, "CASE DETAILS", dx+20, bodyTop+13, dw-40, 23, app.fonts.Micro, muted)
	textAt(hdc, strLimit(c.Label(), 47), dx+20, bodyTop+41, dw-40, 32, app.fonts.Title, white)
	summary := []string{
		"Category: " + c.Category,
		"Operation: " + c.Operation,
		"Title ID: " + c.TitleID,
		fmt.Sprintf("Groups: %d   ·   Reports: %d", len(c.GroupIDs), c.GitHubReports),
		fmt.Sprintf("Local reports: %d", c.LocalOccurrences),
		fmt.Sprintf("Root issue: #%d  %s", c.RootIssue, c.RootIssueState),
		fmt.Sprintf("Fix PR: #%d  %s", c.FixPR, c.FixPRState),
	}
	for i, line := range summary {
		textAt(hdc, strLimit(line, 53), dx+20, bodyTop+90+int32(i)*25, dw-38, 22, app.fonts.Small, map[bool]uint32{true: amber, false: muted}[c.RootConflict && i == 5])
	}
	if c.RootConflict {
		textAt(hdc, "Root links conflict: review before merging.", dx+20, bodyTop+268, dw-40, 23, app.fonts.Small, amber)
	}
	// Put navigation controls at the foot of the details panel. Icons are native
	// and existing click/hover state logic applies to all buttons.
	sy := bodyTop + bodyHeight - 186
	if sy < bodyTop+288 {
		sy = bodyTop + 288
	}
	bw := (dw - 54) / 2
	actions := []struct{ label, id string }{
		{"Open report", "dev-report"}, {"Open root issue", "dev-root"},
		{"Open fix PR", "dev-fixpr"}, {"Copy case", "dev-copy"},
		{"Copy group markers", "dev-copygroups"}, {"Copy root marker", "dev-copyroot"},
		{"New root issue", "dev-create"}, {"Full details", "dev-view"},
	}
	for i, a := range actions {
		by := sy + int32(i/2)*41
		if by+36 > bodyTop+bodyHeight-9 {
			break
		}
		bx := dx + 17 + int32(i%2)*(bw+12)
		button(hdc, a.label, ir(bx, by, bw, 35), a.id, 0, i == 6)
	}
}
