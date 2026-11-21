package core

import (
    "bytes"
    "context"
    "crypto/sha256"
    "encoding/binary"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "hash/crc32"
    "net/http"
    "net/http/httptest"
    "os"
    "path/filepath"
    "strings"
    "testing"
    "time"
)
func qaGame(t *testing.T)(string,[]byte){
    t.Helper()
    dir:=filepath.Join(t.TempDir(),"games","Example Game")
    if err:=os.MkdirAll(dir,0755);err!=nil{t.Fatal(err)}
    xbe:=make([]byte,0x500)
    copy(xbe,"XBEH")
    binary.LittleEndian.PutUint32(xbe[0x104:],0x10000)
    binary.LittleEndian.PutUint32(xbe[0x118:],0x10200)
    binary.LittleEndian.PutUint32(xbe[0x208:],0x4A580001)
    for i,r:=range []rune("QA Game"){binary.LittleEndian.PutUint16(xbe[0x20C+i*2:],uint16(r))}
    binary.LittleEndian.PutUint32(xbe[0x2A0:],4)
    if err:=os.WriteFile(filepath.Join(dir,"default.xbe"),xbe,0644);err!=nil{t.Fatal(err)}
    if err:=os.MkdirAll(filepath.Join(dir,"empty"),0755);err!=nil{t.Fatal(err)}
    return dir,xbe
}
func qaPatch(t *testing.T,xbe []byte,dir string,name string,hexOffset string,hexValue string)string {
    t.Helper()
    p:=filepath.Join(dir,name+".jmp")
    contents:=fmt.Sprintf("system=XBOX\ngame-title=QA Game\nregion=PAL\nversion=4A580001 {%08X}\nauthor=Tester\nnotes=verified\n#Modify\n%s:%s\n",crc32.ChecksumIEEE(xbe),hexOffset,hexValue)
    if e:=os.WriteFile(p,[]byte(contents),0644);e!=nil{t.Fatal(e)}
    return p
}
func TestLibraryScanAndCautiousBatch(t *testing.T){
    folder,orig:=qaGame(t)
    patches:=t.TempDir()
    pf:=qaPatch(t,orig,patches,"Wide","300","FE")
    catalog:=Catalog{Games:[]Game{{Title:"QA Game",Patches:[]CatalogPatch{
        {File:pf,Name:"Wide",TitleID:"4A580001",CRC:fmt.Sprintf("%08X",crc32.ChecksumIEEE(orig)),Author:"Tester"},
    }}}}
    log:=NewVerificationLog(100)
    scan,e:=ScanGameLibrary(context.Background(),filepath.Dir(folder),catalog,log)
    if e!=nil{t.Fatal(e)}
    if len(scan.Games)!=1{t.Fatalf("found %d games, want 1",len(scan.Games))}
    g:=scan.Games[0]
    if g.Status!="ready"||len(g.Patches)!=1||g.Patches[0].Status!="exact"{t.Fatalf("unexpected game match: %+v",g)}
    output:=filepath.Join(t.TempDir(),"patched")
    jobs:=[]BatchJob{{Game:g,Decision:BatchAuto,PatchFile:pf,OutputMask:GameOutputFolder|GameOutputXBE}}
    summary,e:=RunBatchJobs(context.Background(),filepath.Dir(folder),output,jobs,log)
    if e!=nil{t.Fatal(e)}
    if summary.Completed!=1||summary.Failed!=0{t.Fatalf("unexpected batch summary: %+v",summary)}
    dest:=filepath.Join(summary.Outputs[0],"Extracted","default.xbe")
    after,e:=os.ReadFile(dest)
    if e!=nil{t.Fatal(e)}
    if after[0x300]!=0xFE{t.Fatal("patch did not change output")}
    originalAgain,e:=os.ReadFile(filepath.Join(folder,"default.xbe"))
    if e!=nil||!bytes.Equal(orig,originalAgain){t.Fatal("source XBE modified",e)}
    if len(log.Snapshot())<3{t.Fatal("visual log lacked preflight/export/verification entries")}
    summary,e=RunBatchJobs(context.Background(),filepath.Dir(folder),output,jobs,log)
    if e!=nil{t.Fatal(e)}
    if summary.Completed!=0||summary.Failed!=1{t.Fatal("existing output overwritten instead of being rejected")}
}
func TestBatchRefusesAmbiguityAndMovedSource(t *testing.T){
    folder,orig:=qaGame(t)
    patches:=t.TempDir()
    p1:=qaPatch(t,orig,patches,"A","300","EF")
    p2:=qaPatch(t,orig,patches,"B","300","EA")
    crc:=fmt.Sprintf("%08X",crc32.ChecksumIEEE(orig))
    c:=Catalog{Games:[]Game{{Title:"QA",Patches:[]CatalogPatch{{File:p1,TitleID:"4A580001",CRC:crc},{File:p2,TitleID:"4A580001",CRC:crc}}}}}
    scan,e:=ScanGameLibrary(context.Background(),filepath.Dir(folder),c,nil)
    if e!=nil{t.Fatal(e)}
    if scan.Games[0].Status!="review"{t.Fatal("two exact JMPs must be left for manual approval")}
    job:=BatchJob{Game:scan.Games[0],Decision:BatchAuto,PatchFile:p1,OutputMask:GameOutputXBE}
    if e:=VerifyBatchJob(job);e==nil{t.Fatal("ambiguous game approved")}
    job.Decision=BatchReview
    summary,e:=RunBatchJobs(context.Background(),filepath.Dir(folder),filepath.Join(t.TempDir(),"out"),[]BatchJob{job},nil)
    if e!=nil||summary.Manual!=1{t.Fatalf("manual review not preserved: %+v %v",summary,e)}
}
func TestJMPOffsetOverlapHexPreview(t *testing.T){
    src:=bytes.Repeat([]byte{0x11},64)
    newPatch:=func(name string,data byte)*Patch{return &Patch{Filename:name,Headers:map[string]string{},Records:[]Record{{Kind:Offset,Offset:20,Data:[]byte{data},Enabled:true}}}}
    r,e:=CompareJMPOverlaps(src,[]*Patch{newPatch("a.jmp",0x33),newPatch("b.jmp",0x44)},100)
    if e!=nil{t.Fatal(e)}
    if r.Total!=1||r.Different!=1||len(r.Overlaps)!=1{t.Fatalf("wrong overlap: %+v",r)}
    overlap:=r.Overlaps[0]
    if overlap.Offset!=20||overlap.A!=0x33||overlap.B!=0x44||overlap.Same{t.Fatalf("wrong overlapping hex: %+v",overlap)}
    if overlap.OriginalHex==""||overlap.AHex==""||overlap.BHex==""{t.Fatal("hex previews missing")}
}
func TestImageExplorerSingleFileExtraction(t *testing.T){
    folder,_:=qaGame(t)
    iso:=filepath.Join(t.TempDir(),"game.iso")
    if e:=BuildGameXISO(folder,iso);e!=nil{t.Fatal(e)}
    entries,e:=BrowseGameImage(iso)
    if e!=nil{t.Fatal(e)}
    if len(entries)<2{t.Fatal("XISO browser omitted members and directories")}
    out:=filepath.Join(t.TempDir(),"one.xbe")
    if e:=ExtractImageMember(iso,"default.xbe",out);e!=nil{t.Fatal(e)}
    source,_:=os.ReadFile(filepath.Join(folder,"default.xbe"))
    got,_:=os.ReadFile(out)
    if !bytes.Equal(source,got){t.Fatal("single extraction changed bytes")}
    if e:=ExtractImageMember(iso,"default.xbe",out);e==nil{t.Fatal("overwrote existing extracted file")}
}
func TestConfidenceIndexVersioningAndUpdates(t *testing.T){
    p:=strings.Repeat("a",64)
    now:=time.Now().UTC().Format(time.RFC3339)
    r:=ConfidenceReport{ID:"report_12345",PatchSHA256:p,TitleID:"4A580001",SourceCRC32:"AB12CD34",Environment:"hardware",Result:"working",Hardware:"Original Xbox",ReportedUTC:now}
    db:=ConfidenceDatabase{Schema:ConfidenceSchema,Revision:2,Reports:[]ConfidenceReport{r}}
    b,_:=json.Marshal(db)
    if _,e:=ParseConfidenceDatabase(b);e!=nil{t.Fatal(e)}
    root:=t.TempDir()
    server:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,req *http.Request){w.Write(b)}))
    defer server.Close()
    n,e:=SyncConfidenceCache(context.Background(),root,server.Client(),server.URL)
    if e!=nil||n!=1{t.Fatalf("confidence sync: %v count %d",e,n)}
    n,e=SyncConfidenceCache(context.Background(),root,server.Client(),server.URL)
    if e!=nil||n!=0{t.Fatalf("repeat sync should not claim new: %v count %d",e,n)}
    read,e:=LoadConfidenceCache(root)
    if e!=nil||len(ConfidenceForPatch(read,p,"4A580001","AB12CD34"))!=1{t.Fatal("missing hardware log after sync",e)}
    r.Environment="xemu";r.Hardware=""
    r.ID="second_34567"
    db.Reports=append(db.Reports,r);db.Revision=3
    b,_=json.Marshal(db)
    n,e=SyncConfidenceCache(context.Background(),root,server.Client(),server.URL)
    if e!=nil||n!=1{t.Fatalf("expected new emulation report: %v count %d",e,n)}
    report,_:=LoadConfidenceCache(root)
    matched:=ConfidenceForPatch(report,p,"4A580001","AB12CD34")
    if len(matched)!=2||matched[0].Environment==matched[1].Environment{t.Fatal("hardware and emulation records were collapsed")}
    bad:=[]byte(`{"schema":"magicpatchdesk-confidence-v1","revision":4,"reports":[{"id":"bad"}]}`)
    b=bad
    if _,e=SyncConfidenceCache(context.Background(),root,server.Client(),server.URL);e==nil{t.Fatal("bad confidence data accepted")}
    retained,_:=LoadConfidenceCache(root)
    if retained.Revision!=3{t.Fatal("invalid update destroyed previous confidence cache")}
    digest:=sha256.Sum256([]byte("x"))
    if len(hex.EncodeToString(digest[:]))!=64{t.Fatal("unexpected digest size")}
}
