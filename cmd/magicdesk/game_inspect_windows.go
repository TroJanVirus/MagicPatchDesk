//go:build windows

package main

import (
    "context"
    "magicpatchdesk/core"
    "path/filepath"
    "strings"
    "sync"
    "sync/atomic"
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

// Source selections are never scanned in separate unbounded goroutines.
// There is ONE reader for the entire application, plus one replaceable
// pending request. The previous scan receives cooperative cancellation.
type gameInspectionRequest struct {
    path string
    generation uint64
    hwnd uintptr
    ctx context.Context
}

var (
    inspectRequestCh=make(chan gameInspectionRequest,1)
    inspectorOnce sync.Once
    inspectCancel context.CancelFunc // owned by the Win32 UI thread
    inspectorStopped bool           // owned by the Win32 UI thread
    inspectorActive atomic.Int32
    inspectorMaximum atomic.Int32
    inspectorFinished atomic.Uint64
    inspectorCancelled atomic.Uint64
)

func inspectGamePathContext(ctx context.Context,path string)(core.XBEInfo,string,string,error){
    result,err:=core.InspectGameSourceContext(ctx,path)
    return result.Info,result.PartSummary,result.Kind,err
}
func inspectGamePath(path string)(core.XBEInfo,string,string,error){
    return inspectGamePathContext(context.Background(),path)
}

// Called only from the window thread. Canceling promptly releases CCI index
// allocations, source handles and in-progress XBE reads for older sources.
func queueGameInspection(path string,generation uint64,hwnd uintptr){
    if inspectorStopped{return}
    if inspectCancel!=nil{inspectCancel()}
    ctx,cancel:=context.WithCancel(context.Background())
    inspectCancel=cancel
    inspectorOnce.Do(func(){go runGameInspectionWorker()})
    request:=gameInspectionRequest{path:path,generation:generation,hwnd:hwnd,ctx:ctx}
    select{
    case inspectRequestCh<-request:
    default:
        // Only the newest pending file can matter. The in-flight worker
        // already has a canceled context; no additional worker is spawned.
        select{case <-inspectRequestCh:default:}
        inspectRequestCh<-request
    }
}
func stopGameInspections(){
    if inspectorStopped{return}
    inspectorStopped=true
    if inspectCancel!=nil{inspectCancel();inspectCancel=nil}
    // No further UI submissions are possible after WM_DESTROY.
    close(inspectRequestCh)
}

func recordInspectorConcurrency(){
    current:=inspectorActive.Add(1)
    for {
        previous:=inspectorMaximum.Load()
        if previous>=current||inspectorMaximum.CompareAndSwap(previous,current){break}
    }
}
func runGameInspectionWorker(){
    defer catchFatalPanic()
    for req:=range inspectRequestCh{
        if req.ctx.Err()!=nil{inspectorCancelled.Add(1);continue}
        recordInspectorConcurrency()
        info,summary,kind,err:=inspectGamePathContext(req.ctx,req.path)
        inspectorActive.Add(-1)
        if req.ctx.Err()!=nil{
            inspectorCancelled.Add(1)
            continue
        }
        result:=gameInspection{
            path:req.path,generation:req.generation,
            info:info,partSummary:summary,sourceKind:kind,err:err,
        }
        inspectorFinished.Add(1)
        select{
        case gameInspectReady<-result:
        default:
            select{case <-gameInspectReady:default:}
            select{case gameInspectReady<-result:default:}
        }
        if req.hwnd!=0{sysCall(postMsg,req.hwnd,WM_APP+4,0,0)}
    }
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
