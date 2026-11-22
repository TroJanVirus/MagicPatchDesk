package core

import (
    "context"
    "encoding/json"
    "errors"
    "os"
    "path/filepath"
    "strings"
    "sync"
    "testing"
    "time"
)

// Regression for the user's v0.14.0-test.17 stack:
// JSON decoder out of sync - data changing underfoot?
// The local cache is optional and must NEVER take down a source selection.
func TestHubCacheDecoderPanicBecomesError(t *testing.T) {
    value,err:=decodeHubCache([]byte(`{"games":[]}`),func([]byte,any)error{
        panic("JSON decoder out of sync - data changing underfoot?")
    })
    if !errors.Is(err,ErrHubCacheInvalid)||len(value.Games)!=0{
        t.Fatalf("decoder panic was not isolated: %+v %v",value,err)
    }
}
func TestInvalidHubCachePreservedWithoutNetwork(t *testing.T) {
    root:=t.TempDir()
    filename:=filepath.Join(root,"patchhub-index.json")
    original:=[]byte(`{"updated": "bad-json", "games": [`)
    if err:=os.WriteFile(filename,original,0600);err!=nil{t.Fatal(err)}
    if _,err:=ReadHubCacheChecked(root);!errors.Is(err,ErrHubCacheInvalid){
        t.Fatalf("invalid index was accepted: %v",err)
    }
    if _,err:=SyncPatchHub(context.Background(),root);!errors.Is(err,ErrHubCacheInvalid){
        t.Fatalf("sync must report corrupt cache without network: %v",err)
    }
    after,err:=os.ReadFile(filename)
    if err!=nil||string(after)!=string(original){
        t.Fatalf("original index was destroyed: %v",err)
    }
}
func TestHubCacheReadWriteStress(t *testing.T) {
    root:=t.TempDir()
    hub:=HubIndex{GalleryScanVersion:2,Updated:time.Now().UTC(),
        Games:[]HubGame{{Title:"Phantom Crash",Versions:[]HubVersion{{TitleID:"504C0001",Region:"USA",Version:"5"}}}}}
    if err:=saveHubIndex(root,hub);err!=nil{t.Fatal(err)}
    var wg sync.WaitGroup
    errorsCh:=make(chan error,12)
    for i:=0;i<12;i++{
        wg.Add(1)
        go func(i int){
            defer wg.Done()
            for j:=0;j<25;j++{
                if i%4==0{
                    if err:=saveHubIndex(root,hub);err!=nil{errorsCh<-err;return}
                }else{
                    value,err:=ReadHubCacheChecked(root)
                    if err!=nil||len(value.Games)!=1||value.Games[0].Title!="Phantom Crash"{
                        if err==nil{err=errors.New("lost valid cache snapshot")}
                        errorsCh<-err;return
                    }
                }
            }
        }(i)
    }
    wg.Wait()
    close(errorsCh)
    for err:=range errorsCh{t.Fatal(err)}
    if body,err:=os.ReadFile(filepath.Join(root,"patchhub-index.json"));err!=nil||!json.Valid(body){t.Fatalf("non-atomic cache write: %v",err)}
}
func TestMissingAndOversizedHubCache(t *testing.T){
    root:=t.TempDir()
    if cache,err:=ReadHubCacheChecked(root);err!=nil||len(cache.Games)!=0{t.Fatalf("first launch should be offline safe: %v",err)}
    file:=filepath.Join(root,"patchhub-index.json")
    if err:=os.WriteFile(file,[]byte(strings.Repeat("a",(24<<20)+1)),0600);err!=nil{t.Fatal(err)}
    if _,err:=ReadHubCacheChecked(root);!errors.Is(err,ErrHubCacheInvalid){t.Fatalf("oversized cache not bounded: %v",err)}
}

func TestRecoveredBackgroundPanicPreservesActiveSession(t *testing.T){
    root:=t.TempDir()
    session,err:=BeginCrashSession(root,"v0.14.0-test.17",31337)
    if err!=nil{t.Fatal(err)}
    if err:=session.SetStage("game_inspect_pending");err!=nil{t.Fatal(err)}
    if err:=session.SaveRecoveredBackgroundPanic("patchhub_sync","decoder failed",[]byte("synthetic private stack"));err!=nil{t.Fatal(err)}
    if _,err:=os.Stat(session.marker);err!=nil{t.Fatalf("background panic ended app session: %v",err)}
    notices,err:=PendingCrashNotices(root)
    if err!=nil||notices!=1{t.Fatalf("recovered panic not visible in reports: %d %v",notices,err)}
    entries,err:=os.ReadDir(CrashDirectory(root))
    if err!=nil{t.Fatal(err)}
    hasStack:=false
    for _,entry:=range entries {
        if strings.HasPrefix(entry.Name(),"crash-"){
            body,_:=os.ReadFile(filepath.Join(CrashDirectory(root),entry.Name()))
            hasStack=strings.Contains(string(body),"synthetic private stack")
        }
    }
    if !hasStack{t.Fatal("private stack trace not preserved")}
    if err:=session.End();err!=nil{t.Fatal(err)}
    if _,err:=os.Stat(session.marker);!errors.Is(err,os.ErrNotExist){t.Fatal("clean exit marker was not cleared")}
    if crashStage("game_inspect_pending")!="game_inspect_pending"{t.Fatal("source inspection diagnostic stage discarded")}
}
