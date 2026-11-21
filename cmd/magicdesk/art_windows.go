//go:build windows

package main

import (
	"context"
	"magicpatchdesk/core"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

var gdiplus = syscall.NewLazyDLL("gdiplus.dll")
var startGDI = gdiplus.NewProc("GdiplusStartup")
var stopGDI = gdiplus.NewProc("GdiplusShutdown")
var gdiLoad = gdiplus.NewProc("GdipLoadImageFromFile")
var gdiDispose = gdiplus.NewProc("GdipDisposeImage")
var gdiCreate = gdiplus.NewProc("GdipCreateFromHDC")
var gdiDelete = gdiplus.NewProc("GdipDeleteGraphics")
var gdiDraw = gdiplus.NewProc("GdipDrawImageRectI")
var gdiplusToken uintptr
var artCache = map[string]uintptr{}

type gdiplusStartupInput struct {
	Version            uint32
	Callback           uintptr
	SuppressBackground uint32
	SuppressExternal   uint32
}

func initArtwork() {
	input := gdiplusStartupInput{Version: 1}
	startGDI.Call(uintptr(unsafe.Pointer(&gdiplusToken)), uintptr(unsafe.Pointer(&input)), 0)
}
func closeArtwork() {
	for _, p := range artCache {
		if p != 0 {
			gdiDispose.Call(p)
		}
	}
	if gdiplusToken != 0 {
		stopGDI.Call(gdiplusToken)
	}
}
func drawArt(hdc uintptr, path string, x, y, w, h int32) bool {
	if path == "" || gdiplusToken == 0 {
		return false
	}
	art, ok := artCache[path]
	if !ok {
		var loaded uintptr
		status, _, _ := gdiLoad.Call(uintptr(unsafe.Pointer(ptr(path))), uintptr(unsafe.Pointer(&loaded)))
		if status != 0 {
			loaded = 0
		}
		if loaded == 0 {
			return false
		}
		artCache[path] = loaded
		art = loaded
	}
	if art == 0 {
		return false
	}
	var graphics uintptr
	status, _, _ := gdiCreate.Call(hdc, uintptr(unsafe.Pointer(&graphics)))
	if status != 0 || graphics == 0 {
		return false
	}
	v, _, _ := gdiDraw.Call(graphics, art, uintptr(x), uintptr(y), uintptr(w), uintptr(h))
	gdiDelete.Call(graphics)
	return v == 0
}

var coverSlots = make(chan struct{}, 3)
var coverMu sync.Mutex
var coverAttempts = map[string]coverAttempt{}

type coverAttempt struct {
	InFlight bool
	Next     time.Time
}

// Downloads are asynchronous and rate-limited. A failed request gets a retry
// window rather than being marked "downloaded" for the entire app session.
func scheduleCover(source string) string {
	path := core.CoverCachePath(app.root, source)
	if core.ValidCoverFile(path) {
		return path
	}
	// Older releases cached with .img and did not validate the bytes. Reuse only
	// known-good files, otherwise redownload to a .jpg path GDI+ understands.
	oldPath := path[:len(path)-4] + ".img"
	if core.ValidCoverFile(oldPath) {
		return oldPath
	}
	coverMu.Lock()
	item := coverAttempts[path]
	if item.InFlight || time.Now().Before(item.Next) {
		coverMu.Unlock()
		return ""
	}
	item.InFlight = true
	item.Next = time.Now().Add(2 * time.Minute)
	coverAttempts[path] = item
	coverMu.Unlock()
	go func() {
		coverSlots <- struct{}{}
		defer func() { <-coverSlots }()
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()
		result, err := core.DownloadCover(ctx, app.root, source)
		coverMu.Lock()
		item := coverAttempts[path]
		item.InFlight = false
		if err != nil {
			item.Next = time.Now().Add(2 * time.Minute)
		} else {
			item.Next = time.Time{}
		}
		coverAttempts[path] = item
		coverMu.Unlock()
		if err == nil && result != "" && core.ValidCoverFile(result) {
			sysCall(postMsg, app.hwnd, WM_APP+1, 0, 0)
		}
	}()
	return ""
}
