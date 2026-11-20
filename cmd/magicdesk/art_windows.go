//go:build windows

package main

import (
	"context"
	"magicpatchdesk/core"
	"os"
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
		result, _, _ := gdiLoad.Call(uintptr(unsafe.Pointer(ptr(path))), uintptr(unsafe.Pointer(&loaded)))
		if result != 0 {
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

var coverQueue sync.Map
var coverSlots = make(chan struct{}, 3)

// Only visible covers are retrieved, with a strict three-download concurrency
// limit. All artwork is cached locally for subsequent fully offline starts.
func scheduleCover(source string) string {
	path := core.CoverCachePath(app.root, source)
	if _, err := os.Stat(path); err == nil {
		return path
	}
	if _, loaded := coverQueue.LoadOrStore(path, true); !loaded {
		go func() {
			coverSlots <- struct{}{}
			defer func() { <-coverSlots }()
			ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
			defer cancel()
			if _, err := core.DownloadCover(ctx, app.root, source); err == nil {
				sysCall(postMsg, app.hwnd, WM_APP+1, 0, 0)
			}
		}()
	}
	return path
}
