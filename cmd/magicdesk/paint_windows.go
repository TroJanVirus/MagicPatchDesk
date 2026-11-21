//go:build windows

package main

import (
	"fmt"
	"magicpatchdesk/core"
	"path/filepath"
	"strings"
	"unicode"
	"unsafe"
)

const (
	bg       = 0x0C111C
	sidebar  = 0x111A29
	surface  = 0x182335
	surface2 = 0x1F2D44
	muted    = 0x91A3BD
	white    = 0xF5F8FF
	blue     = 0x6C9CFF
	mint     = 0x5AD5BF
	amber    = 0xFFC678
)

func cover(hdc uintptr, x, y int32, title, art, artURL string) {
	h := uint32(2166136261)
	for _, ch := range title {
		h ^= uint32(ch)
		h *= 16777619
	}
	p := []uint32{0x244F66, 0x584273, 0x31555B, 0x5C4C37, 0x384D7A, 0x364F47}
	color := p[h%uint32(len(p))]
	panel(hdc, ir(x, y, 52, 52), color, 13)
	if art == "" && artURL != "" {
		art = scheduleCover(artURL)
	}
	if drawArt(hdc, art, x+2, y+2, 48, 48) {
		return
	}
	initial := "?"
	for _, r := range title {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			initial = strings.ToUpper(string(r))
			break
		}
	}
	txt(hdc, initial, ir(x, y, 52, 52), app.fonts.Title, white, DT_SINGLELINE|DT_CENTER|DT_VCENTER)
}
func paint(hdc uintptr) {
	var r rect
	sysCall(getClientRect, app.hwnd, uintptr(unsafe.Pointer(&r)))
	w, h := r.R-r.L, r.B-r.T
	app.w = w
	app.h = h
	if w < 700 || h < 500 {
		fill(hdc, r, bg)
		textAt(hdc, "Increase window size to use Magic Patch Desk", 20, 20, w-30, 40, app.fonts.Title, white)
		return
	}
	app.hits = app.hits[:0]
	fill(hdc, r, bg)
	drawSidebar(hdc, w, h)
	switch app.tab {
	case 0:
		drawLibrary(hdc, w, h)
	case 1:
		drawConvert(hdc, w, h)
	case 2:
		drawSettings(hdc, w, h)
	case 3:
		drawAbout(hdc, w, h)
	}
	app.mu.Lock()
	status := app.status
	busy := app.busy
	progress := app.progress
	app.mu.Unlock()
	fill(hdc, ir(232, h-42, w-232, 42), 0x101A2A)
	if busy {
		fill(hdc, ir(232, h-43, int32(float64(w-232)*progress), 3), blue)
	}
	textAt(hdc, "●", 253, h-35, 14, 22, app.fonts.Regular, mint)
	textAt(hdc, strLimit(status, 155), 273, h-35, w-340, 24, app.fonts.Small, muted)
	textAt(hdc, "v0.4", w-67, h-35, 55, 24, app.fonts.Small, 0x7C92AD)
}
func drawSidebar(hdc uintptr, w, h int32) {
	fill(hdc, ir(0, 0, 232, h), sidebar)
	panel(hdc, ir(21, 28, 42, 42), 0x3F73E6, 12)
	txt(hdc, "M", ir(21, 28, 42, 42), app.fonts.Title, white, DT_CENTER|DT_SINGLELINE|DT_VCENTER)
	textAt(hdc, "MAGIC", 74, 25, 142, 25, app.fonts.Title, white)
	textAt(hdc, "PATCH DESK", 75, 49, 140, 16, app.fonts.Micro, 0x88A6CE)
	textAt(hdc, "WORKSPACE", 24, 114, 165, 22, app.fonts.Micro, 0x637793)
	nav := []struct {
		label  string
		symbol string
	}{{"Patch Library", "◈"}, {"Convert & Create", "◇"}, {"Settings", "⚙"}, {"About", "ⓘ"}}
	for i, item := range nav {
		y := int32(145 + i*57)
		if app.tab == i {
			panel(hdc, ir(13, y, 206, 47), 0x283F65, 12)
			fill(hdc, ir(13, y+8, 4, 31), blue)
		}
		txt(hdc, item.symbol, ir(25, y, 27, 47), app.fonts.Title, map[bool]uint32{true: blue, false: muted}[app.tab == i], DT_SINGLELINE|DT_VCENTER|DT_CENTER)
		textAt(hdc, item.label, 61, y+8, 145, 30, app.fonts.Medium, map[bool]uint32{true: white, false: muted}[app.tab == i])
		addHit(ir(10, y, 211, 49), "tab", i)
	}
	panel(hdc, ir(17, h-202, 198, 122), 0x1E2A3E, 14)
	textAt(hdc, "PATCH COLLECTION", 31, h-192, 167, 21, app.fonts.Micro, muted)
	app.mu.Lock()
	remote := app.remoteCount
	status := app.syncStatus
	count := app.cat.PatchCount
	app.mu.Unlock()
	textAt(hdc, fmt.Sprintf("%d", count), 31, h-164, 156, 38, app.fonts.Hero, white)
	textAt(hdc, fmt.Sprintf("Local JMPs  ·  Online %d", remote), 31, h-122, 170, 17, app.fonts.Small, muted)
	button(hdc, "↻   Sync GitHub", ir(19, h-66, 194, 42), "sync", 0, false)
	_ = status
}
func drawPageHeader(hdc uintptr, title, subtitle string, w int32) {
	textAt(hdc, title, 263, 30, w-530, 45, app.fonts.Hero, white)
	textAt(hdc, subtitle, 264, 76, w-520, 26, app.fonts.Small, muted)
}
func drawLibrary(hdc uintptr, w, h int32) {
	drawPageHeader(hdc, "Discover & patch", "Select a game, pick a patch, and build a ready-to-use XBE or XISO.", w)
	button(hdc, "Load patch file", ir(w-365, 38, 167, 42), "direct", 0, false)
	button(hdc, "+  Import JMP", ir(w-186, 38, 155, 42), "import", 0, false)
	listX := int32(255)
	listW := int32(353)
	listTop := int32(117)
	listBottom := h - 161
	rightX := int32(627)
	rightW := w - rightX - 23
	panel(hdc, ir(listX, listTop, listW, listBottom-listTop), surface, 17)
	textAt(hdc, "GAME CATALOG", listX+17, listTop+14, listW-25, 22, app.fonts.Micro, muted)
	panel(hdc, ir(listX+15, listTop+47, listW-30, 43), 0x27364F, 11)
	search := app.search
	if search == "" {
		search = "Search games, types, authors…"
	}
	c := uint32(muted)
	if app.search != "" {
		c = white
	}
	textAt(hdc, "⌕", listX+27, listTop+54, 24, 29, app.fonts.Title, blue)
	textAt(hdc, strLimit(search, 43), listX+58, listTop+55, listW-96, 27, app.fonts.Regular, c)
	if app.searchFocus {
		fill(hdc, ir(listX+15, listTop+89, listW-30, 2), blue)
	}
	addHit(ir(listX+15, listTop+47, listW-30, 43), "search", 0)
	app.mu.Lock()
	catalog := app.cat
	app.mu.Unlock()
	filtered := make([]int, 0, len(catalog.Games))
	term := strings.ToLower(strings.TrimSpace(app.search))
	for i, g := range catalog.Games {
		matches := term == "" || strings.Contains(strings.ToLower(g.Title), term)
		if !matches {
			for _, p := range g.Patches {
				if strings.Contains(strings.ToLower(p.Type+" "+p.Author+" "+p.Name+" "+p.TitleID), term) {
					matches = true
					break
				}
			}
		}
		if matches && app.filter > 0 {
			matches = false
			for _, p := range g.Patches {
				typ := strings.ToLower(p.Type)
				switch app.filter {
				case 1:
					matches = strings.Contains(typ, "720")
				case 2:
					matches = strings.Contains(typ, "wide") || strings.Contains(typ, "16:9")
				case 3:
					matches = strings.Contains(typ, "480")
				case 4:
					matches = !strings.Contains(typ, "720") && !strings.Contains(typ, "wide") && !strings.Contains(typ, "480")
				}
				if matches {
					break
				}
			}
		}
		if matches {
			filtered = append(filtered, i)
		}
	}
	textAt(hdc, fmt.Sprintf("%d GAMES  ·  %d PATCHES", len(filtered), catalog.PatchCount), listX+20, listTop+102, listW-40, 19, app.fonts.Micro, 0x7390B4)
	chips := []struct {
		name  string
		width int32
	}{{"All", 50}, {"720p", 59}, {"Wide", 61}, {"480p", 58}, {"Other", 71}}
	chipX := listX + 15
	for idx, chip := range chips {
		color := uint32(0x23314A)
		if idx == app.filter {
			color = 0x345684
		}
		panel(hdc, ir(chipX, listTop+123, chip.width, 25), color, 10)
		txt(hdc, chip.name, ir(chipX, listTop+123, chip.width, 25), app.fonts.Micro, white, DT_SINGLELINE|DT_VCENTER|DT_CENTER)
		addHit(ir(chipX, listTop+123, chip.width, 25), "filter", idx)
		chipX += chip.width + 5
	}
	gameStart := listTop + 160
	gameEnd := listBottom - 12
	perPage := int((gameEnd - gameStart) / 66)
	if perPage < 1 {
		perPage = 1
	}
	if app.gameScroll > len(filtered)-perPage {
		app.gameScroll = len(filtered) - perPage
	}
	if app.gameScroll < 0 {
		app.gameScroll = 0
	}
	for row := 0; row < perPage && row+app.gameScroll < len(filtered); row++ {
		i := filtered[row+app.gameScroll]
		g := catalog.Games[i]
		y := gameStart + int32(row)*66
		selected := i == app.selectedGame
		if selected {
			panel(hdc, ir(listX+10, y, listW-20, 61), 0x2B4468, 12)
		} else if row%2 == 0 {
			panel(hdc, ir(listX+10, y, listW-20, 61), 0x1D2B40, 11)
		}
		cover(hdc, listX+18, y+5, g.Title, g.Cover, g.CoverURL)
		textAt(hdc, strLimit(g.Title, 29), listX+80, y+6, listW-107, 26, app.fonts.Medium, white)
		textAt(hdc, fmt.Sprintf("%d patch%s available", len(g.Patches), map[bool]string{true: "", false: "es"}[len(g.Patches) == 1]), listX+80, y+34, listW-102, 19, app.fonts.Small, muted)
		addHit(ir(listX+10, y, listW-20, 62), "game", i)
	}
	if len(filtered) == 0 {
		textAt(hdc, "No matching games yet", listX+26, gameStart+21, listW-51, 27, app.fonts.Medium, muted)
		textAt(hdc, "Import local JMPs or sync GitHub.", listX+26, gameStart+55, listW-51, 26, app.fonts.Small, muted)
	}
	panel(hdc, ir(rightX, listTop, rightW, listBottom-listTop), surface, 17)
	if app.directFile != "" {
		textAt(hdc, "CUSTOM PATCH FILE", rightX+30, listTop+29, rightW-50, 34, app.fonts.Title, white)
		textAt(hdc, strLimit(filepath.Base(app.directFile), 72), rightX+30, listTop+77, rightW-60, 28, app.fonts.Medium, mint)
		textAt(hdc, "Supports JMP, IPS, encrypted IPS and Xdelta/VCDIFF.", rightX+30, listTop+125, rightW-60, 27, app.fonts.Small, muted)
		textAt(hdc, "Choose an original XBE / ISO below, then Apply Patch.", rightX+30, listTop+160, rightW-60, 27, app.fonts.Regular, white)
		if strings.EqualFold(filepath.Ext(app.directFile), ".eips") {
			textAt(hdc, "ENCRYPTED IPS PASSPHRASE", rightX+30, listTop+215, rightW-60, 25, app.fonts.Micro, muted)
			panel(hdc, ir(rightX+30, listTop+248, rightW-60, 50), surface2, 10)
			placeholder := "Click to enter the password"
			if app.password != "" {
				placeholder = strings.Repeat("•", len([]rune(app.password)))
			}
			textAt(hdc, placeholder, rightX+44, listTop+258, rightW-82, 27, app.fonts.Regular, white)
			addHit(ir(rightX+30, listTop+248, rightW-60, 50), "password", 0)
		}
	} else if app.selectedGame < 0 || app.selectedGame >= len(catalog.Games) {
		textAt(hdc, "Choose a game", rightX+30, listTop+29, rightW-55, 36, app.fonts.Title, white)
		textAt(hdc, "Browse the collection on the left to see patches,", rightX+31, listTop+75, rightW-53, 25, app.fonts.Regular, muted)
		textAt(hdc, "authors, Title IDs, record groups and checks.", rightX+31, listTop+101, rightW-53, 25, app.fonts.Regular, muted)
		panel(hdc, ir(rightX+28, listTop+165, rightW-56, 130), 0x21324A, 16)
		textAt(hdc, "SAFE BY DESIGN", rightX+48, listTop+185, rightW-97, 24, app.fonts.Micro, mint)
		textAt(hdc, "Your original game file is never overwritten.", rightX+48, listTop+217, rightW-96, 29, app.fonts.Medium, white)
		textAt(hdc, "Patch matching is verified before saving a new copy.", rightX+48, listTop+252, rightW-96, 20, app.fonts.Small, muted)
	} else {
		g := catalog.Games[app.selectedGame]
		drawGameDetails(hdc, rightX, rightW, listTop, listBottom, g)
	}
	panel(hdc, ir(255, h-143, w-280, 85), surface, 16)
	textAt(hdc, "TARGET FILE", 271, h-129, 147, 19, app.fonts.Micro, muted)
	input := app.input
	if input == "" {
		input = "Choose an original Xbox executable or ISO…"
	}
	textAt(hdc, strLimit(filepath.Base(input), 72), 272, h-110, w-625, 30, app.fonts.Medium, white)
	if app.inputInfo.Valid {
		textAt(hdc, fmt.Sprintf("%s   ·   %s   ·   CRC %s", app.inputInfo.Title, app.inputInfo.TitleID, app.inputInfo.CRC32), 273, h-81, w-625, 17, app.fonts.Small, mint)
	} else if app.inputErr != "" {
		textAt(hdc, strLimit(app.inputErr, 95), 273, h-82, w-625, 17, app.fonts.Small, amber)
	}
	button(hdc, "Browse file", ir(w-466, h-119, 122, 43), "open", 0, false)
	button(hdc, "PREVIEW", ir(w-333, h-119, 119, 43), "preview", 0, false)
	button(hdc, "APPLY PATCH", ir(w-202, h-119, 175, 43), "apply", 0, true)
}
func drawGameDetails(hdc uintptr, x, width, top, bottom int32, g core.Game) {
	textAt(hdc, strLimit(g.Title, 44), x+22, top+12, width-43, 38, app.fonts.Title, white)
	textAt(hdc, fmt.Sprintf("%d available patches · Select one", len(g.Patches)), x+23, top+50, width-42, 19, app.fonts.Small, muted)
	listY := top + 83
	listRows := 3
	rowH := int32(57)
	if app.patchScroll > len(g.Patches)-listRows {
		app.patchScroll = len(g.Patches) - listRows
	}
	if app.patchScroll < 0 {
		app.patchScroll = 0
	}
	for i := 0; i < listRows && i+app.patchScroll < len(g.Patches); i++ {
		idx := i + app.patchScroll
		p := g.Patches[idx]
		y := listY + int32(i)*rowH
		chosen := idx == app.selectedPatch
		color := uint32(0x26364D)
		if chosen {
			color = 0x30527D
		}
		panel(hdc, ir(x+16, y, width-32, rowH-5), color, 11)
		tag(hdc, strLimit(p.Type, 17), x+26, y+14, 0x3B5271)
		label := p.Name
		if len(g.Title) > 0 && strings.HasPrefix(strings.ToLower(label), strings.ToLower(g.Title)) {
			label = strings.TrimSpace(strings.TrimPrefix(label, label[:len(g.Title)]))
			label = strings.TrimLeft(label, " -·")
		}
		if label == "" {
			label = p.Type
		}
		textAt(hdc, strLimit(label, 70), x+149, y+7, width-179, 24, app.fonts.Medium, white)
		detail := strings.TrimSpace(p.Region + "  ·  " + p.Author)
		if p.TitleID != "" {
			detail += "  ·  " + p.TitleID
		}
		if p.RAM != "" {
			detail += "  ·  " + p.RAM
		}
		textAt(hdc, strLimit(detail, 90), x+150, y+29, width-181, 17, app.fonts.Small, muted)
		addHit(ir(x+16, y, width-32, rowH-5), "patch", idx)
	}
	p := app.parsed
	groupY := listY + rowH*3 + 14
	fill(hdc, ir(x+18, groupY-5, width-36, 1), 0x354258)
	if p == nil {
		txt(hdc, "Choose a patch above to review and toggle its records.", ir(x+25, groupY+16, width-51, 55), app.fonts.Regular, muted, DT_WORDBREAK)
		return
	}
	textAt(hdc, fmt.Sprintf("PATCH RECORD GROUPS  ·  %d RECORDS", len(p.Records)), x+23, groupY, width-43, 23, app.fonts.Micro, muted)
	textAt(hdc, "Toggle a whole group, or expand + to select records.", x+23, groupY+24, width-43, 20, app.fonts.Small, 0x91A3BD)
	rowY := groupY + 56
	visible := int((bottom - rowY - 88) / 38)
	if visible < 1 {
		visible = 1
	}
	type row struct {
		group  int
		record int
	}
	rows := make([]row, 0, len(p.Groups)+len(p.Records))
	for gi, g := range p.Groups {
		rows = append(rows, row{gi, -1})
		if app.expandedGroup == gi {
			for ri, r := range p.Records {
				if strings.EqualFold(r.Group, g.Name) {
					rows = append(rows, row{gi, ri})
				}
			}
		}
	}
	if app.groupScroll > len(rows)-visible {
		app.groupScroll = len(rows) - visible
	}
	if app.groupScroll < 0 {
		app.groupScroll = 0
	}
	for i := 0; i < visible && i+app.groupScroll < len(rows); i++ {
		r := rows[i+app.groupScroll]
		y := rowY + int32(i)*38
		if r.record >= 0 {
			rec := p.Records[r.record]
			panel(hdc, ir(x+34, y, width-52, 34), 0x1F2F45, 8)
			mark(hdc, ir(x+47, y+8, 18, 18), rec.Enabled)
			label := fmt.Sprintf("#%d  %s", r.record+1, rec.Kind)
            if target, targetErr := core.TargetFromComment(rec.Comment); targetErr == nil && target != "default.xbe" {
                label += " → " + target
            }
			if rec.Kind == core.Offset {
				label += fmt.Sprintf("  ·  0x%X", rec.Offset)
			}
			if rec.Kind == core.SearchReplace {
				label += fmt.Sprintf("  ·  %d bytes", len(rec.Find))
			}
			if rec.Kind == core.Append {
				label += fmt.Sprintf("  ·  %d bytes", len(rec.Data))
			}
			textAt(hdc, strLimit(label, 67), x+78, y+5, width-126, 24, app.fonts.Small, muted)
			if !app.busy {
				addHit(ir(x+34, y, width-52, 34), "record", r.record)
			}
			continue
		}
		g := p.Groups[r.group]
		panel(hdc, ir(x+18, y, width-36, 34), 0x24344B, 9)
		mark(hdc, ir(x+47, y+8, 18, 18), g.Enabled)
		if g.Mixed {
			textAt(hdc, "–", x+50, y+6, 14, 21, app.fonts.Micro, white)
		}
		textAt(hdc, strLimit(g.Name, 50), x+78, y+5, width-185, 24, app.fonts.Medium, white)
		textAt(hdc, fmt.Sprintf("×%d", g.Count), x+width-76, y+6, 54, 23, app.fonts.Small, muted)
		if g.Count > 1 {
			chevron := "+"
			if app.expandedGroup == r.group {
				chevron = "−"
			}
			textAt(hdc, chevron, x+26, y+5, 18, 22, app.fonts.Title, blue)
			if !app.busy {
				addHit(ir(x+18, y, 29, 34), "expand", r.group)
			}
		}
		if !app.busy {
			addHit(ir(x+48, y, width-66, 34), "group", r.group)
		}
	}
	foot := bottom - 72
	fill(hdc, ir(x+18, foot-7, width-36, 1), 0x354258)
	version := p.Headers["version"]
	textAt(hdc, "AUTHOR  "+strLimit(p.Headers["author"], 25)+"     •     CRC32  "+strLimit(p.ExpectedCRC(), 37), x+23, foot, width-46, 20, app.fonts.Small, mint)
	textAt(hdc, strLimit(p.Headers["notes"], 93), x+23, foot+24, width-46, 22, app.fonts.Small, muted)
	_ = version
}
func drawConvert(hdc uintptr, w, h int32) {
	drawPageHeader(hdc, "Convert & create", "Convert JMP, IPS, BPS, Xdelta and encrypted IPS using a stock XBE, or create a patch from two XBEs.", w)
	x := int32(265)
	width := w - 294
	panel(hdc, ir(x, 125, width, 620), surface, 19)
	textAt(hdc, "PATCH CONVERTER", x+28, 145, width-40, 29, app.fonts.Title, white)

	textAt(hdc, "01   Stock / original XBE  ·  optional for direct offset patches", x+31, 181, width-55, 20, app.fonts.Medium, muted)
	panel(hdc, ir(x+28, 208, width-226, 43), surface2, 12)
	label := "Choose your original XBE"
	if app.aFile != "" { label = filepath.Base(app.aFile) }
	textAt(hdc, label, x+44, 214, width-253, 31, app.fonts.Medium, white)
	button(hdc, "Choose stock XBE", ir(x+width-190, 209, 160, 40), "a", 0, false)

	textAt(hdc, "02   Source patch  ·  JMP / IPS / BPS / Xdelta / encrypted IPS", x+31, 259, width-50, 20, app.fonts.Medium, muted)
	panel(hdc, ir(x+28, 285, width-226, 43), surface2, 12)
	label = "Choose an existing patch to convert"
	if app.convertPatchFile != "" {
		label = filepath.Base(app.convertPatchFile)
	} else if app.bFile == "" && app.directFile != "" {
		label = filepath.Base(app.directFile) + "  (library)"
	} else if app.bFile == "" && app.parsed != nil {
		label = app.parsed.Filename + "  (library)"
	}
	textAt(hdc, label, x+44, 291, width-253, 31, app.fonts.Medium, white)
	button(hdc, "Choose patch", ir(x+width-190, 286, 160, 40), "cpatch", 0, false)

	textAt(hdc, "OR   Modified XBE  ·  alternative to a source patch", x+31, 336, width-60, 20, app.fonts.Medium, muted)
	panel(hdc, ir(x+28, 362, width-226, 43), surface2, 12)
	label = "Optional: choose a pre-patched XBE"
	if app.bFile != "" { label = filepath.Base(app.bFile) }
	textAt(hdc, label, x+44, 369, width-253, 29, app.fonts.Medium, white)
	button(hdc, "Choose modified", ir(x+width-190, 363, 160, 40), "b", 0, false)

	textAt(hdc, "OUTPUT PATCH FORMAT", x+30, 416, width-50, 20, app.fonts.Micro, muted)
	labels := []string{"JMP  ·  editable", "IPS  ·  classic", "Xdelta  ·  VCDIFF", "EIPS  ·  encrypted", "BPS  ·  CRC32-verified"}
	cell := (width-85)/2
	for i, s := range labels {
		bx := x + 28 + int32(i%2)*(cell+13)
		by := int32(439 + (i/2)*41)
		panel(hdc, ir(bx, by, cell, 35), map[bool]uint32{true: 0x32517B, false: surface2}[app.format == i], 10)
		mark(hdc, ir(bx+10, by+9, 16, 16), app.format == i)
		textAt(hdc, s, bx+34, by+4, cell-40, 28, app.fonts.Small, white)
		addHit(ir(bx, by, cell, 35), "format", i)
	}
	textAt(hdc, "Password (EIPS encryption / decryption; 8+ characters to create)", x+30, 562, width-55, 21, app.fonts.Micro, muted)
	panel(hdc, ir(x+28, 584, width-56, 37), surface2, 10)
	placeholder := "Password for encrypted IPS"
	if app.password != "" { placeholder = strings.Repeat("•", len([]rune(app.password))) }
	textAt(hdc, placeholder, x+44, 590, width-90, 25, app.fonts.Regular, white)
	if app.passFocus { fill(hdc, ir(x+28, 620, width-56, 2), blue) }
	addHit(ir(x+28, 584, width-56, 37), "password", 0)

	mark(hdc, ir(x+32, 631, 18, 18), app.ipsRLE)
	textAt(hdc, "Use IPS RLE encoding (for IPS and encrypted IPS)", x+60, 625, width-83, 28, app.fonts.Small, muted)
	addHit(ir(x+28, 625, width-56, 29), "ipsrle", 0)
    if app.conversionBlocked {
        panel(hdc, ir(x+width-277, 658, 244, 44), 0x3A4150, 12)
        textAt(hdc, "CONVERSION BLOCKED", x+width-259, 665, 230, 29, app.fonts.Medium, 0xABB3BF)
    } else {
        button(hdc, "CONVERT / CREATE PATCH", ir(x+width-277, 658, 244, 44), "convert", 0, true)
    }
	hint := app.conversionHint
	if hint == "" { hint = "Choose a source patch to determine whether the original XBE is required." }
	textAt(hdc, hint, x+30, 707, width-53, 25, app.fonts.Small, muted)
	_ = h
}

func drawSettings(hdc uintptr, w, h int32) {
	drawPageHeader(hdc, "Settings & storage", "Performance and verification controls. Saved locally; no account required.", w)
	x := int32(266)
	width := w - 298
	panel(hdc, ir(x, 129, width, 154), surface, 17)
	textAt(hdc, "PARALLEL FILE WORKERS", x+28, 153, width-55, 29, app.fonts.Title, white)
	textAt(hdc, "Saved automatically · Parallel workers for ISO copying and patch indexing (1–8).", x+29, 194, width-60, 24, app.fonts.Regular, muted)
	button(hdc, "−", ir(x+width-217, 208, 48, 48), "workers", -1, false)
	textAt(hdc, fmt.Sprintf("%d", app.workers), x+width-158, 210, 50, 45, app.fonts.Title, white)
	button(hdc, "+", ir(x+width-95, 208, 48, 48), "workers", 1, false)
	panel(hdc, ir(x, 302, width, 122), surface, 17)
	textAt(hdc, "VERIFICATION SAFEGUARDS", x+28, 319, width-55, 29, app.fonts.Title, white)
	textAt(hdc, "CRC32 and Title ID protection. Override resets OFF each launch.", x+29, 357, width-185, 28, app.fonts.Regular, muted)
	button(hdc, map[bool]string{true: "OVERRIDE ON", false: "ENFORCED"}[app.override], ir(x+width-180, 353, 152, 43), "override", 0, app.override)
	panel(hdc, ir(x, 443, width, 169), surface, 17)
	textAt(hdc, "OFFLINE PATCHES", x+28, 462, width-45, 29, app.fonts.Title, white)
	textAt(hdc, strLimit(app.root, 100), x+29, 499, width-57, 26, app.fonts.Small, muted)
	textAt(hdc, "Copy JMP files into Custom; they remain available without internet.", x+29, 527, width-55, 23, app.fonts.Regular, muted)
	button(hdc, "Open patch folder", ir(x+27, 561, 195, 40), "folder", 0, false)
	button(hdc, "Import single JMP", ir(x+235, 561, 188, 40), "import", 0, false)
	button(hdc, "Rescan", ir(x+438, 561, 112, 40), "refresh", 0, false)
	_ = h
}
func drawAbout(hdc uintptr, w, h int32) {
	drawPageHeader(hdc, "About Magic Patch Desk", "A native Windows workbench for Original Xbox game modifications.", w)
	x := int32(266)
	width := w - 299
	panel(hdc, ir(x, 130, width, 408), surface, 20)
	textAt(hdc, "NATIVE. PORTABLE. MADE FOR XBOX.", x+34, 155, width-65, 40, app.fonts.Title, white)
	lines := []string{"• Native Win32 application compiled with Go — no browser window or runtime installer.", "• Live GitHub collection checking, fully cached offline; add JMPs manually.", "• Original XBE inspection, Title IDs, regions and CRC32 comparison.", "• Group identical JMP comments, expand to toggle individual records or mixed groups.", "• XISO and Xbox full-disc ISO reading, with safe output copy and sector relocation.", "• Parallel file copying with bounded worker counts.", "• Dry-run patch preview, confirmation before writing; safe settings saved locally."}
	for i, v := range lines {
		textAt(hdc, v, x+35, 220+int32(i)*38, width-62, 33, app.fonts.Regular, muted)
	}
	textAt(hdc, "Important: recompiled and altered full-disc images cannot retain original disc signatures.", x+35, 503, width-67, 28, app.fonts.Small, amber)
	_ = h
}
