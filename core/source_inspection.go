package core

import (
    "context"
    "errors"
    "os"
    "path/filepath"
    "strings"
)

// GameSourceInspection is the read-only result of examining a source image.
// It does not contain a file handle or mutable parser state, so it is safe to
// pass across the native Win32 worker/UI message boundary.
type GameSourceInspection struct {
    Info XBEInfo
    PartSummary string
    Kind string
}

// InspectGameSourceContext is the single inspection entry point for XBE,
// XISO, full-disc ISO and numbered/single CCI sources. It never alters an
// original image. Cancellation is checked while walking XDVDFS, indexing
// CCIs and reading executable bytes.
func InspectGameSourceContext(ctx context.Context,path string)(GameSourceInspection,error){
    result:=GameSourceInspection{}
    if err:=ctx.Err();err!=nil{return result,err}
    if CCISourcePath(path){
        result.Kind="cci"
        summary,err:=CCIPartSummary(path)
        if err!=nil{return result,err}
        result.PartSummary=summary
        info,err:=InspectCCIExecutableContext(ctx,path)
        if err!=nil{return result,err}
        result.Info=info
        return result,nil
    }
    if IsNumberedRawImagePart(path){
        summary,err:=RawImagePartSummary(path)
        if err!=nil{return result,err}
        result.PartSummary=summary
    }
    if strings.EqualFold(filepath.Ext(path),".xbe"){
        result.Kind="xbe"
        file,err:=os.Open(path)
        if err!=nil{return result,err}
        defer file.Close()
        stat,err:=file.Stat()
        if err!=nil{return result,err}
        if !stat.Mode().IsRegular(){return result,errors.New("selected XBE is not a regular file")}
        if stat.Size()>512<<20{return result,errors.New("selected XBE exceeds 512 MiB limit")}
        if stat.Size()<0x200{return result,errors.New("selected XBE is too small to contain a valid header")}
        data,err:=readAtContext(ctx,file,0,int(stat.Size()))
        if err!=nil{return result,err}
        if err=ctx.Err();err!=nil{return result,err}
        result.Info,err=InspectXBE(data)
        return result,err
    }
    image,err:=OpenXISOContext(ctx,path)
    if err!=nil{return result,err}
    result.Kind="xiso"
    if image.Base!=0&&!IsNumberedRawImagePart(path){result.Kind="fulliso"}
    entry,err:=image.DefaultXBE()
    if err!=nil{return result,err}
    data,err:=image.ReadFileContext(ctx,entry)
    if err!=nil{return result,err}
    if err=ctx.Err();err!=nil{return result,err}
    result.Info,err=InspectXBE(data)
    return result,err
}
