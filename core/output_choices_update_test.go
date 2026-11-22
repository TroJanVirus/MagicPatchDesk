package core

import (
    "crypto/sha256"
    "encoding/hex"
    "os"
    "path/filepath"
    "strings"
    "testing"
)
func TestExplicitGameOutputsAndPreserveCurrentFormat(t *testing.T){
    samples:=[]struct{kind string;preserve uint32}{
        {"xbe",GameOutputXBE},
        {"folder",GameOutputFolder},
        {"xiso",GameOutputXISO},
        {"cci",GameOutputCCI},
        {"fulliso",GameOutputISO},
    }
    for _,c:=range samples{
        t.Run(c.kind,func(t *testing.T){
            if _,e:=ResolveGameOutputChoice(c.kind,0,false);e==nil{
                t.Fatal("Apply must not become enabled before explicit format selection")
            }
            got,e:=ResolveGameOutputChoice(c.kind,0,true)
            if e!=nil||got!=c.preserve{t.Fatalf("preserve source type got %b, want %b: %v",got,c.preserve,e)}
            allowed,e:=AllowedOutputFormats(c.kind)
            if e!=nil||allowed&c.preserve==0{t.Fatalf("preserved kind not allowed: %b %v",allowed,e)}
            if c.kind!="fulliso"{
                if _,e=ResolveGameOutputChoice(c.kind,GameOutputISO,false);e==nil{
                    t.Fatal("invalid full-disc ISO conversion accepted for non-full-disc input")
                }
            }
        })
    }
    both,e:=ResolveGameOutputChoice("xiso",GameOutputCCI|GameOutputFolder,true)
    if e!=nil||both!=(GameOutputXISO|GameOutputCCI|GameOutputFolder){
        t.Fatalf("preserve plus multi-select did not combine uniquely: %b %v",both,e)
    }
    if _,e:=ResolveGameOutputChoice("xbe",GameOutputXBE|GameOutputCCI,false);e==nil{
        t.Fatal("loose XBE was allowed to invent missing complete-game resources")
    }
    if _,e:=ResolveGameOutputChoice("",GameOutputXBE,false);e==nil{
        t.Fatal("source-less export was accepted")
    }
    if _,e:=ResolveGameOutputChoice("cci",GameOutputAll,false);e==nil{
        t.Fatal("CCI input was incorrectly allowed to make full-disc ISO")
    }
}
func TestPortableUpdateIntentPreventsWrongTargetOrHash(t *testing.T){
    root:=t.TempDir()
    original:=filepath.Join(root,"MagicPatchDesk-Windows-x64.exe")
    stage:=filepath.Join(root,".mpd-update-test.exe")
    if e:=os.WriteFile(original,[]byte("old exe"),0600);e!=nil{t.Fatal(e)}
    newBytes:=[]byte("verified replacement exe")
    if e:=os.WriteFile(stage,newBytes,0600);e!=nil{t.Fatal(e)}
    checksum:=sha256.Sum256(newBytes)
    digest:=hex.EncodeToString(checksum[:])
    if e:=SavePortableUpdateIntent(stage,original,digest,410);e!=nil{t.Fatal(e)}
    if e:=VerifyPortableUpdateIntent(stage,original,digest,410);e!=nil{t.Fatal(e)}
    if e:=VerifyPortableUpdateIntent(stage,original,digest,411);e==nil{t.Fatal("wrong parent accepted")}
    if e:=VerifyPortableUpdateIntent(stage,filepath.Join(root,"other.exe"),digest,410);e==nil{t.Fatal("unexpected update target accepted")}
    if e:=VerifyPortableUpdateIntent(stage,original,strings.Repeat("0",64),410);e==nil{t.Fatal("wrong checksum accepted")}
    if e:=os.WriteFile(stage,[]byte("tampered"),0600);e!=nil{t.Fatal(e)}
    if e:=VerifyPortableUpdateIntent(stage,original,digest,410);e==nil{t.Fatal("tampered download accepted")}
    if e:=os.WriteFile(stage,newBytes,0600);e!=nil{t.Fatal(e)}
    if e:=SavePortableUpdateIntent(stage,original,digest,410);e==nil{
        t.Fatal("update intent unexpectedly overwritten")
    }
    if e:=os.Remove(PortableUpdateIntentPath(stage));e!=nil{t.Fatal(e)}
    if e:=VerifyPortableUpdateIntent(stage,original,digest,410);e==nil{
        t.Fatal("portable updater accepted a target without user-generated authorization")
    }
}
func TestPortableIntentRestrictsStagingToSameFolder(t *testing.T){
    root:=t.TempDir()
    source:=filepath.Join(root,"MagicPatchDesk-Windows-x64.exe")
    content:=[]byte("fake signed artifact")
    if e:=os.WriteFile(source,content,0600);e!=nil{t.Fatal(e)}
    digestBytes:=sha256.Sum256(content)
    digest:=hex.EncodeToString(digestBytes[:])
    outside:=filepath.Join(t.TempDir(),".mpd-update-external.exe")
    if e:=os.WriteFile(outside,content,0600);e!=nil{t.Fatal(e)}
    if e:=SavePortableUpdateIntent(outside,source,digest,14);e==nil{
        t.Fatal("cross-folder stage was allowed")
    }
    normal:=filepath.Join(root,"unexpected.zip")
    if e:=os.WriteFile(normal,content,0600);e!=nil{t.Fatal(e)}
    if e:=SavePortableUpdateIntent(normal,source,digest,14);e==nil{t.Fatal("untrusted stage name accepted")}
}
func TestManagedAndPortableUpdateRollbacksHaveIndependentNames(t *testing.T){
    root:=t.TempDir()
    managed:=filepath.Join(root,"Programs","MagicPatchDesk","MagicPatchDesk.exe")
    installedBackup,e:=UpdateTargetBackup(managed,managed)
    if e!=nil||filepath.Base(installedBackup)!="MagicPatchDesk.previous.exe"{
        t.Fatalf("managed updater backup changed: %s %v",installedBackup,e)
    }
    portable:=filepath.Join(root,"Downloads","MagicPatchDesk-Windows-x64.exe")
    portableBackup,e:=UpdateTargetBackup(portable,managed)
    if e!=nil||filepath.Base(portableBackup)!="MagicPatchDesk-Windows-x64.previous.exe"{
        t.Fatalf("portable backup incorrect: %s %v",portableBackup,e)
    }
    customName:=filepath.Join(root,"Desktop","My Magic Patcher.exe")
    customBackup,e:=UpdateTargetBackup(customName,managed)
    if e!=nil||filepath.Base(customBackup)!="My Magic Patcher.previous.exe"{
        t.Fatalf("renamed portable EXE rejected: %s %v",customBackup,e)
    }
    for _,bad:=range []string{"MagicPatchDeskUpdater.exe","MagicPatchDesk.previous.exe",".mpd-update-staged.exe","notes.txt"}{
        if _,e:=UpdateTargetBackup(filepath.Join(root,bad),managed);e==nil{
            t.Fatalf("helper/backup/non-exe update target accepted: %s",bad)
        }
    }
}
