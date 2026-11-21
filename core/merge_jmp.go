package core

import (
    "bytes"
    "errors"
    "fmt"
    "strings"
)

// CombineJMPs produces one source-verified offset-based JMP. Every source JMP
// is first checked independently against the SAME original, to reveal byte
// overlaps. Patches are then applied in their displayed order. Conflicting
// writes require an explicit second approval; output is re-applied and verified.
// The combined JMP preserves original author and patch-source attribution in
// metadata, but cannot represent multiple target files in one loose XBE.
func CombineJMPs(original []byte,patches []*Patch,approveConflicts bool)([]byte,OverlapReport,error){
    var report OverlapReport
    if len(patches)<2{return nil,report,errors.New("choose at least two JMP patches to combine")}
    if len(original)>64<<20{return nil,report,errors.New("original XBE exceeds 64MiB combine limit")}
    info,e:=InspectXBE(original)
    if e!=nil{return nil,report,fmt.Errorf("cannot combine patches without a valid original XBE: %w",e)}
    for _,p:=range patches{
        if p==nil{return nil,report,errors.New("missing JMP")}
        if e=p.CheckSingleFileOnly();e!=nil{return nil,report,e}
        if e=p.VerifyCRC(original);e!=nil{return nil,report,fmt.Errorf("%s: %w",p.Filename,e)}
        expected:=TitleIDFromVersion(p.Headers["version"])
        if expected!=""&&!strings.EqualFold(expected,info.TitleID){
            return nil,report,fmt.Errorf("%s has a different Title ID than the original XBE",p.Filename)
        }
    }
    report,e=CompareJMPOverlaps(original,patches,256)
    if e!=nil{return nil,report,e}
    if report.Different>0&&!approveConflicts{
        return nil,report,fmt.Errorf("%d conflicting changed byte(s): explicit manual approval is required",report.Different)
    }
    current:=bytes.Clone(original)
    authors:=[]string{}
    names:=[]string{}
    for _,p:=range patches{
        // The original's version was checked above; sequential operations
        // may intentionally depend on changes from earlier JMPs.
        next,e:=p.Apply(current,true)
        if e!=nil{return nil,report,fmt.Errorf("combine %s: %w",p.Filename,e)}
        current=next
        a:=strings.TrimSpace(p.Headers["author"])
        if a==""{a="Unknown"}
        authors=append(authors,a)
        // Target discovery parses filename-like tokens in notes. Escape
        // the dot in original JMP names to preserve source provenance
        // without accidentally rerouting new offset records.
        names=append(names,strings.ReplaceAll(p.Filename,".","[dot]"))
    }
    jmpText,e:=DiffToJMP(original,current,info.Title)
    if e!=nil{return nil,report,e}
    merged,e:=ParseJMP("combined.jmp",[]byte(jmpText))
    if e!=nil{return nil,report,e}
    merged.Headers["author"]=strings.Join(authors," + ")
    merged.Headers["notes"]="Combined from "+strings.Join(names," -> ")+" (source patch authors: "+strings.Join(authors,", ")+")"
    encoded:=SerializeJMP(merged)
    reloaded,e:=ParseJMP("combined.jmp",encoded)
    if e!=nil{return nil,report,e}
    verify,e:=reloaded.Apply(original,false)
    if e!=nil{return nil,report,e}
    if !bytes.Equal(verify,current){return nil,report,errors.New("combined JMP failed independent re-application check")}
    return encoded,report,nil
}
