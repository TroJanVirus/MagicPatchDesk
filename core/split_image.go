package core

import (
    "errors"
    "fmt"
    "io"
    "os"
    "path/filepath"
    "strconv"
    "strings"
)

// Supported raw multi-file layouts are Game.1.iso, Game.2.iso, ... or
// Game.1.xiso, Game.2.xiso, ... . They are concatenated logically; no temporary
// 4GB+ joined disc is written simply to look up default.xbe.
func IsNumberedRawImagePart(path string) bool {
    ext:=strings.ToLower(filepath.Ext(path))
    if ext!=".iso"&&ext!=".xiso"{return false}
    stem:=path[:len(path)-len(filepath.Ext(path))]
    dot:=strings.LastIndex(stem,".")
    if dot<0{return false}
    n,err:=strconv.Atoi(stem[dot+1:])
    return err==nil&&n>=1&&n<=64
}

type rawDiscPart struct {
    path string
    f *os.File
    start int64
    size int64
}
type rawDiscReader struct{
    parts []rawDiscPart
    size int64
}
func (r *rawDiscReader) Close(){
    for i:=range r.parts {
        if r.parts[i].f!=nil{_ = r.parts[i].f.Close();r.parts[i].f=nil}
    }
}

func rawImagePartNames(path string)([]string,error){
    if !IsNumberedRawImagePart(path){return nil,errors.New("expected a numbered .iso or .xiso part")}
    ext:=filepath.Ext(path)
    stem:=strings.TrimSuffix(path,ext)
    dot:=strings.LastIndex(stem,".")
    base:=stem[:dot]
    selected,_:=strconv.Atoi(stem[dot+1:])
    entries,err:=os.ReadDir(filepath.Dir(path))
    if err!=nil{return nil,err}
    prefix:=strings.ToLower(filepath.Base(base))+"."
    parts:=map[int]string{}
    max:=0
    for _,entry:=range entries{
        lower:=strings.ToLower(entry.Name())
        if !strings.HasPrefix(lower,prefix)||!strings.HasSuffix(lower,strings.ToLower(ext)){continue}
        middle:=strings.TrimSuffix(strings.TrimPrefix(lower,prefix),strings.ToLower(ext))
        n,e:=strconv.Atoi(middle)
        if e!=nil||n<1{continue}
        if n>64{return nil,fmt.Errorf("split ISO set exceeds 64 parts: %s",entry.Name())}
        if _,exists:=parts[n];exists{return nil,fmt.Errorf("duplicate split ISO part %d",n)}
        if !entry.Type().IsRegular(){return nil,fmt.Errorf("nonregular split ISO part: %s",entry.Name())}
        parts[n]=filepath.Join(filepath.Dir(path),entry.Name())
        if n>max{max=n}
    }
    if _,ok:=parts[1];!ok{return nil,errors.New("split ISO first part (.1.iso or .1.xiso) is missing")}
    if selected>max{return nil,errors.New("selected split ISO part is missing")}
    out:=make([]string,0,max)
    for i:=1;i<=max;i++{
        p,ok:=parts[i]
        if !ok{return nil,fmt.Errorf("split ISO set is missing part %d",i)}
        out=append(out,p)
    }
    return out,nil
}
func openRawDiscParts(path string)(*rawDiscReader,error){
    paths,err:=rawImagePartNames(path)
    if err!=nil{return nil,err}
    r:=&rawDiscReader{}
    valid:=false
    defer func(){if !valid{r.Close()}}()
    for _,p:=range paths{
        f,e:=os.Open(p)
        if e!=nil{return nil,e}
        stat,e:=f.Stat()
        if e!=nil{f.Close();return nil,e}
        if !stat.Mode().IsRegular()||stat.Size()==0{
            f.Close()
            return nil,fmt.Errorf("empty or nonregular split ISO part: %s",filepath.Base(p))
        }
        if stat.Size()>1<<42 || r.size>1<<46-stat.Size(){
            f.Close()
            return nil,errors.New("split ISO size exceeds safety limit")
        }
        r.parts=append(r.parts,rawDiscPart{path:p,f:f,start:r.size,size:stat.Size()})
        r.size+=stat.Size()
    }
    valid=true
    return r,nil
}
func (r *rawDiscReader) ReadAt(p []byte,off int64)(int,error){
    if off<0{return 0,errors.New("negative split ISO read offset")}
    if len(p)==0{return 0,nil}
    if off>=r.size{return 0,io.EOF}
    done:=0
    for _,part:=range r.parts {
        if off>=part.start+part.size{continue}
        if off<part.start{continue}
        start:=off-part.start
        n:=int64(len(p)-done)
        if n>part.size-start{n=part.size-start}
        count,err:=part.f.ReadAt(p[done:done+int(n)],start)
        done+=count;off+=int64(count)
        if err!=nil&&err!=io.EOF{return done,err}
        if count!=int(n){return done,io.ErrUnexpectedEOF}
        if done==len(p){return done,nil}
    }
    return done,io.EOF
}

// Only complete, contiguous raw split sets can be inspected. Standalone ISO
// files retain the existing OpenXISO behavior.
func RawImagePartSummary(path string)(string,error){
    if !IsNumberedRawImagePart(path){return "",nil}
    files,err:=rawImagePartNames(path)
    if err!=nil{return "",err}
    index:=0
    for i,p:=range files {
        if strings.EqualFold(filepath.Clean(path),filepath.Clean(p)){index=i+1;break}
    }
    if index==0{return "",errors.New("selected raw ISO part not found in set")}
    return fmt.Sprintf("ISO part %d of %d",index,len(files)),nil
}
