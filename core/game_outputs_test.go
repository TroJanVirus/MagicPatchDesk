package core

import (
    "bytes"
    "os"
    "path/filepath"
    "strings"
    "testing"
)

func gameFixture(t *testing.T) string {
    t.Helper()
    dir:=filepath.Join(t.TempDir(),"game")
    for _,p:=range []string{"maps/empty","audio","levels/zone"} {
        if err:=os.MkdirAll(filepath.Join(dir,filepath.FromSlash(p)),0755);err!=nil{t.Fatal(err)}
    }
    for name,data:=range map[string][]byte{
        "default.xbe":[]byte("XBEH\000original game executable"),
        "audio/music.bin":bytes.Repeat([]byte{17,34,51,68},4096),
        "levels/zone/terrain.dat":bytes.Repeat([]byte{0xa5},2800),
        "maps/level.txt":[]byte("hello file map"),
    }{
        if err:=os.WriteFile(filepath.Join(dir,filepath.FromSlash(name)),data,0644);err!=nil{t.Fatal(err)}
    }
    return dir
}
func TestGameXISOFolderRoundTrip(t *testing.T) {
    folder:=gameFixture(t)
    iso:=filepath.Join(t.TempDir(),"test.iso")
    if err:=BuildGameXISO(folder,iso);err!=nil{t.Fatal(err)}
    if err:=VerifyGameXISO(iso,folder);err!=nil{t.Fatal(err)}
    extracted:=filepath.Join(t.TempDir(),"extracted")
    if err:=ExtractGameXISO(iso,extracted);err!=nil{t.Fatal(err)}
    if _,err:=os.Stat(filepath.Join(extracted,"maps","empty"));err!=nil{
        t.Fatalf("empty directory not preserved: %v",err)
    }
    for _,name:=range []string{"default.xbe","audio/music.bin","maps/level.txt","levels/zone/terrain.dat"}{
        a,e:=os.ReadFile(filepath.Join(folder,filepath.FromSlash(name)));if e!=nil{t.Fatal(e)}
        b,e:=os.ReadFile(filepath.Join(extracted,filepath.FromSlash(name)));if e!=nil{t.Fatal(e)}
        if !bytes.Equal(a,b){t.Fatalf("roundtrip changed %s",name)}
    }
    if err:=BuildGameXISO(folder,iso);err==nil {
        t.Fatal("overwrote existing XISO")
    }
}
func TestGameImageMultiFormatOutputs(t *testing.T) {
    original:=gameFixture(t)
    xiso:=filepath.Join(t.TempDir(),"original.iso")
    if err:=BuildGameXISO(original,xiso);err!=nil{t.Fatal(err)}
    output:=filepath.Join(t.TempDir(),"patched")
    patched:=[]byte("XBEH patched file -- not an ISO attacher")
    mask:=GameOutputXISO|GameOutputFolder|GameOutputCCI|GameOutputXBE
    if err:=ExportGameOutputs(xiso,output,mask,map[string][]byte{"default.xbe":patched});err!=nil{t.Fatal(err)}
    b,err:=os.ReadFile(filepath.Join(output,"patched.xbe"))
    if err!=nil{t.Fatal(err)}
    if !bytes.Equal(b,patched){t.Fatal("wrong XBE-only export")}
    extracted:=filepath.Join(output,"Extracted")
    b,err=os.ReadFile(filepath.Join(extracted,"default.xbe"))
    if err!=nil{t.Fatal(err)}
    if !bytes.Equal(b,patched){t.Fatal("patch not in extracted folder")}
    if err:=VerifyGameXISO(filepath.Join(output,"patched.xiso.iso"),extracted);err!=nil{t.Fatal(err)}
    cci:=filepath.Join(output,"patched.cci")
    if _,err:=os.Stat(cci);os.IsNotExist(err){cci=filepath.Join(output,"patched.1.cci")}
    decoded:=filepath.Join(t.TempDir(),"decoded.iso")
    if err:=ExtractCCI(cci,decoded,nil);err!=nil{t.Fatal(err)}
    if err:=VerifyGameXISO(decoded,extracted);err!=nil{t.Fatal(err)}
    if err:=ExportGameOutputs(xiso,output,GameOutputXBE,nil);err==nil {
        t.Fatal("should not overwrite existing output package")
    }
}
func TestGameOutputRejectsIncompleteOrUnsafe(t *testing.T) {
    source:=gameFixture(t)
    replacement:=map[string][]byte{"audio/music.bin":[]byte("changed")}
    dst:=filepath.Join(t.TempDir(),"out")
    if e:=ExportGameOutputs(source,dst,GameOutputXBE,replacement);e==nil || !strings.Contains(e.Error(),"omit"){
        t.Fatalf("multi-file XBE-only conversion should fail: %v",e)
    }
    if e:=ExportGameOutputs(source,dst,GameOutputISO,nil);e==nil {
        t.Fatal("must refuse full-disc ISO from extracted folder")
    }
    if e:=ExportGameOutputs(source,dst,GameOutputXISO,map[string][]byte{"../outside":[]byte("hack")});e==nil {
        t.Fatal("must reject traversal")
    }
    if e:=ExportGameOutputs(source,dst,GameOutputXISO,map[string][]byte{"absent.dat":[]byte("hack")});e==nil{
        t.Fatal("must reject missing patch targets")
    }
    if _,e:=os.Stat(dst);!os.IsNotExist(e){t.Fatal("failure left published output")}
}
func TestGameXISORejectSymlinks(t *testing.T) {
    source:=gameFixture(t)
    victim:=filepath.Join(t.TempDir(),"outside.bin")
    if err:=os.WriteFile(victim,[]byte("outside"),0644);err!=nil{t.Fatal(err)}
    if err:=os.Symlink(victim,filepath.Join(source,"escape.dat"));err!=nil{t.Skipf("symlink unavailable: %v",err)}
    if err:=BuildGameXISO(source,filepath.Join(t.TempDir(),"invalid.iso"));err==nil {
        t.Fatal("unsafe symlink packed")
    }
}


func TestGameImageNamedFileOutput(t *testing.T) {
    original:=gameFixture(t)
    destination:=filepath.Join(t.TempDir(),"named")
    changed:=[]byte("named asset was patched")
    mask:=GameOutputFolder|GameOutputXISO|GameOutputCCI
    if err:=ExportGameOutputs(original,destination,mask,map[string][]byte{"audio/music.bin":changed});err!=nil{
        t.Fatal(err)
    }
    b,err:=os.ReadFile(filepath.Join(destination,"Extracted","audio","music.bin"))
    if err!=nil{t.Fatal(err)}
    if !bytes.Equal(b,changed){t.Fatal("named target missing in extracted result")}
    if err:=VerifyGameXISO(filepath.Join(destination,"named.xiso.iso"),filepath.Join(destination,"Extracted"));err!=nil{t.Fatal(err)}
}

func TestGameImageLooseXBEOnly(t *testing.T) {
    game:=filepath.Join(t.TempDir(),"original.xbe")
    original:=[]byte("XBEH standalone file")
    if err:=os.WriteFile(game,original,0644);err!=nil{t.Fatal(err)}
    output:=filepath.Join(t.TempDir(),"single")
    if err:=ExportGameOutputs(game,output,GameOutputXBE,nil);err!=nil{t.Fatal(err)}
    content,err:=os.ReadFile(filepath.Join(output,"single.xbe"))
    if err!=nil{t.Fatal(err)}
    if !bytes.Equal(content,original){t.Fatal("XBE only changed executable")}
    if err:=ExportGameOutputs(game,filepath.Join(t.TempDir(),"not-possible"),GameOutputCCI,nil);err==nil{
        t.Fatal("standalone XBE should not become fake CCI game image")
    }
}

func TestGameMemberCaseInsensitiveResolution(t *testing.T) {
    root:=gameFixture(t)
    if err:=os.Rename(filepath.Join(root,"default.xbe"),filepath.Join(root,"DEFAULT.XBE"));err!=nil{t.Fatal(err)}
    if err:=os.Rename(filepath.Join(root,"audio","music.bin"),filepath.Join(root,"audio","MUSIC.BIN"));err!=nil{t.Fatal(err)}
    output:=filepath.Join(t.TempDir(),"case-output")
    if err:=ExportGameOutputs(root,output,GameOutputXBE|GameOutputFolder,map[string][]byte{"default.xbe":[]byte("XBEH patched")});err!=nil{t.Fatal(err)}
    b,err:=os.ReadFile(filepath.Join(output,"case-output.xbe"))
    if err!=nil{t.Fatal(err)}
    if string(b)!="XBEH patched"{t.Fatalf("case-insensitive XBE replacement failed: %q",b)}
}
