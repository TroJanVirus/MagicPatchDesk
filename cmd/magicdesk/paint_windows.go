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

var (
	bg uint32       = 0x0C111C
	sidebar uint32  = 0x111A29
	surface uint32  = 0x182335
	surface2 uint32 = 0x1F2D44
	muted uint32    = 0x91A3BD
	white uint32    = 0xF5F8FF
	blue uint32     = 0x6C9CFF
	mint uint32     = 0x5AD5BF
	amber uint32    = 0xFFC678
)

// GDI colours use COLORREF BGR. Preserve high-contrast text across presets.
func setTheme(name string) {
    switch name {
    case "Forest":
        bg,sidebar,surface,surface2=0x11201A,0x193027,0x264238,0x305447
        muted,white,blue,mint,amber=0xB1CDBF,0xF1FAF5,0xC5DA92,0x91E3BD,0xB9D8EF
    case "Plum":
        bg,sidebar,surface,surface2=0x1D1320,0x291B2D,0x3A2940,0x4C3652
        muted,white,blue,mint,amber=0xBCB0CA,0xFCF4FF,0xE8A6D6,0xC6D8FF,0xD6AFEA
    case "Light":
        // Cool paper neutrals, ink-blue text, restrained teal accents, and
        // warm amber for warnings. Colours are RGB; col() converts to GDI.
        bg,sidebar,surface,surface2=0xF6F8FA,0xEAF0F4,0xFFFFFF,0xEDF2F6
        muted,white,blue,mint,amber=0x506574,0x203645,0x0D687E,0x116D53,0x955C18
    default:
        bg,sidebar,surface,surface2=0x0C111C,0x111A29,0x182335,0x1F2D44
        muted,white,blue,mint,amber=0x91A3BD,0xF5F8FF,0x6C9CFF,0x5AD5BF,0xFFC678
    }
}

func cover(hdc uintptr, x, y int32, title, art string, artURLs []string) {
	h := uint32(2166136261)
	for _, ch := range title {
		h ^= uint32(ch)
		h *= 16777619
	}
	p := []uint32{0x244F66, 0x584273, 0x31555B, 0x5C4C37, 0x384D7A, 0x364F47}
	color := p[h%uint32(len(p))]
	panel(hdc, ir(x, y, 52, 52), color, 13)
	if art != "" && drawArt(hdc, art, x+2, y+2, 48, 48) {
		return
	}
	// A stale or corrupt locally imported cover must not hide valid artwork
	// already known from PatchHub or a matching Xbox Title ID.
	if len(artURLs) != 0 && drawArt(hdc, scheduleCoverCandidates(artURLs), x+2, y+2, 48, 48) {
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
    case 9:
        drawHome(hdc,w,h)
	case 0:
		drawLibrary(hdc, w, h)
        if app.galleryOpen{drawPatchGallery(hdc,w,h)}
        if app.notesOpen{drawFullPatchNotes(hdc,w,h)}
        if app.outputPickerOpen{drawLibraryOutputChooser(hdc,w,h)}
	case 1:
		drawConvert(hdc, w, h)
	case 2:
		drawSettings(hdc, w, h)
	case 3:
		drawDeveloperDashboard(hdc, w, h)
	case 4:
		drawAbout(hdc, w, h)
	case 5:
		drawRecent(hdc, w, h)
	case 6:
		drawMaintenance(hdc,w,h)
	case 7:
		drawGameOutputs(hdc,w,h)
    case 8:
        drawManager(hdc,w,h)
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
    statusColor:=uint32(muted)
    if strings.HasPrefix(status,"Duplicate patch found!") {statusColor=0xFF667C}
    if app.crashSetupWarning{
        status="CRASH LOGGING UNAVAILABLE · App data cannot be written; please check permissions"
        statusColor=0xFF667C
    }else if app.recoveredCrashes>0{
        status=fmt.Sprintf("%d unexpected previous exit(s) logged · Click here to review and report",app.recoveredCrashes)
        statusColor=0xFF667C
        addHit(ir(249,h-42,w-330,42),"crash-open-reports",0)
    }
    textAt(hdc,strLimit(status,155),273,h-35,w-340,24,app.fonts.Small,statusColor)
	textAt(hdc, "v0.14", w-67, h-35, 55, 24, app.fonts.Small, 0x7C92AD)
}
func drawSidebar(hdc uintptr,w,h int32){
    fill(hdc,ir(0,0,232,h),sidebar)
    panel(hdc,ir(21,28,42,42),0x3F73E6,12)
    txt(hdc,"M",ir(21,28,42,42),app.fonts.Title,white,DT_CENTER|DT_SINGLELINE|DT_VCENTER)
    textAt(hdc,"MAGIC",74,25,142,25,app.fonts.Title,white)
    textAt(hdc,"PATCH DESK",75,49,140,16,app.fonts.Micro,0x88A6CE)
    compactNav:=h<660
    navHeight:=int32(47)
    navStep:=int32(49)
    navStart:=int32(145)
    navCaptionY:=int32(114)
    if compactNav{
        // Fit all workspaces plus bottom-anchored Settings/About even in a
        // 500px-high client, without painting tab rows over the status bar.
        navHeight=32
        navStep=34
        navStart=109
        navCaptionY=82
    }
    textAt(hdc,"WORKSPACE",24,navCaptionY,165,22,app.fonts.Micro,0x637793)
    type navItem struct{label,symbol string;tab int}
    drawNav:=func(item navItem,y int32){
        selected:=app.tab==item.tab
        if selected{
            panel(hdc,ir(13,y,206,navHeight),0x283F65,12)
            fill(hdc,ir(13,y+5,4,navHeight-10),blue)
        }else if app.hoverAction=="tab"&&app.hoverIndex==item.tab{
            panel(hdc,ir(13,y,206,navHeight),0x223756,12)
        }
        symbolColor,textColor:=uint32(muted),uint32(muted)
        if selected{symbolColor=blue;textColor=white}
        txt(hdc,item.symbol,ir(25,y,27,navHeight),app.fonts.Title,symbolColor,DT_SINGLELINE|DT_VCENTER|DT_CENTER)
        textAt(hdc,item.label,61,y+(navHeight-30)/2,151,30,app.fonts.Medium,textColor)
        addHit(ir(10,y,211,navHeight),"tab",item.tab)
    }
    topNav:=[]navItem{
        {"Home","⌂",9},
        {"Patch Library","◈",0},
        {"Convert Patch","◇",1},
        {"Batch Process","▤",8},
        {"Diagnostics","◉",3},
        {"Recent","◷",5},
        {"Maintenance","⚒",6},
    }
    for i,item:=range topNav{drawNav(item,navStart+int32(i)*navStep)}
    // Keep Settings/About clear of the footer and top workspaces.
    dividerY:=h-156
    if dividerY>navStart+int32(len(topNav))*navStep+4{
        fill(hdc,ir(21,dividerY,191,1),0x354258)
    }
    drawNav(navItem{"Settings","⚙",2},h-147)
    drawNav(navItem{"About","ⓘ",4},h-96)
    // The optional collection widget must not cover any workspace.
    collectionTop:=h-274
    navEnd:=navStart+int32(len(topNav))*navStep
    if collectionTop>navEnd+12 && collectionTop+110<h-158 {
        panel(hdc,ir(17,collectionTop,198,110),0x1E2A3E,14)
        textAt(hdc,"PATCH COLLECTION",31,collectionTop+7,167,21,app.fonts.Micro,muted)
        app.mu.Lock()
        remote:=app.remoteCount
        count:=app.cat.PatchCount
        app.mu.Unlock()
        textAt(hdc,fmt.Sprintf("%d unique patches · %d source files",count,remote),31,collectionTop+34,170,24,app.fonts.Small,white)
        button(hdc,"↻ Sync GitHub",ir(25,collectionTop+70,184,32),"sync",0,false)
    }
}
func drawPageHeader(hdc uintptr,title,subtitle string,w int32){
    headlineWidth:=w-530
    if headlineWidth<120{headlineWidth=120}
    titleFont:=app.fonts.Hero
    if textPixelWidth(hdc,title,titleFont)>headlineWidth{
        titleFont=app.fonts.Title
    }
    textWrapped(hdc,title,263,27,headlineWidth,52,titleFont,white)
    textWrapped(hdc,subtitle,264,79,w-520,39,app.fonts.Small,muted)
}
func drawLibrary(hdc uintptr, w, h int32) {
    app.patchViewport=rect{}
    app.recordViewport=rect{}
    app.patchBar=listScrollBar{}
    app.recordBar=listScrollBar{}
	drawPageHeader(hdc, "Discover & patch", "Select a source and patch, then explicitly choose one or more verified output formats.", w)
	button(hdc, "Load patch file", ir(w-365, 38, 167, 42), "direct", 0, false)
	button(hdc, "+  Import JMP", ir(w-186, 38, 155, 42), "import", 0, false)
    listX:=int32(255)
    // Use the available width instead of trapping long game names inside an
    // unchanging 353px sidebar on wide screens.
    listW:=(w-275)*35/100
    if listW<300{listW=300}
    if listW>455{listW=455}
    listTop:=int32(117)
    listBottom:=h-178
    rightX:=listX+listW+19
    rightW:=w-rightX-23
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
		} else if app.hoverAction == "filter" && app.hoverIndex == idx {
			color = 0x34506F
		}
		panel(hdc, ir(chipX, listTop+123, chip.width, 25), color, 10)
		txt(hdc, chip.name, ir(chipX, listTop+123, chip.width, 25), app.fonts.Micro, white, DT_SINGLELINE|DT_VCENTER|DT_CENTER)
		addHit(ir(chipX, listTop+123, chip.width, 25), "filter", idx)
		chipX += chip.width + 5
	}
	gameStart := listTop + 160
	gameEnd := listBottom - 12
    gameRowH:=int32(79)
    perPage:=int((gameEnd-gameStart)/gameRowH)
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
        y:=gameStart+int32(row)*gameRowH
		selected := i == app.selectedGame
		if selected {
            panel(hdc,ir(listX+10,y,listW-20,gameRowH-5),0x2B4468,12)
		} else if app.hoverAction == "game" && app.hoverIndex == i {
            panel(hdc,ir(listX+10,y,listW-20,gameRowH-5),0x264666,12)
		} else if row%2 == 0 {
            panel(hdc,ir(listX+10,y,listW-20,gameRowH-5),0x1D2B40,11)
		}
        cover(hdc,listX+18,y+12,g.Title,g.Cover,core.CoverCandidates(g))
        // Real wrapping preserves long game titles rather than silently
        // truncating the title after 29 characters.
        textWrapped(hdc,g.Title,listX+80,y+5,listW-106,47,app.fonts.Medium,white)
        textAt(hdc,fmt.Sprintf("%d patch%s available",len(g.Patches),map[bool]string{true:"",false:"es"}[len(g.Patches)==1]),listX+80,y+52,listW-103,18,app.fonts.Small,muted)
        addHit(ir(listX+10,y,listW-20,gameRowH-5),"game",i)
	}
	if len(filtered) == 0 {
		textAt(hdc, "No matching games yet", listX+26, gameStart+21, listW-51, 27, app.fonts.Medium, muted)
		textAt(hdc, "Import local JMPs or sync GitHub.", listX+26, gameStart+55, listW-51, 26, app.fonts.Small, muted)
	}
	panel(hdc, ir(rightX, listTop, rightW, listBottom-listTop), surface, 17)
	if app.directFile != "" {
		textAt(hdc, "CUSTOM PATCH FILE", rightX+30, listTop+29, rightW-50, 34, app.fonts.Title, white)
		textAt(hdc, strLimit(filepath.Base(app.directFile), 72), rightX+30, listTop+77, rightW-60, 28, app.fonts.Medium, mint)
		textAt(hdc, "Supports JMP, XBMC4Gamers PF, IPS, BPS, Xdelta, EIPS and .mpdpack.", rightX+30, listTop+125, rightW-60, 27, app.fonts.Small, muted)
		textAt(hdc, "Choose an original XBE / ISO below, then Apply Patch.", rightX+30, listTop+160, rightW-60, 27, app.fonts.Regular, white)
		if strings.EqualFold(filepath.Ext(app.directFile), ".eips") || strings.EqualFold(filepath.Ext(app.directFile), ".mpdpack") {
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
    panel(hdc,ir(255,h-163,w-280,105),surface,16)
    textAt(hdc,"SOURCE GAME",271,h-155,152,19,app.fonts.Micro,muted)
    source:=chosenGameSource()
    display:="Choose an original XBE, XISO / ISO, CCI or extracted game folder"
    if source!=""{display=filepath.Base(source)}
    if app.inputInfo.Valid{
        display+=fmt.Sprintf("  ·  %s  ·  CRC %s",app.inputInfo.TitleID,app.inputInfo.CRC32)
    }
    if app.inputPartSummary!=""{display+="  ·  "+app.inputPartSummary}
    textAt(hdc,display,272,h-136,w-639,28,app.fonts.Small,white)
    button(hdc,"Open file",ir(w-318,h-151,89,32),"open",0,false)
    button(hdc,"Open folder",ir(w-221,h-151,101,32),"game-folder",0,false)
    button(hdc,"Preview",ir(w-111,h-151,83,32),"preview",0,false)
    outputStatus:="OUTPUTS: "+outputSummary()
    if source!=""&&gameOutputMask==0&&!gameOutputPreserve{
        outputStatus="OUTPUTS: Choose the formats to create from your game and patch"
    }
    // Two lines where needed: do not turn the requested instruction into
    // "OUTPUTS: Choose a source game" or squeeze it into a single short line.
    textWrapped(hdc,outputStatus,272,h-107,w-656,43,app.fonts.Small,muted)
    button(hdc,"CHOOSE OUTPUTS",ir(w-395,h-107,181,43),"output-open",0,false)
    if libraryApplyReady(){
        button(hdc,"APPLY PATCH",ir(w-205,h-107,177,43),"apply",0,true)
    }else{
        r:=ir(w-205,h-107,177,43)
        panel(hdc,r,0x333E52,12)
        txt(hdc,"APPLY PATCH",r,app.fonts.Medium,0x8B99AE,DT_CENTER|DT_SINGLELINE|DT_VCENTER)
        // No hit target: truly disabled until patch, game AND choice exist.
    }
}
func drawGameDetails(hdc uintptr, x, width, top, bottom int32, g core.Game) {
    textWrapped(hdc,g.Title,x+22,top+10,width-43,52,app.fonts.Title,white)
    textAt(hdc,fmt.Sprintf("%d patches · Cover: %s",len(g.Patches),coverStatus(g)),x+23,top+63,width-253,23,app.fonts.Small,muted)
    button(hdc,"Check cover",ir(x+width-224,top+58,108,28),"cover-check",0,false)
    button(hdc,"Copy details",ir(x+width-110,top+58,99,28),"cover-copy",0,false)
    listY:=top+91
    listRows:=3
    if bottom-top<485 {listRows=2} // reserve record space at 720p-height
    rowH:=int32(71)
    app.patchViewport=ir(x+16,listY,width-32,int32(listRows)*rowH)
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
		} else if app.hoverAction == "patch" && app.hoverIndex == idx {
			color = 0x2D496B
		}
		panel(hdc, ir(x+16, y, width-32, rowH-5), color, 11)
        tag(hdc,p.Type,x+26,y+19,0x3B5271)
		label := p.Name
		if len(g.Title) > 0 && strings.HasPrefix(strings.ToLower(label), strings.ToLower(g.Title)) {
			label = strings.TrimSpace(strings.TrimPrefix(label, label[:len(g.Title)]))
			label = strings.TrimLeft(label, " -·")
		}
		if label == "" {
			label = p.Type
		}
        textWrapped(hdc,label,x+149,y+5,width-179,40,app.fonts.Medium,white)
		detail := strings.TrimSpace(p.Region + "  ·  " + p.Author)
		if p.TitleID != "" {
			detail += "  ·  " + p.TitleID
		}
		if p.RAM != "" {
			detail += "  ·  " + p.RAM
		}
        if count:=len(galleryForPatch(p));count>0{detail+=fmt.Sprintf("  ·  %d patch images",count)}
        textAt(hdc,detail,x+150,y+47,width-181,19,app.fonts.Small,muted)

		addHit(ir(x+16, y, width-32, rowH-5), "patch", idx)
	}
    app.patchBar=drawScrollBar(hdc,ir(x+width-29,listY,11,int32(listRows)*rowH-5),
        app.patchScroll,len(g.Patches),listRows,"patch")
	p := app.parsed
	groupY := listY + rowH*int32(listRows) + 14
	fill(hdc, ir(x+18, groupY-5, width-36, 1), 0x354258)
	if p == nil {
		txt(hdc, "Choose a patch above to review and toggle its records.", ir(x+25, groupY+16, width-51, 55), app.fonts.Regular, muted, DT_WORDBREAK)
		return
	}
    textAt(hdc,fmt.Sprintf("PATCH RECORD GROUPS  ·  %d RECORDS",len(p.Records)),x+23,groupY,width-199,23,app.fonts.Micro,muted)
    if app.selectedPatch>=0&&app.selectedPatch<len(g.Patches){
        chosen:=g.Patches[app.selectedPatch]
        shots:=galleryForPatch(chosen)
        if len(shots)>0 {
            button(hdc,fmt.Sprintf("View images (%d)",len(shots)),ir(x+width-168,groupY-1,150,30),"gallery-open",0,false)
        }else if mediaDiscoveryRunning(chosen){
            textAt(hdc,"Finding images…",x+width-164,groupY+2,145,24,app.fonts.Small,muted)
        }else{
            if reason:=mediaDiscoveryReason(chosen);reason!="" {
                button(hdc,"Why no images?",ir(x+width-168,groupY-1,150,30),"gallery-explain",0,false)
            }else{
                button(hdc,"Find images",ir(x+width-168,groupY-1,150,30),"gallery-discover",0,false)
            }
        }
    }
	textAt(hdc, "Toggle a group · + expands records · Read shows full comments and links.", x+23, groupY+24, width-43, 20, app.fonts.Small, 0x91A3BD)
	rowY := groupY + 56
    rowHeight:=int32(60) // two-line group preview plus a separate full comment action
    visible:=int((bottom-rowY-82)/rowHeight)
	if visible < 1 {visible=1}
    app.recordViewport=ir(x+18,rowY,width-36,int32(visible)*rowHeight)
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
        y:=rowY+int32(i)*rowHeight
		if r.record >= 0 {
			rec := p.Records[r.record]
			color := uint32(0x1F2F45)
			if app.hoverAction == "record" && app.hoverIndex == r.record { color = 0x2B4260 }
			panel(hdc,ir(x+34,y,width-52,rowHeight-4),color,8)
			mark(hdc,ir(x+47,y+18,18,18),rec.Enabled)
			label := fmt.Sprintf("#%d  %s", r.record+1, rec.Kind)
            if target, targetErr := p.RecordTarget(rec); targetErr == nil && target != "default.xbe" {
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
			textWrapped(hdc,label,x+78,y+5,width-126,rowHeight-12,app.fonts.Small,muted)
			if !app.busy {
				addHit(ir(x+34,y,width-52,rowHeight-4),"record",r.record)
			}
			continue
		}
		g := p.Groups[r.group]
		groupColor := uint32(0x24344B)
		if app.hoverAction == "group" && app.hoverIndex == r.group {groupColor = 0x325071}
		panel(hdc,ir(x+18,y,width-36,rowHeight-4),groupColor,9)
		mark(hdc,ir(x+47,y+18,18,18),g.Enabled)
		if g.Mixed {
			textAt(hdc,"–",x+50,y+18,14,21,app.fonts.Micro,white)
		}
		// The parser keeps the FULL group comment. Show two readable lines here;
        // the separate Read control opens a scrollable uncropped view.
        preview:=core.FormatJMPNotes(g.Name)
        textWrapped(hdc,preview,x+78,y+6,width-257,rowHeight-12,app.fonts.Medium,white)
		textAt(hdc,fmt.Sprintf("×%d",g.Count),x+width-77,y+17,52,22,app.fonts.Small,muted)
		if g.Count > 1 {
			chevron := "+"
			if app.expandedGroup == r.group {
				chevron = "−"
			}
			textAt(hdc,chevron,x+26,y+17,18,22,app.fonts.Title,blue)
			if !app.busy {
				addHit(ir(x+18,y,29,rowHeight-4),"expand",r.group)
			}
		}
		if !app.busy {
			addHit(ir(x+48,y,width-66,rowHeight-4),"group",r.group)
            button(hdc,"Read",ir(x+width-165,y+10,77,35),"group-details",r.group,false)
		}
	}
    app.recordBar=drawScrollBar(hdc,ir(x+width-29,rowY,11,int32(visible)*rowHeight-3),
        app.groupScroll,len(rows),visible,"records")
	foot := bottom - 72
	fill(hdc, ir(x+18, foot-7, width-36, 1), 0x354258)
	crcLabel := "CRC32 "+p.ExpectedCRC()
    crcColor := mint
    if len(p.CRCs)>0 && app.inputInfo.CRC32!="" {
        match:=false
        for _,expected:=range p.CRCs {
            if fmt.Sprintf("%08X",expected)==strings.ToUpper(app.inputInfo.CRC32) {match=true;break}
        }
        if !match {
            crcLabel="CRC32 MISMATCH! expected "+p.ExpectedCRC()+" / loaded "+strings.ToUpper(app.inputInfo.CRC32)
            crcColor=0xB52F43 // Accessible warning red (RGB; col() converts to GDI)
        } else {
            crcLabel="CRC32 MATCH · "+app.inputInfo.CRC32
        }
    }
    textAt(hdc, "AUTHOR "+strLimit(p.Headers["author"], 20)+"  •  "+strLimit(crcLabel, 80), x+23, foot, width-46, 20, app.fonts.Small, crcColor)
    if p.Headers["notes"]!="" {
        button(hdc,"READ FULL NOTES · formatted + scrollable",ir(x+18,foot+24,width-36,32),"notes-open",0,false)
    }else{
        textAt(hdc,"No JMP notes were provided.",x+23,foot+26,width-46,22,app.fonts.Small,muted)
    }
}
func drawConvert(hdc uintptr, w, h int32) {
	drawPageHeader(hdc, "Convert & create", "Convert JMP, IPS, BPS, Xdelta and encrypted IPS using a stock XBE, or create a patch from two XBEs.", w)
	x := int32(265)
	width := w - 294
	panel(hdc, ir(x, 125, width, 620), surface, 19)
	textAt(hdc, "PATCH CONVERTER", x+28, 145, width-40, 29, app.fonts.Title, white)

    // Start with what the user is converting. The original XBE is step 02:
    // optional for some patch formats and required for byte-level checks.
    textAt(hdc,"01   Source patch  ·  JMP / IPS / BPS / Xdelta / encrypted IPS",x+31,181,width-50,20,app.fonts.Medium,muted)
    panel(hdc,ir(x+28,208,width-226,43),surface2,12)
    label:="Choose an existing patch to convert"
    if app.convertPatchFile!="" {
        label=filepath.Base(app.convertPatchFile)
    }else if app.bFile==""&&app.directFile!="" {
        label=filepath.Base(app.directFile)+"  (library)"
    }else if app.bFile==""&&app.parsed!=nil {
        label=app.parsed.Filename+"  (library)"
    }
    textAt(hdc,label,x+44,214,width-253,31,app.fonts.Medium,white)
    button(hdc,"Choose patch",ir(x+width-190,209,160,40),"cpatch",0,false)

    textAt(hdc,"02   Stock / original XBE  ·  source and verification",x+31,259,width-55,20,app.fonts.Medium,muted)
    panel(hdc,ir(x+28,285,width-226,43),surface2,12)
    label="Choose your original XBE"
    if app.aFile!="" {label=filepath.Base(app.aFile)}
    textAt(hdc,label,x+44,291,width-253,31,app.fonts.Medium,white)
    button(hdc,"Choose stock XBE",ir(x+width-190,286,160,40),"a",0,false)

	textAt(hdc, "OR   Modified XBE  ·  alternative to a source patch", x+31, 336, width-60, 20, app.fonts.Medium, muted)
	panel(hdc, ir(x+28, 362, width-226, 43), surface2, 12)
	label = "Optional: choose a pre-patched XBE"
	if app.bFile != "" { label = filepath.Base(app.bFile) }
	textAt(hdc, label, x+44, 369, width-253, 29, app.fonts.Medium, white)
	button(hdc, "Choose modified", ir(x+width-190, 363, 160, 40), "b", 0, false)

	textAt(hdc, "OUTPUT PATCH FORMAT", x+30, 416, width-50, 20, app.fonts.Micro, muted)
	labels := []string{"JMP  ·  editable", "IPS  ·  classic", "Xdelta  ·  VCDIFF", "EIPS  ·  encrypted", "BPS  ·  CRC32-verified", "ALL FORMATS  ·  PACK"}
	cell := (width-85)/2
	for i, s := range labels {
		bx := x + 28 + int32(i%2)*(cell+13)
		by := int32(439 + (i/2)*41)
		panel(hdc, ir(bx, by, cell, 35), map[bool]uint32{true: 0x32517B, false: surface2}[app.format == i], 10)
		mark(hdc, ir(bx+10, by+9, 16, 16), app.format == i)
		textAt(hdc, s, bx+34, by+4, cell-40, 28, app.fonts.Small, white)
		addHit(ir(bx, by, cell, 35), "format", i)
	}
    // Plain IPS only uses optional RLE; passwords are exclusively for
    // EIPS, all-formats packs containing EIPS, or an encrypted IPS source.
    showPassword:=app.format==3||app.format==5||
        strings.EqualFold(filepath.Ext(app.convertPatchFile),".eips")||
        strings.EqualFold(filepath.Ext(app.directFile),".eips")
    showRLE:=app.format==1||app.format==3||app.format==5
    if showPassword{
        textAt(hdc,"Encrypted IPS password (required for EIPS; optional for pack)",x+30,562,width-55,21,app.fonts.Micro,muted)
        panel(hdc,ir(x+28,584,width-56,37),surface2,10)
        placeholder:="Password for encrypted IPS"
        if app.password!="" {placeholder=strings.Repeat("•",len([]rune(app.password)))}
        textAt(hdc,placeholder,x+44,590,width-90,25,app.fonts.Regular,white)
        if app.passFocus{fill(hdc,ir(x+28,620,width-56,2),blue)}
        addHit(ir(x+28,584,width-56,37),"password",0)
    }
    if showRLE{
        posY:=int32(625)
        if !showPassword{posY=584}
        mark(hdc,ir(x+32,posY+6,18,18),app.ipsRLE)
        textAt(hdc,"Use IPS run-length encoding (RLE)",x+60,posY,width-83,28,app.fonts.Small,muted)
        addHit(ir(x+28,posY,width-56,31),"ipsrle",0)
    }
	if app.packMode {
		button(hdc, "LOCATE TARGET ORIGINALS", ir(x+28, 658, 230, 44), "packsources", 0, false)
	}
    if app.conversionBlocked {
        blockedButton:=ir(x+width-277,658,244,44)
        panel(hdc,blockedButton,0x3A4150,12)
        textAt(hdc,"CONVERSION BLOCKED",x+width-259,665,230,29,app.fonts.Medium,0xABB3BF)
        // Still block conversion; clicking the disabled-looking area explains
        // the exact search/replace or BPS checksum requirement in a dialog.
        addHit(blockedButton,"convert-reason",0)
    }else{
        label:="CONVERT / CREATE PATCH"
        if app.packMode {label="CREATE PATCH PACK"}
        button(hdc,label,ir(x+width-277,658,244,44),"convert",0,true)
    }
    hint:=app.conversionHint
    if hint=="" {hint="Choose a patch to see whether a stock/original XBE is required."}
    if app.conversionBlocked{
        textWrapped(hdc,hint,x+30,697,width-53,47,app.fonts.Small,amber)
    }else{
        textWrapped(hdc,hint,x+30,700,width-53,28,app.fonts.Small,muted)
        textAt(hdc,"Optional modified XBE: compare both results byte-for-byte.",x+30,728,width-53,18,app.fonts.Small,muted)
    }
	_ = h
}

func drawSettings(hdc uintptr, w, h int32) {
    drawPageHeader(hdc,"Settings & diagnostics",
        "Appearance, performance, verification, offline patches and local reports.",w)
    x:=int32(266)
    width:=w-298

    // Dedicated, clearly differentiated appearance section. A preview shows
    // background, surface, accent and ink colours for every theme.
    themeY:=int32(125)
    panel(hdc,ir(x,themeY,width,148),surface,16)
    textAt(hdc,"APPEARANCE",x+24,themeY+9,width-48,27,app.fonts.Title,white)
    textAt(hdc,"Choose a colour theme · saved automatically",x+24,themeY+36,width-48,20,app.fonts.Small,muted)
    themes:=[]struct{name string; background,panel,accent,ink uint32}{
        {"Midnight",0x0C111C,0x182335,0x6C9CFF,0xF5F8FF},
        {"Forest",0x11201A,0x264238,0x91E3BD,0xF1FAF5},
        {"Plum",0x1D1320,0x3A2940,0xE8A6D6,0xFCF4FF},
        {"Light",0xF6F8FA,0xFFFFFF,0x0D687E,0x203645},
    }
    tileW:=(width-66)/4
    if tileW<75{tileW=75}
    for i,t:=range themes {
        tx:=x+24+int32(i)*(tileW+6)
        selected:=app.theme==t.name
        bgCard:=surface2
        if selected {bgCard=0xD5E8EF; if app.theme!="Light"{bgCard=0x2A4161}}
        panel(hdc,ir(tx,themeY+66,tileW,67),bgCard,12)
        if selected{fill(hdc,ir(tx+10,themeY+128,tileW-20,3),blue)}
        rawSwatch(hdc,ir(tx+12,themeY+75,26,26),t.background)
        rawSwatch(hdc,ir(tx+27,themeY+80,22,22),t.panel)
        rawSwatch(hdc,ir(tx+41,themeY+75,19,19),t.accent)
        rawSwatch(hdc,ir(tx+51,themeY+91,12,12),t.ink)
        textAt(hdc,t.name,tx+10,themeY+107,tileW-18,20,app.fonts.Small,white)
        addHit(ir(tx,themeY+66,tileW,67),"theme",i)
    }

    gap:=int32(13)
    leftW:=(width-gap)/2
    rightX:=x+leftW+gap
    rightW:=width-leftW-gap
    y:=int32(285)

    panel(hdc,ir(x,y,leftW,108),surface,15)
    textAt(hdc,"PARALLEL FILE WORKERS",x+22,y+10,leftW-43,27,app.fonts.Title,white)
    textAt(hdc,"ISO copying and patch indexing (1–8).",x+22,y+39,leftW-43,21,app.fonts.Small,muted)
    button(hdc,"−",ir(x+leftW-175,y+68,44,32),"workers",-1,false)
    textAt(hdc,fmt.Sprintf("%d",app.workers),x+leftW-117,y+69,38,30,app.fonts.Title,white)
    button(hdc,"+",ir(x+leftW-65,y+68,43,32),"workers",1,false)

    y=405
    panel(hdc,ir(x,y,leftW,103),surface,15)
    textAt(hdc,"VERIFICATION SAFEGUARDS",x+22,y+9,leftW-44,26,app.fonts.Title,white)
    textAt(hdc,"Enforce the stock XBE's expected CRC32.",x+22,y+37,leftW-44,20,app.fonts.Small,muted)
    button(hdc,map[bool]string{true:"OVERRIDE ON",false:"ENFORCED"}[app.override],
        ir(x+leftW-177,y+64,155,33),"override",0,app.override)

    y=520
    panel(hdc,ir(x,y,leftW,141),surface,15)
    textAt(hdc,"OFFLINE PATCHES",x+22,y+9,leftW-44,27,app.fonts.Title,white)
    textAt(hdc,strLimit(app.root,64),x+22,y+38,leftW-44,22,app.fonts.Small,muted)
    button(hdc,fmt.Sprintf("Duplicates: %d · View",len(app.cat.Duplicates)),
        ir(x+21,y+69,leftW-42,28),"duplicate-audit",0,false)
    cell:=(leftW-60)/3
    for i,c:=range []struct{label,action string}{{"Patch folder","folder"},{"Import JMP / PF","import"},{"Rescan","refresh"}}{
        button(hdc,c.label,ir(x+19+int32(i)*(cell+9),y+105,cell,30),c.action,0,false)
    }

    y=285
    panel(hdc,ir(rightX,y,rightW,376),surface,15)
    textAt(hdc,"PROBLEM REPORT GROUPS",rightX+21,y+11,rightW-42,27,app.fonts.Title,white)
    app.mu.Lock()
    groupCount:=len(app.diagnosticGroups)
    groupIndex:=app.diagnosticSelected
    var g core.DiagnosticGroup
    hasGroup:=groupIndex>=0&&groupIndex<groupCount
    if hasGroup{g=app.diagnosticGroups[groupIndex]}
    app.mu.Unlock()
    headline:="No reports. They remain local until you share them."
    detail:=""
    if hasGroup{
        headline=fmt.Sprintf("Group %d/%d · %s / %s",groupIndex+1,groupCount,g.ErrorCode,g.Operation)
        detail=fmt.Sprintf("%s · Title ID %s · %d reports, %d warnings",g.ID,g.TitleID,g.Occurrences,g.Warnings)
    }
    textAt(hdc,strLimit(headline,83),rightX+21,y+51,rightW-42,24,app.fonts.Small,white)
    textAt(hdc,strLimit(detail,95),rightX+21,y+79,rightW-42,24,app.fonts.Small,muted)
    viewLabel:="View group"
    if hasGroup&&g.Latest.CrashEvidenceID!=""{viewLabel="View crash log"}
    buttons:=[][2]string{{"Previous","diagnostics-prev"},{"Next","diagnostics-next"},
        {viewLabel,"diagnostics-view"},{"Copy summary","diagnostics-copy"},
        {"Find matches","diagnostics-find"},{"Report issue","diagnostics-github"}}
    btnW:=(rightW-61)/2
    for i,c:=range buttons {
        bx:=rightX+21+int32(i%2)*(btnW+12)
        by:=y+128+int32(i/2)*54
        button(hdc,c[0],ir(bx,by,btnW,39),c[1],0,i==5)
    }
    textAt(hdc,"Private evidence and crash logs",rightX+21,y+292,rightW-42,20,app.fonts.Small,muted)
    button(hdc,"Crash logs",ir(rightX+21,y+318,btnW,34),"diagnostics-crashfolder",0,false)
    button(hdc,"Reports folder",ir(rightX+33+btnW,y+318,btnW,34),"diagnostics-folder",0,false)
    _=h
}
// drawGameOutputs scales its vertical layout to fit a typical 720p desktop
// instead of hiding the Create button under the bottom status bar.
func drawGameOutputs(hdc uintptr, w, h int32) {
    drawPageHeader(hdc, "Game image outputs", "Choose exactly which game files and image formats to generate.", w)
    if h<610 {
        textAt(hdc,"Increase the window height to show the complete Game Outputs workspace.",265,148,w-308,44,app.fonts.Regular,amber)
        return
    }
    x:=int32(265)
    width:=w-294
    factor:=float64(1)
    if h<785 {factor=float64(h-180)/624.0}
    y:=func(original int32)int32{return 125+int32(float64(original-125)*factor)}
    hh:=func(original int32)int32 {
        if factor>=1 {return original}
        scaled:=int32(float64(original)*factor)
        if scaled<26{return 26}
        return scaled
    }
    panelHeight:=int32(624)
    if h<785 {panelHeight=h-180}
    panel(hdc,ir(x,125,width,panelHeight),surface,19)
    textAt(hdc,"01   GAME SOURCE",x+31,y(150),width-55,hh(25),app.fonts.Micro,muted)
    source:=chosenGameSource()
    description:="Use a loaded game, choose an image, or locate its extracted folder"
    if source!="" {description=source}
    panel(hdc,ir(x+28,y(179),width-56,hh(43)),surface2,12)
    textAt(hdc,strLimit(description,92),x+42,y(186),width-84,hh(28),app.fonts.Medium,white)
    third:=(width-81)/3
    button(hdc,"Choose image/XBE",ir(x+28,y(232),third,hh(38)),"game-source",0,false)
    button(hdc,"Choose extracted folder",ir(x+40+third,y(232),third,hh(38)),"game-folder",0,false)
    button(hdc,"Use loaded game",ir(x+52+2*third,y(232),third,hh(38)),"game-use-loaded",0,false)
    textAt(hdc,"02   GAME OUTPUT FORMATS  ·  SELECT ONE OR MORE",x+31,y(297),width-55,hh(27),app.fonts.Micro,muted)
    labels:=[]string{"Patched game XBE only","Extracted game folder","Game-partition XISO","Cerbios compressed CCI","Preserve full-disc ISO"}
    cell:=(width-84)/2
    for i,v:=range labels {
        cx:=x+28+int32(i%2)*(cell+15)
        cy:=y(int32(330+(i/2)*59))
        cellHeight:=hh(49)
        active:=gameOutputMask&(uint32(1)<<uint(i))!=0
        fillColor:=uint32(surface2)
        if active{fillColor=0x2E4E78}
        panel(hdc,ir(cx,cy,cell,cellHeight),fillColor,12)
        mark(hdc,ir(cx+13,cy+(cellHeight-18)/2,18,18),active)
        textAt(hdc,v,cx+43,cy+3,cell-49,cellHeight-6,app.fonts.Medium,white)
        addHit(ir(cx,cy,cell,cellHeight),"game-format",i)
    }
    textAt(hdc,"03   PATCH SOURCE  ·  OPTIONAL",x+31,y(518),width-55,hh(28),app.fonts.Micro,muted)
    patch:="No patch: convert/extract original game without modification"
    if app.directFile!="" {patch=filepath.Base(app.directFile)
    }else if app.parsed!=nil {patch=app.parsed.Filename}
    if gameOutputIgnorePatch{patch="Conversion only — selected patch will NOT be applied"}
    panel(hdc,ir(x+28,y(545),width-56,hh(44)),surface2,11)
    textAt(hdc,strLimit(patch,90),x+43,y(552),width-230,hh(28),app.fonts.Regular,white)
    patchAction:="Skip patch"
    if gameOutputIgnorePatch{patchAction="Include patch"}
    button(hdc,patchAction,ir(x+width-181,y(549),142,hh(35)),"game-patch-toggle",0,false)
    textAt(hdc,"Patch Library selects records; use Skip patch for clean image conversions.",x+32,y(604),width-60,hh(25),app.fonts.Small,muted)
    textAt(hdc,"XISO is NOT a full-disc ISO. All outputs are verified before publication.",x+32,y(630),width-60,hh(25),app.fonts.Small,muted)
    button(hdc,"CREATE SELECTED OUTPUTS",ir(x+width-304,y(674),270,hh(47)),"game-export",0,true)
    textAt(hdc,outputNames(gameOutputMask),x+31,y(685),width-345,hh(26),app.fonts.Small,mint)
}

func drawAbout(hdc uintptr, w, h int32) {
    drawPageHeader(hdc, "About Magic Patch Desk", "Native Windows Original Xbox patching and image conversion.", w)
    x:=int32(266)
    width:=w-299
    panel(hdc,ir(x,130,width,605),surface,20)
    textAt(hdc,"NATIVE. PORTABLE. MADE FOR XBOX.",x+34,149,width-65,39,app.fonts.Title,white)
    lines:=[]string{
        "Independent native Go/XDVDFS/CCI implementation — no bundled external tooling.",
        "Games and original executables remain on your computer; originals are preserved.",
        "Image outputs: extracted game folder, game XBE, XISO, CCI and safe full-disc ISO.",
        "Patch support: JMP, IPS, BPS, Xdelta, EIPS, MPD pack and Rocky5 PF.",
    }
    for i,v:=range lines {textAt(hdc,v,x+35,207+int32(i)*34,width-62,29,app.fonts.Regular,muted)}
    textAt(hdc,"REFERENCES AND ACKNOWLEDGEMENTS",x+35,370,width-60,32,app.fonts.Title,white)
    refs:=[]string{
        "Team Resurgent / Repackinator (GPL-3.0): Xbox disc-image workflow reference",
        "Team Cerbios; Team Resurgent / libCCI; purelogiccode / CCISharp: CCI format",
        "Rocky5 / XBMC4Gamers (GPL-3.0): PF patch format and workflow",
        "JayYardley / Xbox Magic Patches and PatchHub: JMP catalogue/source",
    }
    for i,v:=range refs{textAt(hdc,v,x+35,417+int32(i)*37,width-63,30,app.fonts.Regular,muted)}
    textAt(hdc,"References are credited, not bundled. No sources have been copied.",x+35,579,width-67,28,app.fonts.Small,amber)
    button(hdc,"VIEW FULL CREDITS + SOURCE LINKS",ir(x+35,635,323,45),"game-credits",0,false)
    _=h
}
