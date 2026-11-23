//go:build windows

package main

import (
	"context"
	"magicpatchdesk/core"
	"os"
	"fmt"
	"sort"
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
// These diagnostic results never leave local memory unless the user copies them.
// Only the UI thread updates and reads this map.
var artDrawFailures = map[string]string{}

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
	if path == "" { return false }
	if gdiplusToken == 0 {
		artDrawFailures[path] = "GDI+ did not initialize"
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
			artDrawFailures[path] = fmt.Sprintf("GDI+ image load status %d", status)
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
		artDrawFailures[path] = fmt.Sprintf("GDI+ graphics context status %d", status)
		return false
	}
	v, _, _ := gdiDraw.Call(graphics, art, uintptr(x), uintptr(y), uintptr(w), uintptr(h))
	gdiDelete.Call(graphics)
	if v != 0 { artDrawFailures[path] = fmt.Sprintf("GDI+ image drawing status %d", v) } else { delete(artDrawFailures, path) }
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
	LastResult string // safe category only, not a Windows path or HTTP response body
	Attempted int
	LastChecked time.Time
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
		attempted := 0
		lastReason := "No candidates"
		// Bound any single image request, not the whole set. A slow/broken
		// primary URL must not exhaust the timeout of a valid Title-ID fallback.
		for _, source := range job.Sources {
			attempted++
			ctx, cancel := context.WithTimeout(context.Background(), 18*time.Second)
			result, err := core.DownloadCover(ctx, job.Root, source)
			cancel()
			if err == nil && result != "" && core.ValidCoverFile(result) {
				found = true
				lastReason = "Downloaded and decoded"
				break
			}
			if err == nil { lastReason = "Downloaded file failed image validation" } else { lastReason = coverFailureClass(err) }
		}
		coverMu.Lock()
		item := coverAttempts[job.Key]
		item.InFlight = false
		item.LastResult = lastReason
		item.Attempted = attempted
		item.LastChecked = time.Now()
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


// Include only stable, privacy-safe error categories in copyable local
// diagnostics; network libraries sometimes embed local paths in raw errors.
func coverFailureClass(err error) string {
	if err == nil { return "Unknown download failure" }
	e := strings.ToLower(err.Error())
	switch {
	case strings.Contains(e, "404"): return "HTTP 404: image missing"
	case strings.Contains(e, "403"): return "HTTP 403: website denied request"
	case strings.Contains(e, "429"): return "HTTP 429: website rate limit"
	case strings.Contains(e, "500"), strings.Contains(e, "502"), strings.Contains(e, "503"): return "Website HTTP 5xx"
	case strings.Contains(e, "x509"), strings.Contains(e, "certificate"): return "TLS certificate verification failed"
	case strings.Contains(e, "tls"): return "TLS handshake failed"
	case strings.Contains(e, "timeout"), strings.Contains(e, "deadline exceeded"): return "Network timeout"
	case strings.Contains(e, "connection refused"): return "Connection refused"
	case strings.Contains(e, "no such host"), strings.Contains(e, "lookup"): return "DNS lookup failed"
	case strings.Contains(e, "untrusted"): return "Untrusted URL or redirect"
	case strings.Contains(e, "cover too large"), strings.Contains(e, "5 mib"): return "Image too large"
	case strings.Contains(e, "image"), strings.Contains(e, "dimensions"), strings.Contains(e, "jpeg"): return "Image validation rejected"
	default: return "Network or file error (unspecified)"
	}
}

func coverStatus(g core.Game) string {
	if g.Cover != "" && core.ValidCoverFile(g.Cover) {
		if issue := artDrawFailures[g.Cover]; issue != "" { return issue }
		return "Local cover cached"
	}
	candidates := core.CoverCandidates(g)
	if len(candidates)==0 { return "No known image URL or Title ID" }
	for _, u := range candidates {
		p := core.CoverCachePath(app.root,u)
		if core.ValidCoverFile(p) {
			if issue := artDrawFailures[p]; issue != "" { return issue }
			return "Valid image cached"
		}
	}
	key := strings.Join(candidates,"\n")
	coverMu.Lock()
	state := coverAttempts[key]
	coverMu.Unlock()
	switch {
	case state.InFlight: return "Downloading or queued"
	case state.LastResult!="": return state.LastResult
	default: return "Not downloaded yet"
	}
}

func forceSelectedCoverRetry() {
	if app.selectedGame<0 || app.selectedGame>=len(app.cat.Games) {return}
	g:=app.cat.Games[app.selectedGame]
	candidates:=core.CoverCandidates(g)
	if len(candidates)==0 { notify("Cover check: no website URL or valid Xbox Title ID"); return }
	key:=strings.Join(candidates,"\n")
	coverMu.Lock()
	item:=coverAttempts[key]
	if !item.InFlight {
		item.Next=time.Time{}
		coverAttempts[key]=item
	}
	coverMu.Unlock()
	// The next paint schedules a new job, without blocking the native UI.
	redraw()
	notify("Cover check: "+g.Title+" - "+coverStatus(g))
}

func selectedCoverDiagnosticText() string {
	if app.selectedGame<0 || app.selectedGame>=len(app.cat.Games) { return "No game selected" }
	g:=app.cat.Games[app.selectedGame]
	candidates:=core.CoverCandidates(g)
	ids:=map[string]bool{}
	for _,p:=range g.Patches { if p.TitleID!="" { ids[p.TitleID]=true } }
	for _,v:=range g.Versions { if v.TitleID!="" { ids[v.TitleID]=true } }
	allIDs:=make([]string,0,len(ids))
	for id:=range ids { allIDs=append(allIDs,id) }
	sort.Strings(allIDs)
	key:=strings.Join(candidates,"\n")
	coverMu.Lock()
	state:=coverAttempts[key]
	coverMu.Unlock()
	var b strings.Builder
	fmt.Fprintf(&b,"Magic Patch Desk %s — cover diagnosis\n",core.AppVersion)
	fmt.Fprintf(&b,"Game: %s\nXbox Title IDs: %s\nPatchHub folder: %s\n",g.Title,strings.Join(allIDs,", "),g.CoverFolder)
	fmt.Fprintf(&b,"Cover state: %s\n",coverStatus(g))
	if !state.LastChecked.IsZero() { fmt.Fprintf(&b,"Last attempt: %s UTC; %d URLs tried; result: %s\n",state.LastChecked.UTC().Format(time.RFC3339),state.Attempted,state.LastResult) }
	if g.Cover!="" { fmt.Fprintf(&b,"Imported cover exists and decodes: %t\n",core.ValidCoverFile(g.Cover)) }
	fmt.Fprintf(&b,"Website URL candidates: %d\n",len(candidates))
	for i,u:=range candidates {
		p:=core.CoverCachePath(app.root,u)
		fmt.Fprintf(&b,"%d. %s\n   Cached valid: %t\n",i+1,u,core.ValidCoverFile(p))
		if issue:=artDrawFailures[p];issue!=""{fmt.Fprintf(&b,"   Rendering: %s\n",issue)}
	}
	// Never include local directory/user names in a shareable report.
	return b.String()
}
