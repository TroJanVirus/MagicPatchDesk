package core

import (
    "crypto/sha256"
    "errors"
    "fmt"
    "io"
    "os"
    "path/filepath"
    "sort"
    "strings"
)
type ImageBrowserEntry struct{
    Path string
    IsDirectory bool
    Size uint32
}
// BrowseGameImage reads directory records only. It never edits or extracts.
func BrowseGameImage(image string)([]ImageBrowserEntry,error){
    im,e:=OpenXISO(image)
    if e!=nil{return nil,e}
    entries:=make([]ImageBrowserEntry,0,len(im.Directories)+len(im.Files))
    for _,d:=range im.Directories{if _,e=validGameMember(d);e!=nil{return nil,e};entries=append(entries,ImageBrowserEntry{Path:d,IsDirectory:true})}
    for _,f:=range im.Files{if _,e=validGameMember(f.Path);e!=nil{return nil,e};entries=append(entries,ImageBrowserEntry{Path:f.Path,Size:f.Size})}
    sort.Slice(entries,func(i,j int)bool{
        if entries[i].IsDirectory!=entries[j].IsDirectory{return entries[i].IsDirectory}
        return strings.ToLower(entries[i].Path)<strings.ToLower(entries[j].Path)
    })
    return entries,nil
}
// ExtractImageMember extracts one explicitly selected file, never overwrites,
// and rehashes its output to detect failed reads/writes.
func ExtractImageMember(image,member,destination string)error{
    if _,e:=validGameMember(member);e!=nil{return e}
    if _,e:=os.Lstat(destination);e==nil{return errors.New("extraction output already exists")}
    im,e:=OpenXISO(image)
    if e!=nil{return e}
    match,e:=im.ResolveTarget(member)
    if e!=nil{return e}
    if !strings.EqualFold(match.Path,member){return fmt.Errorf("image member resolved to a different path: %s",match.Path)}
    in,e:=os.Open(image)
    if e!=nil{return e}
    defer in.Close()
    out,e:=os.OpenFile(destination,os.O_CREATE|os.O_EXCL|os.O_WRONLY,0644)
    if e!=nil{return e}
    good:=false
    defer func(){if !good {out.Close();os.Remove(destination)}}()
    a:=sha256.New()
    reader:=io.NewSectionReader(in,im.Base+int64(match.StartSector)*sectorSize,int64(match.Size))
    bytesCopied,e:=io.Copy(io.MultiWriter(out,a),reader)
    if e!=nil{return e}
    if bytesCopied!=int64(match.Size){return errors.New("source truncated during member extraction")}
    if e=out.Sync();e!=nil{return e}
    if e=out.Close();e!=nil{return e}
    target,e:=os.Open(destination)
    if e!=nil{return e}
    b:=sha256.New()
    verified,e:=io.Copy(b,target)
    closeErr:=target.Close()
    if e!=nil{return e}
    if closeErr!=nil{return closeErr}
    if verified!=int64(match.Size) || sha256.Sum256(a.Sum(nil))!=sha256.Sum256(b.Sum(nil)){
        return errors.New("image extraction hash or size mismatch")
    }
    good=true
    return nil
}
// Prevent accidental confusion between an extracted member and an entire game.
func SuggestedMemberName(entry ImageBrowserEntry)string{
    if entry.IsDirectory{return ""}
    return filepath.Base(filepath.FromSlash(entry.Path))
}
