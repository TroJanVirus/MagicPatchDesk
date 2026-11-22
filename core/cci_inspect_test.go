package core

import (
    "bytes"
    "io"
    "os"
    "path/filepath"
    "strings"
    "testing"
)

func makeSplitGameCCI(t *testing.T)(string,[]byte,[]byte){
    t.Helper()
    root:=t.TempDir()
    folder:=filepath.Join(root,"game")
    if err:=os.Mkdir(folder,0755);err!=nil{t.Fatal(err)}
    // A valid XBE followed by incompressible bytes forces its data to cross
    // one or more CCI file boundaries. Using a compressed zero fixture would
    // not exercise logical part offsets correctly.
    original:=append([]byte(nil),fakeXBE()...)
    var step uint32=0xA19F72D5
    for i:=0;i<320000;i++{
        step^=step<<13;step^=step>>17;step^=step<<5
        original=append(original,byte(step))
    }
    sidecar:=make([]byte,190000)
    for i:=range sidecar{
        step^=step<<13;step^=step>>17;step^=step<<5
        sidecar[i]=byte(step)
    }
    for path,data:=range map[string][]byte{"default.xbe":original,"extra.bin":sidecar}{
        if err:=os.WriteFile(filepath.Join(folder,path),data,0600);err!=nil{t.Fatal(err)}
    }
    iso:=filepath.Join(root,"game.iso")
    if err:=BuildGameXISO(folder,iso);err!=nil{t.Fatal(err)}
    cci:=filepath.Join(root,"game.cci")
    // Much smaller than the FATX limit for deterministic multipart coverage.
    if err:=compressCCIWithLimit(iso,cci,86016,nil);err!=nil{t.Fatal(err)}
    if _,err:=os.Stat(filepath.Join(root,"game.3.cci"));err!=nil{
        t.Fatalf("split fixture should have at least 3 parts: %v",err)
    }
    return cci,original,sidecar
}
func TestSelectedAnyCCIPartInspectsTheWholeGame(t *testing.T){
    base,xbe,sidecar:=makeSplitGameCCI(t)
    first:=strings.TrimSuffix(base,".cci")+".1.cci"
    middle:=strings.TrimSuffix(base,".cci")+".2.cci"
    third:=strings.TrimSuffix(base,".cci")+".3.cci"
    expected,err:=InspectXBE(xbe)
    if err!=nil{t.Fatal(err)}
    for _,path:=range []string{first,middle,third}{
        t.Run(filepath.Base(path),func(t *testing.T){
            got,err:=ReadCCIGameMember(path,"")
            if err!=nil{t.Fatal(err)}
            if !bytes.Equal(got,xbe){t.Fatal("default.xbe differs across selected split parts")}
            extra,err:=ReadCCIGameMember(path,"extra.bin")
            if err!=nil||!bytes.Equal(extra,sidecar){
                t.Fatalf("sidecar read failed across split set: %v",err)
            }
            info,err:=InspectCCIExecutable(path)
            if err!=nil{t.Fatal(err)}
            if info.TitleID!=expected.TitleID||info.Title!=expected.Title||info.CRC32!=expected.CRC32{
                t.Fatalf("CCI metadata differs from raw XBE: %+v vs %+v",info,expected)
            }
        })
    }
    reader,err:=OpenCCI(middle)
    if err!=nil{t.Fatal(err)}
    block:=make([]byte,4096)
    readAt:=reader.Parts[0].sectors*2048-731
    n,err:=reader.ReadAt(block,readAt)
    if err!=nil||n!=len(block){t.Fatalf("read crossing CCI part boundary: %d %v",n,err)}
    decoded:=filepath.Join(t.TempDir(),"reassembled.iso")
    if err:=ExtractCCI(third,decoded,nil);err!=nil{t.Fatal(err)}
    raw,err:=os.ReadFile(decoded)
    if err!=nil{t.Fatal(err)}
    if !bytes.Equal(block,raw[readAt:readAt+int64(len(block))]){
        t.Fatal("random-access cross-part bytes differ from fully decoded ISO")
    }
    n,err=reader.ReadAt(make([]byte,12),reader.Sectors*2048-4)
    if n!=4||err!=io.EOF{t.Fatalf("truncated reader-at semantics incorrect: %d %v",n,err)}
    reader.Close()
}
func TestIncompleteSplitCCIFailsRatherThanReadingOnePart(t *testing.T){
    path,_,_:=makeSplitGameCCI(t)
    stem:=strings.TrimSuffix(path,".cci")
    if err:=os.Remove(stem+".2.cci");err!=nil{t.Fatal(err)}
    if _,err:=InspectCCIExecutable(stem+".3.cci");err==nil||!strings.Contains(err.Error(),"missing part 2"){
        t.Fatalf("CCI silently skipped a missing middle part: %v",err)
    }
    if err:=os.Remove(stem+".1.cci");err!=nil{t.Fatal(err)}
    if _,err:=InspectCCIExecutable(stem+".3.cci");err==nil||!strings.Contains(strings.ToLower(err.Error()),"first part"){
        t.Fatalf("CCI silently inspected a later part without first: %v",err)
    }
}
