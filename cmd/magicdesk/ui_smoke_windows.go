//go:build windows

package main

import (
    "fmt"
)

// smokePaintAndExit exercises all native workspaces and the common 720p-height
// Game Outputs layout. It is strictly opt-in via --mpd-ui-smoke-v1 and avoids
// all network sync, auto-reporting and patch/image modification.
func smokePaintAndExit() {
    for _,tab:=range []int{0,1,2,3,4,5,6,7,8}{
        app.tab=tab
        redraw()
        sysCall(updateWindow,app.hwnd)
    }
    // Simulate a 1024x720 outer window, including its non-client borders.
    if sysCall(user32.NewProc("MoveWindow"),app.hwnd,55,55,1024,720,1)==0 {
        panic("Win32 MoveWindow failed during UI smoke test")
    }
    app.tab=7
    redraw()
    sysCall(updateWindow,app.hwnd)
    var exportVisible,formatVisible,sourceVisible bool
    for _,h:=range app.hits{
        switch h.Action{
        case "game-export":
            exportVisible=true
            if h.R.B>app.h-42 || h.R.T<100{
                panic(fmt.Sprintf("Game Outputs export button clipped: bottom=%d client=%d",h.R.B,app.h))
            }
        case "game-format":formatVisible=true
        case "game-source":sourceVisible=true
        }
    }
    if !exportVisible || !formatVisible || !sourceVisible {
        panic(fmt.Sprintf("Game Outputs lacks clickable controls (export=%t format=%t source=%t)",exportVisible,formatVisible,sourceVisible))
    }
    app.tab=8
    redraw()
    sysCall(updateWindow,app.hwnd)
    var scanAvailable,modeAvailable bool
    for _,hit:=range app.hits{
        if hit.Action=="manager-scan"{scanAvailable=true}
        if hit.Action=="manager-mode"{modeAvailable=true}
    }
    if !scanAvailable||!modeAvailable{
        panic(fmt.Sprintf("Game manager navigation failed (scan=%t tabs=%t)",scanAvailable,modeAvailable))
    }
    sysCall(user32.NewProc("DestroyWindow"),app.hwnd)
}
