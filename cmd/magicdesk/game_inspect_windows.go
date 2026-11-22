//go:build windows

package main

import (
    "magicpatchdesk/core"
    "os"
    "path/filepath"
    "strings"
)

// gameInspection is an immutable worker result. Background file readers must
// never assign app.inputInfo/status or touch any GDI/selection state: all such
// mutations belong to the owner Win32 UI thread.
type gameInspection struct {
    path string
    generation uint64
    info core.XBEInfo
    partSummary string
    err error
}

func inspectGamePath(path string)(core.XBEInfo,string,error){
    var info core.XBEInfo
    var err error
    var partSummary string
    if core.CCISourcePath(path){
        partSummary,err=core.CCIPartSummary(path)
        if err==nil{info,err=core.InspectCCIExecutable(path)}
        return info,partSummary,err
    }
    if core.IsNumberedRawImagePart(path) {
        partSummary,err=core.RawImagePartSummary(path)
        if err!=nil{return info,partSummary,err}
    }
    var data []byte
    if strings.EqualFold(filepath.Ext(path),".xbe"){
        data,err=os.ReadFile(path)
    }else{
        var image *core.Image
        image,err=core.OpenXISO(path)
        if err==nil{
            var entry core.ImageFile
            entry,err=image.DefaultXBE()
            if err==nil{data,err=image.ReadFile(entry)}
        }
    }
    if err==nil{info,err=core.InspectXBE(data)}
    return info,partSummary,err
}

// The event handler calls this only on the Win32 UI thread, after dropping any
// stale result from a previous file picker operation. The existing selection
// is retained when it already belongs to the same matched game.
func commitGameInspection(result gameInspection) {
    if result.generation!=app.gameInputGeneration||result.path!=app.input{return}
    markCrashStage("game_inspect_commit")
    defer markCrashStage("ui_callback")
    app.mu.Lock()
    app.inputInfo=result.info
    app.inputPartSummary=result.partSummary
    app.inputErr=""
    if result.err!=nil{
        app.inputErr=result.err.Error()
        app.status="Cannot identify game: "+result.err.Error()
    }else{
        app.status="Loaded: "+result.info.Title+" · "+result.info.TitleID+" · CRC32 "+result.info.CRC32
        if result.partSummary!=""{app.status+=" · "+result.partSummary}
    }
    app.mu.Unlock()
    if result.err!=nil{
        recordFailure("game.inspect",result.err,result.path,"","")
        redraw()
        return
    }
    rememberRecent("input",result.path)
    autoSelectXBE(result.info)
    redraw()
}
