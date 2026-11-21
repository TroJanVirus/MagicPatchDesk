//go:build windows

package main

import (
    "context"
    "errors"
    "fmt"
    "magicpatchdesk/core"
    "os"
    "path/filepath"
    "strings"
    "unsafe"
)

// A separate image-output workspace deliberately shares the patch selected
// in Patch Library, but does not force the image source's extension on output.
var gameOutputSource string
var gameOutputMask uint32 = core.GameOutputXISO

func chosenGameSource() string {
    if gameOutputSource!="" {return gameOutputSource}
    return app.input
}
func outputNames(mask uint32) string {
    parts:=[]string{}
    for _,v:=range []struct{mask uint32;name string}{
        {core.GameOutputFolder,"folder"},{core.GameOutputXISO,"XISO"},
        {core.GameOutputCCI,"CCI"},{core.GameOutputXBE,"XBE"},
        {core.GameOutputISO,"full-disc ISO"},
    } {
        if mask&v.mask!=0 {parts=append(parts,v.name)}
    }
    return strings.Join(parts,", ")
}
func gameOutputAction(action string,index int) {
    switch action {
    case "game-source":
        if p:=pickFile(false,"Xbox files\x00*.xbe;*.iso;*.cci\x00All files\x00*.*\x00","Choose original game image or XBE","");p!="" {
            gameOutputSource=p
            switch {
            case core.CCISourcePath(p):gameOutputMask=core.GameOutputCCI
            case strings.EqualFold(filepath.Ext(p),".xbe"):gameOutputMask=core.GameOutputXBE
            default:
                gameOutputMask=core.GameOutputXISO
                if im,e:=core.OpenXISO(p);e==nil && im.Base!=0 {
                    gameOutputMask=core.GameOutputISO
                }
            }
            app.status="Game output source: "+filepath.Base(p)
        }
    case "game-folder":
        // Native open-file dialog locates the *root* default.xbe. This avoids
        // selecting an arbitrary unrelated parent directory by mistake.
        if p:=pickFile(false,"Root game XBE\x00default.xbe\x00Xbox XBE\x00*.xbe\x00","Choose default.xbe inside extracted game folder","");p!="" {
            if !strings.EqualFold(filepath.Base(p),"default.xbe") {
                notify("Select the root default.xbe, not another game executable.")
                return
            }
            gameOutputSource=filepath.Dir(p)
            gameOutputMask=core.GameOutputFolder
            app.status="Extracted game folder selected"
        }
    case "game-use-loaded":
        gameOutputSource=""
        if strings.EqualFold(filepath.Ext(app.input),".xbe"){gameOutputMask=core.GameOutputXBE
        } else if core.CCISourcePath(app.input){gameOutputMask=core.GameOutputCCI
        } else {gameOutputMask=core.GameOutputXISO}
    case "game-format":
        if index>=0 && index<=4 {gameOutputMask ^= uint32(1)<<uint(index)}
    case "game-export":
        startGameOutput()
    case "game-credits":
        url:="https://github.com/TroJanVirus/MagicPatchDesk/blob/main/CREDITS.md"
        sysCall(shellExecute,app.hwnd,uintptr(unsafe.Pointer(ptr("open"))),uintptr(unsafe.Pointer(ptr(url))),0,0,SW_SHOW)
    }
}

func gameOriginals(source string, wanted []string) (map[string][]byte,error) {
    sourceInfo,e:=os.Stat(source)
    if e!=nil{return nil,e}
    sourceFile:=source
    if !sourceInfo.IsDir() && core.CCISourcePath(source){
        scratch,e:=os.MkdirTemp("","mpd-source-cci-*")
        if e!=nil{return nil,e}
        defer os.RemoveAll(scratch)
        sourceFile=filepath.Join(scratch,"decoded.iso")
        if e=core.ExtractCCI(source,sourceFile,nil);e!=nil{return nil,e}
    }
    var image *core.Image
    if !sourceInfo.IsDir() && !strings.EqualFold(filepath.Ext(source),".xbe"){
        image,e=core.OpenXISO(sourceFile)
        if e!=nil{return nil,e}
    }
    out:=make(map[string][]byte,len(wanted))
    for _,name:=range wanted {
        canonical,e:=core.NormaliseTarget(name)
        if e!=nil{return nil,e}
        if image!=nil {
            member,e:=image.ResolveTarget(canonical)
            if e!=nil{return nil,e}
            out[canonical],e=image.ReadFile(member)
            if e!=nil{return nil,e}
            continue
        }
        if sourceInfo.IsDir(){
            p:=filepath.Join(source,filepath.FromSlash(canonical))
            s,e:=os.Lstat(p)
            if e!=nil{return nil,e}
            if !s.Mode().IsRegular(){return nil,fmt.Errorf("source target is not regular: %s",canonical)}
            out[canonical],e=os.ReadFile(p)
            if e!=nil{return nil,e}
            continue
        }
        if canonical!="default.xbe"{return nil,fmt.Errorf("source XBE does not include %s",canonical)}
        out[canonical],e=os.ReadFile(source)
        if e!=nil{return nil,e}
    }
    return out,nil
}

// Every patch member must be present and must successfully verify; never
// silently skip the named files or treat all records as default.xbe edits.
func calculateGameReplacements(source string,selected *core.Patch,direct,password string,override bool)(map[string][]byte,error){
    if direct=="" && selected==nil{return map[string][]byte{},nil}
    ext:=strings.ToLower(filepath.Ext(direct))
    if ext==".pf" {return nil,errors.New("PF must be applied as a complete game-folder operation")}
    if ext==".mpdpack" {
        bytes,e:=readBoundedSource(direct)
        if e!=nil{return nil,e}
        manifest,_,e:=core.ReadPatchPack(bytes)
        if e!=nil{return nil,e}
        targets:=[]string{}
        seen:=map[string]bool{}
        for _,member:=range manifest.Entries {
            if !seen[member.Target] {targets=append(targets,member.Target);seen[member.Target]=true}
        }
        originals,e:=gameOriginals(source,targets)
        if e!=nil{return nil,e}
        return core.ApplyPatchPack(bytes,originals,password)
    }
    p:=selected
    if ext==".jmp" {
        bytes,e:=readBoundedSource(direct)
        if e!=nil{return nil,e}
        p,e=core.ParseJMP(filepath.Base(direct),bytes)
        if e!=nil{return nil,e}
    } else if direct!="" {
        originals,e:=gameOriginals(source,[]string{"default.xbe"})
        if e!=nil{return nil,e}
        newXBE,e:=core.ApplyPatchFile(originals["default.xbe"],direct,password,override)
        if e!=nil{return nil,e}
        return map[string][]byte{"default.xbe":newXBE},nil
    }
    if p==nil{return nil,errors.New("choose a patch or use conversion without a patch")}
    targets,e:=p.TargetFiles()
    if e!=nil{return nil,e}
    if len(targets)==0{return nil,errors.New("no enabled patch records")}
    originalNames:=append([]string{"default.xbe"},targets...)
    originals,e:=gameOriginals(source,originalNames)
    if e!=nil{return nil,e}
    if !override {
        if e=p.VerifyCRC(originals["default.xbe"]);e!=nil{return nil,e}
        if info,e:=core.InspectXBE(originals["default.xbe"]);e==nil {
            expected:=core.TitleIDFromVersion(p.Headers["version"])
            if expected!="" && !strings.EqualFold(info.TitleID,expected){
                return nil,fmt.Errorf("XBE Title ID %s does not match patch %s",info.TitleID,expected)
            }
        }
    }
    outputs:=map[string][]byte{}
    for _,name:=range targets {
        patched,e:=p.ApplyToTarget(originals[name],name,override)
        if e!=nil{return nil,fmt.Errorf("patch %s: %w",name,e)}
        outputs[name]=patched
    }
    return outputs,nil
}

func preparePFGameFolder(source,script string)(string,func(),error){
    stat,e:=os.Stat(source)
    if e!=nil{return "",nil,e}
    if !stat.IsDir() && strings.EqualFold(filepath.Ext(source),".xbe"){
        return "",nil,errors.New("PF operations need an entire game, not a standalone XBE")
    }
    scratch,e:=os.MkdirTemp("","mpd-game-pf-*")
    if e!=nil{return "",nil,e}
    cleanup:=func(){os.RemoveAll(scratch)}
    original:=filepath.Join(scratch,"original")
    if stat.IsDir(){
        e=core.CopyExtractedGame(source,original)
    } else {
        iso:=source
        if core.CCISourcePath(source){
            iso=filepath.Join(scratch,"source.iso")
            e=core.ExtractCCI(source,iso,nil)
        }
        if e==nil{e=core.ExtractGameXISO(iso,original)}
    }
    if e!=nil{cleanup();return "",nil,e}
    data,e:=readBoundedSource(script)
    if e!=nil{cleanup();return "",nil,e}
    pf,e:=core.ParsePF(filepath.Base(script),data)
    if e!=nil{cleanup();return "",nil,e}
    output:=filepath.Join(scratch,"patched")
    _,e=pf.ApplyPFDirectory(context.Background(),original,output,nil)
    if e!=nil{cleanup();return "",nil,e}
    return output,cleanup,nil
}

func startGameOutput(){
    app.mu.Lock()
    if app.busy{app.mu.Unlock();return}
    source,selected,direct:=chosenGameSource(),app.parsed,app.directFile
    mask,password,override:=gameOutputMask,app.password,app.override
    app.mu.Unlock()
    if source=="" {notify("Choose a game image, extracted folder or XBE first.");return}
    if mask==0 {notify("Select at least one output format.");return}
    if mask&core.GameOutputXBE!=0 && mask&core.GameOutputFolder!=0 {
        // Allowed: a folder and an XBE, but each patch member is checked first.
    }
    base:=strings.TrimSuffix(filepath.Base(source),filepath.Ext(source))
    if base==""{base="Game"}
    dest:=pickFile(true,"Output package folder name\x00*.*\x00","Select a NEW output package name",base+"_patched")
    if dest==""{return}
    if info,e:=os.Stat(dest);e==nil && info!=nil{notify("Output already exists; choose a new name.");return}
    if !showPreviewMessage(fmt.Sprintf("Original: %s\n\nRequested formats: %s\n\nOutput package: %s\n\nExisting output is never overwritten. All named patch targets will be checked. Temporary disk space must accommodate extraction and packing.",filepath.Base(source),outputNames(mask),filepath.Base(dest)),true){return}
    app.mu.Lock()
    if app.busy {app.mu.Unlock();return}
    app.busy=true
    app.progress=0
    app.status="Verifying original game and preparing requested output formats..."
    app.mu.Unlock()
    redraw()
    go func(){
        var err error
        workSource:=source
        if strings.EqualFold(filepath.Ext(direct),".pf"){
            if mask&core.GameOutputXBE!=0 {
                err=errors.New("PF may modify additional files: XBE-only output would be incomplete; choose folder, XISO or CCI")
            } else if mask&core.GameOutputISO!=0 {
                err=errors.New("full-disc ISO cannot preserve PF file operations; choose XISO, CCI or folder")
            } else {
                var cleanup func()
                workSource,cleanup,err=preparePFGameFolder(source,direct)
                if cleanup!=nil{defer cleanup()}
            }
        }
        var replacements map[string][]byte
        if err==nil && !strings.EqualFold(filepath.Ext(direct),".pf"){
            replacements,err=calculateGameReplacements(source,selected,direct,password,override)
        }
        if err==nil {err=core.ExportGameOutputs(workSource,dest,mask,replacements)}
        app.mu.Lock()
        app.busy=false
        if err!=nil{
            app.status="Game output failed: "+err.Error()
            app.lastErr=err.Error()
        }else{
            app.status="Game output verified: "+dest
            app.progress=1
        }
        app.mu.Unlock()
        if err!=nil{recordFailure("game.export",err,source,direct,dest)
        } else {rememberRecent("output",dest)}
        redraw()
    }()
}
