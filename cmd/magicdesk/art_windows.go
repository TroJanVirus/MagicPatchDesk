//go:build windows

package main

import (
	"context"
	"magicpatchdesk/core"
	"os"
	"strings"
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
var gdiImageWidth = gdiplus.NewProc("GdipGetImageWidth")
var gdiImageHeight = gdiplus.NewProc("GdipGetImageHeight")
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
// drawArtFitted preserves the original aspect ratio (and pads the available
// space) rather than stretching the screenshots used for patch comparisons.
func drawArtFitted(hdc uintptr,filename string,x,y,w,h int32)bool{
    if filename==""||gdiplusToken==0||w<=0||h<=0{return false}
    art,ok:=artCache[filename]
    if !ok{
        var loaded uintptr
        status,_,_:=gdiLoad.Call(uintptr(unsafe.Pointer(ptr(filename))),uintptr(unsafe.Pointer(&loaded)))
        if status!=0||loaded==0{return false}
        artCache[filename]=loaded
        art=loaded
    }
    if art==0{return false}
    var imageW,imageH uint32
    if status,_,_:=gdiImageWidth.Call(art,uintptr(unsafe.Pointer(&imageW)));status!=0{return false}
    if status,_,_:=gdiImageHeight.Call(art,uintptr(unsafe.Pointer(&imageH)));status!=0{return false}
    if imageW==0||imageH==0{return false}
    tw,th:=w,h
    if int64(w)*int64(imageH) < int64(h)*int64(imageW){
        th=int32(int64(w)*int64(imageH)/int64(imageW))
    }else{
        tw=int32(int64(h)*int64(imageW)/int64(imageH))
    }
    if tw<1||th<1{return false}
    return drawArt(hdc,filename,x+(w-tw)/2,y+(h-th)/2,tw,th)
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

// Only three cover workers may exist; never leave hundreds of goroutines
// waiting for a semaphore during rapid scrolling. Cache works offline.
func scheduleCover(source string) string {
	return scheduleCoverCandidates([]string{source})
}

func scheduleCoverCandidates(sources []string) string {
	if len(sources) == 0 {
		return ""
	}
	for _, source := range sources {
		path := core.CoverCachePath(app.root, source)
		if core.ValidCoverFile(path) {
			return path
		}
		oldPath := path[:len(path)-4] + ".img"
		if core.ValidCoverFile(oldPath) {
			if err := os.Rename(oldPath, path); err == nil {
				return path
			}
			return oldPath
		}
	}
	key := strings.Join(sources, "\n")
	coverMu.Lock()
	item := coverAttempts[key]
	if item.InFlight || time.Now().Before(item.Next) {
		coverMu.Unlock()
		return ""
	}
	select {
	case coverSlots <- struct{}{}:
	default:
		coverMu.Unlock()
		return ""
	}
	item.InFlight = true
	coverAttempts[key] = item
	coverMu.Unlock()
	candidates := append([]string(nil), sources...)
	go func() {
		defer func() { <-coverSlots }()
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()
		found := false
		for _, source := range candidates {
			result, err := core.DownloadCover(ctx, app.root, source)
			if err == nil && result != "" && core.ValidCoverFile(result) {
				found = true
				break
			}
			if ctx.Err() != nil {
				break
			}
		}
		coverMu.Lock()
		item := coverAttempts[key]
		item.InFlight = false
		if found {
			item.Next = time.Time{}
		} else {
			item.Next = time.Now().Add(30 * time.Minute)
		}
		coverAttempts[key] = item
		coverMu.Unlock()
		if found && app.hwnd != 0 {
			sysCall(postMsg, app.hwnd, WM_APP+1, 0, 0)
		}
	}()
	return ""
}
