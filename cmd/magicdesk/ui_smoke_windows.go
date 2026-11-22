//go:build windows

package main

import (
    "fmt"
    "magicpatchdesk/core"
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
    for mode:=0;mode<5;mode++{
        manager.mu.Lock();manager.mode=mode;manager.mu.Unlock()
        redraw()
        sysCall(updateWindow,app.hwnd)
        var scanAvailable,modeAvailable bool
        for _,hit:=range app.hits{
            if hit.Action=="manager-scan"{scanAvailable=true}
            if hit.Action=="manager-mode"{modeAvailable=true}
        }
        if !scanAvailable||!modeAvailable{
            panic(fmt.Sprintf("Game manager mode %d missing native controls (scan=%t tabs=%t)",mode,scanAvailable,modeAvailable))
        }
        if mode==4 {
            controls:=map[string]bool{
                "manager-confidence-revision":false,
                "manager-confidence-ram":false,
                "manager-confidence-cpu":false,
                "manager-confidence-hardware":false,
                "manager-confidence-xemu":false,
            }
            for _,hit:=range app.hits{
                if _,ok:=controls[hit.Action];ok{
                    controls[hit.Action]=true
                    minimumTop:=int32(310)
                    if hit.Action=="manager-confidence-revision"||
                        hit.Action=="manager-confidence-ram"||
                        hit.Action=="manager-confidence-cpu"{
                        minimumTop=380
                    }
                    if hit.R.T<minimumTop||hit.R.B>app.h-42{
                        panic(fmt.Sprintf("Field-report choice %s clipped at 1024x720 (top=%d bottom=%d)",hit.Action,hit.R.T,hit.R.B))
                    }
                }
            }
            for name,exists:=range controls{
                if !exists{panic(fmt.Sprintf("Field-report choice %s missing or unclickable",name))}
            }
            // The same renderer must accommodate known upgraded hardware and
            // an explicit "I don't know" default, without relying on a game.
            manager.mu.Lock()
            manager.xboxRevision="1.6b"
            manager.ramUpgrade="upgraded"
            manager.cpuUpgrade="upgraded"
            manager.mu.Unlock()
            redraw()
            sysCall(updateWindow,app.hwnd)
            manager.mu.Lock()
            manager.xboxRevision="unknown"
            manager.ramUpgrade="unknown"
            manager.cpuUpgrade="unknown"
            manager.mu.Unlock()
        }

    }
    manager.mu.Lock();manager.mode=0;manager.mu.Unlock()
    // Test a multi-image PatchHub patch at the same 1024x720 bounds without
    // performing any network fetch or using copyrighted game/screenshot data.
    app.cat=core.Catalog{Games:[]core.Game{{Title:"Synthetic PatchHub Game",
        Patches:[]core.CatalogPatch{{Name:"Synthetic 720p JMP",
            Gallery:[]core.PatchGalleryImage{
                {Kind:"before",Local:"C:\\missing-gallery-before.png"},
                {Kind:"after",Local:"C:\\missing-gallery-after.png"},
                {Kind:"before",Local:"C:\\missing-gallery-before2.png"},
            },
        }},
    }}}
    app.selectedGame=0
    app.selectedPatch=0
    app.parsed=&core.Patch{Headers:map[string]string{"author":"Test"}}
    app.tab=0
    app.galleryOpen=false
    redraw()
    sysCall(updateWindow,app.hwnd)
    foundButton:=false
    for _,hit:=range app.hits{
        if hit.Action=="gallery-open"{
            foundButton=true
            if hit.R.B>app.h-42{panic("Patch Library gallery button clipped")}
        }
    }
    if !foundButton{panic("Patch Library screenshot viewer button missing")}
    app.galleryOpen=true
    redraw()
    sysCall(updateWindow,app.hwnd)
    modal:=map[string]bool{"gallery-close":false,"gallery-prev":false,"gallery-next":false}
    for _,hit:=range app.hits{
        if _,ok:=modal[hit.Action];ok{
            modal[hit.Action]=true
            if hit.R.B>app.h-42{panic("Patch gallery navigation clipped below app footer")}
        }
    }
    for action,present:=range modal{
        if !present{panic("Patch gallery missing action "+action)}
    }
    handle("gallery-next",0)
    if app.galleryPage!=1{panic("Multi-image gallery cannot page to the second image pair")}
    handle("gallery-close",0)
    if app.galleryOpen{panic("Patch gallery did not close")}
    sysCall(user32.NewProc("DestroyWindow"),app.hwnd)
}
