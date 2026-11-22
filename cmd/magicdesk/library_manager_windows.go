//go:build windows

package main

import (
    "context"
    "crypto/rand"
    "encoding/hex"
    "fmt"
    "magicpatchdesk/core"
    "os"
    "path/filepath"
    "sort"
    "strings"
    "sync"
    "syscall"
    "time"
    "unsafe"
)

type managerModel struct {
    mu sync.Mutex
    root string
    scan core.LibraryScan
    jobs []core.BatchJob
    selected int
    patchIndex int
    mode int // games, browse, hex, verification, confidence
    scroll int
    logScroll int
    browsePath string
    browse []core.ImageBrowserEntry
    browseSelected int
    browseScroll int
    overlapping core.OverlapReport
    overlapError string
    overlapScroll int
    confidence []core.ConfidenceReport
    confidenceScroll int
    reportResult string
    xboxRevision string
    ramUpgrade string
    cpuUpgrade string
    scanning bool
    running bool
    status string
    log *core.VerificationLog
}
var manager = managerModel{selected:-1,browseSelected:-1,reportResult:"working",xboxRevision:"unknown",ramUpgrade:"unknown",cpuUpgrade:"unknown",log:core.NewVerificationLog(700)}

type browseInfo struct {
    Owner uintptr
    Root uintptr
    DisplayName *uint16
    Title *uint16
    Flags uint32
    Callback uintptr
    Data uintptr
    Image int32
}
func chooseDirectory(title string) string {
    var display [260]uint16
    info:=browseInfo{Owner:app.hwnd,DisplayName:&display[0],Title:ptr(title),Flags:0x1|0x40}
    folder:=sysCall(shell32.NewProc("SHBrowseForFolderW"),uintptr(unsafe.Pointer(&info)))
    if folder==0{return ""}
    defer sysCall(syscallOle32.NewProc("CoTaskMemFree"),folder)
    var output [32768]uint16
    if sysCall(shell32.NewProc("SHGetPathFromIDListW"),folder,uintptr(unsafe.Pointer(&output[0])))==0{return ""}
    for i,ch:=range output{if ch==0{return string(utf16Decode(output[:i]))}}
    return ""
}
// This uses Win32 SHBrowseForFolder on Windows 10/11; no directory is modified.
var syscallOle32 = syscallLazyOLE()
func syscallLazyOLE() *syscall.LazyDLL{return syscall.NewLazyDLL("ole32.dll")}
func utf16Decode(v []uint16)[]rune {return []rune(syscall.UTF16ToString(v))}

func defaultBatchMask(g core.LibraryGame)uint32{
    switch g.Kind{
    case "folder":return core.GameOutputFolder
    case "cci":return core.GameOutputCCI
    case "fulliso":return core.GameOutputXISO
    default:return core.GameOutputXISO
    }
}
func chosenManagerPatch(g core.LibraryGame,index int)string{
    if len(g.Patches)==0{return ""}
    if index<0||index>=len(g.Patches){index=0}
    return g.Patches[index].File
}
func managerSelected()(core.BatchJob,int,bool){
    manager.mu.Lock()
    defer manager.mu.Unlock()
    if manager.selected<0||manager.selected>=len(manager.jobs){return core.BatchJob{},0,false}
    return manager.jobs[manager.selected],manager.patchIndex,true
}
func managerScan(){
    root:=chooseDirectory("Select a folder containing extracted Original Xbox games and XISO/CCI files")
    if root==""{return}
    manager.mu.Lock()
    if manager.scanning||manager.running{manager.mu.Unlock();return}
    manager.scanning=true
    manager.root=root
    manager.status="Scanning the game library (read only)…"
    manager.mu.Unlock()
    manager.log.Add("info","Library scan","Inspecting source formats and patch matches")
    redraw()
    go func(){
        app.mu.Lock()
        cat:=app.cat
        app.mu.Unlock()
        ctx,cancel:=context.WithTimeout(context.Background(),4*time.Minute)
        defer cancel()
        scan,e:=core.ScanGameLibrary(ctx,root,cat,manager.log)
        manager.mu.Lock()
        manager.scanning=false
        if e!=nil{
            manager.status="Scan error: "+e.Error()
            manager.log.Add("error","Scan",e.Error())
        }else{
            manager.scan=scan
            manager.jobs=make([]core.BatchJob,len(scan.Games))
            for i,g:=range scan.Games{
                manager.jobs[i]=core.BatchJob{Game:g,Decision:core.BatchReview,OutputMask:defaultBatchMask(g)}
                for _,p:=range g.Patches{if p.Status=="exact"{manager.jobs[i].PatchFile=p.File;manager.jobs[i].PatchSHA256=p.SHA256;break}}
            }
            // Leave ambiguous and complex games at the end of the queue.
            sort.SliceStable(manager.jobs,func(i,j int)bool{
                priority:=func(status string)int{switch status{case "ready":return 0;case "review":return 1;case "unmatched":return 2;default:return 3}}
                return priority(manager.jobs[i].Game.Status)<priority(manager.jobs[j].Game.Status)
            })
            manager.selected=-1
            manager.scroll=0
            manager.status=fmt.Sprintf("%d games scanned; no files modified",len(scan.Games))
        }
        manager.mu.Unlock()
        redraw()
    }()
}
func managerSelectPatchOffset(delta int){
    manager.mu.Lock()
    if manager.selected>=0&&manager.selected<len(manager.jobs){
        count:=len(manager.jobs[manager.selected].Game.Patches)
        if count>0{
            manager.patchIndex=(manager.patchIndex+delta+count)%count
            manager.jobs[manager.selected].PatchFile=chosenManagerPatch(manager.jobs[manager.selected].Game,manager.patchIndex)
            manager.jobs[manager.selected].PatchSHA256=manager.jobs[manager.selected].Game.Patches[manager.patchIndex].SHA256
        }
    }
    manager.mu.Unlock()
    refreshManagerConfidence()
}
func refreshManagerConfidence(){
    job,choice,ok:=managerSelected()
    matches:=[]core.ConfidenceReport{}
    if ok&&choice<len(job.Game.Patches){
        patch:=job.Game.Patches[choice]
        digest,e:=core.ConfidenceFileHash(patch.File)
        if e==nil{
            db,e:=core.LoadConfidenceCache(app.root)
            if e==nil{matches=core.ConfidenceForPatch(db,digest,job.Game.TitleID,job.Game.CRC32)}
        }
    }
    manager.mu.Lock()
    manager.confidence=matches
    manager.confidenceScroll=0
    manager.mu.Unlock()
}
func refreshConfidenceOnUpdate(){
    ctx,cancel:=context.WithTimeout(context.Background(),18*time.Second)
    defer cancel()
    count,e:=core.SyncConfidenceCache(ctx,app.root,nil,"")
    if e!=nil{
        manager.log.Add("warning","Confidence update","Offline or invalid index; keeping local reports")
        return
    }
    if count>0{
        manager.log.Add("pass","Confidence update",fmt.Sprintf("%d new reviewed confidence reports found; updating local logs",count))
        app.mu.Lock()
        app.status=fmt.Sprintf("%d new confidence reports found; updated local logs",count)
        app.mu.Unlock()
    }else{manager.log.Add("info","Confidence update","Reviewed confidence reports are current")}
    refreshManagerConfidence()
    redraw()
}
func managerReviewOrSkip(decision core.BatchDecision){
    manager.mu.Lock()
    if manager.selected>=0&&manager.selected<len(manager.jobs){
        job:=&manager.jobs[manager.selected]
        if decision==core.BatchAuto {
            if job.Game.Status!="ready" {
                manager.status="This game requires manual approval; cannot automatically apply patches"
                manager.mu.Unlock();return
            }
            // Must choose one exact matching JMP, not just any candidate.
            found:=false
            for _,p:=range job.Game.Patches{
                if p.Status=="exact"&&p.File==job.PatchFile{found=true}
            }
            if !found{
                manager.status="Select the verified JMP before enabling unattended processing"
                manager.mu.Unlock();return
            }
        }
        job.Decision=decision
        manager.status=filepath.Base(job.Game.Path)+": "+string(decision)
    }
    manager.mu.Unlock()
}
func managerSetFormat(){
    manager.mu.Lock()
    defer manager.mu.Unlock()
    if manager.selected<0||manager.selected>=len(manager.jobs){return}
    j:=&manager.jobs[manager.selected]
    options:=[]uint32{core.GameOutputFolder,core.GameOutputXISO,core.GameOutputCCI,core.GameOutputXBE}
    if j.Game.Kind=="fulliso"{options=append(options,core.GameOutputISO)}
    i:=0
    for n,v:=range options{if j.OutputMask==v{i=n;break}}
    j.OutputMask=options[(i+1)%len(options)]
}
func managerApproveSafe(){
    if maintenanceDialog("Approve unattended batch processing","Automatically SELECT only games with one exact original-XBE CRC and Title ID matching a single-file JMP. All other games stay in manual review. Nothing will be modified until Run Auto is clicked.",0x24|0x100)!=6{return}
    count:=0
    manager.mu.Lock()
    for i:=range manager.jobs{
        job:=&manager.jobs[i]
        if job.Game.Status=="ready"&&job.PatchFile!="" {
            job.Decision=core.BatchAuto
            count++
        }else if job.Decision!=core.BatchSkip {
            job.Decision=core.BatchReview
        }
    }
    manager.status=fmt.Sprintf("Selected %d exact-match game(s) for unattended processing; others need approval",count)
    manager.mu.Unlock()
}
func managerRunBatch(){
    manager.mu.Lock()
    if manager.running||manager.scanning{manager.mu.Unlock();return}
    root:=manager.root
    jobs:=append([]core.BatchJob(nil),manager.jobs...)
    count:=0
    for _,j:=range jobs{if j.Decision==core.BatchAuto{count++}}
    manager.mu.Unlock()
    if count==0{notify("No games marked Auto. Exact-match games may be approved individually or together.");return}
    dest:=chooseDirectory("Choose a SEPARATE destination for new patched game packages")
    if dest==""{return}
    if maintenanceDialog("Confirm unattended patching",fmt.Sprintf("Create NEW output packages for %d verified game(s)?\n\nSource library: %s\nOutput folder: %s\n\nComplex and uncertain games remain untouched for manual approval; no existing output may be overwritten.",count,root,dest),0x24|0x100)!=6{return}
    manager.mu.Lock()
    manager.running=true
    manager.status="Applying exact-match patches and verifying new game copies…"
    manager.mu.Unlock()
    go func(){
        ctx,cancel:=context.WithTimeout(context.Background(),4*time.Hour)
        defer cancel()
        summary,e:=core.RunBatchJobs(ctx,root,dest,jobs,manager.log)
        manager.mu.Lock()
        manager.running=false
        if e!=nil{manager.status="Batch error: "+e.Error()
        }else{manager.status=fmt.Sprintf("Completed %d; skipped %d; review %d; failed %d",summary.Completed,summary.Skipped,summary.Manual,summary.Failed)}
        manager.mu.Unlock()
        if e!=nil{recordFailure("batch.process",e,root,"",dest)}
        if len(summary.Outputs)>0{rememberRecent("output",dest)}
        redraw()
    }()
}
func managerOpenInOutputs(){
    job,choice,ok:=managerSelected()
    if !ok{return}
    // Validate full-disc vs game-partition ISO off-thread before making any
    // output choice usable, just as the regular Patch Library picker does.
    openInput(job.Game.Path)
    if job.OutputMask!=0 {gameOutputMask=job.OutputMask}
    app.directFile=chosenManagerPatch(job.Game,choice)
    app.parsed=nil
    gameOutputIgnorePatch=app.directFile==""
    app.tab=7
    app.status="Manual review: choose patch record groups and output formats before applying"
}
func managerBrowse(path string){
    if path==""{return}
    entries,e:=core.BrowseGameImage(path)
    manager.mu.Lock()
    if e!=nil{manager.status="Image browsing unavailable: "+e.Error()
    }else{
        manager.browsePath=path
        manager.browse=entries
        manager.browseSelected=-1
        manager.browseScroll=0
        manager.status=fmt.Sprintf("%d image entries indexed read-only",len(entries))
    }
    manager.mu.Unlock()
    if e!=nil{manager.log.Add("warning","Image browser",e.Error())}
}
func managerExtractFile(){
    manager.mu.Lock()
    if manager.browseSelected<0||manager.browseSelected>=len(manager.browse){manager.mu.Unlock();return}
    entry:=manager.browse[manager.browseSelected]
    path:=manager.browsePath
    manager.mu.Unlock()
    if entry.IsDirectory {notify("Select an image file, not a directory.");return}
    dest:=pickFile(true,"Extracted file\x00*.*\x00","Save independent extracted file",core.SuggestedMemberName(entry))
    if dest==""{return}
    if e:=core.ExtractImageMember(path,entry.Path,dest);e!=nil{
        manager.log.Add("error","Extract",e.Error())
        notify("Single-file extraction failed: "+e.Error())
    }else{
        manager.log.Add("pass","Extract",entry.Path+": verified independent copy")
        rememberRecent("output",dest)
        notify("Extracted file verified: "+filepath.Base(dest))
    }
}
func managerBuildOverlap(){
    job,_,ok:=managerSelected()
    if !ok{return}
    list:=[]*core.Patch{}
    for _,entry:=range job.Game.Patches{
        if entry.Status=="incompatible"||!strings.EqualFold(filepath.Ext(entry.File),".jmp"){continue}
        bytes,e:=os.ReadFile(entry.File)
        if e!=nil{continue}
        p,e:=core.ParseJMP(filepath.Base(entry.File),bytes)
        if e==nil&&p.CheckSingleFileOnly()==nil{list=append(list,p)}
    }
    var report core.OverlapReport
    err:=error(nil)
    if len(list)<2{err=fmt.Errorf("at least two source-compatible single-file JMPs are needed for hex overlap comparison")
    }else if job.Game.Kind=="cci"{err=fmt.Errorf("inspect the compressed image's original XBE before comparing byte overlaps")
    }else{
        original,e:=gameOriginals(job.Game.Path,[]string{"default.xbe"})
        if e!=nil{err=e}else{report,err=core.CompareJMPOverlaps(original["default.xbe"],list,512)}
    }
    manager.mu.Lock()
    manager.overlapping=report
    manager.overlapScroll=0
    manager.overlapError=""
    if err!=nil {manager.overlapError=err.Error()
    }else{
        manager.overlapError=fmt.Sprintf("%d overlapping byte(s), %d differing writes; display capped at 512",report.Total,report.Different)
    }
    manager.mu.Unlock()
    if err!=nil{manager.log.Add("warning","Hex overlap",err.Error())
    }else{manager.log.Add("info","Hex overlap",manager.overlapError)}
}
func managerExportCombined(){
    job,_,ok:=managerSelected()
    if !ok{return}
    if job.Game.Kind=="cci"{notify("Inspect the compressed game XBE before combining JMPs.");return}
    originals,e:=gameOriginals(job.Game.Path,[]string{"default.xbe"})
    if e!=nil{notify(e.Error());return}
    patches:=[]*core.Patch{}
    for _,candidate:=range job.Game.Patches {
        if candidate.Status=="incompatible" || !strings.EqualFold(filepath.Ext(candidate.File),".jmp"){continue}
        data,readErr:=os.ReadFile(candidate.File)
        if readErr!=nil{notify(readErr.Error());return}
        p,parseErr:=core.ParseJMP(filepath.Base(candidate.File),data)
        if parseErr!=nil{notify(parseErr.Error());return}
        if parseErr=p.CheckSingleFileOnly();parseErr!=nil{continue}
        patches=append(patches,p)
    }
    if len(patches)<2{notify("Choose a game with two or more compatible single-file JMPs before combining.");return}
    data,report,mergeErr:=core.CombineJMPs(originals["default.xbe"],patches,false)
    if mergeErr!=nil&&report.Different>0{
        message:=fmt.Sprintf("%d overlapping byte changes CONFLICT.\n\nThe JMPs will apply sequentially in the order shown by the HEX OVERLAPS preview. The later patch may deliberately overwrite bytes from the earlier patch.\n\nThis cannot be automatically proven safe. Review the hex rows first. Combine anyway?",report.Different)
        if maintenanceDialog("Conflicting JMP bytes",message,0x24|0x100)!=6{return}
        data,report,mergeErr=core.CombineJMPs(originals["default.xbe"],patches,true)
    }
    if mergeErr!=nil{manager.log.Add("error","Merge JMP",mergeErr.Error());notify(mergeErr.Error());return}
    if maintenanceDialog("Export combined JMP",fmt.Sprintf("Create a NEW combined JMP from %d patches?\n\n%v overlapping bytes; %v conflicting writes.\n\nOriginal sources and authors will be retained in the JMP metadata. The output is independently re-applied and verified before saving.",len(patches),report.Total,report.Different),0x24|0x100)!=6{return}
    dest:=pickFile(true,"Magic patch\x00*.jmp\x00","Save a new combined JMP","Combined_JMP.jmp")
    if dest==""{return}
    if !strings.EqualFold(filepath.Ext(dest),".jmp"){notify("Choose a .jmp output file.");return}
    file,e:=os.OpenFile(dest,os.O_CREATE|os.O_EXCL|os.O_WRONLY,0644)
    if e!=nil{notify("Existing patch must not be overwritten: "+e.Error());return}
    written,e:=file.Write(data)
    if closeErr:=file.Close();e==nil{e=closeErr}
    if e==nil&&written!=len(data){e=fmt.Errorf("combined JMP was not completely written")}
    if e!=nil{os.Remove(dest);notify(e.Error());return}
    manager.log.Add("pass","Merge JMP",fmt.Sprintf("Verified combined JMP from %d original patches",len(patches)))
    rememberRecent("patch",dest)
    notify("Combined JMP verified and saved: "+filepath.Base(dest))
}

func managerConfirm(environment string){
    job,choice,ok:=managerSelected()
    if !ok||choice>=len(job.Game.Patches){notify("Select a game and matching patch to confirm.");return}
    if job.Game.TitleID==""||job.Game.CRC32==""{notify("Only identified original XBE versions can be confirmed.");return}
    if _,e:=loadGitHubCredential();e!=nil{notify("Connect GitHub Issues in Maintenance before sharing a confidence report.");return}
    choicePatch:=job.Game.Patches[choice]
    digest,e:=core.ConfidenceFileHash(choicePatch.File)
    if e!=nil{notify(e.Error());return}
    manager.mu.Lock()
    result:=manager.reportResult
    revision,ram,cpu:=manager.xboxRevision,manager.ramUpgrade,manager.cpuUpgrade
    manager.mu.Unlock()
    hardware:=""
    if environment=="hardware"{hardware="Original Xbox"}
    var random [12]byte
    if _,e=rand.Read(random[:]);e!=nil{notify(e.Error());return}
    report:=core.ConfidenceReport{
        ID:hex.EncodeToString(random[:]),PatchSHA256:digest,TitleID:job.Game.TitleID,
        SourceCRC32:job.Game.CRC32,Environment:environment,Result:result,
        Hardware:hardware,ReportedUTC:time.Now().UTC().Format(time.RFC3339),
    }
    if environment=="hardware"{
        report.XboxRevision=revision
        report.RAMUpgrade=ram
        report.CPUUpgrade=cpu
    }
    // Validation happens before requesting public-submission consent, so
    // incomplete/invalid hardware metadata cannot be silently uploaded.
    if _,e=core.ConfidenceIssueBody(report);e!=nil{notify("Invalid hardware report: "+e.Error());return}
    hardwareDetails:=""
    if environment=="hardware"{
        hardwareDetails="\nRevision: "+core.ConfidenceRevisionLabel(revision)+
            "\nRAM upgrade: "+core.ConfidenceUpgradeLabel(ram)+
            "\nCPU upgrade: "+core.ConfidenceUpgradeLabel(cpu)+"\n"
    }
    msg:=fmt.Sprintf("Publish a PUBLIC field-test report to GitHub Issues?\n\nPatch: %s\nTitle ID: %s\nOriginal XBE CRC: %s\nTest environment: %s\nResult: %s%s\nOnly patch SHA256, source identity and the displayed hardware choices are shared. No game bytes or local path. A maintainer must review it before it appears in the public confidence log.",choicePatch.Name,report.TitleID,report.SourceCRC32,environment,result,hardwareDetails)
    if maintenanceDialog("Confirm compatibility report",msg,0x24|0x100)!=6{return}
    manager.log.Add("info","Confidence submission","Submitting user-confirmed "+environment+" report for review")
    go func(){
        token,e:=loadGitHubCredential()
        if e!=nil{notify(e.Error());return}
        ctx,cancel:=context.WithTimeout(context.Background(),35*time.Second)
        defer cancel()
        number,e:=core.SubmitConfidenceReport(ctx,nil,token,report)
        if e!=nil {
            manager.log.Add("error","Confidence submission","GitHub rejected report; no curated log changed")
            notify("Compatibility report was not submitted: "+e.Error())
        }else{
            manager.log.Add("pass","Confidence submission",fmt.Sprintf("Review issue #%d created; not yet in curated confidence index",number))
            notify(fmt.Sprintf("Compatibility report posted as issue #%d, awaiting review",number))
        }
        redraw()
    }()
}
func managerAction(action string,index int){
    switch action{
    case "manager-scan":managerScan()
    case "manager-mode":
        manager.mu.Lock();manager.mode=index;manager.mu.Unlock()
        if index==2{managerBuildOverlap()}
        if index==4{refreshManagerConfidence()}
    case "manager-select":
        manager.mu.Lock()
        if index>=0&&index<len(manager.jobs){
            if index!=manager.selected {
                // Avoid accidentally attributing another user's / console's
                // hardware modifications to the next game report.
                manager.xboxRevision="unknown"
                manager.ramUpgrade="unknown"
                manager.cpuUpgrade="unknown"
            }
            manager.selected=index;manager.patchIndex=0
        }
        manager.mu.Unlock()
        refreshManagerConfidence()
    case "manager-patch-prev":managerSelectPatchOffset(-1)
    case "manager-patch-next":managerSelectPatchOffset(1)
    case "manager-auto":managerReviewOrSkip(core.BatchAuto)
    case "manager-review":managerReviewOrSkip(core.BatchReview)
    case "manager-skip":managerReviewOrSkip(core.BatchSkip)
    case "manager-all-safe":managerApproveSafe()
    case "manager-run":managerRunBatch()
    case "manager-format":managerSetFormat()
    case "manager-open":managerOpenInOutputs()
    case "manager-image-choose":
        p:=pickFile(false,"Xbox image\x00*.iso;*.xiso\x00All files\x00*.*\x00","Browse Xbox XISO files before patching","")
        if p!=""{managerBrowse(p)}
    case "manager-image-current":
        job,_,ok:=managerSelected();if ok{managerBrowse(job.Game.Path)}
    case "manager-image-select":
        manager.mu.Lock();if index>=0&&index<len(manager.browse){manager.browseSelected=index};manager.mu.Unlock()
    case "manager-image-extract":managerExtractFile()
    case "manager-log-clear":manager.log.Clear();manager.mu.Lock();manager.logScroll=0;manager.mu.Unlock()
    case "manager-log-prev":
        manager.mu.Lock();manager.logScroll+=8;manager.mu.Unlock()
    case "manager-log-next":
        manager.mu.Lock();manager.logScroll-=8;if manager.logScroll<0{manager.logScroll=0};manager.mu.Unlock()
    case "manager-merge":managerExportCombined()
    case "manager-confidence-refresh":go refreshConfidenceOnUpdate()
    case "manager-confidence-result":
        manager.mu.Lock()
        switch manager.reportResult{case "working":manager.reportResult="partial";case "partial":manager.reportResult="failed";default:manager.reportResult="working"}
        manager.mu.Unlock()
    case "manager-confidence-revision":
        manager.mu.Lock()
        values:=core.ConfidenceXboxRevisions()
        for i,v:=range values{if v==manager.xboxRevision {manager.xboxRevision=values[(i+1)%len(values)];break}}
        manager.mu.Unlock()
    case "manager-confidence-ram":
        manager.mu.Lock()
        values:=core.ConfidenceUpgradeStatuses()
        for i,v:=range values{if v==manager.ramUpgrade{manager.ramUpgrade=values[(i+1)%len(values)];break}}
        manager.mu.Unlock()
    case "manager-confidence-cpu":
        manager.mu.Lock()
        values:=core.ConfidenceUpgradeStatuses()
        for i,v:=range values{if v==manager.cpuUpgrade{manager.cpuUpgrade=values[(i+1)%len(values)];break}}
        manager.mu.Unlock()
    case "manager-confidence-xemu":managerConfirm("xemu")
    case "manager-confidence-hardware":managerConfirm("hardware")
    }
}
func managerWheel(delta int) {
    manager.mu.Lock()
    defer manager.mu.Unlock()
    lines:=delta/120
    switch manager.mode {
    case 0:manager.scroll-=lines*3;if manager.scroll<0{manager.scroll=0}
    case 1:manager.browseScroll-=lines*3;if manager.browseScroll<0{manager.browseScroll=0}
    case 2:manager.overlapScroll-=lines*3;if manager.overlapScroll<0{manager.overlapScroll=0}
    case 3:manager.logScroll+=lines*3;if manager.logScroll<0{manager.logScroll=0}
    case 4:manager.confidenceScroll-=lines*3;if manager.confidenceScroll<0{manager.confidenceScroll=0}
    }
}
