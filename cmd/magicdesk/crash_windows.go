//go:build windows

package main

import (
    "fmt"
    "magicpatchdesk/core"
    "os"
    "path/filepath"
    "runtime/debug"
    "syscall"
    "unsafe"
)

var crashSession *core.CrashSession
var crashStderr *os.File

var (
    procSetStdHandle = k32.NewProc("SetStdHandle")
    procOpenCrashProcess = k32.NewProc("OpenProcess")
    procGetExitCode = k32.NewProc("GetExitCodeProcess")
    procCloseCrashHandle = k32.NewProc("CloseHandle")
    crashAdvapi = syscall.NewLazyDLL("advapi32.dll")
    procRegCrashCreate = crashAdvapi.NewProc("RegCreateKeyExW")
    procRegCrashSet = crashAdvapi.NewProc("RegSetValueExW")
    procRegCrashClose = crashAdvapi.NewProc("RegCloseKey")
)

func crashProcessAlive(pid int) bool {
    if pid<=0{return false}
    const processQueryLimitedInformation=0x1000
    handle,_,callErr:=procOpenCrashProcess.Call(processQueryLimitedInformation,0,uintptr(pid))
    if handle==0{return callErr==syscall.Errno(5)} // access denied means likely alive
    defer procCloseCrashHandle.Call(handle)
    var code uint32
    ok,_,_:=procGetExitCode.Call(handle,uintptr(unsafe.Pointer(&code)))
    return ok!=0&&code==259 // STILL_ACTIVE
}

// WER LocalDumps is a Windows-supported best-effort fallback when Go's panic
// recovery does not run (including a native access violation). It is scoped
// to this executable under HKCU; no administrator rights or third-party DLL
// are required. Dumps may contain private memory and are NEVER uploaded.
func configureCrashWER(root string) error {
    exe,e:=os.Executable()
    if e!=nil{return e}
    folder:=filepath.Join(core.CrashDirectory(root),"Dumps")
    if e=os.MkdirAll(folder,0700);e!=nil{return e}
    key:="Software\\Microsoft\\Windows\\Windows Error Reporting\\LocalDumps\\"+filepath.Base(exe)
    utfKey,e:=syscall.UTF16PtrFromString(key)
    if e!=nil{return e}
    var handle uintptr
    result,_,_:=procRegCrashCreate.Call(0x80000001,uintptr(unsafe.Pointer(utfKey)),0,0,0,0x0002,0,uintptr(unsafe.Pointer(&handle)),0)
    if result!=0||handle==0{return fmt.Errorf("WER registry access returned status %d",result)}
    defer procRegCrashClose.Call(handle)
    setDWORD:=func(name string,value uint32)error{
        keyName,_:=syscall.UTF16PtrFromString(name)
        ret,_,_:=procRegCrashSet.Call(handle,uintptr(unsafe.Pointer(keyName)),0,4,uintptr(unsafe.Pointer(&value)),4)
        if ret!=0{return fmt.Errorf("WER %s returned status %d",name,ret)}
        return nil
    }
    folderName,_:=syscall.UTF16PtrFromString("DumpFolder")
    wide,e:=syscall.UTF16FromString(folder)
    if e!=nil{return e}
    ret,_,_:=procRegCrashSet.Call(handle,uintptr(unsafe.Pointer(folderName)),0,1,uintptr(unsafe.Pointer(&wide[0])),uintptr(len(wide)*2))
    if ret!=0{return fmt.Errorf("WER DumpFolder returned status %d",ret)}
    if e=setDWORD("DumpType",1);e!=nil{return e} // limited minidump
    return setDWORD("DumpCount",10)
}

func beginCrashProtection(root string) (int,error) {
    count,e:=core.RecoverAbandonedCrashSessions(root,crashProcessAlive)
    if e!=nil{return 0,e}
    crashSession,e=core.BeginCrashSession(root,core.AppVersion,os.Getpid())
    if e!=nil{return count,e}
    // Windows GUI processes often lack a usable stderr handle. Keep a private
    // runtime panic log open throughout the session, even for native crashes.
    logPath:=core.RuntimeCrashLogPath(root,crashSession.ID())
    f,e:=os.OpenFile(logPath,os.O_CREATE|os.O_WRONLY|os.O_EXCL,0600)
    if e==nil{
        crashStderr=f
        os.Stderr=f
        procSetStdHandle.Call(^uintptr(11),f.Fd()) // STD_ERROR_HANDLE == -12
    }
    // WER is best-effort. Session recovery and caught-panic logs still work
    // if local policy disables it or registry access fails.
    _=configureCrashWER(root)
    notices,e:=core.PendingCrashNotices(root)
    if e!=nil{return count,e}
    return notices,nil
}
func markCrashStage(stage string) {
    if crashSession!=nil { _=crashSession.SetStage(stage) }
}

func catchFatalPanic(){
    if value:=recover();value!=nil{
        captureFatalPanic(value)
    }
}
func captureFatalPanic(value any){
    // Never continue after an unhandled callback panic: UI state may be
    // inconsistent. Write the private trace synchronously before exit.
    defer os.Exit(2)
    if crashSession!=nil{
        reason:=fmt.Sprint(value)
        if len(reason)>2048{reason=reason[:2048]}
        _=crashSession.SaveCaughtCrash(reason,debug.Stack())
    }
}

func endCrashProtection(){
    if crashSession!=nil{_ = crashSession.End()}
    if crashStderr!=nil{
        path:=crashStderr.Name()
        _=crashStderr.Close()
        // Clean sessions do not accumulate empty crash logs.
        if info,e:=os.Stat(path);e==nil&&info.Size()==0{_ =os.Remove(path)}
    }
}
