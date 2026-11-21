package core

import (
    "bytes"
    "fmt"
    "math/rand"
    "os"
    "path/filepath"
    "strings"
    "testing"
)

func TestXboxGameMemberRejectsWindowsDeviceAliases(t *testing.T) {
    for _,path:=range []string{"CON.txt.foo","AUX.log","folder/PRN.txt_more","folder/con .txt","NUL.bin","folder/COM1.log","LPT9.txt.extra","../escape","folder//gap"} {
        if _,err:=validGameMember(path);err==nil {
            t.Errorf("unsafe XDVDFS member accepted: %s",path)
        }
    }
    for _,path:=range []string{"default.xbe","media/texture.bin","foo.CON","folder/fine.ext.more"}{
        if _,err:=validGameMember(path);err!=nil{
            t.Errorf("valid XDVDFS member rejected: %s: %v",path,err)
        }
    }
}

func TestGameXISORejectsMissingOrExtraDirectory(t *testing.T) {
    root:=gameFixture(t)
    iso:=filepath.Join(t.TempDir(),"game.iso")
    if err:=BuildGameXISO(root,iso);err!=nil{t.Fatal(err)}
    if err:=VerifyGameXISO(iso,root);err!=nil{t.Fatal(err)}
    if err:=os.Remove(filepath.Join(root,"maps","empty"));err!=nil{t.Fatal(err)}
    if err:=VerifyGameXISO(iso,root);err==nil{
        t.Fatal("missing empty source folder went undetected")
    }
    if err:=os.MkdirAll(filepath.Join(root,"maps","empty"),0755);err!=nil{t.Fatal(err)}
    if err:=os.MkdirAll(filepath.Join(root,"unexpected","empty"),0755);err!=nil{t.Fatal(err)}
    if err:=VerifyGameXISO(iso,root);err==nil{
        t.Fatal("extra source directory went undetected")
    }
}

func TestReadGameMemberIndependentOfSourceCase(t *testing.T) {
    root:=gameFixture(t)
    data,err:=ReadGameMember(root,"AUDIO/MUSIC.BIN")
    if err!=nil{t.Fatal(err)}
    if len(data)!=16384{t.Fatalf("wrong member length: %d",len(data))}
    if _,err:=ReadGameMember(root,"audio/../music.bin");err==nil{
        t.Fatal("traversal accepted in game-member reader")
    }
    if _,err:=ReadGameMember(root,"does-not-exist.dat");err==nil{
        t.Fatal("missing member read as valid")
    }
    // Case-insensitive modified asset must be placed in its actual source
    // directory, without manufacturing a second file.
    dest:=filepath.Join(t.TempDir(),"bundle")
    modified:=[]byte("checked case-insensitive target")
    if err:=ExportGameOutputs(root,dest,GameOutputFolder|GameOutputXISO,map[string][]byte{"AUDIO/MUSIC.BIN":modified});err!=nil{t.Fatal(err)}
    result,err:=os.ReadFile(filepath.Join(dest,"Extracted","audio","music.bin"))
    if err!=nil{t.Fatal(err)}
    if !bytes.Equal(result,modified){t.Fatal("named patched asset was not updated")}
}

func TestCCIRejectsSplitGapsAndAmbiguousInput(t *testing.T) {
    d:=t.TempDir()
    first:=filepath.Join(d,"game.1.cci")
    third:=filepath.Join(d,"game.3.cci")
    for _,name:=range []string{first,third}{
        if err:=os.WriteFile(name,[]byte("test stub"),0644);err!=nil{t.Fatal(err)}
    }
    if _,err:=discoverCCIParts(first);err==nil||!strings.Contains(err.Error(),"missing part 2"){
        t.Fatalf("missing middle CCI segment was silently omitted: %v",err)
    }
    if err:=os.WriteFile(filepath.Join(d,"game.2.cci"),[]byte("test stub"),0644);err!=nil{t.Fatal(err)}
    parts,err:=discoverCCIParts(first)
    if err!=nil{t.Fatal(err)}
    if len(parts)!=3{t.Fatalf("discovered %d parts, expected 3",len(parts))}
    if err:=os.WriteFile(filepath.Join(d,"game.cci"),[]byte("standalone"),0644);err!=nil{t.Fatal(err)}
    if _,err:=discoverCCIParts(filepath.Join(d,"game.cci"));err==nil||!strings.Contains(err.Error(),"ambiguous"){
        t.Fatalf("ambiguous CCI base image wasn't rejected: %v",err)
    }
    if _,err:=discoverCCIParts(third);err!=nil{
        t.Fatalf("explicit split part should find complete set: %v",err)
    }
    if _,err:=discoverCCIParts(filepath.Join(d,"game.4.cci"));err==nil{
        t.Fatal("nonexistent split part accepted")
    }
}

func TestCCISmallSplitRoundTripAndMissingSegment(t *testing.T){
    source:=gameFixture(t)
    noisy:=make([]byte,90<<10)
    rng:=rand.New(rand.NewSource(7123))
    if _,err:=rng.Read(noisy);err!=nil{t.Fatal(err)}
    if err:=os.WriteFile(filepath.Join(source,"levels","noise.dat"),noisy,0644);err!=nil{t.Fatal(err)}
    iso:=filepath.Join(t.TempDir(),"original.iso")
    if err:=BuildGameXISO(source,iso);err!=nil{t.Fatal(err)}
    output:=filepath.Join(t.TempDir(),"split.cci")
    if err:=compressCCIWithLimit(iso,output,8192,nil);err!=nil{t.Fatal(err)}
    first:=strings.TrimSuffix(output,".cci")+".1.cci"
    second:=strings.TrimSuffix(output,".cci")+".2.cci"
    if _,err:=os.Stat(first);err!=nil{t.Fatalf("CCI did not split: %v",err)}
    if _,err:=os.Stat(second);err!=nil{t.Fatalf("CCI missing part 2: %v",err)}
    out:=filepath.Join(t.TempDir(),"roundtrip.iso")
    if err:=ExtractCCI(first,out,nil);err!=nil{t.Fatal(err)}
    if err:=VerifyGameXISO(out,source);err!=nil{t.Fatalf("CCI split roundtrip: %v",err)}
    // The set must fail closed on a gap instead of returning a deceptively
    // smaller but apparently valid CCI image.
    if err:=os.Remove(second);err!=nil{t.Fatal(err)}
    if _,err:=OpenCCI(first);err==nil{
        t.Fatal("missing middle segment didn't fail")
    }else{t.Logf("Missing CCI segment correctly rejected: %v",err)}
}

func TestCCIStandalonePartsAreNotCrossSelected(t *testing.T) {
    d:=t.TempDir()
    path:=filepath.Join(d,"game.cci")
    if err:=os.WriteFile(path,[]byte("standalone stub"),0644);err!=nil{t.Fatal(err)}
    got,err:=discoverCCIParts(path)
    if err!=nil{t.Fatal(err)}
    if len(got)!=1 || got[0]!=path{t.Fatal(fmt.Sprintf("unexpected standalone CCI selection: %#v",got))}
}
