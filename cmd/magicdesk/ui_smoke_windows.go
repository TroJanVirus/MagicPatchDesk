//go:build windows

package main

import (
    "fmt"
    "magicpatchdesk/core"
    "runtime"
    "strings"
    "syscall"
    "unsafe"
)

// smokePaintAndExit exercises all native workspaces and the common 720p-height
// Game Outputs layout. It is strictly opt-in via --mpd-ui-smoke-v1 and avoids
// all network sync, auto-reporting and patch/image modification.
func smokePaintAndExit() {
    // Validate the actual release executable, not just a mock UI drawing:
    // the complete native caption and official JMP .rsrc group icon must exist.
    caption:=make([]uint16,256)
    n:=sysCall(user32.NewProc("GetWindowTextW"),app.hwnd,
        uintptr(unsafe.Pointer(&caption[0])),uintptr(len(caption)))
    if n==0||syscall.UTF16ToString(caption[:n])!="Magic Patch Desk | Powered by Patch Hub"{
        panic("Window caption does not match the requested Patch Hub branding")
    }
    icon:=sysCall(user32.NewProc("LoadIconW"),sysCall(getModule,0),1)
    if icon==0{panic("Official JMP icon missing from compiled Windows EXE resources")}
    smokeVerifiedCoverDrawing()
    if app.tab!=9 {panic("Home must be the initial landing tab")}
    app.tab=9
    redraw()
    sysCall(updateWindow,app.hwnd)
    homeControls:=map[string]bool{
        "home-open-game":false,
        "home-library":false,
        "home-patchhub":false,
        "tab":false,
    }
    for _,item:=range app.hits {
        if _,ok:=homeControls[item.Action];ok{
            if item.Action!="tab"||item.Index==9{homeControls[item.Action]=true}
        }
    }
    for action,present:=range homeControls{
        if !present{panic("Home missing clickable action: "+action)}
    }
    handle("home-library",0)
    if app.tab!=0{panic("Home Browse patches does not navigate to Patch Library")}
    handle("home-convert",0)
    if app.tab!=1{panic("Home Convert Patch shortcut does not work")}

    for _,tab:=range []int{9,0,1,2,3,4,5,6,8}{
        app.tab=tab
        // Force collection while native text and paint paths are busy.
        // This helps expose unsafe Go-pointer-to-uintptr lifetimes.
        runtime.GC()
        redraw()
        sysCall(updateWindow,app.hwnd)
    }
    if sysCall(user32.NewProc("MoveWindow"),app.hwnd,55,55,1024,720,1)==0{
        panic("Win32 MoveWindow failed during UI smoke test")
    }
    // Every theme must retain four clickable colour swatches after resizing
    // to a 720p-height window. UI painting must not require network access.
    for _,theme:=range []string{"Midnight","Forest","Plum","Light"}{
        app.theme=theme
        setTheme(theme)
        updateWindowsTitleBarTheme()
        app.tab=2
        redraw()
        sysCall(updateWindow,app.hwnd)
        choices:=map[int]bool{}
        for _,hit:=range app.hits{
            if hit.Action=="theme" {
                choices[hit.Index]=true
                if hit.R.T<125||hit.R.B>app.h-42 {
                    panic(fmt.Sprintf("Theme %s colour chooser clipped (top=%d bottom=%d)",theme,hit.R.T,hit.R.B))
                }
            }
        }
        if len(choices)!=4 {panic(fmt.Sprintf("Theme %s lacks four clickable colour previews",theme))}
    }
    app.theme="Midnight"
    setTheme(app.theme)
    updateWindowsTitleBarTheme()
    // Source patch must be the FIRST numbered conversion step; the original
    // XBE belongs below it as step 02. The hit order catches UI regressions
    // without relying on brittle screenshot text OCR.
    app.tab=1
    redraw()
    sysCall(updateWindow,app.hwnd)
    sourceTop,originalTop:=int32(-1),int32(-1)
    for _,hit:=range app.hits{
        if hit.Action=="cpatch"{sourceTop=hit.R.T}
        if hit.Action=="a"{originalTop=hit.R.T}
    }
    if sourceTop<0||originalTop<0||sourceTop>=originalTop{
        panic(fmt.Sprintf("Converter source patch must precede original XBE (patch=%d original=%d)",sourceTop,originalTop))
    }

    // Regression: a library JMP containing only absolute offset writes
    // defaults to direct IPS (not a source-dependent all-formats pack).
    // Choosing BPS explicitly must preserve that selection and explain the
    // unavoidable original/target checksum requirement. Search/replace must
    // never be offered as a source-free IPS conversion.
    savedParsed,savedDirect:=app.parsed,app.directFile
    savedInput,savedModified,savedSelected:=app.convertPatchFile,app.bFile,app.aFile
    savedFormat,savedExplicit:=app.format,app.convertFormatChosen
    offsetJMP:=[]byte("#Jay's Magic Patcher\nsystem=Xbox\ngame-title=Smoke\nregion=\nversion=\nauthor=\nnotes=\n#Offset\n0x300:AABBCC\n")
    patch,e:=core.ParseJMP("offset.jmp",offsetJMP)
    if e!=nil{panic(fmt.Sprintf("Offset JMP smoke fixture invalid: %v",e))}
    app.parsed=patch
    app.directFile=""
    app.convertPatchFile=""
    app.bFile=""
    app.aFile=""
    app.convertFormatChosen=false
    app.format=core.AllPatchFormats
    handle("tab",1)
    if app.format!=1||app.conversionBlocked{
        panic(fmt.Sprintf("Library offset JMP not source-free IPS: format=%d blocked=%t reason=%q",app.format,app.conversionBlocked,app.conversionHint))
    }
    handle("format",4)
    if !app.convertFormatChosen||!app.conversionBlocked||!strings.Contains(app.conversionHint,"BPS"){
        panic("BPS incorrectly offered source-free or missing original checksum explanation")
    }
    handle("tab",0)
    handle("tab",1)
    if app.format!=4 {panic("Explicit BPS format was overwritten by library tab navigation")}
    searchJMP:=[]byte("#Jay's Magic Patcher\nsystem=Xbox\ngame-title=Smoke\nregion=\nversion=\nauthor=\nnotes=\n#Find\n414243\n444546\n")
    searchPatch,e:=core.ParseJMP("search.jmp",searchJMP)
    if e!=nil{panic(fmt.Sprintf("Search JMP smoke fixture invalid: %v",e))}
    app.parsed=searchPatch
    handle("format",1)
    if !app.conversionBlocked||!strings.Contains(strings.ToLower(app.conversionHint),"search-and-replace"){
        panic("Search-and-replace JMP did not show reason for blocked source-free IPS")
    }
    app.parsed,app.directFile=savedParsed,savedDirect
    app.convertPatchFile,app.bFile,app.aFile=savedInput,savedModified,savedSelected
    app.format,app.convertFormatChosen=savedFormat,savedExplicit

    // Reproduce reported sequence: select a real library patch first, then
    // choose its matching XBE. The parsed patch and record toggles remain
    // intact; an older inspector result must be dropped.
    smokePreselectedPatchThenXBE()
    smokeISOPickerValidation()

    // Test the real Win32 button hit rectangles and Segoe UI text metrics at
    // the usual 1024px workspace and the narrower stacked-settings layout.
    for _,windowWidth:=range []int32{1024,800} {
        if sysCall(user32.NewProc("MoveWindow"),app.hwnd,55,55,uintptr(windowWidth),720,1)==0 {
            panic("Cannot resize Maintenance smoke-test window")
        }
        app.tab=6
        redraw()
        sysCall(updateWindow,app.hwnd)
        required:=map[string]string{
            "maint-beta":"Previews on",
            "maint-autocheck":"Check + sync at start",
            "maint-program":"App folder",
            "maint-data":"Data folder",
        }
        if !maintenancePref.Beta{required["maint-beta"]="Stable only"}
        if maintenancePref.NoAutoCheck{required["maint-autocheck"]="Manual check + sync"}
        smokeDC:=sysCall(user32.NewProc("GetDC"),app.hwnd)
        if smokeDC==0{panic("Cannot measure native Maintenance button text")}
        bounds:=map[string]rect{}
        for _,hit:=range app.hits{
            if label,ok:=required[hit.Action];ok {
                bounds[hit.Action]=hit.R
                // The compact fallback is the smallest font the button code
                // actually uses; if it still cannot fit, reflow is required.
                if textPixelWidth(smokeDC,label,app.fonts.ButtonTiny)>hit.R.R-hit.R.L-20 {
                    panic(fmt.Sprintf("Maintenance caption clipped at %dpx: %s = %q",windowWidth,hit.Action,label))
                }
            }
        }
        sysCall(user32.NewProc("ReleaseDC"),app.hwnd,smokeDC)
        for name:=range required {
            if _,ok:=bounds[name];!ok{panic("Missing Maintenance control "+name)}
        }
        for a,ra:=range bounds{
            for b,rb:=range bounds{
                if a>=b{continue}
                if ra.L<rb.R&&ra.R>rb.L&&ra.T<rb.B&&ra.B>rb.T {
                    panic(fmt.Sprintf("Maintenance controls overlap at %dpx: %s and %s",windowWidth,a,b))
                }
            }
        }
    }
    if sysCall(user32.NewProc("MoveWindow"),app.hwnd,55,55,1024,720,1)==0 {
        panic("Cannot restore Win32 smoke-test window")
    }
    app.tab=0
    if chosenGameSource()=="" {
        help:=outputSummary()
        if !strings.Contains(strings.ToLower(help),"formats")||strings.Contains(help,"source game") {
            panic("OUTPUTS prompt does not explain format selection")
        }
    }

    // Long JMP comments must never be accessible solely through a clipped
    // group-row label. The Read hit opens an independent, scrollable modal
    // with complete comments and trusted, explicitly opened HTTP(S) URLs.
    oldCatalog,oldSelectedGame,oldSelectedPatch:=app.cat,app.selectedGame,app.selectedPatch
    oldParsed,oldExpanded:=app.parsed,app.expandedGroup
    oldNotesTarget,oldNotesLink,oldNotesOpen:=app.notesTargetGroup,app.notesLinkIndex,app.notesOpen
    longComment:=`Widescreen Support (Everything is less stretched, and the HUD is centred correctly) Please read these important setup instructions before applying the patch. <a href="https://example.org/patch-guide?game=phantom&amp;mode=wide">Detailed installation guide</a>`
    fixture:=`#Jay's Magic Patcher
system=Xbox
game-title=UI fixture
region=NTSC
version=
author=QA
notes=Patch notes <a href="https://example.org/notes">notes link</a>
#`+longComment+`
0x300:0102
`
    viewerPatch,err:=core.ParseJMP("viewer-fixture.jmp",[]byte(fixture))
    if err!=nil{panic(fmt.Sprintf("Invalid comment viewer fixture: %v",err))}
    app.cat=core.Catalog{Games:[]core.Game{{
        Title:"Viewer Fixture",
        Patches:[]core.CatalogPatch{{Name:"Widescreen test",Type:"Widescreen",Author:"QA"}},
    }}}
    app.selectedGame=0
    app.selectedPatch=0
    app.parsed=viewerPatch
    app.expandedGroup=-1
    app.tab=0
    redraw()
    sysCall(updateWindow,app.hwnd)
    readFound,toggleFound:=false,false
    for _,hit:=range app.hits{
        if hit.Action=="group"&&hit.Index==0{toggleFound=true}
        if hit.Action=="group-details"&&hit.Index==0{
            readFound=true
            click,_:=mouseHit(hit.R.L+5,hit.R.T+5)
            if click.Action!="group-details"{panic("Read action is covered by group-toggle hitbox")}
        }
    }
    if !readFound||!toggleFound{panic("Group must offer separate toggle and Read details controls")}
    beforeEnabled:=viewerPatch.Groups[0].Enabled
    handle("group-details",0)
    if !app.notesOpen||app.notesTargetGroup!=0||
        !strings.Contains(selectedPatchNoteText(),"Detailed installation guide")||
        !strings.Contains(selectedPatchNoteText(),"Please read these important setup instructions")||
        len(selectedPatchNoteLinks())!=1{
        panic("Full JMP group details or its external link were not available")
    }
    if viewerPatch.Groups[0].Enabled!=beforeEnabled{panic("Opening group details accidentally toggled patch")}
    redraw()
    sysCall(updateWindow,app.hwnd)
    modalActions:=map[string]bool{"notes-copy":false,"notes-link-open":false,"notes-link-copy":false,"notes-close":false}
    for _,hit:=range app.hits{if _,ok:=modalActions[hit.Action];ok{modalActions[hit.Action]=true}}
    for key,ok:=range modalActions{if !ok{panic("Comment viewer missing action "+key)}}
    handle("notes-close",0)
    handle("notes-open",0)
    if app.notesTargetGroup!=-1||len(selectedPatchNoteLinks())!=1{
        panic("Read Full Notes did not return to notes= header links")
    }
    handle("notes-close",0)
    app.cat,app.selectedGame,app.selectedPatch=oldCatalog,oldSelectedGame,oldSelectedPatch
    app.parsed,app.expandedGroup=oldParsed,oldExpanded
    app.notesTargetGroup,app.notesLinkIndex,app.notesOpen=oldNotesTarget,oldNotesLink,oldNotesOpen

    // A standalone Game Outputs tab must NOT be navigable: its previous
    // single-game functionality now belongs to Patch Library.
    app.tab=0
    redraw()
    sysCall(updateWindow,app.hwnd)
    for _,hit:=range app.hits{
        if hit.Action=="tab"&&hit.Index==7{panic("Legacy Game Outputs tab still appears")}
    }
    // Settings and About must be anchored to the bottom even in 720p height.
    navBottom:=map[int]bool{2:false,4:false,8:false}
    for _,hit:=range app.hits{
        if hit.Action=="tab" {
            if _,ok:=navBottom[hit.Index];ok{
                if hit.Index==2||hit.Index==4{
                    if hit.R.T<app.h-150{panic("Settings/About not bottom-anchored")}
                }
                navBottom[hit.Index]=true
            }
        }
    }
    for index,exists:=range navBottom{if !exists{panic(fmt.Sprintf("Missing sidebar tab %d",index))}}

    // Explicit format gate: merely selecting a game and a patch is not enough.
    app.input="SyntheticGame.xiso"
    selectGameSource(app.input)
    // Model the successful asynchronous ISO validation explicitly. The
    // output chooser must remain locked while the real scan is pending.
    if gameOutputValidated||availableGameOutputs()!=0{
        panic("XISO exposed outputs before completed filesystem verification")
    }
    setGameOutputValidation(app.input,"xiso",true)
    app.parsed=&core.Patch{Headers:map[string]string{"notes":"safe synthetic patch"}}
    redraw()
    sysCall(updateWindow,app.hwnd)
    applyExists,chooserExists,folderExists:=false,false,false
    for _,hit:=range app.hits{
        if hit.Action=="apply"{applyExists=true}
        if hit.Action=="output-open"{chooserExists=true}
        if hit.Action=="game-folder"{folderExists=true}
    }
    if applyExists||!chooserExists||!folderExists{
        panic(fmt.Sprintf("Patch Library output gate invalid before choice (apply=%t chooser=%t folder=%t)",applyExists,chooserExists,folderExists))
    }
    app.outputPickerOpen=true
    redraw()
    sysCall(updateWindow,app.hwnd)
    preserveExists,options,isoDisabled:=false,0,true
    for _,hit:=range app.hits{
        if hit.Action=="game-preserve"{preserveExists=true}
        if hit.Action=="game-format"{
            options++
            if hit.Index==4{isoDisabled=false}
        }
    }
    if !preserveExists||options!=4||!isoDisabled{
        panic(fmt.Sprintf("Multi-select output chooser lacks preserved format/source constraints (preserve=%t options=%d fulliso-disabled=%t)",preserveExists,options,isoDisabled))
    }
    app.outputPickerOpen=false
    gameOutputMask=core.GameOutputXISO|core.GameOutputCCI
    gameOutputPreserve=true
    redraw()
    sysCall(updateWindow,app.hwnd)
    applyExists=false
    for _,hit:=range app.hits{if hit.Action=="apply"{applyExists=true}}
    if !applyExists||!libraryApplyReady(){panic("Apply Patch remains disabled after explicit valid multi-format selection")}
    app.input="SyntheticOther.xbe"
    selectGameSource(app.input)
    if gameOutputMask!=0||gameOutputPreserve||libraryApplyReady(){
        panic("New source inherited prior output choice without explicit consent")
    }

    // Conditional diagnostic-connection controls: Connect only when the
    // Windows Credential Manager connection is absent.
    githubConnected.Store(false)
    app.tab=6
    redraw()
    sysCall(updateWindow,app.hwnd)
    controls:=map[string]bool{}
    for _,hit:=range app.hits{
        if strings.HasPrefix(hit.Action,"maint-"){controls[hit.Action]=true}
    }
    if !controls["maint-connect"]||controls["maint-disconnect"]||controls["maint-reports"]||controls["maint-sync"]{
        panic("Disconnected Maintenance exposes private reporting actions")
    }
    githubConnected.Store(true)
    redraw()
    sysCall(updateWindow,app.hwnd)
    controls=map[string]bool{}
    for _,hit:=range app.hits{
        if strings.HasPrefix(hit.Action,"maint-"){controls[hit.Action]=true}
    }
    if controls["maint-connect"]||!controls["maint-disconnect"]||!controls["maint-reports"]||!controls["maint-sync"]{
        panic("Connected Maintenance lacks conditional diagnostic actions")
    }
    githubConnected.Store(false)

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
    checkGalleryActions:=func(wantPrev,wantNext bool){
        prev,next,closeButton,disabled:=false,false,false,0
        for _,hit:=range app.hits{
            if hit.Action=="gallery-prev"{prev=true}
            if hit.Action=="gallery-next"{next=true}
            if hit.Action=="gallery-close"{closeButton=true}
            if hit.Action=="gallery-disabled"{disabled++}
            if strings.HasPrefix(hit.Action,"gallery-") && hit.R.B>app.h-42 {
                panic("Patch gallery control clipped below app footer")
            }
        }
        if prev!=wantPrev||next!=wantNext||!closeButton||disabled!=1{
            panic(fmt.Sprintf("Gallery boundary buttons wrong: prev=%t next=%t disabled=%d close=%t",prev,next,disabled,closeButton))
        }
    }
    // Initial page: Previous is disabled and non-clickable.
    checkGalleryActions(false,true)
    handle("gallery-disabled",0)
    if !app.galleryOpen||app.galleryPage!=0{panic("Disabled Previous closed or paged the gallery")}
    handle("gallery-next",0)
    if app.galleryPage!=1{panic("Multi-image gallery cannot page to the second image pair")}
    redraw()
    sysCall(updateWindow,app.hwnd)
    checkGalleryActions(true,false)
    handle("gallery-disabled",0)
    if !app.galleryOpen||app.galleryPage!=1{panic("Disabled Next closed or paged the gallery")}
    handle("gallery-next",0)
    if app.galleryPage!=1{panic("Gallery paging exceeded last image pair")}
    handle("gallery-prev",0)
    if app.galleryPage!=0{panic("Gallery Previous failed to return to first image pair")}
    handle("gallery-close",0)
    if app.galleryOpen{panic("Patch gallery did not close")}
    // Same-patch click must deselect it (and restore the empty instructions)
    // without unloading the currently chosen game.
    sourceBefore:=chosenGameSource()
    handle("patch",0)
    if app.selectedPatch!=-1||app.parsed!=nil||app.selectedGame!=0||
        chosenGameSource()!=sourceBefore||app.galleryOpen{
        panic("Second patch click failed to return to unselected patch state")
    }
    // Restore synthetic patch for the remaining independent-scroll tests.
    app.selectedPatch=0
    app.parsed=&core.Patch{Headers:map[string]string{"author":"Test"}}

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
    // Final Windows-only stress suite uses real on-disk ISO/CCI/XBE fixtures
    // and switches each source hundreds of times with a patch selected.
    smokeRapidMixedSourceSwitch()
    sysCall(user32.NewProc("DestroyWindow"),app.hwnd)
}
