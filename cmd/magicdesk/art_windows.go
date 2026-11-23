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

// The image loader must not depend on further mouse movement or a successful
// sibling download to make progress. Exactly three workers consume a bounded
// queue, avoiding both dropped requests and unbounded goroutine growth.
const coverWorkerCount = 3
var coverMu sync.Mutex
var coverAttempts = map[string]coverAttempt{}
var coverJobs = make(chan coverRequest, 48)
var coverWorkers sync.Once

type coverAttempt struct {
	InFlight bool // queued or currently downloading
	Next time.Time
}
type coverRequest struct {
	Key string
	Sources []string
	Root string
	HWND uintptr
}

func coverWorkerLoop() {
	defer catchFatalPanic()
	for job := range coverJobs {
		found := false
		// Bound any single image request, not the whole set. A slow/broken
		// primary URL must not exhaust the timeout of a valid Title-ID fallback.
		for _, source := range job.Sources {
			ctx, cancel := context.WithTimeout(context.Background(), 18*time.Second)
			result, err := core.DownloadCover(ctx, job.Root, source)
			cancel()
			if err == nil && result != "" && core.ValidCoverFile(result) {
				found = true
				break
			}
		}
		coverMu.Lock()
		item := coverAttempts[job.Key]
		item.InFlight = false
		if found {
			item.Next = time.Time{}
		} else {
			// Transient failures retry instead of staying invisible for half
			// an hour; repeated redraws do not hammer PatchHub.
			item.Next = time.Now().Add(2 * time.Minute)
		}
		coverAttempts[job.Key] = item
		coverMu.Unlock()
		// Always repaint, including failures. It also gives any visible cover
		// which was rejected by a full queue another chance to enqueue.
		if job.HWND != 0 {
			sysCall(postMsg, job.HWND, WM_APP+1, 0, 0)
		}
	}
}

func startCoverWorkers() {
	coverWorkers.Do(func() {
		for i := 0; i < coverWorkerCount; i++ {
			go coverWorkerLoop()
		}
	})
}

// scheduleCoverCandidates is called only on the UI thread while painting.
// It never blocks on network access or waits for a free worker, and it first
// returns a validated cached cover regardless of the previous retry state.
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
	job := coverRequest{Key: key, Sources: append([]string(nil), sources...), Root: app.root, HWND: app.hwnd}
	// Queuing a request claims its identity atomically; a full queue leaves it
	// eligible for retry on the next completion-triggered repaint.
	item.InFlight = true
	coverAttempts[key] = item
	coverMu.Unlock()
	startCoverWorkers()
	select {
	case coverJobs <- job:
	default:
		coverMu.Lock()
		item = coverAttempts[key]
		item.InFlight = false
		coverAttempts[key] = item
		coverMu.Unlock()
	}
	return ""
}

func scheduleCover(source string) string {
	return scheduleCoverCandidates([]string{source})
}
