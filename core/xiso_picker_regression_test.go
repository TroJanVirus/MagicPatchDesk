package core

import (
    "bytes"
    "encoding/binary"
    "errors"
    "io"
    "os"
    "path/filepath"
    "strings"
    "testing"
)

// Build a real game-partition XDVDFS rather than the older synthetic
// multi-file test fixture, which deliberately overlaps the cfg sector with
// a large default.xbe and is unsuitable for byte-for-byte identity tests.
func isoPickerFixture(t *testing.T,full bool)string{
    t.Helper()
    folder:=t.TempDir()
    if err:=os.WriteFile(filepath.Join(folder,"default.xbe"),fakeXBE(),0600);err!=nil{t.Fatal(err)}
    if err:=os.WriteFile(filepath.Join(folder,"extra.bin"),[]byte("not part of executable"),0600);err!=nil{t.Fatal(err)}
    game:=filepath.Join(t.TempDir(),"game.iso")
    if err:=BuildGameXISO(folder,game);err!=nil{t.Fatal(err)}
    if !full{return game}
    original,err:=os.Open(game)
    if err!=nil{t.Fatal(err)}
    defer original.Close()
    fullPath:=filepath.Join(t.TempDir(),"full.iso")
    out,err:=os.Create(fullPath)
    if err!=nil{t.Fatal(err)}
    if _,err=out.Seek(0x02080000,io.SeekStart);err!=nil{out.Close();t.Fatal(err)}
    if _,err=io.Copy(out,original);err!=nil{out.Close();t.Fatal(err)}
    if err=out.Close();err!=nil{t.Fatal(err)}
    return fullPath
}

// The exact workflow is: choose a library patch, then choose an ISO. The
// parser must identify its real default.xbe without altering the game image.
func TestXISOIdentificationForPatchSelectedBeforeISO(t *testing.T){
    for _,full:=range []bool{false,true}{
        name:="game-partition"
        if full{name="full-disc"}
        t.Run(name,func(t *testing.T){
            path:=isoPickerFixture(t,full)
            before,err:=os.ReadFile(path)
            if err!=nil{t.Fatal(err)}
            image,err:=OpenXISO(path)
            if err!=nil{t.Fatal(err)}
            if (image.Base!=0)!=full{t.Fatalf("wrong disc layout classification: %x",image.Base)}
            entry,err:=image.DefaultXBE()
            if err!=nil{t.Fatal(err)}
            data,err:=image.ReadFile(entry)
            if err!=nil{t.Fatal(err)}
            want:=fakeXBE()
            if !bytes.Equal(data,want){t.Fatal("ISO produced different default.xbe bytes")}
            id,err:=InspectXBE(data)
            if err!=nil||!id.Valid||id.TitleID==""||id.CRC32==""{
                t.Fatalf("embedded XBE identification failed: %+v %v",id,err)
            }
            after,err:=os.ReadFile(path)
            if err!=nil{t.Fatal(err)}
            if !bytes.Equal(before,after){t.Fatal("opening ISO changed original source bytes")}
        })
    }
}

// A skewed real-world or malformed XDVDFS search tree must not consume
// unbounded call stack space. Entries are intentionally in one long chain.
func TestXDVDFSLongSkewedDirectoryTreeDoesNotRecurse(t *testing.T){
    const entries=7500
    const tableStart=34*2048
    const fileSector=380
    fixture:=make([]byte,850000)
    copy(fixture[0x10000:],volumeMagic)
    copy(fixture[0x107EC:],volumeMagic)
    binary.LittleEndian.PutUint32(fixture[0x10014:],34)
    binary.LittleEndian.PutUint32(fixture[0x10018:],entries*32)
    for i:=0;i<entries;i++{
        off:=tableStart+i*32
        if i+1<entries {
            binary.LittleEndian.PutUint16(fixture[off+2:],uint16((i+1)*8))
        }
        binary.LittleEndian.PutUint32(fixture[off+4:],fileSector)
        binary.LittleEndian.PutUint32(fixture[off+8:],1)
        fixture[off+12]=0x20
        fixture[off+13]=1
        fixture[off+14]='a'
    }
    last:=tableStart+(entries-1)*32
    xbe:=fakeXBE()
    binary.LittleEndian.PutUint32(fixture[last+8:],uint32(len(xbe)))
    fixture[last+13]=11
    copy(fixture[last+14:],"default.xbe")
    copy(fixture[fileSector*2048:],xbe)
    path:=filepath.Join(t.TempDir(),"deep.iso")
    if err:=os.WriteFile(path,fixture,0600);err!=nil{t.Fatal(err)}
    im,err:=OpenXISO(path)
    if err!=nil{t.Fatal(err)}
    if len(im.Files)!=entries{t.Fatalf("missing tree nodes: %d",len(im.Files))}
    entry,err:=im.DefaultXBE()
    if err!=nil{t.Fatal(err)}
    got,err:=im.ReadFile(entry)
    if err!=nil||!bytes.Equal(got,xbe){t.Fatalf("deeply nested XISO yielded bad XBE: %v",err)}
}

func TestXISORejectsTruncationAndInvalidDirectoryBounds(t *testing.T){
    path:=isoPickerFixture(t,false)
    im,err:=OpenXISO(path)
    if err!=nil{t.Fatal(err)}
    entry,err:=im.DefaultXBE()
    if err!=nil{t.Fatal(err)}
    // The source can be replaced/truncated between directory scanning and
    // member reading. Reject the later invalid extent, not partial bytes.
    end:=im.Base+int64(entry.StartSector)*sectorSize+int64(entry.Size)-1
    if err:=os.Truncate(path,end);err!=nil{t.Fatal(err)}
    if _,err=im.ReadFile(entry);err==nil||
        !strings.Contains(err.Error(),"outside the current disc image"){
        t.Fatalf("truncated XBE member not rejected: %v",err)
    }

    malformed:=isoPickerFixture(t,false)
    f,err:=os.OpenFile(malformed,os.O_RDWR,0)
    if err!=nil{t.Fatal(err)}
    invalid:=make([]byte,4)
    binary.LittleEndian.PutUint32(invalid,65<<20)
    _,err=f.WriteAt(invalid,0x10018)
    f.Close()
    if err!=nil{t.Fatal(err)}
    if _,err=OpenXISO(malformed);err==nil{t.Fatal("unsafe root table size accepted")}
}

type shortButNilReader struct{}
func (shortButNilReader) ReadAt(b []byte,offset int64)(int,error){
    if len(b)==0{return 0,nil}
    return len(b)-1,nil
}
func TestImageReadAtRejectsShortSuccess(t *testing.T){
    _,err:=readAt(shortButNilReader{},0,2048)
    if !errors.Is(err,io.ErrUnexpectedEOF){
        t.Fatalf("short read with nil error was accepted: %v",err)
    }
}
func TestXISORejectsNonRegularFile(t *testing.T){
    if _,err:=OpenXISO(t.TempDir());err==nil{
        t.Fatal("directory was accepted as a disc image")
    }
}

func TestXISOHeadersAndDirectoryCorruptionReturnsErrorNotPanic(t *testing.T){
    src:=isoPickerFixture(t,false)
    original,err:=os.ReadFile(src)
    if err!=nil{t.Fatal(err)}
    // Mutate volume metadata, directory offsets and filename entries. Some
    // mutations may remain valid; none may panic, overflow or require reading
    // an entire untrusted image into RAM.
    offsets:=[]int{0x10000,0x10014,0x10018,0x107EC,34*2048,34*2048+2,
        34*2048+4,34*2048+8,34*2048+12,34*2048+13,34*2048+64}
    path:=filepath.Join(t.TempDir(),"mutated.iso")
    for caseNumber:=0;caseNumber<100;caseNumber++{
        data:=append([]byte(nil),original...)
        offset:=offsets[caseNumber%len(offsets)]
        if offset>=len(data){t.Fatal("bad test offset")}
        data[offset]^=byte(1+caseNumber*7)
        if err:=os.WriteFile(path,data,0600);err!=nil{t.Fatal(err)}
        func(){
            defer func(){
                if v:=recover();v!=nil{
                    t.Fatalf("malformed ISO triggered Go panic in case %d: %v",caseNumber,v)
                }
            }()
            im,err:=OpenXISO(path)
            if err==nil{
                if item,e:=im.DefaultXBE();e==nil{
                    _,_=im.ReadFile(item)
                }
            }
        }()
    }
}
