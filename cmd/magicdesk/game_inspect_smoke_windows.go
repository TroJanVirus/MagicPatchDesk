//go:build windows

package main

import (
    "magicpatchdesk/core"
    "fmt"
)

// This reproduces the reported order of user actions in memory:
// choose a library JMP first, then select the matching original XBE. The
// selected JMP and any individually toggled records must survive matching.
// Older builds unconditionally reselected and reparsed the JMP after the XBE
// had loaded, while inspection also modified UI-owned fields from a worker.
func smokePreselectedPatchThenXBE(){
    oldCatalog,oldGame,oldPatch:=app.cat,app.selectedGame,app.selectedPatch
    oldParsed,oldInput,oldInfo:=app.parsed,app.input,app.inputInfo
    oldStatus,oldGeneration:=app.status,app.gameInputGeneration
    defer func(){
        app.cat,app.selectedGame,app.selectedPatch=oldCatalog,oldGame,oldPatch
        app.parsed,app.input,app.inputInfo=oldParsed,oldInput,oldInfo
        app.status,app.gameInputGeneration=oldStatus,oldGeneration
    }()

    selected:=&core.Patch{
        Headers:map[string]string{"author":"QA","notes":"Do not change toggles"},
        Groups:[]core.Group{{Name:"Widescreen",Count:1,Enabled:false}},
        Records:[]core.Record{{Kind:core.Offset,Group:"Widescreen",Enabled:false,Offset:0x300,Data:[]byte{1}}},
    }
    app.cat=core.Catalog{Games:[]core.Game{{
        Title:"Phantom Crash",
        Patches:[]core.CatalogPatch{{
            Name:"{Widescreen} (NTSC) [Jay]",
            TitleID:"504C0001",CRC:"1A09C9A8",
            File:"nonexistent-for-selected-patch-test.jmp",
        }},
    }}}
    app.selectedGame=0
    app.selectedPatch=0
    app.parsed=selected
    info:=core.XBEInfo{Title:"Phantom Crash",TitleID:"504C0001",CRC32:"1A09C9A8",Valid:true}
    autoSelectXBE(info)
    if app.selectedGame!=0||app.selectedPatch!=0||app.parsed!=selected||
        app.parsed.Groups[0].Enabled||app.parsed.Records[0].Enabled{
        panic("Selecting matching XBE reparsed or changed already selected JMP")
    }
    // Delayed workers may finish after the user has picked another XBE.
    // The token/path must reject those results without altering UI state.
    app.gameInputGeneration=43
    app.input="currently-selected.xbe"
    oldInfoSnapshot:=app.inputInfo
    oldStatusSnapshot:=app.status
    commitGameInspection(gameInspection{
        generation:42,path:"previously-selected.xbe",info:info,
    })
    if app.inputInfo!=oldInfoSnapshot||app.status!=oldStatusSnapshot||
        app.parsed!=selected{
        panic(fmt.Sprintf("Stale game inspection changed UI state: %+v",app.inputInfo))
    }
}
