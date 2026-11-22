package core

import (
    "bytes"
    "crypto/sha256"
    "errors"
    "fmt"
    "io"
    "os"
    "path/filepath"
    "strings"
)

// ErrDuplicatePatch reports that the exact imported JMP bytes are already
// indexed in either the GitHub or Custom patch library. This is intentionally
// a no-op: no source or cached file is modified.
var ErrDuplicatePatch = errors.New("duplicate patch found")

type DuplicatePatchError struct{ Existing string }
func (e *DuplicatePatchError) Error() string {
    return "Duplicate patch found! No need to add."
}
func (e *DuplicatePatchError) Unwrap() error { return ErrDuplicatePatch }

const maxImportedPatchBytes = 16 << 20

// existingIdenticalPatch compares the complete bytes, not a filename, Title
// ID, or CRC32. A different revision with the same name is NOT a duplicate.
// WalkDir does not traverse directory symlinks; symlink files are ignored.
func existingIdenticalPatch(root, extension string, incoming []byte) (string,error) {
    wanted := sha256.Sum256(incoming)
    incomingIsJMP:=strings.EqualFold(extension,".jmp")
    for _, library := range []string{"Custom","GitHub"} {
        directory:=filepath.Join(root,library)
        var matching string
        err:=filepath.WalkDir(directory,func(path string,d os.DirEntry,walkErr error)error{
            if errors.Is(walkErr,os.ErrNotExist){return nil}
            if walkErr!=nil{return walkErr}
            if d.IsDir(){return nil}
            if d.Type()&os.ModeSymlink!=0||!strings.EqualFold(filepath.Ext(path),extension){return nil}
            stat,e:=d.Info()
            if e!=nil{return e}
            if !stat.Mode().IsRegular()||stat.Size()<=0||stat.Size()>maxImportedPatchBytes{return nil}
            if stat.Size()!=int64(len(incoming))&&!incomingIsJMP{return nil}
            file,e:=os.Open(path)
            if e!=nil{return e}
            sum:=sha256.New()
            n,e:=io.Copy(sum,io.LimitReader(file,maxImportedPatchBytes+1))
            closeErr:=file.Close()
            if e!=nil{return e}
            if closeErr!=nil{return closeErr}
            if n!=stat.Size(){return fmt.Errorf("library patch changed while hashing: %s",filepath.Base(path))}
            if n==int64(len(incoming))&&bytes.Equal(sum.Sum(nil),wanted[:]){
                matching=path
                return filepath.SkipAll
            }
            if incomingIsJMP{
                candidate,e:=os.ReadFile(path)
                if e!=nil{return e}
                if EquivalentJMP(incoming,candidate){
                    matching=path
                    return filepath.SkipAll
                }
            }
            return nil
        })
        if err!=nil{return "",err}
        if matching!=""{return matching,nil}
    }
    return "",nil
}

// writeImportedPatchExclusive never replaces an existing same-named file. A
// distinct patch with a colliding filename receives a numbered local filename
// to prevent losing earlier versions or changing their catalogue provenance.
func writeImportedPatchExclusive(directory,name string,data []byte)(string,error){
    ext:=filepath.Ext(name)
    stem:=strings.TrimSuffix(name,ext)
    for suffix:=0;suffix<10000;suffix++ {
        destination:=filepath.Join(directory,name)
        if suffix>0{destination=filepath.Join(directory,fmt.Sprintf("%s (import %d)%s",stem,suffix+1,ext))}
        f,e:=os.OpenFile(destination,os.O_WRONLY|os.O_CREATE|os.O_EXCL,0644)
        if errors.Is(e,os.ErrExist){continue}
        if e!=nil{return "",e}
        n,e:=f.Write(data)
        if e==nil&&n!=len(data){e=io.ErrShortWrite}
        if e==nil{e=f.Sync()}
        closeErr:=f.Close()
        if e==nil{e=closeErr}
        if e!=nil{_ =os.Remove(destination);return "",e}
        return destination,nil
    }
    return "",errors.New("too many local patches with the same filename")
}
