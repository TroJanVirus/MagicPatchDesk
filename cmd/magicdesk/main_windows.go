//go:build windows

package main

import (
	"context"
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
	invalidateRect         = user32.NewProc("InvalidateRect")
	beginPaint             = user32.NewProc("BeginPaint")
	endPaint               = user32.NewProc("EndPaint")
	getClientRect          = user32.NewProc("GetClientRect")
	drawText               = user32.NewProc("DrawTextW")
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
type fonts struct{ Regular, Medium, Title, Hero, Small, Micro uintptr }
type appState struct {
	mu            sync.Mutex
	hwnd          uintptr
	root          string
	cat           core.Catalog
	selectedGame  int
	selectedPatch int
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
	override      bool
	status        string
	syncStatus    string
	remoteCount   int
	busy          bool
	progress      float64
	aFile, bFile  string
	directFile    string
	format        int
	password      string
	passFocus     bool
	hits          []hit
	w, h          int32
	fonts         fonts
	lastErr       string
}

var app appState

func sysCall(p *syscall.LazyProc, a ...uintptr) uintptr { r, _, _ := p.Call(a...); return r }
func font(size int32, weight int32) uintptr {
	return sysCall(createFont, uintptr(int64(-size)), 0, 0, 0, uintptr(weight), 0, 0, 0, 1, 0, 0, 0, 5, uintptr(unsafe.Pointer(ptr("Segoe UI"))))
}
func addHit(r rect, act string, index int) { app.hits = append(app.hits, hit{r, act, index}) }
func fill(hdc uintptr, r rect, c uint32) {
	b := sysCall(createSolidBrush, col(c))
	sysCall(fillRect, hdc, uintptr(unsafe.Pointer(&r)), b)
	sysCall(deleteObject, b)
}
func panel(hdc uintptr, r rect, color uint32, radius int32) {
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
	if s == "" {
		return
	}
	sysCall(setBkMode, hdc, TRANSPARENT)
	sysCall(setTextColor, hdc, col(color))
	old := sysCall(selectObject, hdc, size)
	utf := syscall.StringToUTF16(s)
	sysCall(drawText, hdc, uintptr(unsafe.Pointer(&utf[0])), uintptr(len(utf)-1), uintptr(unsafe.Pointer(&r)), uintptr(flags))
	sysCall(selectObject, hdc, old)
}
func textAt(hdc uintptr, s string, x, y, w, h int32, f uintptr, color uint32) {
	txt(hdc, s, ir(x, y, w, h), f, color, DT_SINGLELINE|DT_VCENTER|DT_END_ELLIPSIS)
}
func button(hdc uintptr, label string, r rect, action string, idx int, main bool) {
	if main {
		panel(hdc, r, 0x447FF5, 14)
	} else {
		panel(hdc, r, 0x283247, 12)
	}
	c := uint32(0xEDF3FF)
	txt(hdc, label, r, app.fonts.Medium, c, DT_SINGLELINE|DT_VCENTER|DT_CENTER|DT_END_ELLIPSIS)
	addHit(r, action, idx)
}
func tag(hdc uintptr, label string, x, y int32, bg uint32) {
	wd := int32(12 + len([]rune(label))*7)
	if wd > 170 {
		wd = 170
	}
	panel(hdc, ir(x, y, wd, 23), bg, 11)
	txt(hdc, label, ir(x+6, y, wd-12, 22), app.fonts.Micro, 0xE5EEFF, DT_SINGLELINE|DT_VCENTER|DT_CENTER)
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
	app.mu.Lock()
	app.status = s
	app.mu.Unlock()
	sysCall(postMsg, app.hwnd, WM_APP+1, 0, 0)
}
func initApp() {
	runtime.LockOSThread()
	app.workers = 4
	app.selectedGame = -1
	app.selectedPatch = -1
	app.root, _ = core.GetLibraryDir()
	app.status = "Loading local patch library…"
	app.syncStatus = "Connecting to GitHub…"
	initArtwork()
	defer closeArtwork()
	app.fonts = fonts{Regular: font(17, 400), Medium: font(17, 600), Title: font(22, 700), Hero: font(30, 700), Small: font(14, 400), Micro: font(13, 600)}
	sysCall(setDPI)
	instance := sysCall(getModule, 0)
	name := ptr("MagicPatchDeskNative")
	class := wndClass{Size: uint32(unsafe.Sizeof(wndClass{})), WndProc: syscall.NewCallback(windowProc), Instance: instance, Cursor: sysCall(loadCursor, 0, IDC_ARROW), Name: name}
	if sysCall(regClass, uintptr(unsafe.Pointer(&class))) == 0 {
		panic("RegisterClassEx failed")
	}
	app.hwnd = sysCall(createWindow, 0, uintptr(unsafe.Pointer(name)), uintptr(unsafe.Pointer(ptr("Magic Patch Desk  |  Xbox Patch Hub"))), WS_OVERLAPPEDWINDOW|WS_VISIBLE, 100, 80, 1320, 840, 0, 0, instance, 0)
	if app.hwnd == 0 {
		panic("CreateWindowEx failed")
	}
	dark := int32(1)
	sysCall(dwmAttr, app.hwnd, 20, uintptr(unsafe.Pointer(&dark)), 4)
	sysCall(dragAccept, app.hwnd, 1)
	sysCall(showWindow, app.hwnd, SW_SHOW)
	sysCall(updateWindow, app.hwnd)
	go reloadAndSync()
	go syncPatchHubMetadata()
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
func main() { initApp() }
func reloadCatalog() {
	app.mu.Lock()
	workers := app.workers
	prevTitle := ""
	prevFile := ""
	if app.selectedGame >= 0 && app.selectedGame < len(app.cat.Games) {
		prevTitle = app.cat.Games[app.selectedGame].Title
		if app.selectedPatch >= 0 && app.selectedPatch < len(app.cat.Games[app.selectedGame].Patches) {
			prevFile = app.cat.Games[app.selectedGame].Patches[app.selectedPatch].File
		}
	}
	app.mu.Unlock()
	c, err := core.LoadCatalog(app.root, workers)
	if err != nil {
		notify(err.Error())
		return
	}
	app.mu.Lock()
	app.cat = c
	app.selectedGame = -1
	app.selectedPatch = -1
	app.parsed = nil
	if prevTitle != "" {
		for i, g := range c.Games {
			if strings.EqualFold(g.Title, prevTitle) {
				app.selectedGame = i
				for k, cp := range g.Patches {
					if cp.File == prevFile {
						app.selectedPatch = k
						break
					}
				}
				break
			}
		}
	}
	info := app.inputInfo
	app.mu.Unlock()
	if info.Valid && prevTitle == "" {
		autoSelectXBE(info)
	} else if prevFile != "" {
		app.mu.Lock()
		idx := app.selectedPatch
		app.mu.Unlock()
		if idx >= 0 {
			selectPatch(idx)
		}
	}
	redraw()
}

// Match by the embedded XBE Title ID and its whole-file CRC32 when possible.
// Suggest a patch, never apply it automatically.
func autoSelectXBE(info core.XBEInfo) {
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
	app.input = path
	app.inputErr = "Reading executable…"
	app.inputInfo = core.XBEInfo{}
	redraw()
	go func() {
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
		app.inputInfo = info
		if err != nil {
			app.inputErr = err.Error()
		} else {
			app.inputErr = ""
			app.status = "Loaded: " + info.Title + " · " + info.TitleID + " · CRC32 " + info.CRC32
		}
		app.mu.Unlock()
		if err == nil {
			autoSelectXBE(info)
		}
		redraw()
	}()
}
func selectPatch(idx int) {
	app.mu.Lock()
	if app.selectedGame < 0 || app.selectedGame >= len(app.cat.Games) || idx < 0 || idx >= len(app.cat.Games[app.selectedGame].Patches) {
		app.mu.Unlock()
		return
	}
	path := app.cat.Games[app.selectedGame].Patches[idx].File
	app.selectedPatch = idx
	app.groupScroll = 0
	app.mu.Unlock()
	b, err := os.ReadFile(path)
	if err != nil {
		notify(err.Error())
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
		if ext == ".xbe" {
			var b []byte
			b, err = os.ReadFile(source)
			if err == nil {
				var output []byte
				output, err = apply(b)
				if err == nil {
					err = atomicWrite(dest, output)
				}
			}
		} else {
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
		redraw()
	}()
}

func convertFiles() {
	if app.aFile == "" {
		notify("Choose the original XBE before converting.")
		return
	}
	if app.bFile == "" && app.parsed == nil && app.directFile == "" {
		notify("Choose a modified XBE, or select a patch from the library.")
		return
	}
	format := app.format
	ext := []string{".jmp", ".ips", ".xdelta", ".eips"}[format]
	output := pickFile(true, "Patch file\x00*.jmp;*.ips;*.eips;*.xdelta\x00All files\x00*.*\x00", "Save converted patch", "ConvertedPatch"+ext)
	if output == "" {
		return
	}
	app.mu.Lock()
	app.busy = true
	app.mu.Unlock()
	redraw()
	aPath, bPath, pass, direct, selected := app.aFile, app.bFile, app.password, app.directFile, app.parsed
	go func() {
		a, err := os.ReadFile(aPath)
		if err == nil {
			var b []byte
			if bPath != "" {
				b, err = os.ReadFile(bPath)
			} else if direct != "" {
				b, err = core.ApplyPatchFile(a, direct, pass, false)
			} else {
				b, err = selected.Apply(a, false)
			}
			if err == nil {
				var out []byte
				out, err = core.BuildPatch(a, b, format, pass, strings.TrimSuffix(filepath.Base(aPath), filepath.Ext(aPath)))
				if err == nil {
					err = atomicWrite(output, out)
				}
			}
		}
		app.mu.Lock()
		app.busy = false
		if err != nil {
			app.status = "Conversion failed: " + err.Error()
		} else {
			app.status = "Saved " + strings.ToUpper(ext) + " patch: " + output
		}
		app.mu.Unlock()
		redraw()
	}()
}

func handle(action string, index int) {
	switch action {
	case "tab":
		app.tab = index
		app.gameScroll = 0
	case "game":
		app.directFile = ""
		app.selectedGame = index
		app.selectedPatch = -1
		app.parsed = nil
		app.patchScroll = 0
		app.groupScroll = 0
	case "patch":
		app.directFile = ""
		selectPatch(index)
	case "group":
		if app.parsed != nil && index < len(app.parsed.Groups) {
			g := app.parsed.Groups[index]
			app.parsed.EnableGroup(g.Name, !g.Enabled)
		}
	case "direct":
		if path := pickFile(false, "Patch files\x00*.ips;*.eips;*.xdelta;*.vcdiff;*.jmp\x00All files\x00*.*\x00", "Choose patch file", ""); path != "" {
			app.directFile = path
			app.status = "Custom patch loaded: " + filepath.Base(path)
			redraw()
		}
	case "open":
		if p := pickFile(false, "Xbox files\x00*.xbe;*.iso\x00XBE executable\x00*.xbe\x00Xbox disc image\x00*.iso\x00All files\x00*.*\x00", "Choose XBE or Xbox ISO", ""); p != "" {
			openInput(p)
		}
	case "sync":
		go reloadAndSync()
	case "apply":
		startPatch()
	case "filter":
		app.filter = index
		app.gameScroll = 0
	case "search":
		app.searchFocus = true
		app.passFocus = false
	case "import":
		if p := pickFile(false, "Magic patches\x00*.jmp\x00All files\x00*.*\x00", "Import JMP file", ""); p != "" {
			if _, err := core.ImportJMP(app.root, p); err != nil {
				notify("Import error: " + err.Error())
			} else {
				go reloadCatalog()
				notify("Imported local JMP: " + filepath.Base(p))
			}
		}
	case "folder":
		path := filepath.Join(app.root, "Custom")
		os.MkdirAll(path, 0755)
		sysCall(shellExecute, app.hwnd, uintptr(unsafe.Pointer(ptr("open"))), uintptr(unsafe.Pointer(ptr(path))), 0, 0, SW_SHOW)
	case "refresh":
		go reloadCatalog()
	case "workers":
		app.workers += index
		if app.workers < 1 {
			app.workers = 1
		}
		if app.workers > 8 {
			app.workers = 8
		}
	case "override":
		app.override = !app.override
	case "a":
		if p := pickFile(false, "XBE files\x00*.xbe\x00", "Choose original XBE", ""); p != "" {
			app.aFile = p
		}
	case "b":
		if p := pickFile(false, "XBE files\x00*.xbe\x00", "Choose modified XBE", ""); p != "" {
			app.bFile = p
		}
	case "format":
		app.format = index
	case "password":
		app.passFocus = true
		app.searchFocus = false
	case "convert":
		convertFiles()
	}
	redraw()
}
func windowProc(hwnd uintptr, msg uint32, wp, lp uintptr) uintptr {
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
	case WM_LBUTTONUP:
		x, y := lo(lp), hi(lp)
		app.searchFocus = false
		app.passFocus = false
		for i := len(app.hits) - 1; i >= 0; i-- {
			h := app.hits[i]
			if inside(h.R, x, y) {
				handle(h.Action, h.Index)
				return 0
			}
		}
		redraw()
		return 0
	case WM_MOUSEWHEEL:
		delta := hi(wp)
		if delta != 0 {
			var p point
			sysCall(user32.NewProc("GetCursorPos"), uintptr(unsafe.Pointer(&p)))
			sysCall(user32.NewProc("ScreenToClient"), hwnd, uintptr(unsafe.Pointer(&p)))
			if p.X < 610 {
				app.gameScroll -= int(delta/120) * 3
				if app.gameScroll < 0 {
					app.gameScroll = 0
				}
			} else {
				app.patchScroll -= int(delta/120) * 2
				if app.patchScroll < 0 {
					app.patchScroll = 0
				}
				app.groupScroll -= int(delta / 120)
				if app.groupScroll < 0 {
					app.groupScroll = 0
				}
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
		if wp == 27 {
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
				go func() {
					n, e := core.ImportFolder(app.root, path)
					if e != nil {
						notify("Folder import error: " + e.Error())
					} else {
						reloadCatalog()
						notify(fmt.Sprintf("Imported %d JMP files from %s", n, filepath.Base(path)))
					}
				}()
			} else if ext == ".jmp" {
				if _, err := core.ImportJMP(app.root, path); err != nil {
					notify(err.Error())
				} else {
					go reloadCatalog()
					notify("Imported " + filepath.Base(path))
				}
			} else if ext == ".xbe" || ext == ".iso" {
				openInput(path)
			}
		}
		sysCall(dragFinish, wp)
		redraw()
		return 0
	case WM_APP + 1:
		redraw()
		return 0
	}
	return sysCall(defWindow, hwnd, uintptr(msg), wp, lp)
}
