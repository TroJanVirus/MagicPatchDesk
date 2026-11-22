package core

import (
    "context"
    "errors"
    "fmt"
    "io"
    "path/filepath"
    "strings"
)

// ReadAt presents the full contiguous CCI split set as a read-only logical
// stream. It never writes or concatenates any source part on disk. Caller
// closes the CCIReader; it is safe to read bytes spanning part boundaries.
func (r *CCIReader) ReadAt(dst []byte, offset int64)(int,error){
    return r.ReadAtContext(context.Background(),dst,offset)
}
func (r *CCIReader) ReadAtContext(ctx context.Context,dst []byte,offset int64)(int,error){
    if err:=ctx.Err();err!=nil{return 0,err}
    if offset<0{return 0,errors.New("negative CCI read offset")}
    if len(dst)==0{return 0,nil}
    if r==nil||len(r.Parts)==0{return 0,errors.New("CCI reader is closed")}
    total:=r.Sectors*cciSectorSize
    if offset>=total{return 0,io.EOF}
    copied:=0
    sector:=make([]byte,cciSectorSize)
    for copied<len(dst)&&offset<total {
        if err:=ctx.Err();err!=nil{return copied,err}
        n:=int64(len(dst)-copied)
        within:=offset%cciSectorSize
        if n>cciSectorSize-within{n=cciSectorSize-within}
        if err:=r.ReadSector(offset/cciSectorSize,sector);err!=nil{return copied,err}
        copied+=copy(dst[copied:],sector[int(within):int(within+n)])
        offset+=n
    }
    if copied<len(dst){return copied,io.EOF}
    return copied,nil
}

// ReadCCIGameMember opens and validates the complete numbered split set from
// ANY chosen part (including .2.cci etc.), parses its XDVDFS volume, and
// extracts one named member using its actual logical sector location. Supplying
// an empty member selects default.xbe (or the only XBE if no default exists).
// It never decodes the entire game to a temporary ISO.
func ReadCCIGameMember(path,member string)([]byte,error){
    return ReadCCIGameMemberContext(context.Background(),path,member)
}
func ReadCCIGameMemberContext(ctx context.Context,path,member string)([]byte,error){
    items,err:=ReadCCIGameMembersContext(ctx,path,[]string{member})
    if err!=nil{return nil,err}
    return items[member],nil
}

// ReadCCIGameMembers opens the split headers and XDVDFS directory once,
// keeping all requested targets inside a single verified logical read.
func ReadCCIGameMembers(path string,members []string)(map[string][]byte,error){
    return ReadCCIGameMembersContext(context.Background(),path,members)
}

type cciContextReaderAt struct {
    ctx context.Context
    reader *CCIReader
}
func (c cciContextReaderAt) ReadAt(b []byte,off int64)(int,error){
    return c.reader.ReadAtContext(c.ctx,b,off)
}
func ReadCCIGameMembersContext(ctx context.Context,path string,members []string)(map[string][]byte,error){
    if err:=ctx.Err();err!=nil{return nil,err}
    r,err:=OpenCCIContext(ctx,path)
    if err!=nil{return nil,err}
    defer r.Close()
    source:=cciContextReaderAt{ctx:ctx,reader:r}
    image,err:=openXDVDFSReaderAtContext(ctx,source,r.Sectors*cciSectorSize,path)
    if err!=nil{return nil,fmt.Errorf("CCI does not contain a readable Xbox XDVDFS image: %w",err)}
    result:=make(map[string][]byte,len(members))
    for _,member:=range members{
        if err:=ctx.Err();err!=nil{return nil,err}
        if _,already:=result[member];already{continue}
        var target ImageFile
        if member==""{
            target,err=image.DefaultXBE()
        }else{
            target,err=image.ResolveTarget(member)
        }
        if err!=nil{return nil,err}
        if target.Size>512<<20{return nil,errors.New("CCI member exceeds 512 MiB safety limit")}
        result[member],err=readAtContext(ctx,source,image.Base+int64(target.StartSector)*sectorSize,int(target.Size))
        if err!=nil{return nil,fmt.Errorf("CCI member %s: %w",member,err)}
    }
    return result,nil
}

// CCIPartSummary describes which member of a numbered set the user chose.
// This is filesystem-only discovery; InspectCCIExecutable independently
// validates CCIM headers and the actual logical XDVDFS data.
func CCIPartSummary(path string)(string,error){
    parts,err:=discoverCCIParts(path)
    if err!=nil{return "",err}
    if len(parts)==1{return "single CCI image",nil}
    selected:=0
    for i,part:=range parts{
        if strings.EqualFold(filepath.Clean(part),filepath.Clean(path)){
            selected=i+1
            break
        }
    }
    if selected==0 {return "",errors.New("selected CCI part is not in discovered set")}
    return fmt.Sprintf("CCI part %d of %d",selected,len(parts)),nil
}

// InspectCCIExecutable returns the same Title ID, game name and CRC32 as
// InspectXBE would for an extracted default.xbe. Only the required XDVDFS
// directory and XBE sectors are read, even for multi-gigabyte CCI sets.
func InspectCCIExecutable(path string)(XBEInfo,error){
    return InspectCCIExecutableContext(context.Background(),path)
}
func InspectCCIExecutableContext(ctx context.Context,path string)(XBEInfo,error){
    data,err:=ReadCCIGameMemberContext(ctx,path,"")
    if err!=nil{return XBEInfo{},err}
    if err:=ctx.Err();err!=nil{return XBEInfo{},err}
    return InspectXBE(data)
}
