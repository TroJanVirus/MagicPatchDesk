package core

import (
    "bytes"
    "context"
    "errors"
    "os"
    "path/filepath"
    "testing"
)

func sourceInspectionFixtures(t *testing.T)(string,string,string,XBEInfo){
    t.Helper()
    root:=t.TempDir()
    game:=filepath.Join(root,"game")
    if err:=os.MkdirAll(game,0755);err!=nil{t.Fatal(err)}
    xbe:=fakeXBE()
    raw:=filepath.Join(root,"default.xbe")
    if err:=os.WriteFile(raw,xbe,0600);err!=nil{t.Fatal(err)}
    if err:=os.WriteFile(filepath.Join(game,"default.xbe"),xbe,0600);err!=nil{t.Fatal(err)}
    if err:=os.WriteFile(filepath.Join(game,"extra.bin"),[]byte("original Xbox asset"),0600);err!=nil{t.Fatal(err)}
    iso:=filepath.Join(root,"game.iso")
    if err:=BuildGameXISO(game,iso);err!=nil{t.Fatal(err)}
    cci:=filepath.Join(root,"game.cci")
    if err:=CompressCCI(iso,cci,nil);err!=nil{t.Fatal(err)}
    info,err:=InspectXBE(xbe)
    if err!=nil{t.Fatal(err)}
    return raw,iso,cci,info
}

func TestRepeatedMixedSourcesInspectSameUnmodifiedXBE(t *testing.T){
    xbe,iso,cci,want:=sourceInspectionFixtures(t)
    sources:=[]struct{path,kind string}{
        {iso,"xiso"},{cci,"cci"},{xbe,"xbe"},
    }
    before:=map[string][]byte{}
    for _,source:=range sources{
        data,err:=os.ReadFile(source.path)
        if err!=nil{t.Fatal(err)}
        before[source.path]=data
    }
    for i:=0;i<120;i++{
        source:=sources[i%len(sources)]
        inspection,err:=InspectGameSourceContext(context.Background(),source.path)
        if err!=nil{t.Fatalf("inspection #%d (%s): %v",i,source.kind,err)}
        if inspection.Kind!=source.kind||!inspection.Info.Valid||
            inspection.Info.CRC32!=want.CRC32||inspection.Info.TitleID!=want.TitleID{
            t.Fatalf("inspection #%d mismatch: %+v vs %+v",i,inspection,want)
        }
    }
    for _,source:=range sources{
        after,err:=os.ReadFile(source.path)
        if err!=nil{t.Fatal(err)}
        if !bytes.Equal(before[source.path],after){t.Fatal("source image modified: "+source.path)}
    }
}

func TestCancelledMixedSourcesReturnContextCanceled(t *testing.T){
    xbe,iso,cci,_:=sourceInspectionFixtures(t)
    paths:=[]string{xbe,iso,cci}
    for i:=0;i<90;i++{
        ctx,cancel:=context.WithCancel(context.Background())
        cancel()
        _,err:=InspectGameSourceContext(ctx,paths[i%len(paths)])
        if !errors.Is(err,context.Canceled){
            t.Fatalf("cancelled scan %d did not stop immediately: %v",i,err)
        }
    }
}
func TestCancelledXDVDFSReaderStopsDuringDirectoryTraversal(t *testing.T){
    src:=makeMultifileXISO(t,false)
    ctx,cancel:=context.WithCancel(context.Background())
    cancel()
    if _,err:=OpenXISOContext(ctx,src);!errors.Is(err,context.Canceled){
        t.Fatalf("cancelled XDVDFS parser continued: %v",err)
    }
}
func TestCancelledCCIReaderStopsBeforeIndexAllocation(t *testing.T){
    _,_,cci,_:=sourceInspectionFixtures(t)
    ctx,cancel:=context.WithCancel(context.Background())
    cancel()
    if _,err:=OpenCCIContext(ctx,cci);!errors.Is(err,context.Canceled){
        t.Fatalf("cancelled CCI index still opened: %v",err)
    }
}
