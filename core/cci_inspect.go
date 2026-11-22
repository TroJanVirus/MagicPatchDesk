package core

import (
    "errors"
    "fmt"
    "io"
)

// ReadAt presents the full contiguous CCI split set as a read-only logical
// stream. It never writes or concatenates any source part on disk. Caller
// closes the CCIReader; it is safe to read bytes spanning part boundaries.
func (r *CCIReader) ReadAt(dst []byte, offset int64)(int,error){
    if offset<0{return 0,errors.New("negative CCI read offset")}
    if len(dst)==0{return 0,nil}
    if r==nil||len(r.Parts)==0{return 0,errors.New("CCI reader is closed")}
    total:=r.Sectors*cciSectorSize
    if offset>=total{return 0,io.EOF}
    copied:=0
    sector:=make([]byte,cciSectorSize)
    for copied<len(dst)&&offset<total {
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
    r,err:=OpenCCI(path)
    if err!=nil{return nil,err}
    defer r.Close()
    image,err:=openXDVDFSReaderAt(r,r.Sectors*cciSectorSize,path)
    if err!=nil{return nil,fmt.Errorf("CCI does not contain a readable Xbox XDVDFS image: %w",err)}
    var target ImageFile
    if member==""{
        target,err=image.DefaultXBE()
    }else{
        target,err=image.ResolveTarget(member)
    }
    if err!=nil{return nil,err}
    if target.Size>512<<20{return nil,errors.New("CCI member exceeds 512 MiB safety limit")}
    return readAt(r,image.Base+int64(target.StartSector)*sectorSize,int(target.Size))
}

// InspectCCIExecutable returns the same Title ID, game name and CRC32 as
// InspectXBE would for an extracted default.xbe. Only the required XDVDFS
// directory and XBE sectors are read, even for multi-gigabyte CCI sets.
func InspectCCIExecutable(path string)(XBEInfo,error){
    data,err:=ReadCCIGameMember(path,"")
    if err!=nil{return XBEInfo{},err}
    return InspectXBE(data)
}
