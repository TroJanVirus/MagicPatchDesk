package core

import (
    "crypto/rand"
    "encoding/hex"
    "encoding/json"
    "errors"
    "fmt"
    "os"
    "path/filepath"
    "regexp"
    "strings"
    "sync"
    "time"
)

const crashSessionSchema="magicpatchdesk-crash-session-v1"
var crashEvidenceIDRE=regexp.MustCompile(`^[a-f0-9]{20}$`)

type crashSessionMarker struct {
    Schema string `json:"schema"`
    ID string `json:"id"`
    PID int `json:"pid"`
    StartedUTC string `json:"started_utc"`
    AppVersion string `json:"app_version"`
    Stage string `json:"stage"`
    SourceKind string `json:"source_kind,omitempty"`
    SourceSwitches int `json:"source_switches,omitempty"`
    RecentSourceKinds []string `json:"recent_source_kinds,omitempty"`
}

type CrashSession struct {
    mu sync.Mutex
    root string
    marker string
    data crashSessionMarker
}

func CrashDirectory(root string) string{return filepath.Join(root,"Reports","Crash")}
func crashMarkerPath(root,id string)string{return filepath.Join(CrashDirectory(root),"session-"+id+".json")}
func crashNoticePath(root,id string)string{return filepath.Join(CrashDirectory(root),"notice-"+id)}
func recordCrashNotice(root,id string)error{
    if !crashEvidenceIDRE.MatchString(id){return errors.New("invalid crash evidence ID")}
    f,e:=os.OpenFile(crashNoticePath(root,id),os.O_CREATE|os.O_WRONLY|os.O_EXCL,0600)
    if errors.Is(e,os.ErrExist){return nil}
    if e!=nil{return e}
    return f.Close()
}
func PendingCrashNotices(root string)(int,error){
    entries,e:=os.ReadDir(CrashDirectory(root))
    if errors.Is(e,os.ErrNotExist){return 0,nil}
    if e!=nil{return 0,e}
    count:=0
    for _,entry:=range entries{
        if entry.IsDir(){continue}
        name:=entry.Name()
        if strings.HasPrefix(name,"notice-")&&crashEvidenceIDRE.MatchString(strings.TrimPrefix(name,"notice-")){count++}
    }
    return count,nil
}
func ClearCrashNotices(root string)error{
    entries,e:=os.ReadDir(CrashDirectory(root))
    if errors.Is(e,os.ErrNotExist){return nil}
    if e!=nil{return e}
    for _,entry:=range entries{
        if entry.IsDir(){continue}
        name:=entry.Name()
        if strings.HasPrefix(name,"notice-")&&crashEvidenceIDRE.MatchString(strings.TrimPrefix(name,"notice-")){
            if e=os.Remove(filepath.Join(CrashDirectory(root),name));e!=nil{return e}
        }
    }
    return nil
}

func CrashEvidencePath(root,id string)string{
    if !crashEvidenceIDRE.MatchString(id){return ""}
    return filepath.Join(CrashDirectory(root),"crash-"+id+".log")
}
func RuntimeCrashLogPath(root,id string)string{
    if !crashEvidenceIDRE.MatchString(id){return ""}
    return filepath.Join(CrashDirectory(root),"runtime-"+id+".log")
}
func crashStage(value string)string{
    switch value{
    case "startup","idle","import_patch","import_folder","read_xbe","auto_match","parse_patch","patch_preview","patch_apply","convert_patch","ui_callback","game_source_open","game_inspect_pending","game_inspect_commit","library_sync","patchhub_sync","other":
        return value
    default:return "other"
    }
}
// Crash breadcrumbs contain only file format categories, never filenames,
// patch bytes, absolute paths or user information. Keep at most eight hops.
func crashSourceCode(path string) string {
    switch strings.ToLower(filepath.Ext(path)) {
    case ".xbe": return "xbe"
    case ".iso": return "iso"
    case ".xiso": return "xiso"
    case ".cci": return "cci"
    default: return "other"
    }
}
func safeCrashSourceCode(value string) string {
    switch value {
    case "xbe","iso","xiso","cci","other":return value
    default:return ""
    }
}
func reportCrashSourceKind(value string) string {
    switch value {
    case "xbe": return "loose-executable"
    case "iso","xiso": return "disc-image"
    case "cci": return "cerbios-cci"
    case "other": return "other-file"
    default: return "unspecified"
    }
}
func safeCrashSourceTrail(values []string) []string {
    trail:=make([]string,0,8)
    for _,value:=range values {
        if kind:=safeCrashSourceCode(value);kind!="" {trail=append(trail,kind)}
        if len(trail)>8 {trail=trail[len(trail)-8:]}
    }
    return trail
}
func safeCrashSwitchCount(value int) int {
    if value<0{return 0}
    if value>1000000{return 1000000}
    return value
}
func (s *CrashSession) RecordSourceSelection(path string) error {
    if s==nil{return nil}
    kind:=crashSourceCode(path)
    s.mu.Lock()
    defer s.mu.Unlock()
    s.data.Stage="game_source_open"
    s.data.SourceKind=kind
    s.data.SourceSwitches=safeCrashSwitchCount(s.data.SourceSwitches+1)
    s.data.RecentSourceKinds=safeCrashSourceTrail(append(s.data.RecentSourceKinds,kind))
    return writeCrashMarker(s.marker,s.data)
}
func writeCrashMarker(path string,m crashSessionMarker)error{
    data,e:=json.Marshal(m)
    if e!=nil{return e}
    if len(data)>2048{return errors.New("crash marker exceeds limit")}
    temp,e:=os.CreateTemp(filepath.Dir(path),".crash-session-*")
    if e!=nil{return e}
    defer os.Remove(temp.Name())
    if e=temp.Chmod(0600);e!=nil{temp.Close();return e}
    if _,e=temp.Write(data);e!=nil{temp.Close();return e}
    if e=temp.Sync();e!=nil{temp.Close();return e}
    if e=temp.Close();e!=nil{return e}
    return os.Rename(temp.Name(),path)
}
func BeginCrashSession(root,appVersion string,pid int)(*CrashSession,error){
    if root==""||pid<=0{return nil,errors.New("crash logging needs valid app directory and process ID")}
    dir:=CrashDirectory(root)
    if e:=os.MkdirAll(dir,0700);e!=nil{return nil,e}
    var nonce [10]byte
    if _,e:=rand.Read(nonce[:]);e!=nil{return nil,e}
    id:=hex.EncodeToString(nonce[:])
    m:=crashSessionMarker{Schema:crashSessionSchema,ID:id,PID:pid,
        StartedUTC:time.Now().UTC().Format(time.RFC3339),AppVersion:appVersion,Stage:"startup"}
    path:=crashMarkerPath(root,id)
    if e:=writeCrashMarker(path,m);e!=nil{return nil,e}
    return &CrashSession{root:root,marker:path,data:m},nil
}
func (s *CrashSession) ID()string{
    if s==nil{return ""}
    s.mu.Lock();defer s.mu.Unlock()
    return s.data.ID
}
func (s *CrashSession) SetStage(stage string)error{
    if s==nil{return nil}
    s.mu.Lock();defer s.mu.Unlock()
    s.data.Stage=crashStage(stage)
    return writeCrashMarker(s.marker,s.data)
}
func (s *CrashSession) End()error{
    if s==nil{return nil}
    s.mu.Lock();defer s.mu.Unlock()
    e:=os.Remove(s.marker)
    if errors.Is(e,os.ErrNotExist){return nil}
    return e
}
// SaveCaughtCrash writes a local, full stack trace WITHOUT transmitting it,
// then a separate privacy-filtered diagnostic compatible with the existing
// Settings / Reports and GitHub issue workflows. The raw panic value and stack
// must never be copied into the shareable JSON report.
func (s *CrashSession) SaveCaughtCrash(reason string,stack []byte)error{
    if s==nil{return errors.New("crash session unavailable")}
    s.mu.Lock();defer s.mu.Unlock()
    dir:=CrashDirectory(s.root)
    if e:=os.MkdirAll(dir,0700);e!=nil{return e}
    path:=CrashEvidencePath(s.root,s.data.ID)
    f,e:=os.OpenFile(path,os.O_CREATE|os.O_WRONLY|os.O_EXCL,0600)
    if e!=nil{return e}
    if len(stack)>256<<10{stack=stack[:256<<10]}
    _,writeErr:=fmt.Fprintf(f,"MagicPatchDesk private crash evidence — DO NOT UPLOAD WITHOUT REVIEW\nTime: %s\nVersion: %s\nLast action: %s\nReason: %s\n\n%s\n",time.Now().UTC().Format(time.RFC3339),s.data.AppVersion,s.data.Stage,reason,stack)
    syncErr:=f.Sync()
    closeErr:=f.Close()
    if writeErr!=nil{return writeErr}
    if syncErr!=nil{return syncErr}
    if closeErr!=nil{return closeErr}
    report,e:=NewFailureReport(FailureContext{AppVersion:s.data.AppVersion,
        Operation:"process.unhandled_panic."+s.data.Stage},errors.New("unhandled panic"))
    if e!=nil{return e}
    report.CrashEvidenceID=s.data.ID
    if _,e=SaveFailureReport(s.root,report);e!=nil{return e}
    if e=recordCrashNotice(s.root,s.data.ID);e!=nil{return e}
    return os.Remove(s.marker)
}
// SaveRecoveredBackgroundPanic records an isolated background-task failure
// without ending the process session or turning a recovered panic into an
// incorrect "unexpected exit" report on the next launch.
func (s *CrashSession) SaveRecoveredBackgroundPanic(stage, reason string, stack []byte) error {
    if s == nil { return errors.New("crash session unavailable") }
    s.mu.Lock()
    defer s.mu.Unlock()
    var nonce [10]byte
    if _,e:=rand.Read(nonce[:]);e!=nil{return e}
    evidenceID:=hex.EncodeToString(nonce[:])
    if len(reason)>2048{reason=reason[:2048]}
    if len(stack)>256<<10{stack=stack[:256<<10]}
    dir:=CrashDirectory(s.root)
    if e:=os.MkdirAll(dir,0700);e!=nil{return e}
    f,e:=os.OpenFile(CrashEvidencePath(s.root,evidenceID),os.O_CREATE|os.O_EXCL|os.O_WRONLY,0600)
    if e!=nil{return e}
    _,writeErr:=fmt.Fprintf(f,"MagicPatchDesk private recovered background panic — DO NOT UPLOAD WITHOUT REVIEW\nTime: %s\nVersion: %s\nOperation: %s\nReason: %s\n\n%s\n",time.Now().UTC().Format(time.RFC3339),s.data.AppVersion,crashStage(stage),reason,stack)
    syncErr:=f.Sync()
    closeErr:=f.Close()
    if writeErr!=nil{return writeErr}
    if syncErr!=nil{return syncErr}
    if closeErr!=nil{return closeErr}
    report,e:=NewFailureReport(FailureContext{AppVersion:s.data.AppVersion,
        Operation:"process.recovered_panic."+crashStage(stage)},errors.New("recovered background panic"))
    if e!=nil{return e}
    report.CrashEvidenceID=evidenceID
    report.SourceKind=reportCrashSourceKind(s.data.SourceKind)
    report.SourceSwitchCount=safeCrashSwitchCount(s.data.SourceSwitches)
    report.RecentSourceKinds=safeCrashSourceTrail(s.data.RecentSourceKinds)
    if _,e=SaveFailureReport(s.root,report);e!=nil{return e}
    return recordCrashNotice(s.root,evidenceID)
}

// RecoverAbandonedCrashSessions safely imports evidence of a process that
// ended without performing normal shutdown. An unexpected exit is NOT proof
// of a Go panic; power-off and forced process termination also qualify.
func RecoverAbandonedCrashSessions(root string,isAlive func(int)bool)(int,error){
    dir:=CrashDirectory(root)
    entries,e:=os.ReadDir(dir)
    if errors.Is(e,os.ErrNotExist){return 0,nil}
    if e!=nil{return 0,e}
    count:=0
    for _,entry:=range entries{
        if count>=50{break}
        name:=entry.Name()
        if entry.IsDir()||!strings.HasPrefix(name,"session-")||!strings.HasSuffix(name,".json"){continue}
        fileID:=strings.TrimSuffix(strings.TrimPrefix(name,"session-"),".json")
        if !crashEvidenceIDRE.MatchString(fileID){continue}
        path:=filepath.Join(dir,name)
        st,e:=entry.Info()
        if e!=nil||!st.Mode().IsRegular()||st.Size()>2048{continue}
        body,e:=os.ReadFile(path)
        if e!=nil{continue}
        var marker crashSessionMarker
        if json.Unmarshal(body,&marker)!=nil||marker.Schema!=crashSessionSchema||marker.ID!=fileID||marker.PID<=0{continue}
        if isAlive!=nil&&isAlive(marker.PID){continue}
        // A process terminated abruptly without passing Go's panic handler.
        // Create a human-readable private log even when Windows/Go supplied
        // no stack or minidump. The sanitized JSON is a separate document.
        logPath:=CrashEvidencePath(root,marker.ID)
        if _,e=os.Stat(logPath);errors.Is(e,os.ErrNotExist){
            note:=fmt.Sprintf("MagicPatchDesk private unexpected-exit evidence — DO NOT UPLOAD WITHOUT REVIEW\nPrior session started: %s\nVersion: %s\nLast action: %s\nPID: %d\nLast source type: %s\nSource selections: %d\nRecent source types: %s\n\nThis session did not close normally; it may have crashed, been killed or lost power. The cause cannot be inferred from this marker alone. Examine runtime-%s.log or a Windows Error Reporting dump if present.\n",marker.StartedUTC,marker.AppVersion,crashStage(marker.Stage),marker.PID,reportCrashSourceKind(safeCrashSourceCode(marker.SourceKind)),safeCrashSwitchCount(marker.SourceSwitches),strings.Join(safeCrashSourceTrail(marker.RecentSourceKinds)," > "),marker.ID)
            if e=os.WriteFile(logPath,[]byte(note),0600);e!=nil{return count,e}
        }
        report,e:=NewFailureReport(FailureContext{
            AppVersion:marker.AppVersion,
            Operation:"process.unexpected_exit."+crashStage(marker.Stage),
        },errors.New("unexpected process exit"))
        if e!=nil{return count,e}
        report.CrashEvidenceID=marker.ID
        report.SourceKind=reportCrashSourceKind(safeCrashSourceCode(marker.SourceKind))
        report.SourceSwitchCount=safeCrashSwitchCount(marker.SourceSwitches)
        report.RecentSourceKinds=safeCrashSourceTrail(marker.RecentSourceKinds)
        if _,e=SaveFailureReport(root,report);e!=nil{return count,e}
        if e=recordCrashNotice(root,marker.ID);e!=nil{return count,e}
        if e=os.Remove(path);e!=nil{return count,e}
        count++
    }
    return count,nil
}
