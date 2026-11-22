//go:build windows

package main

import (
    "fmt"
    "magicpatchdesk/core"
    "os"
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

// Microsoft documents automatic local minidumps under an administrative HKLM
// WER LocalDumps registry key. The app never silently writes an undocumented
// HKCU substitute or changes system crash-dump policy. Session markers,
// runtime stderr and caught-panic traces work without administrator access.
// If stronger native crash dumps are needed, docs/CRASH_RECOVERY_AND_UI.md
// describes how to opt in through the documented Windows configuration.
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
