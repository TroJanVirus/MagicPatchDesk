package core

import (
    "context"
    "crypto/sha256"
    "errors"
    "fmt"
    "os"
    "path/filepath"
    "regexp"
    "sort"
    "strings"
)

// LibraryGame is one detected game, not an assumption about a patchable title.
// A game is safe for automatic processing only with exactly one verified JMP,
// an exact source CRC and Title ID, and no unsupported named-file behaviour.
type LibraryGame struct {
    Path string
    Kind string
    Title string
    TitleID string
    CRC32 string
    Status string // ready, review, unmatched, error
    Reason string
    Patches []LibraryPatch
}
type LibraryPatch struct {
    File string
    Name string
    Author string
    SHA256 string
    Status string // exact, review, incompatible
    Reason string
}
type LibraryScan struct {
    Root string
    Games []LibraryGame
    Checked int
    Warnings []string
}
var librarySplitSuffix=regexp.MustCompile(`(?i)\.(\d+)\.cci$`)
var libraryCRC=regexp.MustCompile(`(?i)\b[0-9a-f]{8}\b`)

func libraryCandidate(path string)bool{
    ext:=strings.ToLower(filepath.Ext(path))
    if ext==".iso" || ext==".xiso"{return true}
    if ext!=".cci"{return false}
    match:=librarySplitSuffix.FindStringSubmatch(path)
    return len(match)==0 || match[1]=="1"
}
func loadLibraryIdentity(game *LibraryGame)error {
    if game.Kind=="cci" {
        return errors.New("compressed CCI metadata needs extraction; manually review this game")
    }
    var content []byte
    if game.Kind=="folder" {
        target,e:=resolveGameMember(game.Path,"default.xbe")
        if e!=nil{return e}
        stat,e:=os.Lstat(target)
        if e!=nil{return e}
        if !stat.Mode().IsRegular()||stat.Size()>64<<20{return errors.New("game XBE exceeds safe 64MiB scan limit")}
        data,e:=ReadGameMember(game.Path,"default.xbe")
        if e!=nil{return e}
        content=data
    }else {
        im,e:=OpenXISO(game.Path)
        if e!=nil{return e}
        if im.Base!=0 {game.Kind="fulliso"}else{game.Kind="xiso"}
        entry,e:=im.DefaultXBE()
        if e!=nil{return e}
        if entry.Size>64<<20 {return errors.New("game XBE exceeds 64MiB identity scan limit")}
        content,e=im.ReadFile(entry)
        if e!=nil{return e}
    }
    if len(content)>64<<20{return errors.New("game XBE exceeds 64MiB identity scan limit")}
    info,e:=InspectXBE(content)
    if e!=nil{return fmt.Errorf("cannot identify game XBE: %w",e)}
    game.Title=info.Title
    game.TitleID=info.TitleID
    game.CRC32=info.CRC32
    return nil
}
func matchingLibraryPatches(game *LibraryGame,catalog Catalog) {
    exact:=0
    for _,entry:=range catalog.Games {
        for _,patch:=range entry.Patches {
            if patch.Issue!=""{continue}
            // Unidentified generic patches cannot be safely associated with
            // games using only a similar filename or title.
            if game.TitleID=="" || patch.TitleID=="" || !strings.EqualFold(game.TitleID,patch.TitleID){continue}
            item:=LibraryPatch{File:patch.File,Name:patch.Name,Author:patch.Author,Status:"review"}
            crcSet:=libraryCRC.FindAllString(strings.ToUpper(patch.CRC),-1)
            if len(crcSet)==0 {
                item.Reason="Title ID matches; exact original CRC is not specified"
            }else {
                found:=false
                for _,crc:=range crcSet{if strings.EqualFold(crc,game.CRC32){found=true;break}}
                if found && strings.EqualFold(filepath.Ext(patch.File),".jmp") {
                    // Named-file target and record complexity can only be
                    // approved after reading the actual patch.
                    b,e:=os.ReadFile(patch.File)
                    if e==nil{
                        sum:=sha256.Sum256(b)
                        item.SHA256=fmt.Sprintf("%x",sum)
                        p,e:=ParseJMP(filepath.Base(patch.File),b)
                        if e==nil && p.CheckSingleFileOnly()==nil {
                            item.Status="exact"
                            item.Reason="Matching Title ID and exact source XBE CRC32; single-file JMP"
                            exact++
                        }else{item.Reason="Patch targets multiple files or requires manual inspection"}
                    }else{item.Reason="Patch file could not be read"}
                }else if !found {
                    item.Status="incompatible"
                    item.Reason="Title ID matches but original XBE CRC32 differs"
                }else{
                    item.Reason="Source identity matches but patch format requires manual review"
                }
            }
            game.Patches=append(game.Patches,item)
        }
    }
    sort.Slice(game.Patches,func(i,j int)bool{
        a,b:=game.Patches[i],game.Patches[j]
        order:=map[string]int{"exact":0,"review":1,"incompatible":2}
        if order[a.Status]!=order[b.Status]{return order[a.Status]<order[b.Status]}
        return strings.ToLower(a.Name)<strings.ToLower(b.Name)
    })
    switch {
    case exact==1:
        // Even if additional review/incompatible patches exist, only the exact
        // CRC-backed JMP is eligible for unattended mode.
        game.Status="ready";game.Reason="One exact verified JMP candidate"
    case exact>1:
        game.Status="review";game.Reason="Multiple exact patches: user must choose and review byte overlaps"
    case len(game.Patches)>0:
        game.Status="review";game.Reason="Possible patches need inspection or have mismatching source CRCs"
    default:
        game.Status="unmatched";game.Reason="No patches with a matching Title ID in the local catalogue"
    }
}

// ScanGameLibrary only READS games. In particular it never extracts a CCI,
// patches any file or traverses symlinks. It returns review-required entries
// instead of guessing when identity cannot be verified.
func ScanGameLibrary(ctx context.Context,root string,catalog Catalog,log *VerificationLog)(LibraryScan,error){
    result:=LibraryScan{Root:root}
    info,err:=os.Lstat(root)
    if err!=nil{return result,err}
    if !info.IsDir(){return result,errors.New("library scan root must be a real directory")}
    const maxWalk=50000
    walked:=0
    seen:=map[string]bool{}
    err=filepath.WalkDir(root,func(path string,d os.DirEntry,walkErr error) error {
        if walkErr!=nil{return walkErr}
        if ctx!=nil&&ctx.Err()!=nil{return ctx.Err()}
        walked++
        if walked>maxWalk{return errors.New("library scan exceeds 50,000 entries; select a smaller library folder")}
        if d.Type()&os.ModeSymlink!=0{
            result.Warnings=append(result.Warnings,"Ignored symlink: "+filepath.Base(path))
            return nil
        }
        var g LibraryGame
        if d.IsDir(){
            if path==root{return nil}
            entries,e:=os.ReadDir(path)
            if e!=nil{return e}
            for _,item:=range entries{
                if strings.EqualFold(item.Name(),"default.xbe") && !item.IsDir(){
                    if item.Type()&os.ModeSymlink==0{g=LibraryGame{Path:path,Kind:"folder"}}
                    break
                }
            }
            if g.Path==""{return nil}
        }else{
            if !libraryCandidate(path){return nil}
            switch strings.ToLower(filepath.Ext(path)){case ".cci":g.Kind="cci";default:g.Kind="xiso"}
            g.Path=path
        }
        canonical,_:=filepath.Abs(g.Path)
        if seen[strings.ToLower(canonical)]{return nil}
        seen[strings.ToLower(canonical)]=true
        result.Checked++
        if e:=loadLibraryIdentity(&g);e!=nil{
            g.Title=strings.TrimSuffix(filepath.Base(g.Path),filepath.Ext(g.Path))
            g.Status="review"
            g.Reason=e.Error()
            log.Add("warning","Scan",g.Title+": "+g.Reason)
        }else{
            matchingLibraryPatches(&g,catalog)
            log.Add("pass","Identify",g.Title+" ["+g.TitleID+"] "+g.Status)
        }
        result.Games=append(result.Games,g)
        if d.IsDir(){return filepath.SkipDir}
        return nil
    })
    if err!=nil{return result,err}
    sort.Slice(result.Games,func(i,j int)bool{return strings.ToLower(result.Games[i].Path)<strings.ToLower(result.Games[j].Path)})
    return result,nil
}
