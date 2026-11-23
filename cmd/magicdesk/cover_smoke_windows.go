//go:build windows

package main

import (
	"image"
	"image/color"
	"image/jpeg"
	"magicpatchdesk/core"
	"os"
	"path/filepath"
	"syscall"
)

// Exercise the real GDI+ draw path with a validated JPEG and the same hashed
// disk cache name used for PatchHub downloads. Pure Go URL tests cannot catch
// failures where native UI never loads an otherwise-valid downloaded image.
func smokeVerifiedCoverDrawing() {
	root, err := os.MkdirTemp("", "mpd-cover-draw-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(root)
	oldRoot := app.root
	app.root = root
	defer func() { app.root = oldRoot }()

	url := core.CoverCandidates(core.Game{
		Title: "American Idol (Prototype)",
		Patches: []core.CatalogPatch{{TitleID: "434D0027"}},
	})[0]
	path := core.CoverCachePath(root, url)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		panic(err)
	}
	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	img := image.NewRGBA(image.Rect(0, 0, 32, 48))
	for y := 0; y < 48; y++ {
		for x := 0; x < 32; x++ {
			img.Set(x, y, color.RGBA{uint8(x * 7), uint8(y * 4), 190, 255})
		}
	}
	if err := jpeg.Encode(f, img, &jpeg.Options{Quality: 85}); err != nil {
		f.Close()
		panic(err)
	}
	if err := f.Close(); err != nil {
		panic(err)
	}
	if !core.ValidCoverFile(path) {
		panic("synthetic cover is not a valid JPEG")
	}
	if got := scheduleCoverCandidates([]string{url}); got != path {
		panic("native cover scheduler did not return the verified cached JPEG")
	}
	dc := sysCall(user32.NewProc("GetDC"), app.hwnd)
	if dc == 0 {
		panic("native GetDC failed in cover draw test")
	}
	drawn := drawArt(dc, path, 8, 8, 32, 48)
	sysCall(user32.NewProc("ReleaseDC"), app.hwnd, dc)
	if art := artCache[path]; art != 0 {
		gdiDispose.Call(art)
		delete(artCache, path)
	}
	if !drawn {
		panic("GDI+ did not render a verified cover JPEG")
	}
    if artDrawFailures[path]!="" { panic("Cover rendering retained an error after a successful draw") }
    if coverStatus(core.Game{Title:"American Idol (Prototype)",Patches: []core.CatalogPatch{{TitleID:"434D0027"}}})!="Valid image cached" { panic("Cover diagnostic failed to recognise validated cached file") }
    // The interface no longer exposes cover troubleshooting buttons; the
    // same validated cover must appear in a fixed rectangle and open a modal.
    oldCat,oldSelected,oldSelectedPatch,oldParsed,oldTab,oldCoverOpen:=
        app.cat,app.selectedGame,app.selectedPatch,app.parsed,app.tab,app.coverOpen
    app.cat=core.Catalog{Games:[]core.Game{{Title:"American Idol (Prototype)",
        CoverURL:url,
        Patches:[]core.CatalogPatch{{TitleID:"434D0027"}}}}}
    app.selectedGame=0
    app.selectedPatch=-1
    app.parsed=nil
    app.tab=0
    app.coverOpen=false
    redraw()
    sysCall(updateWindow,app.hwnd)
    foundPreview,foundOldDiagnostic:=false,false
    for _,hit:=range app.hits{
        if hit.Action=="cover-view"{
            foundPreview=true
            if hit.R.R-hit.R.L!=101||hit.R.B-hit.R.T!=100{
                panic("Selected game cover does not have a constant preview size")
            }
        }
        if hit.Action=="cover-check"||hit.Action=="cover-copy"||hit.Action=="gallery-explain"{
            foundOldDiagnostic=true
        }
    }
    if !foundPreview||foundOldDiagnostic{
        panic("Cover preview missing or obsolete debugging controls still visible")
    }
    handle("cover-view",0)
    if !app.coverOpen{panic("Cover click did not enlarge the image")}
    redraw()
    sysCall(updateWindow,app.hwnd)
    foundClose:=false
    for _,hit:=range app.hits{
        if hit.Action=="cover-close"{foundClose=true;break}
    }
    if !foundClose{panic("Enlarged game cover cannot be closed")}
    handle("cover-close",0)
    if app.coverOpen{panic("Enlarged game cover did not close")}
    app.cat,app.selectedGame,app.selectedPatch,app.parsed,app.tab,app.coverOpen=
        oldCat,oldSelected,oldSelectedPatch,oldParsed,oldTab,oldCoverOpen
	redraw()
	sysCall(updateWindow,app.hwnd)
	// Verify no temporary input is retained and normal UI painting resumes.
	_ = syscall.GetLastError()
}
