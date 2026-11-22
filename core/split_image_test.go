package core

import (
    "bytes"
    "os"
    "path/filepath"
    "strings"
    "testing"
)

func makeRawSplitGameFixture(t *testing.T)(string,[]byte,[]byte){
    t.Helper()
    root:=t.TempDir()
    folder:=filepath.Join(root,"game")
    if err:=os.Mkdir(folder,0755);err!=nil{t.Fatal(err)}
    xbe:=append([]byte(nil),fakeXBE()...)
    sidecar:=make([]byte,240000)
    var step uint32=0xA19F72D5
    for i:=0;i<230000;i++{
        step^=step<<13;step^=step>>17;step^=step<<5
        xbe=append(xbe,byte(step))
    }
    for i:=range sidecar{
        step^=step<<13;step^=step>>17;step^=step<<5
        sidecar[i]=byte(step)
    }
    for name,data:=range map[string][]byte{"default.xbe":xbe,"extra.bin":sidecar}{
        if err:=os.WriteFile(filepath.Join(folder,name),data,0600);err!=nil{t.Fatal(err)}
    }
    iso:=filepath.Join(root,"game.iso")
    if err:=BuildGameXISO(folder,iso);err!=nil{t.Fatal(err)}
    data,err:=os.ReadFile(iso)
    if err!=nil{t.Fatal(err)}
    if len(data)<350001{t.Fatalf("unexpectedly small fixture %d",len(data))}
    stem:=filepath.Join(root,"split")
    for i,chunk:=range [][]byte{data[:100001],data[100001:330001],data[330001:]} {
        path:=stem+"."+string(rune('1'+i))+".iso"
        if err:=os.WriteFile(path,chunk,0600);err!=nil{t.Fatal(err)}
    }
    return stem+".2.iso",xbe,sidecar
}
func TestAnyNumberedRawISORecognizesEmbeddedXBEAndExportsFullGame(t *testing.T){
    selected,xbe,sidecar:=makeRawSplitGameFixture(t)
    wantedInfo,err:=InspectXBE(xbe)
    if err!=nil{t.Fatal(err)}
    for _,part:=range []string{".1.iso",".2.iso",".3.iso"} {
        path:=strings.TrimSuffix(selected,".2.iso")+part
        image,err:=OpenXISO(path)
        if err!=nil{t.Fatalf("opening %s: %v",part,err)}
        entry,err:=image.DefaultXBE()
        if err!=nil{t.Fatal(err)}
        data,err:=image.ReadFile(entry)
        if err!=nil||!bytes.Equal(data,xbe){t.Fatalf("default.xbe across %s: %v",part,err)}
        info,err:=InspectXBE(data)
        if err!=nil||info.CRC32!=wantedInfo.CRC32||info.TitleID!=wantedInfo.TitleID{
            t.Fatalf("wrong XBE identity from %s: %+v %v",part,info,err)
        }
        named,err:=image.ResolveTarget("extra.bin")
        if err!=nil{t.Fatal(err)}
        other,err:=image.ReadFile(named)
        if err!=nil||!bytes.Equal(other,sidecar){t.Fatalf("sidecar across %s: %v",part,err)}
        label,err:=RawImagePartSummary(path)
        if err!=nil||!strings.Contains(label,"of 3"){t.Fatalf("wrong selected part label: %q %v",label,err)}
    }
    folder:=filepath.Join(t.TempDir(),"extracted")
    if err:=ExtractGameXISO(selected,folder);err!=nil{t.Fatal(err)}
    actual,err:=os.ReadFile(filepath.Join(folder,"default.xbe"))
    if err!=nil||!bytes.Equal(actual,xbe){t.Fatalf("extraction from part 2 wrong: %v",err)}
    extra,err:=os.ReadFile(filepath.Join(folder,"extra.bin"))
    if err!=nil||!bytes.Equal(extra,sidecar){t.Fatalf("sidecar from split ISO wrong: %v",err)}
    dst:=filepath.Join(t.TempDir(),"patched")
    replacement:=append([]byte(nil),xbe...)
    replacement[0x500]^=0x5A
    if err:=ExportGameOutputs(selected,dst,GameOutputXBE|GameOutputXISO,map[string][]byte{"default.xbe":replacement});err!=nil{t.Fatal(err)}
    actual,err=os.ReadFile(filepath.Join(dst,"patched.xbe"))
    if err!=nil||!bytes.Equal(actual,replacement){t.Fatalf("split game patched XBE output incorrect: %v",err)}
    if _,err=OpenXISO(filepath.Join(dst,"patched.xiso.iso"));err!=nil{t.Fatalf("rebuilt split source XISO unreadable: %v",err)}
    if err=ExportGameOutputs(selected,filepath.Join(t.TempDir(),"unsafe"),GameOutputISO,nil);err==nil{
        t.Fatal("split ISO was treated as a single complete original full-disc image")
    }
}
func TestMissingNumberedRawISOPartsAreNotSilentlyIgnored(t *testing.T){
    selected,_,_:=makeRawSplitGameFixture(t)
    stem:=strings.TrimSuffix(selected,".2.iso")
    if err:=os.Remove(stem+".2.iso");err!=nil{t.Fatal(err)}
    if _,err:=OpenXISO(stem+".3.iso");err==nil||!strings.Contains(err.Error(),"missing part 2"){
        t.Fatalf("missing middle part not detected: %v",err)
    }
    if err:=os.Remove(stem+".1.iso");err!=nil{t.Fatal(err)}
    if _,err:=OpenXISO(stem+".3.iso");err==nil||!strings.Contains(err.Error(),"first part"){
        t.Fatalf("missing first part not detected: %v",err)
    }
}
func TestRawSplitPartNeverCopiedOrCompressedAsStandaloneISO(t *testing.T){
    selected,_,_:=makeRawSplitGameFixture(t)
    out:=filepath.Join(t.TempDir(),"wrong.iso")
    if err:=CopyImage(selected,out,2,nil);err==nil{
        t.Fatal("single numbered fragment was copied as complete image")
    }
    if err:=CompressCCI(selected,filepath.Join(t.TempDir(),"wrong.cci"),nil);err==nil{
        t.Fatal("single numbered raw fragment was compressed as complete CCI")
    }
}

func TestNumberedXISOSuffixSupportsPartTwoAndMissingParts(t *testing.T){
    selected,xbe,_:=makeRawSplitGameFixture(t)
    stem:=strings.TrimSuffix(selected,".2.iso")
    for i:=1;i<=3;i++{
        old:=stem+"."+string(rune('0'+i))+".iso"
        neu:=stem+"."+string(rune('0'+i))+".xiso"
        if err:=os.Rename(old,neu);err!=nil{t.Fatal(err)}
    }
    path:=stem+".2.xiso"
    if !IsNumberedRawImagePart(path){t.Fatal("numbered XISO extension ignored")}
    im,err:=OpenXISO(path)
    if err!=nil{t.Fatal(err)}
    entry,err:=im.DefaultXBE()
    if err!=nil{t.Fatal(err)}
    data,err:=im.ReadFile(entry)
    if err!=nil||!bytes.Equal(data,xbe){t.Fatalf("selected .2.xiso gave wrong default.xbe: %v",err)}
}

func TestDiscPartSetGroupingForDragAndDrop(t *testing.T){
    for _,pair:=range [][2]string{
        {"Halo.1.cci","Halo.3.cci"},
        {"Halo.1.iso","Halo.2.iso"},
        {"Halo.1.xiso","Halo.4.xiso"},
    }{
        a,b:=DiscPartSetKey(pair[0]),DiscPartSetKey(pair[1])
        if a==""||a!=b{t.Fatalf("numbered parts not grouped: %q and %q",a,b)}
    }
    if DiscPartSetKey("Halo.iso")!=""||DiscPartSetKey("Halo.xbe")!=""{
        t.Fatal("standalone disc/XBE incorrectly classified as a numbered set")
    }
}
