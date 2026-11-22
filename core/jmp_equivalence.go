package core

import (
    "bytes"
    "crypto/sha256"
    "encoding/json"
    "sort"
    "strings"
)

// EquivalentJMP compares *behavior*, not only byte-for-byte or filename.
// Use this only after ordinary JMP parsing has validated the file. It keeps
// different source identity, per-record targeting, record order, group
// activation and bytes distinct. Author credit and prose changes alone must
// not turn the same patch edits into a second library entry.
func EquivalentJMP(a,b []byte) bool {
    if bytes.Equal(a,b) || bytes.Equal(canonicalJMPText(a),canonicalJMPText(b)) {return true}
    left,errA:=ParseJMP("a.jmp",a)
    right,errB:=ParseJMP("b.jmp",b)
    if errA!=nil||errB!=nil{return false}
    first,okA:=semanticJMPFingerprint(left)
    second,okB:=semanticJMPFingerprint(right)
    return okA&&okB&&first==second
}

func canonicalJMPText(data []byte)[]byte{
    s:=strings.TrimPrefix(string(data),"\ufeff")
    s=strings.ReplaceAll(strings.ReplaceAll(s,"\r\n","\n"),"\r","\n")
    lines:=strings.Split(s,"\n")
    out:=make([]string,0,len(lines))
    for _,v:=range lines{
        v=strings.TrimSpace(v)
        if v==""{continue}
        out=append(out,v)
    }
    return []byte(strings.Join(out,"\n"))
}
type recordFingerprint struct{
    Kind RecordKind `json:"kind"`
    Target string `json:"target"`
    Group string `json:"group"`
    Enabled bool `json:"enabled"`
    Offset int64 `json:"offset,omitempty"`
    Find []byte `json:"find,omitempty"`
    Replace []byte `json:"replace,omitempty"`
    Data []byte `json:"data,omitempty"`
}
type jmpFingerprint struct{
    System string `json:"system"`
    Game string `json:"game"`
    Region string `json:"region"`
    Identity string `json:"identity"`
    CRCs []uint32 `json:"crcs"`
    Records []recordFingerprint `json:"records"`
}
func semanticJMPFingerprint(p *Patch)([32]byte,bool){
    var empty [32]byte
    if p==nil||len(p.Records)==0{return empty,false}
    id:=TitleIDFromVersion(p.Headers["version"])
    crcs:=append([]uint32(nil),p.CRCs...)
    sort.Slice(crcs,func(i,j int)bool{return crcs[i]<crcs[j]})
    identity:=id
    if id==""||len(crcs)==0{
        // Do not merge patches for two games or releases merely because the
        // current byte edits happen to be the same.
        identity=strings.ToLower(strings.Join(strings.Fields(p.Headers["version"])," "))
    }
    key:=jmpFingerprint{
        System:strings.ToLower(strings.TrimSpace(p.Headers["system"])),
        Game:CoverMatchKey(p.Headers["game-title"]),
        Region:strings.ToLower(strings.TrimSpace(p.Headers["region"])),
        Identity:identity,CRCs:crcs,
        Records:make([]recordFingerprint,0,len(p.Records)),
    }
    for _,r:=range p.Records{
        target,err:=p.RecordTarget(r)
        if err!=nil {
            // Several file names inside instructional notes make the patch
            // ambiguous for application, but still permit conservative
            // import deduplication if the *same targets* are named.
            explicit,e:=FileTargetsInText(r.Comment)
            if e!=nil||len(explicit)!=0{return empty,false}
            names,e:=FileTargetsInText(FormatJMPNotes(p.Headers["notes"]))
            if e!=nil||len(names)<2{return empty,false}
            target="ambiguous:"+strings.Join(names,"|")
        }
        key.Records=append(key.Records,recordFingerprint{
            Kind:r.Kind,Target:target,Group:strings.ToLower(strings.TrimSpace(r.Group)),
            Enabled:r.Enabled,Offset:r.Offset,Find:r.Find,Replace:r.Replace,Data:r.Data,
        })
    }
    packed,err:=json.Marshal(key)
    if err!=nil{return empty,false}
    return sha256.Sum256(packed),true
}
