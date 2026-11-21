package core

import (
    "encoding/hex"
    "errors"
    "fmt"
    "strings"
    "sync"
    "time"
)

// VerificationEntry is a local, human-readable verification event. It is never
// transmitted by this type; use DiagnosticReport for privacy-filtered upload.
type VerificationEntry struct {
    AtUTC time.Time
    Level string // info, pass, warning, error
    Step string
    Detail string
}
type VerificationLog struct {
    mu sync.Mutex
    entries []VerificationEntry
    limit int
}
func NewVerificationLog(limit int) *VerificationLog {
    if limit<1{limit=400}
    if limit>5000{limit=5000}
    return &VerificationLog{limit:limit}
}
func (l *VerificationLog) Add(level,step,detail string) {
    if l==nil{return}
    switch level{case "info","pass","warning","error":default:level="info"}
    detail=strings.TrimSpace(detail)
    l.mu.Lock()
    defer l.mu.Unlock()
    l.entries=append(l.entries,VerificationEntry{AtUTC:time.Now().UTC(),Level:level,Step:step,Detail:detail})
    if len(l.entries)>l.limit{
        copy(l.entries,l.entries[len(l.entries)-l.limit:])
        l.entries=l.entries[:l.limit]
    }
}
func (l *VerificationLog) Snapshot() []VerificationEntry {
    if l==nil{return nil}
    l.mu.Lock()
    defer l.mu.Unlock()
    out:=make([]VerificationEntry,len(l.entries))
    copy(out,l.entries)
    return out
}
func (l *VerificationLog) Clear(){if l==nil{return};l.mu.Lock();l.entries=nil;l.mu.Unlock()}

// ByteOverlap describes one potentially conflicting changed byte from two
// JMPs evaluated against THE SAME ORIGINAL. It does not silently choose an
// ordering; the UI should display conflicting bytes as warnings.
type ByteOverlap struct {
    Offset int64
    PatchA string
    PatchB string
    Original byte
    A byte
    B byte
    Same bool
    OriginalHex string
    AHex string
    BHex string
}
type OverlapReport struct {
    Overlaps []ByteOverlap
    Total int
    Different int
    Truncated bool
}

// CompareJMPOverlaps independently applies each JMP to the original. It
// refuses files too large for interactive comparison and limits visible entries.
// Source CRC verification is never bypassed for this analysis.
func CompareJMPOverlaps(original []byte, patches []*Patch, maximum int) (OverlapReport,error) {
    var report OverlapReport
    if len(original)>64<<20{return report,errors.New("hex overlap preview is limited to 64 MiB")}
    if len(patches)<2{return report,errors.New("select two or more JMPs")}
    if maximum<=0 || maximum>4096{maximum=256}
    seen:=map[int]struct{patch string;value byte}{}
    for i,p:=range patches{
        if p==nil{return report,fmt.Errorf("missing JMP %d",i+1)}
        if err:=p.CheckSingleFileOnly();err!=nil{return report,fmt.Errorf("JMP %s: %w",p.Filename,err)}
        changed,err:=p.Apply(original,false)
        if err!=nil{return report,fmt.Errorf("JMP %s: %w",p.Filename,err)}
        for at:=0;at<len(changed);at++{
            orig:=byte(0)
            if at<len(original){orig=original[at]}
            value:=changed[at]
            if at<len(original) && value==orig{continue}
            if previous,ok:=seen[at];ok{
                if previous.patch==p.Filename{continue}
                report.Total++
                different:=previous.value!=value
                if different{report.Different++}
                if len(report.Overlaps)<maximum{
                    width:=12
                    start:=at-width/2
                    if start<0{start=0}
                    stop:=at+width/2+1
                    if stop>len(original){stop=len(original)}
                    originalWindow:=original[start:stop]
                    afterA:=append([]byte(nil),originalWindow...)
                    afterB:=append([]byte(nil),originalWindow...)
                    if at<len(original) {afterA[at-start]=previous.value;afterB[at-start]=value}
                    report.Overlaps=append(report.Overlaps,ByteOverlap{
                        Offset:int64(at),PatchA:previous.patch,PatchB:p.Filename,
                        Original:orig,A:previous.value,B:value,Same:!different,
                        OriginalHex:strings.ToUpper(hex.EncodeToString(originalWindow)),
                        AHex:strings.ToUpper(hex.EncodeToString(afterA)),
                        BHex:strings.ToUpper(hex.EncodeToString(afterB)),
                    })
                }else{report.Truncated=true}
            }else{
                if len(seen)>=1<<20{return report,errors.New("JMP changes exceed the safe hex-overlap comparison budget")}
                seen[at]=struct{patch string;value byte}{p.Filename,value}
            }
        }
    }
    return report,nil
}
