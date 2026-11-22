package core

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
)

const sectorSize int64 = 2048

var partitionBases = []int64{0, 0x0FD90000, 0x02080000, 0x18300000}
var volumeMagic = []byte("MICROSOFT*XBOX*MEDIA")

type Image struct {
	Path       string
	Base       int64
	RootSector uint32
	RootSize   uint32
	Files      []ImageFile
	Directories []string // preserve empty directories during extraction/repacking
}
type ImageFile struct {
	Path        string
	StartSector uint32
	Size        uint32
	EntryAt     int64
}

func get32(b []byte) uint32 { return binary.LittleEndian.Uint32(b) }
func readAt(f io.ReaderAt, where int64, n int) ([]byte, error) {
    if where<0||n<0{return nil,errors.New("negative image read range")}
    b:=make([]byte,n)
    count,err:=f.ReadAt(b,where)
    if err!=nil{return nil,err}
    if count!=n{return nil,io.ErrUnexpectedEOF}
    return b,nil
}
func OpenXISO(path string) (*Image,error) {
    if IsNumberedRawImagePart(path) {
        parts,err:=openRawDiscParts(path)
        if err!=nil{return nil,err}
        defer parts.Close()
        return openXDVDFSReaderAt(parts,parts.size,path)
    }
    f,err:=os.Open(path)
    if err!=nil{return nil,err}
    defer f.Close()
    st,err:=f.Stat()
    if err!=nil{return nil,err}
    if !st.Mode().IsRegular(){return nil,errors.New("disc image is not a regular file")}
    return openXDVDFSReaderAt(f,st.Size(),path)
}

// Use the same bounded XDVDFS parser for ordinary ISOs and for a logical
// multi-part CCI sector stream. This avoids extracting multi-gigabyte CCI
// images just to locate a few executable sectors.
const maxXDVDFSEntries = 200000
const maxXDVDFSDirectoryBytes int64 = 128 << 20

type imageScanLimits struct {
    entries int
    directoryBytes int64
}

func openXDVDFSReaderAt(f io.ReaderAt, imageSize int64, path string) (*Image,error) {
    if imageSize<0{return nil,errors.New("negative Xbox image size")}
	for _, base := range partitionBases {
        if base+0x10800>imageSize {
			continue
		}
		v, err := readAt(f, base+0x10000, int(sectorSize))
		if err != nil || !bytes.Equal(v[:20], volumeMagic) || !bytes.Equal(v[0x7EC:0x800], volumeMagic) {
			continue
		}
		root := get32(v[0x14:])
		rs := get32(v[0x18:])
		if root == 0 || rs == 0 || rs > 64<<20 {
			return nil, fmt.Errorf("invalid XISO directory table")
		}
		im := &Image{Path: path, Base: base, RootSector: root, RootSize: rs}
        visited:=map[string]bool{}
        limits:=&imageScanLimits{}
        if err:=im.walk(f,imageSize,"",root,rs,visited,0,limits);err!=nil{
			return nil, err
		}
		if len(im.Files) == 0 {
			return nil, errors.New("XISO has no files")
		}
		return im, nil
	}
	return nil, errors.New("no recognized original Xbox XDVDFS volume (XISO / XGD1 / XGD2 / XGD3)")
}
func (im *Image) walk(f io.ReaderAt, imageSize int64, prefix string, sector uint32, size uint32, visited map[string]bool, depth int, limits *imageScanLimits) error {
	if depth > 24 {
		return errors.New("XISO directory nesting too deep")
	}
	if size == 0 {
		return nil
	}
    if size>32<<20{return errors.New("directory table too large")}
    limits.directoryBytes+=int64(size)
    if limits.directoryBytes>maxXDVDFSDirectoryBytes{
        return errors.New("XDVDFS directory metadata exceeds safe 128 MiB limit")
    }
	dirAt := im.Base + int64(sector)*sectorSize
	key := fmt.Sprintf("%d:%d", dirAt, size)
	if visited[key] {
		return errors.New("directory cycle detected")
	}
	visited[key] = true
	if dirAt < 0 || dirAt+int64(size) > imageSize {
		return errors.New("directory table outside image")
	}
	table, err := readAt(f, dirAt, int(size))
	if err != nil {
		return err
	}
    // XDVDFS directory entries form a binary search tree. Iterative
    // traversal avoids unbounded Go stack growth on malformed skewed trees.
    nodes:=map[int]bool{}
    pending:=[]int{0}
    for len(pending)>0{
        off:=pending[len(pending)-1]
        pending=pending[:len(pending)-1]
        if nodes[off]{return errors.New("directory tree cycle")}
        nodes[off]=true
        limits.entries++
        if limits.entries>maxXDVDFSEntries{
            return errors.New("XDVDFS has too many directory entries")
        }
        if off<0||off+14>len(table){return errors.New("directory entry outside table")}
        b:=table[off:]
        if b[0]==0xFF&&b[1]==0xFF{continue}
        left:=int(binary.LittleEndian.Uint16(b[0:2]))*4
        right:=int(binary.LittleEndian.Uint16(b[2:4]))*4
        sector:=get32(b[4:])
        filesize:=get32(b[8:])
        attr:=b[12]
        length:=int(b[13])
        if length==0||off+14+length>len(table){
            return errors.New("invalid XISO filename length")
        }
        name:=string(b[14:14+length])
        if strings.ContainsAny(name,"/\\")||name==".."||name=="."{
            return errors.New("XISO contains invalid filename")
        }
        pth:=name
        if prefix!=""{pth=prefix+"/"+name}
        if attr&0x10!=0{
            im.Directories=append(im.Directories,pth)
            if sector!=0&&filesize!=0 {
                if err:=im.walk(f,imageSize,pth,sector,filesize,visited,depth+1,limits);err!=nil{
                    return err
                }
            }
        }else{
            at:=im.Base+int64(sector)*sectorSize
            if at<0||at+int64(filesize)>imageSize{
                return fmt.Errorf("XISO file outside image: %s",pth)
            }
            im.Files=append(im.Files,ImageFile{pth,sector,filesize,dirAt+int64(off)})
        }
        if right>0{pending=append(pending,right)}
        if left>0{pending=append(pending,left)}
    }
    return nil
}
func (im *Image) XBEs() []ImageFile {
	out := []ImageFile{}
	for _, f := range im.Files {
		if strings.HasSuffix(strings.ToLower(f.Path), ".xbe") {
			out = append(out, f)
		}
	}
	return out
}
func (im *Image) DefaultXBE() (ImageFile, error) {
	for _, f := range im.XBEs() {
		if strings.EqualFold(f.Path, "default.xbe") {
			return f, nil
		}
	}
	list := im.XBEs()
	if len(list) == 1 {
		return list[0], nil
	}
	if len(list) == 0 {
		return ImageFile{}, errors.New("no XBE found in image")
	}
	return ImageFile{}, fmt.Errorf("multiple XBEs found; default.xbe is absent (cannot auto-select safely)")
}
func (im *Image) ReadFile(entry ImageFile) ([]byte, error) {
    if entry.Size>512<<20{return nil,errors.New("XDVDFS member exceeds 512 MiB safety limit")}
    if IsNumberedRawImagePart(im.Path){
        parts,err:=openRawDiscParts(im.Path)
        if err!=nil{return nil,err}
        defer parts.Close()
        offset:=im.Base+int64(entry.StartSector)*sectorSize
        if offset<0 || offset+int64(entry.Size)>parts.size {
            return nil,errors.New("split ISO member extends outside the complete disc")
        }
        return readAt(parts,offset,int(entry.Size))
    }
    f,err:=os.Open(im.Path)
    if err!=nil{return nil,err}
    defer f.Close()
    st,err:=f.Stat()
    if err!=nil{return nil,err}
    offset:=im.Base+int64(entry.StartSector)*sectorSize
    if !st.Mode().IsRegular()||offset<0||offset+int64(entry.Size)>st.Size(){
        return nil,errors.New("XDVDFS member is outside the current disc image")
    }
    return readAt(f,offset,int(entry.Size))
}

// CopyImage performs bounded parallel ReadAt/WriteAt copies. The caller chooses worker count (1-8).
func CopyImage(source, dest string, workers int, onProgress func(done, total int64)) error {
    if IsNumberedRawImagePart(source){
        return errors.New("cannot copy one split ISO part as a complete image; use Patch Library to export a new XISO/CCI from the complete set")
    }
	if workers < 1 {
		workers = 1
	}
	if workers > 8 {
		workers = 8
	}
	src, err := os.Open(source)
	if err != nil {
		return err
	}
	defer src.Close()
	st, err := src.Stat()
	if err != nil {
		return err
	}
	total := st.Size()
	dst, err := os.OpenFile(dest, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer dst.Close()
	ok := false
	defer func() {
		if !ok {
			os.Remove(dest)
		}
	}()
	if err := dst.Truncate(total); err != nil {
		return err
	}
	const step int64 = 4 << 20
	tasks := make(chan int64, workers*2)
	var wg sync.WaitGroup
	var firstError error
	var failOnce sync.Once
	var done atomic.Int64
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			b := make([]byte, step)
			for off := range tasks {
				n := int(step)
				if rem := total - off; rem < step {
					n = int(rem)
				}
				read, e := src.ReadAt(b[:n], off)
				if e == nil && read != n {
					e = io.ErrUnexpectedEOF
				}
				if e == nil {
					written, writeErr := dst.WriteAt(b[:n], off)
					e = writeErr
					if e == nil && written != n {
						e = io.ErrShortWrite
					}
				}
				if e != nil {
					failOnce.Do(func() { firstError = e })
					continue
				}
				progress := done.Add(int64(n))
				if onProgress != nil {
					onProgress(progress, total)
				}
			}
		}()
	}
	for off := int64(0); off < total; off += step {
		tasks <- off
	}
	close(tasks)
	wg.Wait()
	if firstError != nil {
		return firstError
	}
	if err := dst.Sync(); err != nil {
		return err
	}
	ok = true
	return nil
}

// PatchImage copies to a new file and updates the XDVDFS directory record for the modified XBE.
// In-place growth is allowed only up to the source's allocated sectors. Relocation of an XBE
// beyond its allocated sectors is supported on XISO-only images, never full Redump disc images.
func PatchImage(input, output string, patch *Patch, overrideCRC bool, workers int, onProgress func(int64, int64)) error {
	return PatchImageWithTransform(input, output, func(src []byte) ([]byte, error) { return patch.Apply(src, overrideCRC) }, workers, onProgress)
}
func PatchImageWithTransform(input, output string, transform func([]byte) ([]byte, error), workers int, onProgress func(int64, int64)) error {
	if transform == nil {
		return errors.New("no patch supplied")
	}

	if filepath.Clean(input) == filepath.Clean(output) {
		return errors.New("output must differ from source")
	}
	im, err := OpenXISO(input)
	if err != nil {
		return err
	}
	entry, err := im.DefaultXBE()
	if err != nil {
		return err
	}
	src, err := im.ReadFile(entry)
	if err != nil {
		return err
	}
	if _, err = InspectXBE(src); err != nil {
		return fmt.Errorf("embedded XBE: %w", err)
	}
	newXBE, err := transform(src)
	if err != nil {
		return err
	}
	if len(newXBE) > int(^uint32(0)) {
		return errors.New("patched XBE too large for XDVDFS")
	}
	oldSectors := (int64(entry.Size) + 2047) / 2048
	newSectors := (int64(len(newXBE)) + 2047) / 2048
	if newSectors > oldSectors && im.Base != 0 {
		return errors.New("XBE growth beyond allocated sectors in a full-disc ISO is unsafe; repack to a game-partition XISO first")
	}
	if err := CopyImage(input, output, workers, onProgress); err != nil {
		return err
	}
	complete := false
	defer func() {
		if !complete {
			os.Remove(output)
		}
	}()
	f, err := os.OpenFile(output, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	defer f.Close()
	at := im.Base + int64(entry.StartSector)*sectorSize
	if newSectors > oldSectors {
		if im.Base != 0 {
			return errors.New("growth beyond allocated sectors in a full-disc image is unsafe; extract/repack as XISO first")
		}
		st, err := f.Stat()
		if err != nil {
			return err
		}
		at = (st.Size() + 2047) &^ int64(2047)
		relSector := (at - im.Base) / sectorSize
		if relSector > int64(^uint32(0)) {
			return errors.New("XISO exceeds 32-bit sector limit")
		}
		entry.StartSector = uint32(relSector)
	}
	if _, err := f.WriteAt(newXBE, at); err != nil {
		return err
	}
	if padding := newSectors*sectorSize - int64(len(newXBE)); padding > 0 {
		fill := bytes.Repeat([]byte{0xFF}, int(padding))
		if _, err = f.WriteAt(fill, at+int64(len(newXBE))); err != nil {
			return err
		}
	}
	entryBytes := make([]byte, 8)
	binary.LittleEndian.PutUint32(entryBytes, entry.StartSector)
	binary.LittleEndian.PutUint32(entryBytes[4:], uint32(len(newXBE)))
	if _, err := f.WriteAt(entryBytes, entry.EntryAt+4); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	complete = true
	return nil
}
