//go:build windows

package main

import (
    "encoding/binary"
    "fmt"
    "magicpatchdesk/core"
    "os"
    "path/filepath"
    "runtime"
    "time"
    "unsafe"
)

// This exercises the user's exact sequence using ACTUAL on-disk inputs:
// choose a JMP, then alternate ISO / CCI / XBE repeatedly. The program
// must not accumulate one active inspector per click or lose the selected JMP.
func smokeRapidMixedSourceSwitch(){
    directory,err:=os.MkdirTemp("","mpd-rapid-source-smoke-*")
    if err!=nil{panic(err)}
    defer os.RemoveAll(directory)

    xbe:=make([]byte,0x1000)
    copy(xbe,"XBEH")
    binary.LittleEndian.PutUint32(xbe[0x104:],0x10000)
    binary.LittleEndian.PutUint32(xbe[0x118:],0x10180)
    binary.LittleEndian.PutUint32(xbe[0x188:],0x4D530064)
    for i,r:=range []rune("Halo 2"){
        binary.LittleEndian.PutUint16(xbe[0x18C+i*2:],uint16(r))
    }
    binary.LittleEndian.PutUint32(xbe[0x220:],1)
    rawXBE:=filepath.Join(directory,"default.xbe")
    if err=os.WriteFile(rawXBE,xbe,0600);err!=nil{panic(err)}
    gameDir:=filepath.Join(directory,"game")
    if err=os.MkdirAll(gameDir,0700);err!=nil{panic(err)}
    if err=os.WriteFile(filepath.Join(gameDir,"default.xbe"),xbe,0600);err!=nil{panic(err)}
    iso:=filepath.Join(directory,"game.iso")
    if err=core.BuildGameXISO(gameDir,iso);err!=nil{panic(err)}
    cci:=filepath.Join(directory,"game.cci")
    if err=core.CompressCCI(iso,cci,nil);err!=nil{panic(err)}
    expected,err:=core.InspectXBE(xbe)
    if err!=nil{panic(err)}

    app.tab=0
    app.cat=core.Catalog{Games:[]core.Game{{
        Title:"Halo 2",
        Patches:[]core.CatalogPatch{{Name:"Widescreen",TitleID:expected.TitleID,
            CRC:"No CRC32 required",File:"not-a-real-test-patch.jmp"}},
    }}}
    app.selectedGame=0
    app.selectedPatch=0
    chosen:=&core.Patch{
        Headers:map[string]string{"author":"test"},
        Groups:[]core.Group{{Name:"Widescreen",Count:1,Enabled:false}},
        Records:[]core.Record{{Kind:core.Offset,Group:"Widescreen",
            Enabled:false,Offset:0x390,Data:[]byte{1}}},
    }
    app.parsed=chosen
    app.directFile=""

    beforeHandles:=smokeProcessHandleCount()
    runtime.GC()
    var beforeMemory runtime.MemStats
    runtime.ReadMemStats(&beforeMemory)

    // Replacing sources must not allocate 270 independent index readers or
    // leave handles alive. The last request must win despite stale posts.
    sources:=[]string{iso,cci,rawXBE}
    for i:=0;i<270;i++{
        openInput(sources[i%len(sources)])
    }
    // Force one more switch while an older request might still be in flight.
    openInput(cci)
    targetGeneration:=app.gameInputGeneration
    deadline:=time.NewTimer(20*time.Second)
    defer deadline.Stop()
    for {
        select{
        case completed:=<-gameInspectReady:
            commitGameInspection(completed)
            if completed.generation==targetGeneration{
                if completed.err!=nil{panic(fmt.Sprintf("Latest CCI inspection failed: %v",completed.err))}
                goto latestInspected
            }
        case <-deadline.C:
            panic(fmt.Sprintf("Repeated source switching hung; active=%d completed=%d cancelled=%d",
                inspectorActive.Load(),inspectorFinished.Load(),inspectorCancelled.Load()))
        }
    }
latestInspected:
    if app.input!=cci||!app.inputInfo.Valid||app.inputInfo.CRC32!=expected.CRC32||
        app.inputKind!="cci"||gameOutputKind!="cci"||!gameOutputValidated{
        panic(fmt.Sprintf("Latest source identity lost: input=%q valid=%t kind=%s CRC=%s",
            app.input,app.inputInfo.Valid,app.inputKind,app.inputInfo.CRC32))
    }
    if inspectorMaximum.Load()>1||inspectorActive.Load()>0{
        panic(fmt.Sprintf("Overlapping source workers detected: maximum=%d active=%d",
            inspectorMaximum.Load(),inspectorActive.Load()))
    }
    if app.selectedGame!=0||app.selectedPatch!=0||app.parsed!=chosen||
        chosen.Groups[0].Enabled||chosen.Records[0].Enabled{
        panic("Repeated source switching changed the selected patch or its record toggles")
    }

    // Immediately repeat with a loose XBE as the final requested source.
    // The previous CCI result must never replace the final XBE.
    for i:=0;i<90;i++{openInput(sources[i%len(sources)])}
    openInput(rawXBE)
    finalGeneration:=app.gameInputGeneration
    deadline2:=time.NewTimer(20*time.Second)
    defer deadline2.Stop()
    for{
        select{
        case completed:=<-gameInspectReady:
            commitGameInspection(completed)
            if completed.generation==finalGeneration{
                if completed.err!=nil{panic(completed.err)}
                goto xbeInspected
            }
        case <-deadline2.C:
            panic("Latest XBE blocked by obsolete ISO/CCI inspection")
        }
    }
xbeInspected:
    if app.input!=rawXBE||app.inputKind!="xbe"||
        app.inputInfo.CRC32!=expected.CRC32||
        app.parsed!=chosen||chosen.Records[0].Enabled||
        inspectorMaximum.Load()>1{
        panic("Final XBE identity or selected patch corrupted after ISO/CCI/XBE stress")
    }

    runtime.GC()
    var afterMemory runtime.MemStats
    runtime.ReadMemStats(&afterMemory)
    if afterMemory.Alloc>beforeMemory.Alloc+96<<20{
        panic(fmt.Sprintf("Large retained memory growth after repeated CCI switching: before=%d after=%d",
            beforeMemory.Alloc,afterMemory.Alloc))
    }
    afterHandles:=smokeProcessHandleCount()
    if beforeHandles!=0&&afterHandles>beforeHandles+32{
        panic(fmt.Sprintf("Possible leaked source handles: before=%d after=%d",beforeHandles,afterHandles))
    }
}

func smokeProcessHandleCount()uint32{
    process:=sysCall(k32.NewProc("GetCurrentProcess"))
    var count uint32
    proc:=k32.NewProc("GetProcessHandleCount")
    r,_,_:=proc.Call(process,uintptr(unsafe.Pointer(&count)))
    if r==0{return 0}
    return count
}
