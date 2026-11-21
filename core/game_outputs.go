package core

// Native game image conversion. Reference acknowledgements and source/code
// provenance are documented in CREDITS.md. This implementation does not
// incorporate Repackinator or other GPL-licensed project source code.
import (
    "crypto/sha256"
    "encoding/binary"
    "errors"
    "fmt"
    "io"
    "os"
    "path/filepath"
    "sort"
    "strings"
)

const (
    GameOutputXBE uint32 = 1 << iota
    GameOutputFolder
    GameOutputXISO
    GameOutputCCI
    GameOutputISO
)
const GameOutputAll = GameOutputXBE | GameOutputFolder | GameOutputXISO | GameOutputCCI | GameOutputISO

func validGameMember(p string) (string, error) {
    p = strings.ReplaceAll(p, "\\", "/")
    if p == "" || strings.HasPrefix(p, "/") || strings.Contains(p, ":") {
        return "", errors.New("unsafe game member path")
    }
    parts := strings.Split(p, "/")
    for _, part := range parts {
        if part == "" || part == "." || part == ".." || len(part) > 255 {
            return "", fmt.Errorf("unsafe game member name %q", part)
        }
        for _, r := range part {
            if r < 32 || r > 126 || strings.ContainsRune("<>:\"/\\|?*", r) {
                return "", fmt.Errorf("unsupported XDVDFS filename character in %q", part)
            }
        }
        if strings.HasSuffix(part, ".") || strings.HasSuffix(part, " ") {
            return "", fmt.Errorf("unsafe trailing dot/space in %q", part)
        }
        name := strings.ToUpper(strings.TrimSuffix(part, filepath.Ext(part)))
        switch name {
        case "CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9", "LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9":
            return "", fmt.Errorf("reserved filesystem filename %q", part)
        }
    }
    return strings.Join(parts, "/"), nil
}

func ensureNewOutput(dest string) error {
    _, err := os.Lstat(dest)
    if err == nil { return fmt.Errorf("output already exists: %s", filepath.Base(dest)) }
    if !errors.Is(err, os.ErrNotExist) { return err }
    return nil
}

// ExtractGameXISO writes every XDVDFS member, including empty directories,
// into an independent destination. Only verified, root-contained paths are used.
func ExtractGameXISO(image, destination string) error {
    if err := ensureNewOutput(destination); err != nil { return err }
    im, err := OpenXISO(image)
    if err != nil { return err }
    src, err := os.Open(image)
    if err != nil { return err }
    defer src.Close()
    tmp, err := os.MkdirTemp(filepath.Dir(destination), ".mpd-extract-*")
    if err != nil { return err }
    defer os.RemoveAll(tmp)
    seen := map[string]bool{}
    for _, dir := range im.Directories {
        name, e := validGameMember(dir)
        if e != nil { return e }
        key := strings.ToLower(name)
        if seen[key] { return fmt.Errorf("duplicate XDVDFS path: %s", name) }
        seen[key] = true
        if e = os.MkdirAll(filepath.Join(tmp, filepath.FromSlash(name)), 0755); e != nil { return e }
    }
    for _, member := range im.Files {
        name, e := validGameMember(member.Path)
        if e != nil { return e }
        key := strings.ToLower(name)
        if seen[key] { return fmt.Errorf("duplicate XDVDFS path: %s", name) }
        seen[key] = true
        target := filepath.Join(tmp, filepath.FromSlash(name))
        if e = os.MkdirAll(filepath.Dir(target), 0755); e != nil { return e }
        f, e := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
        if e != nil { return e }
        _, e = io.Copy(f, io.NewSectionReader(src, im.Base+int64(member.StartSector)*sectorSize, int64(member.Size)))
        if closeErr := f.Close(); e == nil { e = closeErr }
        if e != nil { return fmt.Errorf("extract %s: %w", name, e) }
    }
    if err = ensureNewOutput(destination); err != nil { return err }
    return os.Rename(tmp, destination)
}

// CopyExtractedGame is symlink-safe and never modifies the source directory.
func CopyExtractedGame(source, dest string) error {
    if err := ensureNewOutput(dest); err != nil { return err }
    tmp, err := os.MkdirTemp(filepath.Dir(dest), ".mpd-folder-*")
    if err != nil { return err }
    defer os.RemoveAll(tmp)
    seen := map[string]bool{}
    err = filepath.WalkDir(source, func(name string, entry os.DirEntry, walkErr error) error {
        if walkErr != nil { return walkErr }
        rel, e := filepath.Rel(source, name)
        if e != nil { return e }
        if rel == "." { return nil }
        clean, e := validGameMember(filepath.ToSlash(rel))
        if e != nil { return e }
        key := strings.ToLower(clean)
        if seen[key] { return fmt.Errorf("case-insensitive game path collision: %s", clean) }
        seen[key] = true
        if entry.Type()&os.ModeSymlink != 0 { return fmt.Errorf("symlinks are forbidden: %s", clean) }
        target := filepath.Join(tmp, filepath.FromSlash(clean))
        if entry.IsDir() { return os.MkdirAll(target, 0755) }
        info, e := entry.Info()
        if e != nil { return e }
        if !info.Mode().IsRegular() { return fmt.Errorf("non-regular game file: %s", clean) }
        in, e := os.Open(name)
        if e != nil { return e }
        out, e := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
        if e != nil { in.Close(); return e }
        _, e = io.Copy(out, in)
        if closeErr := in.Close(); e == nil { e = closeErr }
        if closeErr := out.Close(); e == nil { e = closeErr }
        return e
    })
    if err != nil { return err }
    if err = ensureNewOutput(dest); err != nil { return err }
    return os.Rename(tmp, dest)
}

type xdvNode struct {
    name string
    relative string
    folder bool
    children []*xdvNode
    sector uint32
    size uint32
    table []byte
}

func collectXDVTree(source string) (*xdvNode, error) {
    root := &xdvNode{folder:true}
    paths := map[string]*xdvNode{".":root}
    duplicates := map[string]bool{}
    err := filepath.WalkDir(source, func(full string, entry os.DirEntry, walkErr error) error {
        if walkErr != nil { return walkErr }
        rel, e := filepath.Rel(source, full)
        if e != nil { return e }
        if rel == "." { return nil }
        rel, e = validGameMember(filepath.ToSlash(rel))
        if e != nil { return e }
        if duplicates[strings.ToLower(rel)] { return fmt.Errorf("case-insensitive duplicate: %s", rel) }
        duplicates[strings.ToLower(rel)] = true
        if strings.Count(rel, "/") > 23 { return errors.New("XDVDFS nesting exceeds 24 levels") }
        if entry.Type()&os.ModeSymlink != 0 { return fmt.Errorf("symlink forbidden: %s", rel) }
        info, e := entry.Info()
        if e != nil { return e }
        if !entry.IsDir() && !info.Mode().IsRegular() { return fmt.Errorf("non-regular file: %s", rel) }
        if info.Size() > int64(^uint32(0)) { return fmt.Errorf("XDVDFS file exceeds 4 GiB: %s", rel) }
        parent := filepath.ToSlash(filepath.Dir(rel))
        if parent == "" { parent = "." }
        p := paths[parent]
        if p == nil { return fmt.Errorf("missing parent for %s", rel) }
        node := &xdvNode{name:filepath.Base(rel), relative:rel, folder:entry.IsDir(), size:uint32(info.Size())}
        p.children = append(p.children,node)
        paths[rel] = node
        return nil
    })
    if err != nil { return nil, err }
    found := false
    for _, n := range root.children {
        if strings.EqualFold(n.name,"default.xbe") && !n.folder { found = true }
    }
    if !found { return nil, errors.New("root default.xbe missing; cannot build bootable game XISO") }
    return root,nil
}

func prepareXDVTables(root *xdvNode) error {
    if !root.folder { return errors.New("internal XDVDFS directory error") }
    sort.Slice(root.children,func(i,j int)bool {
        return strings.ToLower(root.children[i].name)<strings.ToLower(root.children[j].name)
    })
    total := 0
    for _, child := range root.children {
        if child.folder {
            if err := prepareXDVTables(child); err != nil { return err }
        }
        n := 14+len(child.name)
        total += (n+3)&^3
        if total > 262144 { return errors.New("XDVDFS directory table exceeds pointer limit") }
    }
    root.table = make([]byte,total)
    root.size = uint32(total)
    return nil
}

func xdvWalk(root *xdvNode, visit func(*xdvNode)error) error {
    if err:=visit(root); err!=nil {return err}
    for _, child := range root.children {
        if err:=xdvWalk(child,visit); err!=nil{return err}
    }
    return nil
}
func assignXDV(root *xdvNode) (uint64,error) {
    next := uint64(33) // volume descriptor is at logical sector 32.
    err:=xdvWalk(root,func(n *xdvNode)error {
        if n.folder && n.size>0 {
            if next>uint64(^uint32(0)) {return errors.New("XISO directory sector overflow")}
            n.sector=uint32(next)
            next += (uint64(n.size)+2047)/2048
        }
        return nil
    })
    if err!=nil{return 0,err}
    err=xdvWalk(root,func(n *xdvNode)error {
        if !n.folder {
            if next>uint64(^uint32(0)) {return errors.New("XISO file sector overflow")}
            n.sector=uint32(next)
            next += (uint64(n.size)+2047)/2048
        }
        return nil
    })
    if err!=nil{return 0,err}
    if next>uint64(^uint32(0)) {return 0,errors.New("XISO too large")}
    return next,nil
}
func fillXDVTables(root *xdvNode) error {
    if root.folder {
        offset:=0
        for i,n:=range root.children {
            entry:=root.table[offset:]
            if i+1<len(root.children) {
                next:=offset+((14+len(n.name)+3)&^3)
                if next/4>65535 {return errors.New("XDVDFS sibling pointer overflow")}
                binary.LittleEndian.PutUint16(entry[2:],uint16(next/4))
            }
            binary.LittleEndian.PutUint32(entry[4:],n.sector)
            binary.LittleEndian.PutUint32(entry[8:],n.size)
            if n.folder {entry[12]=0x10} else {entry[12]=0x20}
            entry[13]=byte(len(n.name))
            copy(entry[14:],n.name)
            offset += (14+len(n.name)+3)&^3
        }
    }
    for _,n:=range root.children {
        if err:=fillXDVTables(n); err!=nil{return err}
    }
    return nil
}

// BuildGameXISO constructs a trimmed game-partition XDVDFS image (not a
// reconstructed Redump/full-disc ISO), then verifies every input file hash.
func BuildGameXISO(folder,destination string) error {
    if err:=ensureNewOutput(destination);err!=nil{return err}
    root,err:=collectXDVTree(folder)
    if err!=nil{return err}
    if err=prepareXDVTables(root);err!=nil{return err}
    next,err:=assignXDV(root)
    if err!=nil{return err}
    if err=fillXDVTables(root);err!=nil{return err}
    tmp,err:=os.CreateTemp(filepath.Dir(destination),".mpd-xiso-*")
    if err!=nil{return err}
    tempName:=tmp.Name()
    defer os.Remove(tempName)
    closePending:=true
    defer func(){if closePending {tmp.Close()}}()
    if err=tmp.Truncate(int64(next)*sectorSize);err!=nil{return err}
    vol:=make([]byte,2048)
    copy(vol,volumeMagic)
    copy(vol[0x7ec:],volumeMagic)
    binary.LittleEndian.PutUint32(vol[0x14:],root.sector)
    binary.LittleEndian.PutUint32(vol[0x18:],root.size)
    if _,err=tmp.WriteAt(vol,32*sectorSize);err!=nil{return err}
    err=xdvWalk(root,func(n *xdvNode)error {
        if n.folder && len(n.table)>0 {
            _,e:=tmp.WriteAt(n.table,int64(n.sector)*sectorSize)
            return e
        }
        if n.folder {return nil}
        in,e:=os.Open(filepath.Join(folder,filepath.FromSlash(n.relative)))
        if e!=nil{return e}
        defer in.Close()
        count,e:=io.Copy(io.NewOffsetWriter(tmp,int64(n.sector)*sectorSize),in)
        if e!=nil{return e}
        if count!=int64(n.size) {return fmt.Errorf("source changed during packing: %s",n.relative)}
        return nil
    })
    if err!=nil{return err}
    if err=tmp.Sync();err!=nil{return err}
    if err=tmp.Close();err!=nil{return err}
    closePending=false
    if err=VerifyGameXISO(tempName,folder);err!=nil{return err}
    if err=ensureNewOutput(destination);err!=nil{return err}
    return os.Rename(tempName,destination)
}

func VerifyGameXISO(image,folder string) error {
    im,err:=OpenXISO(image)
    if err!=nil{return err}
    src,err:=os.Open(image)
    if err!=nil{return err}
    defer src.Close()
    files:=map[string]ImageFile{}
    for _, entry:=range im.Files {
        name,e:=validGameMember(entry.Path)
        if e!=nil{return e}
        key:=strings.ToLower(name)
        if _,ok:=files[key];ok{return fmt.Errorf("duplicate XISO member: %s",name)}
        files[key]=entry
    }
    count:=0
    err=filepath.WalkDir(folder,func(path string, d os.DirEntry, e error)error{
        if e!=nil{return e}
        if d.IsDir(){return nil}
        if !d.Type().IsRegular(){return errors.New("non-regular file during verification")}
        rel,e:=filepath.Rel(folder,path)
        if e!=nil{return e}
        rel,e=validGameMember(filepath.ToSlash(rel))
        if e!=nil{return e}
        entry,ok:=files[strings.ToLower(rel)]
        if !ok{return fmt.Errorf("missing output XISO member: %s",rel)}
        f,e:=os.Open(path)
        if e!=nil{return e}
        defer f.Close()
        a:=sha256.New()
        b:=sha256.New()
        n,e:=io.Copy(a,f)
        if e!=nil{return e}
        if n!=int64(entry.Size){return fmt.Errorf("XISO member size mismatch: %s",rel)}
        if _,e=io.Copy(b,io.NewSectionReader(src,im.Base+int64(entry.StartSector)*sectorSize,int64(entry.Size)));e!=nil{return e}
        if !strings.EqualFold(fmt.Sprintf("%x",a.Sum(nil)),fmt.Sprintf("%x",b.Sum(nil))){return fmt.Errorf("XISO member hash mismatch: %s",rel)}
        count++
        return nil
    })
    if err!=nil{return err}
    if count!=len(files){return fmt.Errorf("XISO member count mismatch: %d vs %d",count,len(files))}
    return nil
}

// ExportGameOutputs makes a single independently published directory
// containing all requested artifacts. Previous output directories are NEVER
// overwritten. Changes are applied once to the extracted staging tree.
func ExportGameOutputs(source, destination string, mask uint32, replacements map[string][]byte) error {
    if mask==0 || mask&^GameOutputAll!=0{return errors.New("choose one or more supported output formats")}
    if err:=ensureNewOutput(destination);err!=nil{return err}
    srcInfo,err:=os.Stat(source)
    if err!=nil{return err}
    if srcInfo.IsDir() {
        a,_:=filepath.Abs(source)
        b,_:=filepath.Abs(destination)
        if strings.EqualFold(a,b) || strings.HasPrefix(strings.ToLower(b),strings.ToLower(a)+string(os.PathSeparator)) {
            return errors.New("output cannot be inside source game folder")
        }
    }
    if mask&GameOutputXBE!=0 {
        for target:=range replacements {
            if !strings.EqualFold(strings.ReplaceAll(target,"\\","/"),"default.xbe") {
                return errors.New("XBE-only output would omit other modified game files; deselect XBE-only")
            }
        }
    }
    stage,err:=os.MkdirTemp(filepath.Dir(destination),".mpd-output-*")
    if err!=nil{return err}
    defer os.RemoveAll(stage)
    root:=filepath.Join(stage,"_work")
    inputISO:=""
    fullDisc:=false
    looseXBE:=!srcInfo.IsDir() && strings.EqualFold(filepath.Ext(source),".xbe")
    if srcInfo.IsDir() {
        err=CopyExtractedGame(source,root)
    } else if looseXBE {
        if mask &^GameOutputXBE!=0{return errors.New("loose XBE lacks other game files; only XBE output is possible")}
        err=os.Mkdir(root,0755)
        if err==nil {
            src,e:=os.Open(source)
            if e!=nil{return e}
            dst,e:=os.OpenFile(filepath.Join(root,"default.xbe"),os.O_CREATE|os.O_EXCL|os.O_WRONLY,0644)
            if e!=nil{src.Close();return e}
            _,err=io.Copy(dst,src)
            if e=src.Close();err==nil{err=e}
            if e=dst.Close();err==nil{err=e}
        }
    } else {
        inputISO=source
        if CCISourcePath(source) {
            inputISO=filepath.Join(stage,"_original.iso")
            err=ExtractCCI(source,inputISO,nil)
            if err!=nil{return err}
        }
        im,e:=OpenXISO(inputISO)
        if e!=nil{return e}
        fullDisc=im.Base!=0
        err=ExtractGameXISO(inputISO,root)
    }
    if err!=nil{return err}
    for name,data:=range replacements {
        clean,e:=validGameMember(name)
        if e!=nil{return e}
        target:=filepath.Join(root,filepath.FromSlash(clean))
        st,e:=os.Lstat(target)
        if e!=nil{return fmt.Errorf("patch target not in source: %s: %w",clean,e)}
        if !st.Mode().IsRegular(){return fmt.Errorf("patch target is not regular: %s",clean)}
        if e=os.WriteFile(target,data,0644);e!=nil{return e}
    }
    if mask&GameOutputISO!=0 && (!fullDisc || inputISO=="" || CCISourcePath(source)) {
        return errors.New("full-disc ISO output requires an actual full-disc ISO input; choose XISO or CCI instead")
    }
    name:=filepath.Base(destination)
    if mask&GameOutputXBE!=0 {
        xbePath:=filepath.Join(root,"default.xbe")
        xbe, e:=os.Open(xbePath)
        if e!=nil{return fmt.Errorf("default.xbe missing: %w",e)}
        out,e:=os.OpenFile(filepath.Join(stage,name+".xbe"),os.O_CREATE|os.O_EXCL|os.O_WRONLY,0644)
        if e!=nil{xbe.Close();return e}
        _,err=io.Copy(out,xbe)
        if e=xbe.Close();err==nil{err=e}
        if e=out.Close();err==nil{err=e}
        if err!=nil{return err}
    }
    xiso:=filepath.Join(stage,name+".xiso.iso")
    if mask&(GameOutputXISO|GameOutputCCI)!=0 {
        if err=BuildGameXISO(root,xiso);err!=nil{return err}
    }
    if mask&GameOutputCCI!=0 {
        cci:=filepath.Join(stage,name+".cci")
        if err=CompressCCI(xiso,cci,nil);err!=nil{return err}
        selected:=cci
        if _,e:=os.Stat(selected);errors.Is(e,os.ErrNotExist){selected=filepath.Join(stage,name+".1.cci")}
        roundtrip:=filepath.Join(stage,"_verify.iso")
        if err=ExtractCCI(selected,roundtrip,nil);err!=nil{return err}
        if err=VerifyGameXISO(roundtrip,root);err!=nil{return err}
        if err=os.Remove(roundtrip);err!=nil{return err}
    }
    if mask&GameOutputISO!=0 {
        destISO:=filepath.Join(stage,name+".full.iso")
        if len(replacements)>0 {
            err=PatchImageFiles(inputISO,destISO,replacements,4,nil)
        } else {
            err=CopyImage(inputISO,destISO,4,nil)
        }
        if err!=nil{return err}
        // In-place full-disc writes are only permitted when sectors are safe.
        im,e:=OpenXISO(destISO)
        if e!=nil{return e}
        for rel,data:=range replacements {
            member,e:=im.ResolveTarget(rel)
            if e!=nil{return e}
            stored,e:=im.ReadFile(member)
            if e!=nil{return e}
            if sha256.Sum256(stored)!=sha256.Sum256(data){return fmt.Errorf("ISO verification mismatch: %s",rel)}
        }
    }
    if mask&GameOutputXISO==0 && mask&GameOutputCCI!=0 {
        if err=os.Remove(xiso);err!=nil{return err}
    }
    if inputISO!=source && inputISO!="" {if err=os.Remove(inputISO);err!=nil{return err}}
    if mask&GameOutputFolder!=0 {
        if err=os.Rename(root,filepath.Join(stage,"Extracted"));err!=nil{return err}
    } else {
        if err=os.RemoveAll(root);err!=nil{return err}
    }
    if err=ensureNewOutput(destination);err!=nil{return err}
    return os.Rename(stage,destination)
}
