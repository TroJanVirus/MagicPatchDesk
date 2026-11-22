//go:build windows

package main

import (
    "magicpatchdesk/core"
    "errors"
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
    sourceKind string
    err error
}

func inspectGamePath(path string)(core.XBEInfo,string,string,error){
    var info core.XBEInfo
    var err error
    var partSummary string
    kind:=""
    if core.CCISourcePath(path){
        kind="cci"
        partSummary,err=core.CCIPartSummary(path)
        if err==nil{info,err=core.InspectCCIExecutable(path)}
        return info,partSummary,kind,err
    }
    if core.IsNumberedRawImagePart(path) {
        partSummary,err=core.RawImagePartSummary(path)
        if err!=nil{return info,partSummary,kind,err}
    }
    var data []byte
    if strings.EqualFold(filepath.Ext(path),".xbe"){
        kind="xbe"
        var st os.FileInfo
        st,err=os.Stat(path)
        if err==nil{
            switch{
            case !st.Mode().IsRegular():
                err=errors.New("selected XBE is not a regular file")
            case st.Size()>512<<20:
                err=errors.New("selected XBE exceeds the 512 MiB safety limit")
            case st.Size()<0x200:
                err=errors.New("selected XBE is too small to contain a valid header")
            }
        }
        if err==nil{data,err=os.ReadFile(path)}
    }else{
        var image *core.Image
        image,err=core.OpenXISO(path)
        if err==nil{
            kind="xiso"
            if image.Base!=0&&!core.IsNumberedRawImagePart(path){kind="fulliso"}
            var entry core.ImageFile
            entry,err=image.DefaultXBE()
            if err==nil{data,err=image.ReadFile(entry)}
        }
    }
    if err==nil{info,err=core.InspectXBE(data)}
    return info,partSummary,kind,err
}

// Output permissions are derived only from a successfully parsed source.
// This helper never touches disk: the file dialog, painting and output
// chooser cannot accidentally scan a multi-gigabyte ISO on the GUI thread.
func setGameOutputValidation(path,kind string,valid bool){
    selected:=chosenGameSource()
    same:=selected==path
    folder:=strings.EqualFold(filepath.Base(path),"default.xbe")&&
        selected==filepath.Dir(path)
    if !same&&!folder{return}
    gameOutputKind=""
    gameOutputValidated=false
    if !valid{return}
    if folder{gameOutputKind="folder"}else{gameOutputKind=kind}
    gameOutputValidated=gameOutputKind!=""
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
    app.inputKind=""
    setGameOutputValidation(result.path,result.sourceKind,result.err==nil)
    if result.err!=nil{
        app.inputErr=result.err.Error()
        app.status="Cannot identify game: "+result.err.Error()
    }else{
        app.inputKind=result.sourceKind
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
