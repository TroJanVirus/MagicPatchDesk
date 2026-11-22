//go:build windows

//go:generate python ../../scripts/build_windows_icon.py

package main

import (
	"context"
    "errors"
	"fmt"
	"magicpatchdesk/core"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"
)

const (
	WM_DESTROY          = 0x0002
	WM_PAINT            = 0x000F
	WM_ERASEBKGND       = 0x0014
	WM_SIZE             = 0x0005
	WM_CLOSE            = 0x0010
	WM_LBUTTONUP        = 0x0202
	WM_LBUTTONDOWN      = 0x0201
	WM_MOUSEMOVE        = 0x0200
	WM_MOUSELEAVE       = 0x02A3
	WM_TIMER            = 0x0113
	TME_LEAVE           = 0x00000002
	IDC_HAND            = 32649
	WM_MOUSEWHEEL       = 0x020A
	WM_CHAR             = 0x0102
	WM_KEYDOWN          = 0x0100
	WM_DROPFILES        = 0x0233
	WM_APP              = 0x8000
	WM_SETCURSOR        = 0x0020
	WS_OVERLAPPEDWINDOW = 0x00CF0000
	WS_VISIBLE          = 0x10000000
	DT_SINGLELINE       = 0x20
	DT_VCENTER          = 0x4
	DT_CENTER           = 0x1
	DT_END_ELLIPSIS     = 0x8000
	DT_WORDBREAK        = 0x10
	DT_NOPREFIX         = 0x800 // Display literal ampersands in headings and labels
	TRANSPARENT         = 1
	IDC_ARROW           = 32512
	SW_SHOW             = 5
	OFN_PATHMUSTEXIST   = 0x800
	OFN_FILEMUSTEXIST   = 0x1000
	OFN_OVERWRITEPROMPT = 0x2
	OFN_EXPLORER        = 0x80000
	SM_CXSCREEN         = 0
	SM_CYSCREEN         = 1
)

type point struct{ X, Y int32 }
type trackMouseEvent struct {
	Size, Flags uint32
	HWND        uintptr
	HoverTime   uint32
}
type rect struct{ L, T, R, B int32 }
type paintStruct struct {
	HDC       uintptr
	Erase     int32
	Paint     rect
	Restore   int32
	IncUpdate int32
	Reserved  [32]byte
}
type wndClass struct {
	Size       uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   uintptr
	Icon       uintptr
	Cursor     uintptr
	Background uintptr
	Menu       *uint16
	Name       *uint16
	SmallIcon  uintptr
}
type msg struct {
	HWND    uintptr
	Message uint32
	Pad     uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      point
	Private uint32
}
type openName struct {
	Size          uint32
	Owner         uintptr
	Instance      uintptr
	Filter        *uint16
	CustomFilter  *uint16
	MaxCustom     uint32
	FilterIndex   uint32
	File          *uint16
	MaxFile       uint32
	FileTitle     *uint16
	MaxTitle      uint32
	InitialDir    *uint16
	Title         *uint16
	Flags         uint32
	FileOffset    uint16
	FileExtension uint16
	DefExt        *uint16
	CustData      uintptr
	Hook          uintptr
	Template      *uint16
	Reserved      uintptr
	DwReserved    uint32
	FlagsEx       uint32
}

var (
	user32                 = syscall.NewLazyDLL("user32.dll")
	gdi32                  = syscall.NewLazyDLL("gdi32.dll")
	k32                    = syscall.NewLazyDLL("kernel32.dll")
	dwm                    = syscall.NewLazyDLL("dwmapi.dll")
	shell32                = syscall.NewLazyDLL("shell32.dll")
	dlg32                  = syscall.NewLazyDLL("comdlg32.dll")
	regClass               = user32.NewProc("RegisterClassExW")
	createWindow           = user32.NewProc("CreateWindowExW")
	defWindow              = user32.NewProc("DefWindowProcW")
	destroyWindow          = user32.NewProc("DestroyWindow")
	showWindow             = user32.NewProc("ShowWindow")
	updateWindow           = user32.NewProc("UpdateWindow")
	getMessage             = user32.NewProc("GetMessageW")
	translateMsg           = user32.NewProc("TranslateMessage")
	dispatchMsg            = user32.NewProc("DispatchMessageW")
	postQuit               = user32.NewProc("PostQuitMessage")
	postMsg                = user32.NewProc("PostMessageW")
	trackMouse             = user32.NewProc("TrackMouseEvent")
	setTimer               = user32.NewProc("SetTimer")
	killTimer              = user32.NewProc("KillTimer")
	setCursor              = user32.NewProc("SetCursor")
	setCapture             = user32.NewProc("SetCapture")
	releaseCapture         = user32.NewProc("ReleaseCapture")
	invalidateRect         = user32.NewProc("InvalidateRect")
	beginPaint             = user32.NewProc("BeginPaint")
	endPaint               = user32.NewProc("EndPaint")
	getClientRect          = user32.NewProc("GetClientRect")
	drawText               = user32.NewProc("DrawTextW")
    getTextExtent           = gdi32.NewProc("GetTextExtentPoint32W")
	fillRect               = user32.NewProc("FillRect")
	setBkMode              = gdi32.NewProc("SetBkMode")
	setTextColor           = gdi32.NewProc("SetTextColor")
	createSolidBrush       = gdi32.NewProc("CreateSolidBrush")
	createCompatibleDC     = gdi32.NewProc("CreateCompatibleDC")
	createCompatibleBitmap = gdi32.NewProc("CreateCompatibleBitmap")
	deleteDC               = gdi32.NewProc("DeleteDC")
	bitBlt                 = gdi32.NewProc("BitBlt")
	createPen              = gdi32.NewProc("CreatePen")
	selectObject           = gdi32.NewProc("SelectObject")
	deleteObject           = gdi32.NewProc("DeleteObject")
	roundRect              = gdi32.NewProc("RoundRect")
	createFont             = gdi32.NewProc("CreateFontW")
	loadCursor             = user32.NewProc("LoadCursorW")
	getModule              = k32.NewProc("GetModuleHandleW")
	getOpenName            = dlg32.NewProc("GetOpenFileNameW")
	getSaveName            = dlg32.NewProc("GetSaveFileNameW")
	dragAccept             = shell32.NewProc("DragAcceptFiles")
	dragQuery              = shell32.NewProc("DragQueryFileW")
	dragFinish             = shell32.NewProc("DragFinish")
	shellExecute           = shell32.NewProc("ShellExecuteW")
	dwmAttr                = dwm.NewProc("DwmSetWindowAttribute")
	setDPI                 = user32.NewProc("SetProcessDPIAware")
)

func ptr(s string) *uint16           { return syscall.StringToUTF16Ptr(s) }
func col(hex uint32) uintptr         { return uintptr((hex&0xFF)<<16 | (hex & 0xFF00) | ((hex >> 16) & 0xFF)) }
func ir(x, y, w, h int32) rect       { return rect{x, y, x + w, y + h} }
func inside(r rect, x, y int32) bool { return x >= r.L && x < r.R && y >= r.T && y < r.B }
func lo(v uintptr) int32             { return int32(int16(uint16(v & 0xFFFF))) }
func hi(v uintptr) int32             { return int32(int16(uint16((v >> 16) & 0xFFFF))) }
func strLimit(v string, max int) string {
	r := []rune(v)
	if len(r) > max {
		return string(r[:max-1]) + "…"
	}
	return v
}

type hit struct {
	R      rect
	Action string
	Index  int
}
type fonts struct{ Regular, Medium, Title, Hero, Small, Micro, ButtonCompact, ButtonTiny uintptr }
type appState struct {
	mu            sync.Mutex
	hwnd          uintptr
	root          string
	cat           core.Catalog
	selectedGame  int
	selectedPatch int
    galleryOpen bool
    galleryPage int
	expandedGroup int
	parsed        *core.Patch
	input         string
	inputInfo     core.XBEInfo
	inputErr      string
	search        string
	searchFocus   bool
	tab           int
	gameScroll    int
	filter        int
	patchScroll   int
	groupScroll   int
	workers       int
    theme         string
	override      bool
	status        string
	syncStatus    string
	remoteCount   int
	busy          bool
	progress      float64
	aFile, bFile  string
	convertPatchFile string
	conversionHint string
	conversionBlocked bool
	packMode bool
	packPlan core.PatchPackPlan
	packSources map[string]string
	ipsRLE bool
	directFile string
    format        int
    convertFormatChosen bool // keep a deliberately chosen output across patch selections
	password      string
	passFocus     bool
	hits          []hit
	w, h          int32
	fonts         fonts
	lastErr       string
    recoveredCrashes int
    crashSetupWarning bool
    outputPickerOpen bool
    notesOpen bool
    notesScroll int
    mouseY int32
    scrollDragStartY int32
    scrollDragStartValue int
    patchViewport rect
    recordViewport rect
    patchBar listScrollBar
    recordBar listScrollBar
    notesBar listScrollBar
	recentFiles []core.RecentFile
	recentSelected int
	recentOffset int
	diagnosticGroups  []core.DiagnosticGroup
	diagnosticSelected int
	dashboardIssues []core.DashboardIssue
	dashboardCases []core.DeveloperCase
	dashboardBusy bool
	dashboardStatus string
	dashboardFetchedAt time.Time
	dashboardIncomplete bool
	dashboardSort core.DeveloperSort
	dashboardSelected int
	dashboardScroll int
	hoverAction string
	hoverIndex int
	hoverFade uint8
	pressedAction string
	pressedIndex int
	mouseDown bool
	mouseTracked bool
}

var app appState

func sysCall(p *syscall.LazyProc, a ...uintptr) uintptr { r, _, _ := p.Call(a...); return r }
func font(size int32, weight int32) uintptr {
	return sysCall(createFont, uintptr(int64(-size)), 0, 0, 0, uintptr(weight), 0, 0, 0, 1, 0, 0, 0, 5, uintptr(unsafe.Pointer(ptr("Segoe UI"))))
}
func addHit(r rect, act string, index int) { app.hits = append(app.hits, hit{r, act, index}) }
func fill(hdc uintptr, r rect, c uint32) {
    c=themeSurfaceColor(c)
	b := sysCall(createSolidBrush, col(c))
	sysCall(fillRect, hdc, uintptr(unsafe.Pointer(&r)), b)
	sysCall(deleteObject, b)
}
func panel(hdc uintptr, r rect, color uint32, radius int32) {
    color=themeSurfaceColor(color)
	brush := sysCall(createSolidBrush, col(color))
	pen := sysCall(createPen, 0, 1, col(color))
	o1 := sysCall(selectObject, hdc, brush)
	o2 := sysCall(selectObject, hdc, pen)
	sysCall(roundRect, hdc, uintptr(r.L), uintptr(r.T), uintptr(r.R), uintptr(r.B), uintptr(radius), uintptr(radius))
	sysCall(selectObject, hdc, o1)
	sysCall(selectObject, hdc, o2)
	sysCall(deleteObject, brush)
	sysCall(deleteObject, pen)
}
func txt(hdc uintptr, s string, r rect, size uintptr, color uint32, flags uint32) {
    txtRaw(hdc,s,r,size,themeTextColor(color),flags)
}
func txtRaw(hdc uintptr, s string, r rect, size uintptr, color uint32, flags uint32) {
	if s == "" {
		return
	}
	sysCall(setBkMode, hdc, TRANSPARENT)
	sysCall(setTextColor, hdc, col(color))
	old := sysCall(selectObject, hdc, size)
	utf := syscall.StringToUTF16(s)
	sysCall(drawText, hdc, uintptr(unsafe.Pointer(&utf[0])), uintptr(len(utf)-1), uintptr(unsafe.Pointer(&r)), uintptr(flags|DT_NOPREFIX))
	sysCall(selectObject, hdc, old)
}
func textAt(hdc uintptr, s string, x, y, w, h int32, f uintptr, color uint32) {
    txt(hdc,s,ir(x,y,w,h),f,color,DT_SINGLELINE|DT_VCENTER|DT_END_ELLIPSIS)
}

// Use two lines for user-facing descriptions with enough vertical room.
// The rectangle limits the text so it cannot paint over adjacent controls.
func textWrapped(hdc uintptr,s string,x,y,w,h int32,f uintptr,color uint32){
    if w<=0||h<=0{return}
    txt(hdc,s,ir(x,y,w,h),f,color,DT_WORDBREAK|DT_END_ELLIPSIS)
}

type measuredSize struct{ Width,Height int32 }
func textPixelWidth(hdc uintptr,label string,fontHandle uintptr) int32 {
    if label==""||hdc==0||fontHandle==0{return 0}
    utf:=syscall.StringToUTF16(label)
    if len(utf)<2{return 0}
    old:=sysCall(selectObject,hdc,fontHandle)
    var size measuredSize
    ok:=sysCall(getTextExtent,hdc,uintptr(unsafe.Pointer(&utf[0])),uintptr(len(utf)-1),uintptr(unsafe.Pointer(&size)))
    sysCall(selectObject,hdc,old)
    if ok==0{return 0}
    return size.Width
}

// Native GDI hover and press feedback: the background eases between states,
// a soft shadow appears on hover, and pressed buttons move one pixel.
func blendHex(from, to uint32, strength uint8) uint32 {
	t := uint32(strength)
	result := uint32(0)
	for shift := uint(0); shift <= 16; shift += 8 {
		a, b := (from>>shift)&0xFF, (to>>shift)&0xFF
		result |= ((a*(255-t) + b*t) / 255) << shift
	}
	return result
}
func button(hdc uintptr, label string, r rect, action string, idx int, main bool) {
	hovered := app.hoverAction == action && app.hoverIndex == idx
	pressed := hovered && app.mouseDown && app.pressedAction == action && app.pressedIndex == idx
	strength := uint8(0)
	if hovered {
		strength = app.hoverFade
	}
	color := uint32(0x283247)
	light := uint32(0x405A84)
    if app.theme=="Light" {
        color,light=0xE5EDF2,0xCFDFE8
        if main {color,light=0x11617A,0x147B90}
    }else if main {
		color = 0x447FF5
		light = 0x72A7FF
	}
	color = blendHex(color, light, strength)
	if pressed {
		color = blendHex(color, 0x10233E, 120)
	}
	if strength > 30 {
		panel(hdc, ir(r.L-1, r.T+2, r.R-r.L+2, r.B-r.T+2), blendHex(0x111A29, 0x244776, strength/3), 14)
	}
	dy := int32(0)
	if pressed {
		dy = 2
	}
	rr := ir(r.L, r.T+dy, r.R-r.L, r.B-r.T-dy)
	panel(hdc, rr, color, 14)
	if strength > 20 && !pressed {
		fill(hdc, ir(rr.L+12, rr.B-3, rr.R-rr.L-24, 2), blendHex(color, 0xFFFFFF, strength/3))
	}
	buttonText:=uint32(0xEDF3FF)
    if app.theme=="Light" {
        buttonText=0x183646
        if main {buttonText=0xFFFFFF}
    }
    // Measure actual Segoe UI widths. Fit the whole label with a slightly
    // smaller caption font before resorting to multi-line layout.
    inset:=int32(10)
    available:=rr.R-rr.L-inset*2
    selectedFont:=app.fonts.Medium
    if available>0&&textPixelWidth(hdc,label,selectedFont)>available {
        selectedFont=app.fonts.ButtonCompact
    }
    if available>0&&textPixelWidth(hdc,label,selectedFont)>available {
        selectedFont=app.fonts.ButtonTiny
    }
    flags:=uint32(DT_SINGLELINE|DT_VCENTER|DT_CENTER|DT_END_ELLIPSIS)
    textRect:=ir(rr.L+inset,rr.T+dy/2,rr.R-rr.L-2*inset,rr.B-rr.T)
    if available>0&&textPixelWidth(hdc,label,selectedFont)>available&&rr.B-rr.T>=40 {
        flags=DT_WORDBREAK|DT_CENTER|DT_END_ELLIPSIS
        textRect=ir(rr.L+inset,rr.T+3,rr.R-rr.L-2*inset,rr.B-rr.T-6)
    }
    txtRaw(hdc,label,textRect,selectedFont,buttonText,flags)
	addHit(r, action, idx)
}

func mouseHit(x, y int32) (hit, bool) {
	for i := len(app.hits) - 1; i >= 0; i-- {
		if inside(app.hits[i].R, x, y) {
			return app.hits[i], true
		}
	}
	return hit{}, false
}
func hoverAt(hwnd uintptr, x, y int32) {
	h, found := mouseHit(x, y)
	action := ""
	index := 0
	if found {
		action = h.Action
		index = h.Index
	}
	if app.hoverAction != action || app.hoverIndex != index {
		app.hoverAction, app.hoverIndex = action, index
		sysCall(setTimer, hwnd, 1, 16, 0)
		redraw()
	}
	if !app.mouseTracked {
		ev := trackMouseEvent{Size: uint32(unsafe.Sizeof(trackMouseEvent{})), Flags: TME_LEAVE, HWND: hwnd}
		sysCall(trackMouse, uintptr(unsafe.Pointer(&ev)))
		app.mouseTracked = true
	}
	cursor := uintptr(IDC_ARROW)
	if found {
		cursor = IDC_HAND
	}
	sysCall(setCursor, sysCall(loadCursor, 0, cursor))
}

// The old 7px-per-character estimate clipped the "Widescreen" badge
// because Segoe UI glyph widths are not constant. Measure with real GDI.
func tagWidth(hdc uintptr,label string)int32{
    wd:=textPixelWidth(hdc,label,app.fonts.Micro)+18
    if wd<48{wd=48}
    if wd>190{wd=190}
    return wd
}
func tag(hdc uintptr,label string,x,y int32,bg uint32){
    wd:=tagWidth(hdc,label)
    panel(hdc,ir(x,y,wd,23),bg,11)
    txt(hdc,label,ir(x+7,y,wd-14,23),app.fonts.Micro,0xE5EEFF,DT_SINGLELINE|DT_VCENTER|DT_CENTER|DT_END_ELLIPSIS)
}
func mark(hdc uintptr, r rect, active bool) {
	panel(hdc, r, 0x2D3951, 8)
	if active {
		panel(hdc, ir(r.L+4, r.T+4, r.R-r.L-8, r.B-r.T-8), 0x6D9CFF, 5)
	}
}
func snapshot() (core.Catalog, *core.Patch) {
	app.mu.Lock()
	defer app.mu.Unlock()
	return app.cat, app.parsed
}
func redraw() {
	if app.hwnd != 0 {
		sysCall(invalidateRect, app.hwnd, 0, 0)
	}
}
func notify(s string) {
	reportNotifyError(s)
	app.mu.Lock()
	app.status = s
	app.mu.Unlock()
	sysCall(postMsg, app.hwnd, WM_APP+1, 0, 0)
}
func initApp() {
    defer func(){
        if p:=recover();p!=nil{captureFatalPanic(p)}
        endCrashProtection()
    }()
	runtime.LockOSThread()
	app.workers = 4
    app.theme="Midnight"
	app.ipsRLE = true
	app.expandedGroup = -1
	app.selectedGame = -1
	app.selectedPatch = -1
    app.root, _ = core.GetLibraryDir()
    if !smokeWindowsUI {
        recovered,e:=beginCrashProtection(app.root)
        if e==nil{app.recoveredCrashes=recovered}else{app.crashSetupWarning=true}
    }
	loadMaintenanceSettings(app.root)
    initGitHubConnectionState()
	refreshRecentFiles()
	refreshDiagnosticGroups()
	if prefs, err := core.LoadPreferences(app.root); err == nil {
		app.workers, app.filter = prefs.Workers, prefs.Filter
        app.theme=prefs.Theme
        if prefs.GameOutputFormats!=nil {gameOutputFormats=prefs.GameOutputFormats}
	}
	setTheme(app.theme)
	app.status = "Loading local patch library…"
	app.syncStatus = "Connecting to GitHub…"
	initArtwork()
	defer closeArtwork()
	app.fonts = fonts{Regular: font(17, 400), Medium: font(17, 600), Title: font(22, 700), Hero: font(30, 700), Small: font(14, 400), Micro: font(13, 600), ButtonCompact: font(15, 600), ButtonTiny: font(13, 600)}
	sysCall(setDPI)
	instance := sysCall(getModule, 0)
	name := ptr("MagicPatchDeskNative")
    // Resource group ID 1 is the transparent user-provided JMP logo linked
    // into the EXE by scripts/build_windows_icon.py. Windows uses the same
    // embedded icon for the taskbar, window caption and Explorer shortcut.
    appIcon:=sysCall(user32.NewProc("LoadIconW"),instance,1)
	class := wndClass{Size: uint32(unsafe.Sizeof(wndClass{})), WndProc: syscall.NewCallback(windowProc), Instance: instance, Icon:appIcon, SmallIcon:appIcon, Cursor: sysCall(loadCursor, 0, IDC_ARROW), Name: name}
	if sysCall(regClass, uintptr(unsafe.Pointer(&class))) == 0 {
		panic("RegisterClassEx failed")
	}
	app.hwnd = sysCall(createWindow, 0, uintptr(unsafe.Pointer(name)), uintptr(unsafe.Pointer(ptr("Magic Patch Desk | Powered by Patch Hub"))), WS_OVERLAPPEDWINDOW|WS_VISIBLE, 100, 80, 1320, 840, 0, 0, instance, 0)
	if app.hwnd == 0 {
		panic("CreateWindowEx failed")
	}
	updateWindowsTitleBarTheme()
	sysCall(dragAccept, app.hwnd, 1)
	sysCall(showWindow, app.hwnd, SW_SHOW)
    if smokeWindowsUI {
        smokePaintAndExit()
        return
    }
    // The single Check at Start preference now governs BOTH the application
    // update check and public GitHub/PatchHub library refresh. Offline users
    // still see the cached catalogue and can run either check manually.
    if maintenancePref.NoAutoCheck {go reloadCatalog()}else{checkForNewRelease(true)}
	if maintenancePref.AutoReports && isGitHubConnected() {go syncStoredReports()}
	sysCall(updateWindow, app.hwnd)
    markCrashStage("ui_callback")
	var m msg
	for {
		v := sysCall(getMessage, uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(v) <= 0 {
			break
		}
		sysCall(translateMsg, uintptr(unsafe.Pointer(&m)))
		sysCall(dispatchMsg, uintptr(unsafe.Pointer(&m)))
	}
}
var smokeWindowsUI bool

func main() {
    if len(os.Args)==2 && os.Args[1]=="--mpd-ui-smoke-v1" {
        smokeWindowsUI=true
    }
    if helperMode(){return}
    defer catchFatalPanic()
    initApp()
}
// File scans and GitHub sync run on workers. Never swap the catalogue or
// selection from a worker while Win32 is drawing a patch list. A partially
// updated slice/string header can make native GDI dereference invalid memory.
var catalogueReady = make(chan core.Catalog,4)
var autoMatchReady = make(chan core.XBEInfo,4)

func publishCatalogue(c core.Catalog){
    select{
    case catalogueReady<-c:
    default:
        select{case <-catalogueReady:default:}
        select{case catalogueReady<-c:default:}
    }
    if app.hwnd!=0 {sysCall(postMsg,app.hwnd,WM_APP+2,0,0)}
}

func commitCatalogueOnUI(c core.Catalog){
    app.mu.Lock()
    prevTitle,prevFile:="",""
    if app.selectedGame>=0&&app.selectedGame<len(app.cat.Games){
        prevTitle=app.cat.Games[app.selectedGame].Title
        if app.selectedPatch>=0&&app.selectedPatch<len(app.cat.Games[app.selectedGame].Patches){
            prevFile=app.cat.Games[app.selectedGame].Patches[app.selectedPatch].File
        }
    }
    app.cat=c
    app.selectedGame=-1
    app.selectedPatch=-1
    app.parsed=nil
    if prevTitle!=""{
        for i,g:=range c.Games{
            if strings.EqualFold(g.Title,prevTitle){
                app.selectedGame=i
                for j,p:=range g.Patches{
                    if p.File==prevFile{app.selectedPatch=j;break}
                }
                break
            }
        }
    }
    info:=app.inputInfo
    selected:=app.selectedPatch
    app.mu.Unlock()
    if info.Valid && prevTitle==""{
        autoSelectXBE(info)
    }else if prevFile!=""&&selected>=0{
        selectPatch(selected)
    }
    redraw()
}

func reloadCatalog(){
    app.mu.Lock()
    workers,root:=app.workers,app.root
    app.mu.Unlock()
    c,err:=core.LoadCatalog(root,workers)
    if err!=nil{notify(err.Error());return}
    publishCatalogue(c)
}

func queueAutoMatch(info core.XBEInfo){
    select{
    case autoMatchReady<-info:
    default:
        select{case <-autoMatchReady:default:}
        select{case autoMatchReady<-info:default:}
    }
    if app.hwnd!=0{sysCall(postMsg,app.hwnd,WM_APP+3,0,0)}
}

// Match by the embedded XBE Title ID and its whole-file CRC32 when possible.
// Suggest a patch, never apply it automatically.
func autoSelectXBE(info core.XBEInfo) {
    markCrashStage("auto_match")
    defer markCrashStage("ui_callback")
	app.mu.Lock()
	catalog := app.cat
	app.mu.Unlock()
	bestGame, bestPatch, bestScore := -1, -1, 0
	for i, g := range catalog.Games {
		for k, cp := range g.Patches {
			score := 0
			if cp.TitleID != "" && strings.EqualFold(cp.TitleID, info.TitleID) {
				score = 6
			} else if info.Title != "" && strings.EqualFold(strings.TrimSpace(g.Title), strings.TrimSpace(info.Title)) {
				score = 2
			}
			if score > 0 && cp.CRC != "No CRC32 required" && strings.Contains(strings.ToUpper(cp.CRC), info.CRC32) {
				score += 20
			}
			if score > bestScore {
				bestGame, bestPatch, bestScore = i, k, score
			}
		}
	}
	if bestScore == 0 {
		return
	}
	app.mu.Lock()
	app.selectedGame = bestGame
	app.patchScroll = 0
	app.selectedPatch = -1
	app.parsed = nil
	app.mu.Unlock()
	if bestScore >= 20 {
		selectPatch(bestPatch)
		notify("Verified patch match found: " + catalog.Games[bestGame].Patches[bestPatch].Name)
	} else {
		notify(fmt.Sprintf("Found %d possible patch(es) for %s — choose one and verify CRC / region.", len(catalog.Games[bestGame].Patches), info.Title))
	}
}

func reloadAndSync() {
	reloadCatalog()
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	res, err := core.SyncGithub(ctx, app.root, notify)
	app.mu.Lock()
	if err != nil {
		app.syncStatus = "Offline · saved patches available"
		app.status = "GitHub check failed; offline library remains usable: " + err.Error()
	} else {
		app.remoteCount = res.RemoteCount
		app.syncStatus = fmt.Sprintf("GitHub: %d current JMP patches", res.RemoteCount)
		app.status = res.Message
	}
	app.mu.Unlock()
	if err == nil && res.Updated {
		reloadCatalog()
	}
	redraw()
}

// Patch Hub's public web index supplies optional game cover art and all known
// title listings. The JMP collection still syncs independently from GitHub.
func syncPatchHubMetadata() {
	ctx, cancel := context.WithTimeout(context.Background(), 32*time.Second)
	defer cancel()
	count, err := core.SyncPatchHub(ctx, app.root)
	if err != nil {
		return
	} // A network failure never blocks offline usage.
	reloadCatalog()
	notify(fmt.Sprintf("Patch Hub index ready: %d game titles · JMPs from GitHub / Custom", count))
}

func pickFile(save bool, filter, title, defaultName string) string {
	name := make([]uint16, 32768)
	if save && defaultName != "" {
		copy(name, syscall.StringToUTF16(defaultName))
	}
	f := append(utf16.Encode([]rune(filter)), 0)
	fn := openName{Size: uint32(unsafe.Sizeof(openName{})), Owner: app.hwnd, Filter: &f[0], File: &name[0], MaxFile: uint32(len(name)), Title: ptr(title), Flags: OFN_EXPLORER | OFN_PATHMUSTEXIST}
	if save {
		fn.Flags |= OFN_OVERWRITEPROMPT
	} else {
		fn.Flags |= OFN_FILEMUSTEXIST
	}
	p := getOpenName
	if save {
		p = getSaveName
	}
	if sysCall(p, uintptr(unsafe.Pointer(&fn))) == 0 {
		return ""
	}
	return syscall.UTF16ToString(name)
}
func openInput(path string) {
    selectGameSource(path)
    markCrashStage("read_xbe")
    if core.CCISourcePath(path) {
        reader,err:=core.OpenCCI(path)
        if err!=nil {notify("Cannot load CCI: "+err.Error());return}
        reader.Close()
        app.mu.Lock()
        app.input=path
        app.inputErr="CCI source ready; XBE is checked during preview"
        app.inputInfo=core.XBEInfo{}
        app.status="CCI loaded; select patch and preview"
        app.mu.Unlock()
        rememberRecent("input",path)
        redraw()
        return
    }

	app.input = path
	app.inputErr = "Reading executable…"
	app.inputInfo = core.XBEInfo{}
	redraw()
	go func() {
        defer catchFatalPanic()
		var data []byte
		var err error
		if strings.EqualFold(filepath.Ext(path), ".xbe") {
			data, err = os.ReadFile(path)
		} else {
			var im *core.Image
			im, err = core.OpenXISO(path)
			if err == nil {
				var entry core.ImageFile
				entry, err = im.DefaultXBE()
				if err == nil {
					data, err = im.ReadFile(entry)
				}
			}
		}
		var info core.XBEInfo
		if err == nil {
			info, err = core.InspectXBE(data)
		}
		app.mu.Lock()
        if app.input!=path {app.mu.Unlock();return} // drop a stale asynchronous read
		app.inputInfo = info
		if err != nil {
			app.inputErr = err.Error()
		} else {
			app.inputErr = ""
			app.status = "Loaded: " + info.Title + " · " + info.TitleID + " · CRC32 " + info.CRC32
		}
		app.mu.Unlock()
		if err == nil {
            // Another input may have been chosen while this I/O completed.
            app.mu.Lock();same:=app.input==path;app.mu.Unlock()
            if !same{return}
			rememberRecent("input",path)
			queueAutoMatch(info)
		}
		redraw()
	}()
}
func selectPatch(idx int) {
    markCrashStage("parse_patch")
    defer markCrashStage("ui_callback")
	app.mu.Lock()
	if app.selectedGame < 0 || app.selectedGame >= len(app.cat.Games) || idx < 0 || idx >= len(app.cat.Games[app.selectedGame].Patches) {
		app.mu.Unlock()
		return
	}
	path := app.cat.Games[app.selectedGame].Patches[idx].File
	app.selectedPatch = idx
	app.groupScroll = 0
	app.expandedGroup = -1
	app.mu.Unlock()
	b, err := os.ReadFile(path)
	if err != nil {
		notify(err.Error())
		return
	}
	if strings.EqualFold(filepath.Ext(path), ".pf") {
        if _, err := core.ParsePF(filepath.Base(path),b);err!=nil{notify(err.Error());return}
        app.mu.Lock()
        app.directFile=path
        app.parsed=nil
        app.mu.Unlock()
        notify("XBMC4Gamers PF selected: "+filepath.Base(path)+" · Choose game's default.xbe or ISO")
        discoverSelectedPatchMedia(false)
        redraw()
        return
    }
	p, err := core.ParseJMP(filepath.Base(path), b)
	if err != nil {
		notify(err.Error())
		return
	}
	app.mu.Lock()
	app.parsed = p
	app.mu.Unlock()
    discoverSelectedPatchMedia(false)
	redraw()
}

// atomicWrite only replaces a destination after the complete file has been written
// and flushed. On errors, the original destination remains untouched.
func atomicWrite(dest string, data []byte) error {
	f, err := os.CreateTemp(filepath.Dir(dest), ".magicdesk-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), dest)
}

func atomicPatchISO(source, dest string, apply func([]byte) ([]byte, error), workers int, progress func(int64, int64)) error {
	f, err := os.CreateTemp(filepath.Dir(dest), ".magicdesk-*.iso")
	if err != nil {
		return err
	}
	intermediate := f.Name()
	f.Close()
	os.Remove(intermediate)
	defer os.Remove(intermediate)
	if err = core.PatchImageWithTransform(source, intermediate, apply, workers, progress); err != nil {
		return err
	}
	return os.Rename(intermediate, dest)
}

// Preferences intentionally exclude the CRC/Title-ID bypass: it resets OFF at launch.
func savePreferences() {
	if err := core.SavePreferences(app.root, core.Preferences{Workers: app.workers, Filter: app.filter, Theme: app.theme, GameOutputFormats: gameOutputFormats}); err != nil {
		notify("Could not save interface preferences: " + err.Error())
	}
}

var messageBox = user32.NewProc("MessageBoxW")

// The preview is a dry run on the embedded XBE only, never a disc-image copy.
func previewTarget(source string, selected *core.Patch, direct, password string, override bool) (core.PatchPreview, string, error) {
	var input []byte
	var err error
	xbeName := filepath.Base(source)
	var im *core.Image
	if strings.EqualFold(filepath.Ext(source), ".xbe") {
		input, err = os.ReadFile(source)
	} else {
		im, err = core.OpenXISO(source)
		if err == nil {
			var entry core.ImageFile
			entry, err = im.DefaultXBE()
			if err == nil {
				xbeName = entry.Path
				input, err = im.ReadFile(entry)
			}
		}
	}
	if err != nil {
		return core.PatchPreview{}, "", err
	}
	if len(input) > 512<<20 {
		return core.PatchPreview{}, "", fmt.Errorf("XBE larger than 512 MiB")
	}
	if _, err = core.InspectXBE(input); err != nil {
		return core.PatchPreview{}, "", err
	}
	var result core.PatchPreview
	if direct == "" {
		if selected == nil {
			return core.PatchPreview{}, "", fmt.Errorf("no patch selected")
		}
		result, err = selected.Preview(input, override)
	} else if strings.EqualFold(filepath.Ext(direct), ".jmp") {
		var data []byte
		data, err = os.ReadFile(direct)
		if err == nil {
			var patch *core.Patch
			patch, err = core.ParseJMP(filepath.Base(direct), data)
			if err == nil {
				result, err = patch.Preview(input, override)
			}
		}
	} else {
		var out []byte
		out, err = core.ApplyPatchFile(input, direct, password, override)
		if err == nil {
			result = core.PreviewDifference(input, out)
		}
	}
	if err != nil {
		return core.PatchPreview{}, "", err
	}
	if im != nil {
		entry, e := im.DefaultXBE()
		if e != nil {
			return core.PatchPreview{}, "", e
		}
		oldSectors := (int64(entry.Size) + 2047) / 2048
		newSectors := (result.PatchedSize + 2047) / 2048
		if newSectors > oldSectors && im.Base != 0 {
			return core.PatchPreview{}, "", fmt.Errorf("full-disc ISO has insufficient XBE sectors; repack to game-partition XISO before patching")
		}
	}
	return result, xbeName, nil
}

func previewMessage(p core.PatchPreview, filename string) string {
	summary := fmt.Sprintf("Target: %s\n\nOriginal XBE: %d bytes  (CRC32 %s)\nPatched XBE: %d bytes  (CRC32 %s)\nChanged existing bytes: %d\nAppended bytes: %d\nEnabled JMP records: %d", filename, p.OriginalSize, p.OriginalCRC32, p.PatchedSize, p.PatchedCRC32, p.ModifiedBytes, p.AppendedBytes, p.SelectedRecords)
	if len(p.Records) > 0 {
		summary += "\n\nFirst selected records:"
		for i, r := range p.Records {
			if i >= 8 {
				summary += fmt.Sprintf("\n... and %d more", len(p.Records)-8)
				break
			}
			summary += fmt.Sprintf("\n%d. %s (%s) at 0x%X · %d bytes", r.Number, strLimit(r.Group, 33), r.Kind, r.Offset, r.Size)
		}
	}
	if p.PatchedSize == p.OriginalSize && p.ModifiedBytes == 0 {
		summary += "\n\nWARNING: The selected patch makes no byte changes."
	}
	return summary
}

func showPreviewMessage(body string, confirmation bool) bool {
	flags := uintptr(0x40) // MB_ICONINFORMATION
	if confirmation {
		body += "\n\nApply this patch and save the output copy?"
		flags = 0x24 | 0x100 // MB_YESNO|MB_ICONQUESTION|MB_DEFBUTTON2
	}
	title := "Patch preview · Magic Patch Desk"
	if confirmation {
		title = "Confirm patch · Magic Patch Desk"
	}
	result := sysCall(messageBox, app.hwnd, uintptr(unsafe.Pointer(ptr(body))), uintptr(unsafe.Pointer(ptr(title))), flags)
	return !confirmation || result == 6 // IDYES
}

func startPreview() {
	app.mu.Lock()
	if app.busy {
		app.mu.Unlock()
		return
	}
	selected, source, direct, password, override := app.parsed, app.input, app.directFile, app.password, app.override
	if source == "" || (selected == nil && direct == "") {
		app.mu.Unlock()
		notify("Choose an XBE/ISO and a patch before previewing.")
		return
	}
	app.mu.Unlock()
    if handled,e:=maybeStartCCIPatch(true,selected,direct,source,password,app.workers,override);handled {if e!=nil{notify(e.Error())};return}
    if handled,e:=maybeStartPF(true,direct,source,app.workers);handled {
        if e!=nil{recordFailure("pf.preview.preflight",e,source,direct,"");notify(e.Error())};return
    }
    if handled, e := maybeStartPack(true, direct, source, app.workers, password); handled {
        if e != nil { notify(e.Error()) }
        return
    }
    if handled, e := maybeStartTargeted(true, selected, direct, source, app.workers, override); handled {
        if e != nil { notify(e.Error()) }
        return
    }
    app.mu.Lock()
	app.busy = true
	app.status = "Analysing patch without writing anything…"
	app.mu.Unlock()
	redraw()
	go func() {
		report, xbeName, err := previewTarget(source, selected, direct, password, override)
		if err == nil {
			showPreviewMessage(previewMessage(report, xbeName), false)
		}
		app.mu.Lock()
		app.busy = false
		if err != nil {
			app.status = "Preview failed: " + err.Error()
		} else {
			app.status = "Dry-run preview complete. No files modified."
		}
		app.mu.Unlock()
		if err != nil {recordFailure("patch.preview",err,source,direct,"")}
		redraw()
	}()
}

func startPatch() {
	app.mu.Lock()
	if app.busy {
		app.mu.Unlock()
		return
	}
	selected := app.parsed
	source := app.input
	direct := app.directFile
	workers := app.workers
	override := app.override
	password := app.password
	app.mu.Unlock()
	if selected == nil && direct == "" {
		notify("Select a library patch or load a custom patch file first.")
		return
	}
	if source == "" {
		notify("Choose an original XBE or Xbox ISO first.")
		return
	}
    if handled,e:=maybeStartCCIPatch(false,selected,direct,source,password,workers,override);handled {if e!=nil{notify(e.Error())};return}
    if handled,e:=maybeStartPF(false,direct,source,workers);handled {
        if e!=nil{recordFailure("pf.apply.preflight",e,source,direct,"");notify(e.Error())};return
    }
    if handled, e := maybeStartPack(false, direct, source, workers, password); handled {
        if e != nil { notify(e.Error()) }
        return
    }
    if handled, e := maybeStartTargeted(false, selected, direct, source, workers, override); handled {
        if e != nil { notify(e.Error()) }
        return
    }
	ext := strings.ToLower(filepath.Ext(source))
	base := strings.TrimSuffix(filepath.Base(source), filepath.Ext(source))
	dest := pickFile(true, "Xbox executable or disc image\x00*.xbe;*.iso\x00All files\x00*.*\x00", "Save patched copy", base+"_patched"+ext)
	if dest == "" {
		return
	}
	if strings.EqualFold(filepath.Clean(source), filepath.Clean(dest)) {
		notify("The output must not overwrite your original file.")
		return
	}
	var apply func([]byte) ([]byte, error)
	if direct != "" {
		apply = func(in []byte) ([]byte, error) { return core.ApplyPatchFile(in, direct, password, override) }
	} else {
		apply = func(in []byte) ([]byte, error) { return selected.Apply(in, override) }
	}
	app.mu.Lock()
	app.busy = true
	app.progress = 0
	app.status = "Applying patch…"
	app.mu.Unlock()
	redraw()
	go func() {
		var err error
		var report core.PatchPreview
		var target string
		report, target, err = previewTarget(source, selected, direct, password, override)
		if err == nil && !showPreviewMessage(previewMessage(report, target), true) {
			app.mu.Lock()
			app.busy = false
			app.status = "Patch cancelled after preview; no files modified."
			app.mu.Unlock()
			redraw()
			return
		}
		if err == nil && ext == ".xbe" {
			var b []byte
			b, err = os.ReadFile(source)
			if err == nil {
				var output []byte
				output, err = apply(b)
				if err == nil {
					err = atomicWrite(dest, output)
				}
			}
		} else if err == nil {
			err = atomicPatchISO(source, dest, apply, workers, func(done, total int64) {
				if total > 0 {
					app.mu.Lock()
					app.progress = float64(done) / float64(total)
					app.mu.Unlock()
					if done == total || done%(128<<20) < 4<<20 {
						redraw()
					}
				}
			})
		}
		app.mu.Lock()
		app.busy = false
		if err != nil {
			app.status = "Patch failed: " + err.Error()
			app.lastErr = err.Error()
		} else {
			app.status = "SUCCESS · Saved patched copy: " + dest
			app.progress = 1
		}
		app.mu.Unlock()
        if err==nil {
            manager.log.Add("pass","Patch","Patched copy saved; original input preserved")
            rememberRecent("output",dest)
        }else{
            manager.log.Add("error","Patch","Patch preview or write failed: "+err.Error())
            recordFailure("patch.apply",err,source,direct,ext)
        }
		redraw()
	}()
}

func convertFiles() { runConversion() }

func handle(action string, index int) {
	switch action {
	case "maint-install", "maint-check", "maint-update", "maint-beta", "maint-autocheck", "maint-reports", "maint-connect", "maint-disconnect", "maint-sync", "maint-data", "maint-program", "maint-cci-decompress", "maint-cci-compress":
		maintenanceAction(action)
	case "game-source", "game-folder", "game-use-loaded", "game-format", "game-preserve", "game-export", "game-credits", "game-patch-toggle":
		gameOutputAction(action,index)
	case "manager-scan", "manager-mode", "manager-select", "manager-patch-prev", "manager-patch-next", "manager-auto", "manager-review", "manager-skip", "manager-all-safe", "manager-run", "manager-format", "manager-open", "manager-image-choose", "manager-image-current", "manager-image-select", "manager-image-extract", "manager-log-clear", "manager-log-prev", "manager-log-next", "manager-confidence-refresh", "manager-confidence-result", "manager-confidence-xemu", "manager-confidence-hardware", "manager-confidence-revision", "manager-confidence-ram", "manager-confidence-cpu", "manager-merge":
        managerAction(action,index)
	case "tab":
        if index==7 {index=0} // discontinued standalone Game Outputs
        app.outputPickerOpen=false
        app.notesOpen=false
        app.galleryOpen=false
		app.tab = index
		if index == 5 { refreshRecentFiles() }
        if index==1 {
            // Source-free IPS is the useful default for a library JMP.
            // A format deliberately selected by the user always wins.
            if !app.convertFormatChosen && app.convertPatchFile=="" &&
                (app.parsed!=nil||strings.EqualFold(filepath.Ext(app.directFile),".jmp")){
                app.format=1
            }
            refreshConversionHint()
        }
		app.gameScroll = 0
		if index == 3 { refreshDeveloperDashboardLocal() }
	case "game":
		app.packSources = nil
		app.directFile = ""
		app.selectedGame = index
        app.notesOpen=false
        app.notesScroll=0
        app.galleryOpen=false
        app.galleryPage=0
		app.selectedPatch = -1
		app.parsed = nil
		app.patchScroll = 0
		app.groupScroll = 0
		app.expandedGroup = -1
	case "patch":
        app.notesOpen=false
        app.notesScroll=0
        app.galleryOpen=false
        app.galleryPage=0
		app.packSources = nil
		app.directFile = ""
		selectPatch(index)
    case "output-open":
        if chosenGameSource()=="" {notify("Choose a game file or extracted game folder first.");return}
        app.outputPickerOpen=true
        app.notesOpen=false
        app.galleryOpen=false
    case "output-close":
        app.outputPickerOpen=false
    case "crash-open-reports":
        app.tab=2
        app.recoveredCrashes=0
        _=core.ClearCrashNotices(app.root)
        refreshDiagnosticGroups()
    case "notes-open":
        if selectedPatchNoteText()!="" {app.notesOpen=true;app.notesScroll=0}
    case "notes-close":
        app.notesOpen=false
    case "notes-copy":
        if e:=copyDiagnosticText(selectedPatchNoteText());e!=nil{notify("Cannot copy notes: "+e.Error())}
    case "patch-scroll-track","records-scroll-track","notes-scroll-track":
        section:=strings.TrimSuffix(action,"-scroll-track")
        setScrollSection(section,valueFromScrollTrack(scrollSectionBar(section),app.mouseY))
    case "patch-scroll-thumb","records-scroll-thumb","notes-scroll-thumb":
        // Drag handled directly by the mouse capture loop.
    case "gallery-discover":
        discoverSelectedPatchMedia(true)
    case "gallery-explain":
        app.mu.Lock()
        var item core.CatalogPatch
        if app.selectedGame>=0 && app.selectedGame<len(app.cat.Games) {
            group:=app.cat.Games[app.selectedGame]
            if app.selectedPatch>=0&&app.selectedPatch<len(group.Patches) {item=group.Patches[app.selectedPatch]}
        }
        app.mu.Unlock()
        reason:=mediaDiscoveryReason(item)
        if reason==""{reason="No discovery result recorded yet."}
        showPreviewMessage("PatchHub image lookup details:\n\n"+reason+"\n\nImages belong to the selected game's patch folder, regardless of region or image filename. The public PatchHub media index lists all JPG/PNG files in each patch folder. Retry from the selected patch after syncing the site index.",false)
        // An explicit retry remains available by reselecting the patch.
    case "gallery-open":
        images,_:=selectedLibraryGallery()
        if len(images)>0{app.galleryOpen=true;app.galleryPage=0}
    case "gallery-close":
        app.galleryOpen=false
    case "gallery-prev":
        if app.galleryPage>0{app.galleryPage--}
    case "gallery-next":
        images,_:=selectedLibraryGallery()
        if app.galleryPage+1<(len(images)+1)/2{app.galleryPage++}
	case "group":
		if !app.busy && app.parsed != nil && index >= 0 && index < len(app.parsed.Groups) {
			g := app.parsed.Groups[index]
			app.parsed.EnableGroup(g.Name, !g.Enabled)
		}
	case "expand":
		if index == app.expandedGroup {
			app.expandedGroup = -1
		} else {
			app.expandedGroup = index
		}
	case "record":
		if !app.busy && app.parsed != nil && index >= 0 && index < len(app.parsed.Records) {
			app.parsed.SetRecordEnabled(index, !app.parsed.Records[index].Enabled)
		}
	case "direct":
		if path := pickFile(false, "Patch files\x00*.ips;*.eips;*.xdelta;*.vcdiff;*.bps;*.jmp;*.pf;*.mpdpack\x00All files\x00*.*\x00", "Choose patch file", ""); path != "" {
			app.directFile = path
			rememberRecent("patch",path)
			app.packSources = nil
			refreshConversionHint()
			app.status = "Custom patch loaded: " + filepath.Base(path)
			redraw()
		}
	case "open":
        if p:=pickFile(false,"Xbox files\x00*.xbe;*.iso;*.xiso;*.cci\x00XBE executable\x00*.xbe\x00Xbox XISO / ISO\x00*.iso;*.xiso\x00Cerbios CCI\x00*.cci\x00All files\x00*.*\x00","Choose original Xbox game file","");p!=""{
            openInput(p)
        }
	case "sync":
		go reloadAndSync()
	case "apply":
        if !libraryApplyReady(){
            notify("Choose a source game, a patch, and at least one supported output format before applying.")
            return
        }
        startGameOutput()
	case "preview":
		startPreview()
	case "filter":
		app.filter = index
		app.gameScroll = 0
		savePreferences()
	case "search":
		app.searchFocus = true
		app.passFocus = false
	case "import":
        func(){
            defer markCrashStage("ui_callback")
        markCrashStage("import_patch")
		if p := pickFile(false, "Magic patches\x00*.jmp;*.pf\x00All files\x00*.*\x00", "Import JMP or XBMC4Gamers PF", ""); p != "" {
            if saved,err:=core.ImportJMP(app.root,p);err!=nil{
                if errors.Is(err,core.ErrDuplicatePatch){
                    // A duplicate is a harmless no-op, not a modal error. Red
                    // footer text and the local verification log explain why
                    // nothing was added; do not reset selection or rescan.
                    app.mu.Lock()
                    app.status="Duplicate patch found! No need to add."
                    app.mu.Unlock()
                    manager.log.Add("error","Import JMP","Duplicate patch found! No need to add.")
                }else{
                    manager.log.Add("error","Import JMP","Could not import patch: "+err.Error())
                    notify("Import error: "+err.Error())
                }
            }else{
                manager.log.Add("pass","Import JMP","Imported independent patch copy: "+filepath.Base(saved))
                go reloadCatalog()
                notify("Imported local patch: "+filepath.Base(saved))
            }
		}
        }()
	case "dev-refresh", "dev-sort", "dev-select", "dev-prev", "dev-next", "dev-github", "dev-view", "dev-copy", "dev-root", "dev-report", "dev-fixpr", "dev-copygroups", "dev-copyroot", "dev-create":
		developerAction(action,index)
	case "diagnostics-folder", "diagnostics-crashfolder", "diagnostics-view", "diagnostics-copy", "diagnostics-find", "diagnostics-prev", "diagnostics-next", "diagnostics-github":
		diagnosticsAction(action)
	case "recent-select", "recent-prev", "recent-next", "recent-open", "recent-folder", "recent-copy", "recent-remove", "recent-clear":
		recentAction(action,index)
	case "folder":
		path := filepath.Join(app.root, "Custom")
		os.MkdirAll(path, 0755)
		sysCall(shellExecute, app.hwnd, uintptr(unsafe.Pointer(ptr("open"))), uintptr(unsafe.Pointer(ptr(path))), 0, 0, SW_SHOW)
    case "duplicate-audit":
        app.mu.Lock()
        duplicates:=append([]core.CatalogDuplicate(nil),app.cat.Duplicates...)
        root:=app.root
        app.mu.Unlock()
        var report strings.Builder
        fmt.Fprintf(&report,"Magic Patch Desk - deduplicated catalogue entries\\nCount: %d\\n\\n",len(duplicates))
        for i,d:=range duplicates {
            kept,_:=filepath.Rel(root,d.Kept)
            collapsed,_:=filepath.Rel(root,d.Collapsed)
            fmt.Fprintf(&report,"%d. Kept: %s\\n   Collapsed: %s\\n\\n",i+1,kept,collapsed)
        }
        destination:=filepath.Join(root,"duplicate-patch-audit.txt")
        if err:=os.WriteFile(destination,[]byte(report.String()),0600);err!=nil {
            notify("Cannot create duplicate audit: "+err.Error())
        }else {
            sysCall(shellExecute,app.hwnd,uintptr(unsafe.Pointer(ptr("open"))),uintptr(unsafe.Pointer(ptr(destination))),0,0,SW_SHOW)
            notify(fmt.Sprintf("Catalogue deduplication audit: %d collapsed entries",len(duplicates)))
        }
	case "refresh":
		go reloadCatalog()
	case "theme":
        names:=[]string{"Midnight","Forest","Plum","Light"}
        if index>=0 && index<len(names) {
            app.theme=names[index]
            setTheme(app.theme)
            updateWindowsTitleBarTheme()
            savePreferences()
            redraw()
        }
    case "workers":
		app.workers += index
		if app.workers < 1 {
			app.workers = 1
		}
		if app.workers > 8 {
			app.workers = 8
		}
		savePreferences()
	case "override":
		app.override = !app.override
	case "packsources":
		choosePackSources()
	case "a":
		if p := pickFile(false, "XBE or XISO\x00*.xbe;*.iso\x00", "Choose stock/original XBE or XISO", ""); p != "" {
			app.aFile = p
			rememberRecent("convert_original",p)
			refreshConversionHint()
		}
	case "cpatch":
		if p := pickFile(false, "Patch files\x00*.jmp;*.pf;*.ips;*.eips;*.xdelta;*.vcdiff;*.bps\x00All files\x00*.*\x00", "Choose source patch", ""); p != "" {
			app.convertPatchFile = p
			rememberRecent("convert_patch",p)
            if !app.convertFormatChosen {
                // Previous builds silently switched every imported JMP to
                // All Formats Pack, which always needs originals to create
                // BPS/Xdelta. Do not hide the source-free IPS converter.
                switch strings.ToLower(filepath.Ext(p)){
                case ".jmp":app.format=1
                case ".pf":app.format=0
                }
            }
			app.packSources = nil
			app.bFile = "" // A source patch and a modified XBE are alternative inputs.
			refreshConversionHint()
		}
	case "b":
		if p := pickFile(false, "XBE files\x00*.xbe\x00", "Choose modified XBE", ""); p != "" {
			app.bFile = p
			rememberRecent("convert_modified",p)
			// If a source patch is selected, this XBE is an independent
			// expected-result fixture, not a replacement for that patch.
			refreshConversionHint()
		}
	case "ipsrle":
		app.ipsRLE = !app.ipsRLE
    case "format":
        app.format=index
        app.convertFormatChosen=true
        refreshConversionHint()
	case "password":
		app.passFocus = true
		app.searchFocus = false
    case "convert-reason":
        // Display the full validation reason. A status bar or clipped footer
        // must never be the sole place explaining why conversion is blocked.
        if app.conversionBlocked {
            reason:=app.conversionHint
            if reason=="" {reason="Choose a valid source patch and output format first."}
            proc:=user32.NewProc("MessageBoxW")
            sysCall(proc,app.hwnd,uintptr(unsafe.Pointer(ptr(reason))),uintptr(unsafe.Pointer(ptr("Why conversion is blocked"))),0x40)
        }
    case "convert":
        convertFiles()
	}
	redraw()
}
func windowProc(hwnd uintptr, msg uint32, wp, lp uintptr) (out uintptr) {
    defer catchFatalPanic()
    return windowProcBody(hwnd,msg,wp,lp)
}
func windowProcBody(hwnd uintptr, msg uint32, wp, lp uintptr) uintptr {
	switch msg {
	case WM_DESTROY:
		sysCall(postQuit, 0)
		return 0
	case WM_CLOSE:
		sysCall(destroyWindow, hwnd)
		return 0
	case WM_SIZE:
		app.w = lo(lp)
		app.h = hi(lp)
		redraw()
		return 0
	case WM_ERASEBKGND:
		return 1 // Whole client is painted with an offscreen buffer, avoiding flicker.
	case WM_PAINT:
		var ps paintStruct
		hdc := sysCall(beginPaint, hwnd, uintptr(unsafe.Pointer(&ps)))
		var rr rect
		sysCall(getClientRect, hwnd, uintptr(unsafe.Pointer(&rr)))
		width, height := rr.R-rr.L, rr.B-rr.T
		if width > 0 && height > 0 {
			mem := sysCall(createCompatibleDC, hdc)
			bitmap := sysCall(createCompatibleBitmap, hdc, uintptr(width), uintptr(height))
			if mem != 0 && bitmap != 0 {
				prev := sysCall(selectObject, mem, bitmap)
				paint(mem)
				sysCall(bitBlt, hdc, 0, 0, uintptr(width), uintptr(height), mem, 0, 0, 0x00CC0020) // SRCCOPY
				sysCall(selectObject, mem, prev)
			} else {
				paint(hdc)
			}
			if bitmap != 0 {
				sysCall(deleteObject, bitmap)
			}
			if mem != 0 {
				sysCall(deleteDC, mem)
			}
		}
		sysCall(endPaint, hwnd, uintptr(unsafe.Pointer(&ps)))
		return 0
	case WM_MOUSEMOVE:
        x,y:=lo(lp),hi(lp)
        app.mouseY=y
        if app.mouseDown && strings.HasSuffix(app.pressedAction,"-scroll-thumb"){
            section:=strings.TrimSuffix(app.pressedAction,"-scroll-thumb")
            value:=valueFromThumbDrag(scrollSectionBar(section),app.scrollDragStartValue,app.scrollDragStartY,y)
            setScrollSection(section,value)
            redraw()
            return 0
        }
		hoverAt(hwnd,x,y)
		return 0
	case WM_MOUSELEAVE:
		app.mouseTracked = false
		app.hoverAction = ""
		app.hoverIndex = 0
		sysCall(setTimer, hwnd, 1, 16, 0)
		redraw()
		return 0
	case WM_TIMER:
		if wp == 1 {
			target := int(app.hoverFade)
			desired := 0
			if app.hoverAction != "" {
				desired = 255
			}
			if target < desired {
				target += 46
				if target > desired {
					target = desired
				}
			} else if target > desired {
				target -= 46
				if target < desired {
					target = desired
				}
			}
			app.hoverFade = uint8(target)
			redraw()
			if target == desired {
				sysCall(killTimer, hwnd, 1)
			}
			return 0
		}
	case WM_LBUTTONDOWN:
		x, y := lo(lp), hi(lp)
        app.mouseY=y
		h, ok := mouseHit(x, y)
		app.mouseDown = ok
        if ok&&strings.HasSuffix(h.Action,"-scroll-thumb"){
            section:=strings.TrimSuffix(h.Action,"-scroll-thumb")
            app.scrollDragStartY=y
            switch section{
            case "patch":app.scrollDragStartValue=app.patchScroll
            case "records":app.scrollDragStartValue=app.groupScroll
            case "notes":app.scrollDragStartValue=app.notesScroll
            }
        }
		if ok {
			app.pressedAction, app.pressedIndex = h.Action, h.Index
			sysCall(setCapture, hwnd)
		}
		redraw()
		return 0
	case WM_LBUTTONUP:
		x, y := lo(lp), hi(lp)
        app.mouseY=y
		pressedAction, pressedIndex := app.pressedAction, app.pressedIndex
		wasDown := app.mouseDown
		app.mouseDown = false
		app.pressedAction = ""
		sysCall(releaseCapture)
		app.searchFocus = false
		app.passFocus = false
        if wasDown&&strings.HasSuffix(pressedAction,"-scroll-thumb"){
            section:=strings.TrimSuffix(pressedAction,"-scroll-thumb")
            setScrollSection(section,valueFromThumbDrag(scrollSectionBar(section),app.scrollDragStartValue,app.scrollDragStartY,y))
            redraw()
            return 0
        }
		h, ok := mouseHit(x, y)
		if ok && wasDown && h.Action == pressedAction && h.Index == pressedIndex {
			handle(h.Action, h.Index)
			return 0
		}
		redraw()
		return 0
	case WM_MOUSEWHEEL:
        delta:=int(int16(hi(wp)))
        if delta==0{return 0}
        if app.tab==8{managerWheel(delta);redraw();return 0}
        if app.tab==3{
            app.mu.Lock()
            app.dashboardSelected-=delta/120*2
            if app.dashboardSelected<0{app.dashboardSelected=0}
            if app.dashboardSelected>=len(app.dashboardCases)&&len(app.dashboardCases)>0{app.dashboardSelected=len(app.dashboardCases)-1}
            app.mu.Unlock()
            redraw()
            return 0
        }
        if app.tab==0{
            if app.notesOpen{
                changeScrollByWheel("notes",delta)
                redraw()
                return 0
            }
            if app.galleryOpen{return 0}
            var pos point
            sysCall(user32.NewProc("GetCursorPos"),uintptr(unsafe.Pointer(&pos)))
            sysCall(user32.NewProc("ScreenToClient"),hwnd,uintptr(unsafe.Pointer(&pos)))
            switch{
            case inRect(app.patchViewport,pos.X,pos.Y):
                changeScrollByWheel("patch",delta)
            case inRect(app.recordViewport,pos.X,pos.Y):
                changeScrollByWheel("records",delta)
            case pos.X>=255&&pos.X<608:
                app.gameScroll-=delta/120*3
                if app.gameScroll<0{app.gameScroll=0}
            }
            redraw()
        }
        return 0
	case WM_CHAR:
		if app.passFocus {
			if wp == 8 {
				r := []rune(app.password)
				if len(r) > 0 {
					app.password = string(r[:len(r)-1])
				}
			} else if wp >= 32 && wp < 0x10000 && len([]rune(app.password)) < 100 {
				app.password += string(rune(wp))
			}
			refreshConversionHint()
			redraw()
			return 0
		}
		if app.searchFocus {
			if wp == 8 {
				r := []rune(app.search)
				if len(r) > 0 {
					app.search = string(r[:len(r)-1])
				}
			} else if wp >= 32 && wp < 0x10000 && len([]rune(app.search)) < 100 {
				app.search += string(rune(wp))
			}
			app.gameScroll = 0
			redraw()
		}
		return 0
	case WM_KEYDOWN:
		if handleQuickShortcut(wp) {return 0}
		if wp == 27 {
            app.outputPickerOpen=false
            app.notesOpen=false
            app.galleryOpen=false
			app.searchFocus = false
			app.passFocus = false
			redraw()
		}
		return 0
	case WM_DROPFILES:
		count := sysCall(dragQuery, wp, 0xFFFFFFFF, 0, 0)
		for i := uintptr(0); i < count; i++ {
			n := sysCall(dragQuery, wp, i, 0, 0)
			buf := make([]uint16, n+1)
			sysCall(dragQuery, wp, i, uintptr(unsafe.Pointer(&buf[0])), n+1)
			path := syscall.UTF16ToString(buf)
			ext := strings.ToLower(filepath.Ext(path))
			if info, err := os.Stat(path); err == nil && info.IsDir() {
                if app.tab==0 {
                    xbe:=filepath.Join(path,"default.xbe")
                    if st,e:=os.Stat(xbe);e==nil&&st.Mode().IsRegular(){
                        openInput(xbe)
                        selectGameSource(path)
                        app.status="Extracted game folder selected; choose a patch and output formats"
                        continue
                    }
                }
				go func() {
					n, e := core.ImportFolder(app.root, path)
					if e != nil {
						notify("Folder import error: " + e.Error())
					} else {
						reloadCatalog()
						notify(fmt.Sprintf("Imported %d JMP / PF files from %s", n, filepath.Base(path)))
					}
				}()
			} else if ext == ".mpdpack" {
				rememberRecent("patch",path)
				app.directFile = path
				app.tab = 0
				app.packSources = nil
				app.status = "Loaded patch pack: " + filepath.Base(path)
			} else if app.tab == 1 && (ext == ".jmp" || ext == ".ips" || ext == ".eips" || ext == ".xdelta" || ext == ".vcdiff" || ext == ".bps" || ext == ".pf") {
                app.convertPatchFile=path
                rememberRecent("convert_patch",path)
                if !app.convertFormatChosen&&ext==".jmp"{app.format=1}
                app.packSources=nil
				app.bFile = ""
				refreshConversionHint()
			app.status = "Conversion source patch: " + filepath.Base(path)
			} else if ext == ".jmp" || ext == ".pf" {
                if _,err:=core.ImportJMP(app.root,path);err!=nil {
                    if errors.Is(err,core.ErrDuplicatePatch){
                        app.mu.Lock()
                        app.status="Duplicate patch found! No need to add."
                        app.mu.Unlock()
                        manager.log.Add("error","Import JMP","Duplicate patch found! No need to add.")
                    }else{notify("Import error: "+err.Error())}
                }else{
                    go reloadCatalog()
                    notify("Imported "+filepath.Base(path))
                }
			} else if ext == ".xbe" || ext == ".iso" || ext == ".xiso" || ext == ".cci" {
                openInput(path)
			}
		}
		sysCall(dragFinish, wp)
		redraw()
		return 0
    case WM_APP + 2:
        // Draw/selection objects must be published on the owner UI thread.
        var newest core.Catalog
        found:=false
        for {
            select {
            case newest=<-catalogueReady:
                found=true
            default:
                if found {commitCatalogueOnUI(newest)}
                return 0
            }
        }
    case WM_APP + 3:
        for {
            select{
            case info:=<-autoMatchReady:
                app.mu.Lock()
                valid:=app.inputInfo.Valid&&app.inputInfo.TitleID==info.TitleID&&
                    app.inputInfo.CRC32==info.CRC32
                app.mu.Unlock()
                if valid {autoSelectXBE(info)}
            default:
                return 0
            }
        }
	case WM_APP + 1:
		redraw()
		return 0
	}
	return sysCall(defWindow, hwnd, uintptr(msg), wp, lp)
}
