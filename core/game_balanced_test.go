package core

import (
    "encoding/binary"
    "fmt"
    "os"
    "path/filepath"
    "testing"
)

// A thousands-of-files directory should use XDVDFS binary search pointers,
// rather than the degenerate right-only chain from the first implementation.
func TestGameXISOUsesBalancedDirectoryTrees(t *testing.T) {
    root:=gameFixture(t)
    for i:=0;i<513;i++{
        name:=filepath.Join(root,"files",fmt.Sprintf("entry%04d.dat",i))
        if err:=os.MkdirAll(filepath.Dir(name),0755);err!=nil{t.Fatal(err)}
        if err:=os.WriteFile(name,[]byte{byte(i),byte(i>>8)},0644);err!=nil{t.Fatal(err)}
    }
    iso:=filepath.Join(t.TempDir(),"many.iso")
    if err:=BuildGameXISO(root,iso);err!=nil{t.Fatal(err)}
    if err:=VerifyGameXISO(iso,root);err!=nil{t.Fatal(err)}
    im,err:=OpenXISO(iso)
    if err!=nil{t.Fatal(err)}
    var sub ImageFile
    _=sub
    f,err:=os.Open(iso)
    if err!=nil{t.Fatal(err)}
    defer f.Close()
    for _,dir:=range im.Directories {
        if dir!="files"{continue}
        // Find the root-level "files" directory entry: XDVDFS child records
        // have a 4-byte-aligned search-tree offset.
        directory:=make([]byte,im.RootSize)
        if _,err:=f.ReadAt(directory,im.Base+int64(im.RootSector)*sectorSize);err!=nil{t.Fatal(err)}
        var childSector uint32
        var childSize uint32
        var scan func(int) bool
        scan=func(pos int)bool{
            if pos<0||pos+14>len(directory){return false}
            b:=directory[pos:]
            namelen:=int(b[13])
            if pos+14+namelen>len(directory){return false}
            if string(b[14:14+namelen])=="files"{
                childSector=binary.LittleEndian.Uint32(b[4:])
                childSize=binary.LittleEndian.Uint32(b[8:])
                return true
            }
            l:=int(binary.LittleEndian.Uint16(b[:2]))*4
            r:=int(binary.LittleEndian.Uint16(b[2:4]))*4
            if l>0&&scan(l){return true}
            if r>0&&scan(r){return true}
            return false
        }
        if !scan(0){t.Fatal("missing files directory entry")}
        buf:=make([]byte,childSize)
        if _,err:=f.ReadAt(buf,im.Base+int64(childSector)*sectorSize);err!=nil{t.Fatal(err)}
        left:=binary.LittleEndian.Uint16(buf[:2])
        right:=binary.LittleEndian.Uint16(buf[2:4])
        if left==0||right==0{t.Fatalf("513-file directory is not balanced: left=%d right=%d",left,right)}
        return
    }
    t.Fatal("missing files directory")
}

func TestGameXISOSectorAndEntryPadding(t *testing.T){
    folder:=gameFixture(t)
    image:=filepath.Join(t.TempDir(),"padding.iso")
    if err:=BuildGameXISO(folder,image);err!=nil{t.Fatal(err)}
    stat,err:=os.Stat(image)
    if err!=nil{t.Fatal(err)}
    if stat.Size()%0x10000!=0{
        t.Fatalf("XDVDFS image size %d not 64KiB-aligned",stat.Size())
    }
    im,err:=OpenXISO(image)
    if err!=nil{t.Fatal(err)}
    f,err:=os.Open(image)
    if err!=nil{t.Fatal(err)}
    defer f.Close()
    table:=make([]byte,im.RootSize)
    if _,err:=f.ReadAt(table,im.Base+int64(im.RootSector)*sectorSize);err!=nil{t.Fatal(err)}
    checked:=0
    visited:=map[int]bool{}
    var check func(int)
    check=func(offset int){
        if visited[offset]{
            t.Fatalf("duplicate XDVDFS directory pointer %d",offset)
        }
        visited[offset]=true
        if offset+14>len(table){t.Fatalf("entry outside table at %d",offset)}
        b:=table[offset:]
        n:=14+int(b[13])
        aligned:=(n+3)&^3
        if offset+aligned>len(table){t.Fatalf("invalid entry length at %d",offset)}
        for i:=n;i<aligned;i++{
            if b[i]!=0xFF{
                t.Fatalf("entry padding at offset %d byte %d is %02X, expected FF",offset,i,b[i])
            }
            checked++
        }
        left:=int(binary.LittleEndian.Uint16(b[:2]))*4
        right:=int(binary.LittleEndian.Uint16(b[2:4]))*4
        if left!=0{check(left)}
        if right!=0{check(right)}
    }
    check(0)
    if checked==0{t.Fatal("fixture did not check any directory padding bytes")}
}
