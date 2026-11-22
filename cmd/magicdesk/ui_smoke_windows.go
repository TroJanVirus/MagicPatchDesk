//go:build windows

package main

import (
    "fmt"
    "magicpatchdesk/core"
    "strings"
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
    // Exercise the two separately scrollable regions and long formatted notes.
    app.cat.Games[0].Patches=append(app.cat.Games[0].Patches,
        core.CatalogPatch{Name:"Variant B"},core.CatalogPatch{Name:"Variant C"},
        core.CatalogPatch{Name:"Variant D"},core.CatalogPatch{Name:"Variant E"},
        core.CatalogPatch{Name:"Variant F"},core.CatalogPatch{Name:"Variant G"})
    for i:=0;i<14;i++{
        name:=fmt.Sprintf("Record group %d",i)
        app.parsed.Groups=append(app.parsed.Groups,core.Group{Name:name,Count:1,Enabled:true})
        app.parsed.Records=append(app.parsed.Records,core.Record{Kind:core.Offset,Group:name,Offset:int64(i*4),Data:[]byte{1},Enabled:true})
    }
    app.parsed.Headers["notes"]="Translation Mod.<br><br>"+strings.Repeat("Step: apply to your original game file.<br>",100)
    redraw()
    sysCall(updateWindow,app.hwnd)
    if app.patchBar.Max<=0||app.recordBar.Max<=0{
        panic(fmt.Sprintf("Independent patch and record scrollbars missing: patch=%d records=%d",app.patchBar.Max,app.recordBar.Max))
    }
    scrollbarHits:=map[string]bool{"patch-scroll-thumb":false,"records-scroll-thumb":false,"notes-open":false}
    for _,hit:=range app.hits{
        if _,ok:=scrollbarHits[hit.Action];ok{scrollbarHits[hit.Action]=true}
    }
    for k,v:=range scrollbarHits{if !v{panic("Missing patch/record scrollbar or full-note button: "+k)}}
    app.patchScroll=0
    app.groupScroll=0
    changeScrollByWheel("records",-120)
    oldRecords:=app.groupScroll
    if oldRecords==0||app.patchScroll!=0{panic("Record wheel moved both sections")}
    changeScrollByWheel("patch",-120)
    if app.patchScroll==0||app.groupScroll!=oldRecords{panic("Patch wheel moved record section")}
    handle("notes-open",0)
    redraw()
    sysCall(updateWindow,app.hwnd)
    if !app.notesOpen||app.notesBar.Max<=0{panic("Full formatted JMP notes are not scrollable")}
    notesControls:=map[string]bool{"notes-copy":false,"notes-close":false,"notes-scroll-thumb":false}
    for _,hit:=range app.hits{
        if _,ok:=notesControls[hit.Action];ok{notesControls[hit.Action]=true}
    }
    for name,v:=range notesControls{if !v{panic("Full JMP notes viewer missing "+name)}}
    changeScrollByWheel("notes",-120)
    if app.notesScroll==0{panic("Full notes cannot scroll")}
    handle("notes-close",0)
    if app.notesOpen{panic("Notes modal did not close")}
    // Converter controls must be relevant to the chosen patch format.
    app.tab=1
    app.convertPatchFile=""
    app.directFile=""
    for format:=0;format<6;format++{
        app.format=format
        redraw()
        sysCall(updateWindow,app.hwnd)
        wantPassword:=format==3||format==5
        wantRLE:=format==1||format==3||format==5
        hasPassword,hasRLE:=false,false
        for _,hit:=range app.hits{
            if hit.Action=="password"{hasPassword=true}
            if hit.Action=="ipsrle"{hasRLE=true}
        }
        if hasPassword!=wantPassword||hasRLE!=wantRLE{
            panic(fmt.Sprintf("Unrelated converter IPS controls shown for format %d (password=%t RLE=%t)",format,hasPassword,hasRLE))
        }
    }
    app.format=0
    app.convertPatchFile="encrypted.eips"
    redraw()
    sysCall(updateWindow,app.hwnd)
    decryptPassword:=false
    for _,hit:=range app.hits{if hit.Action=="password"{decryptPassword=true}}
    if !decryptPassword{panic("Encrypted IPS input cannot provide required decryption password")}
    sysCall(user32.NewProc("DestroyWindow"),app.hwnd)
}
