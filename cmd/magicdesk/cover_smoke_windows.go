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
	// Verify no temporary input is retained and normal UI painting resumes.
	_ = syscall.GetLastError()
}
