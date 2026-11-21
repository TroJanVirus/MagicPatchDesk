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
        // Windows treats CON.txt.more like CON: inspect everything before the first dot.
        // Trimming spaces catches Win32 aliases such as "CON .txt".
        name := strings.ToUpper(strings.TrimRight(strings.SplitN(part, ".", 2)[0], " "))
        switch name {
        case "CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9", "LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9":
            return "", fmt.Errorf("reserved filesystem filename %q", part)
        }
    }
    return strings.Join(parts, "/"), nil
}

// resolveGameMember preserves the source filename spelling, while matching
// patch targets case-insensitively as XDVDFS does. It rejects ambiguous names.
func resolveGameMember(root, member string) (string,error) {
    canonical,err:=validGameMember(member)
    if err!=nil{return "",err}
    current:=root
    for _,part:=range strings.Split(canonical,"/") {
        entries,e:=os.ReadDir(current)
        if e!=nil{return "",e}
        found:=""
        for _,entry:=range entries {
            if strings.EqualFold(entry.Name(),part) {
                if found!="" {return "",fmt.Errorf("ambiguous case-insensitive target: %s",canonical)}
                found=entry.Name()
            }
        }
        if found==""{return "",fmt.Errorf("game member not found: %s",canonical)}
        current=filepath.Join(current,found)
        info,e:=os.Lstat(current)
        if e!=nil{return "",e}
        if info.Mode()&os.ModeSymlink!=0{return "",fmt.Errorf("symlink target forbidden: %s",canonical)}
    }
    return current,nil
}

// ReadGameMember resolves a patch target case-insensitively, rejects symlinks,
// and limits memory use before returning a file for a named patch operation.
func ReadGameMember(root, member string) ([]byte,error) {
    target,err:=resolveGameMember(root,member)
    if err!=nil{return nil,err}
    info,err:=os.Lstat(target)
    if err!=nil{return nil,err}
    if !info.Mode().IsRegular(){return nil,fmt.Errorf("game target is not a regular file: %s",member)}
    if info.Size()>512<<20{return nil,fmt.Errorf("game target exceeds 512 MiB patch limit: %s",member)}
    file,err:=os.Open(target)
    if err!=nil{return nil,err}
    defer file.Close()
    b,err:=io.ReadAll(io.LimitReader(file,info.Size()+1))
    if err!=nil{return nil,err}
    if int64(len(b))!=info.Size(){return nil,fmt.Errorf("game target changed during read: %s",member)}
    return b,nil
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
        written, copyErr := io.Copy(f, io.NewSectionReader(src, im.Base+int64(member.StartSector)*sectorSize, int64(member.Size)))
        if closeErr := f.Close(); copyErr == nil { copyErr = closeErr }
        if copyErr != nil { return fmt.Errorf("extract %s: %w", name, copyErr) }
        if written != int64(member.Size) { return fmt.Errorf("truncated XISO file %s: copied %d of %d bytes", name, written, member.Size) }
    }
    if err = ensureNewOutput(destination); err != nil { return err }
    return os.Rename(tmp, destination)
}

// CopyExtractedGame is symlink-safe and never modifies the source directory.
func CopyExtractedGame(source, dest string) error {
    info,err:=os.Lstat(source)
    if err!=nil{return err}
    if !info.IsDir(){return errors.New("source must be a real game directory, not a symlink")}
    a,_:=filepath.Abs(source)
    b,_:=filepath.Abs(dest)
    if strings.EqualFold(a,b) || strings.HasPrefix(strings.ToLower(b),strings.ToLower(a)+string(os.PathSeparator)){
        return errors.New("output cannot be inside source folder")
    }
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
        copied, copyErr := io.Copy(out, in)
        if closeErr := in.Close(); copyErr == nil { copyErr = closeErr }
        if closeErr := out.Close(); copyErr == nil { copyErr = closeErr }
        if copyErr != nil { return copyErr }
        if copied != info.Size() { return fmt.Errorf("source file changed during copy: %s", clean) }
        return nil
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
    info,err:=os.Lstat(source)
    if err!=nil{return nil,err}
    if !info.IsDir(){return nil,errors.New("source must be an actual game directory")}
    root := &xdvNode{folder:true}
    paths := map[string]*xdvNode{".":root}
    duplicates := map[string]bool{}
    err = filepath.WalkDir(source, func(full string, entry os.DirEntry, walkErr error) error {
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
    // XDVDFS directory entry padding is 0xFF, not an implicit zero.
    for i:=range root.table {root.table[i]=0xFF}
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
// fillXDVTables emits a balanced binary-search tree with the root at offset 0.
// XDVDFS directory lookups follow left/right pointers; a right-only linked
// chain degrades to O(N) depth for folders containing thousands of files.
func fillXDVTables(root *xdvNode) error {
    if root.folder && len(root.children)>0 {
        order:=make([]*xdvNode,0,len(root.children))
        var appendBalanced func(int,int)
        appendBalanced=func(start,end int){
            if start>=end{return}
            middle:=start+(end-start)/2
            order=append(order,root.children[middle])
            appendBalanced(start,middle)
            appendBalanced(middle+1,end)
        }
        appendBalanced(0,len(root.children))
        positions:=make(map[*xdvNode]int,len(order))
        offset:=0
        for _,n:=range order {
            positions[n]=offset
            offset+=(14+len(n.name)+3)&^3
        }
        if offset!=len(root.table){return errors.New("internal XDVDFS directory table size mismatch")}
        var emit func(int,int)(*xdvNode,error)
        emit=func(start,end int)(*xdvNode,error){
            if start>=end{return nil,nil}
            middle:=start+(end-start)/2
            n:=root.children[middle]
            left,e:=emit(start,middle)
            if e!=nil{return nil,e}
            right,e:=emit(middle+1,end)
            if e!=nil{return nil,e}
            entry:=root.table[positions[n]:]
            // Clear the 14-byte header while preserving 0xFF name padding.
            for i:=0;i<14;i++{entry[i]=0}
            if left!=nil {
                pos:=positions[left]/4
                if pos==0||pos>65535{return nil,errors.New("XDVDFS left pointer overflow")}
                binary.LittleEndian.PutUint16(entry[0:],uint16(pos))
            }
            if right!=nil {
                pos:=positions[right]/4
                if pos==0||pos>65535{return nil,errors.New("XDVDFS right pointer overflow")}
                binary.LittleEndian.PutUint16(entry[2:],uint16(pos))
            }
            binary.LittleEndian.PutUint32(entry[4:],n.sector)
            binary.LittleEndian.PutUint32(entry[8:],n.size)
            if n.folder {entry[12]=0x10}else{entry[12]=0x20}
            entry[13]=byte(len(n.name))
            copy(entry[14:],n.name)
            return n,nil
        }
        if _,err:=emit(0,len(root.children));err!=nil{return err}
    }
    for _,n:=range root.children {
        if err:=fillXDVTables(n);err!=nil{return err}
    }
    return nil
}

// BuildGameXISO constructs a trimmed game-partition XDVDFS image (not a
// reconstructed Redump/full-disc ISO), then verifies every input file hash.
func BuildGameXISO(folder,destination string) error {
    sourceAbs,_:=filepath.Abs(folder)
    destAbs,_:=filepath.Abs(destination)
    if strings.EqualFold(sourceAbs,destAbs) || strings.HasPrefix(strings.ToLower(destAbs),strings.ToLower(sourceAbs)+string(os.PathSeparator)){
        return errors.New("XISO output must be outside the extracted source folder")
    }
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
    // Physical XDVDFS images are padded to a 64 KiB boundary; directory
    // and file LBAs remain relative to the same game-partition origin.
    imageBytes:=int64(next)*sectorSize
    paddedBytes:=(imageBytes+0xFFFF)&^int64(0xFFFF)
    if err=tmp.Truncate(paddedBytes);err!=nil{return err}
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
        count,copyErr:=io.CopyN(io.NewOffsetWriter(tmp,int64(n.sector)*sectorSize),in,int64(n.size))
        info,statErr:=in.Stat()
        closeErr:=in.Close()
        if copyErr!=nil{return fmt.Errorf("pack %s: %w",n.relative,copyErr)}
        if statErr!=nil{return statErr}
        if closeErr!=nil{return closeErr}
        if count!=int64(n.size) || info.Size()!=int64(n.size) {return fmt.Errorf("source changed during packing: %s",n.relative)}
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

// VerifyGameXISO independently reopens the packed XDVDFS tree and compares
// every member's size and SHA256 plus directory presence (including empty dirs).
// It also detects case collisions and rejects symlinked input paths.
func VerifyGameXISO(image,folder string) error {
    info,err:=os.Lstat(folder)
    if err!=nil{return err}
    if !info.IsDir(){return errors.New("verification requires an actual game directory")}
    im,err:=OpenXISO(image)
    if err!=nil{return err}
    src,err:=os.Open(image)
    if err!=nil{return err}
    defer src.Close()
    files:=map[string]ImageFile{}
    dirs:=map[string]bool{}
    for _, dir:=range im.Directories{
        name,e:=validGameMember(dir)
        if e!=nil{return e}
        key:=strings.ToLower(name)
        if dirs[key]{return fmt.Errorf("duplicate XISO directory: %s",name)}
        dirs[key]=true
    }
    for _, entry:=range im.Files {
        name,e:=validGameMember(entry.Path)
        if e!=nil{return e}
        key:=strings.ToLower(name)
        if _,ok:=files[key];ok{return fmt.Errorf("duplicate XISO member: %s",name)}
        if dirs[key]{return fmt.Errorf("XISO path is both file and folder: %s",name)}
        files[key]=entry
    }
    seenFiles:=map[string]bool{}
    seenDirs:=map[string]bool{}
    err=filepath.WalkDir(folder,func(p string,d os.DirEntry,walkErr error)error{
        if walkErr!=nil{return walkErr}
        rel,e:=filepath.Rel(folder,p)
        if e!=nil{return e}
        if rel=="."{return nil}
        rel,e=validGameMember(filepath.ToSlash(rel))
        if e!=nil{return e}
        key:=strings.ToLower(rel)
        if d.Type()&os.ModeSymlink!=0{return fmt.Errorf("symbolic link forbidden during verification: %s",rel)}
        if d.IsDir(){
            if seenDirs[key]{return fmt.Errorf("duplicate source directory: %s",rel)}
            seenDirs[key]=true
            if !dirs[key]{return fmt.Errorf("missing output XISO directory: %s",rel)}
            return nil
        }
        if !d.Type().IsRegular(){return fmt.Errorf("non-regular source file: %s",rel)}
        if seenFiles[key]{return fmt.Errorf("duplicate source file: %s",rel)}
        seenFiles[key]=true
        entry,ok:=files[key]
        if !ok{return fmt.Errorf("missing output XISO member: %s",rel)}
        f,e:=os.Open(p)
        if e!=nil{return e}
        hashSource:=sha256.New()
        readErr:=error(nil)
        n,copyErr:=io.Copy(hashSource,f)
        readErr=copyErr
        if closeErr:=f.Close();readErr==nil{readErr=closeErr}
        if readErr!=nil{return readErr}
        if n!=int64(entry.Size){return fmt.Errorf("XISO member size mismatch: %s",rel)}
        hashImage:=sha256.New()
        copied,e:=io.Copy(hashImage,io.NewSectionReader(src,im.Base+int64(entry.StartSector)*sectorSize,int64(entry.Size)))
        if e!=nil{return e}
        if copied!=int64(entry.Size){return fmt.Errorf("XISO member truncated: %s",rel)}
        if fmt.Sprintf("%x",hashSource.Sum(nil))!=fmt.Sprintf("%x",hashImage.Sum(nil)){
            return fmt.Errorf("XISO member SHA256 mismatch: %s",rel)
        }
        return nil
    })
    if err!=nil{return err}
    if len(seenFiles)!=len(files){return fmt.Errorf("XISO file count mismatch: %d vs %d",len(seenFiles),len(files))}
    if len(seenDirs)!=len(dirs){return fmt.Errorf("XISO directory count mismatch: %d vs %d",len(seenDirs),len(dirs))}
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
        target,e:=resolveGameMember(root,clean)
        if e!=nil{return e}
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
        xbePath,e:=resolveGameMember(root,"default.xbe")
        if e!=nil{return fmt.Errorf("default.xbe missing: %w",e)}
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
