package core

import (
    "bytes"
    "context"
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "errors"
    "fmt"
    "io"
    "net/http"
    "os"
    "path/filepath"
    "regexp"
    "sort"
    "strings"
    "time"
)

const ConfidenceSchema = "magicpatchdesk-confidence-v1"
const ConfidenceURL = "https://raw.githubusercontent.com/TroJanVirus/MagicPatchDesk/main/confidence/reports.v1.json"
var confidenceID = regexp.MustCompile(`^[a-z0-9_-]{8,64}$`)
var confidenceSHA = regexp.MustCompile(`^[0-9a-f]{64}$`)

// ConfidenceReport is a reviewable field-test event, not a numeric score.
// Curated reports are stored in the public repository, but issue submissions
// are untrusted until reviewed and deliberately imported.
type ConfidenceReport struct {
    ID string `json:"id"`
    PatchSHA256 string `json:"patch_sha256"`
    TitleID string `json:"title_id"`
    SourceCRC32 string `json:"source_crc32"`
    Environment string `json:"environment"` // xemu or hardware
    Result string `json:"result"` // working, partial or failed
    Hardware string `json:"hardware,omitempty"`
    Note string `json:"note,omitempty"`
    ReportedUTC string `json:"reported_utc"`
    SourceIssue int `json:"source_issue,omitempty"`
}
type ConfidenceDatabase struct {
    Schema string `json:"schema"`
    Revision int `json:"revision"`
    Reports []ConfidenceReport `json:"reports"`
}
func validateConfidenceReport(r ConfidenceReport)error{
    if !confidenceID.MatchString(r.ID){return errors.New("invalid confidence report ID")}
    if !confidenceSHA.MatchString(r.PatchSHA256){return errors.New("confidence report patch SHA256 must be 64 lowercase hexadecimal digits")}
    if !diagnosticTitleIDRE.MatchString(r.TitleID)||!diagnosticCRC32RE.MatchString(r.SourceCRC32){return errors.New("confidence report needs verified Title ID and source CRC32")}
    if r.Environment!="xemu" && r.Environment!="hardware"{return errors.New("confidence environment must be xemu or hardware")}
    if r.Result!="working" && r.Result!="partial" && r.Result!="failed"{return errors.New("invalid confidence result")}
    if len(r.Hardware)>80||len(r.Note)>300||strings.ContainsAny(r.Note,"\r\n\x00")||strings.ContainsAny(r.Hardware,"\r\n\x00"){return errors.New("confidence text contains unsupported characters or exceeds limit")}
    if r.Environment=="hardware" && strings.TrimSpace(r.Hardware)==""{return errors.New("hardware test must identify original Xbox console type")}
    t,e:=time.Parse(time.RFC3339,r.ReportedUTC)
    if e!=nil||t.After(time.Now().UTC().Add(48*time.Hour)){return errors.New("invalid confidence report timestamp")}
    if r.SourceIssue<0{return errors.New("invalid confidence source issue number")}
    return nil
}
func ParseConfidenceDatabase(data []byte)(ConfidenceDatabase,error){
    var db ConfidenceDatabase
    if len(data)>2<<20{return db,errors.New("confidence index exceeds 2MiB")}
    decoder:=json.NewDecoder(bytes.NewReader(data))
    decoder.DisallowUnknownFields()
    if err:=decoder.Decode(&db);err!=nil{return db,err}
    var extra any
    if decoder.Decode(&extra)!=io.EOF{return ConfidenceDatabase{},errors.New("confidence index contains trailing data")}
    if db.Schema!=ConfidenceSchema||db.Revision<1||len(db.Reports)>2000{return ConfidenceDatabase{},errors.New("invalid confidence index schema or size")}
    seen:=map[string]bool{}
    for _,report:=range db.Reports{
        if e:=validateConfidenceReport(report);e!=nil{return ConfidenceDatabase{},fmt.Errorf("confidence %s: %w",report.ID,e)}
        if seen[report.ID]{return ConfidenceDatabase{},errors.New("duplicate confidence report ID")}
        seen[report.ID]=true
    }
    return db,nil
}
func ConfidenceFileHash(path string)(string,error){
    info,e:=os.Lstat(path)
    if e!=nil{return "",e}
    if !info.Mode().IsRegular() || info.Size()>32<<20{return "",errors.New("patch must be a regular file smaller than 32MiB")}
    f,e:=os.Open(path)
    if e!=nil{return "",e}
    defer f.Close()
    sum:=sha256.New()
    count,e:=io.Copy(sum,io.LimitReader(f,info.Size()+1))
    if e!=nil{return "",e}
    if count!=info.Size(){return "",errors.New("patch changed while hashing")}
    return hex.EncodeToString(sum.Sum(nil)),nil
}
func ConfidenceForPatch(db ConfidenceDatabase,digest,titleID,crc string)[]ConfidenceReport{
    out:=[]ConfidenceReport{}
    for _,r:=range db.Reports{
        if r.PatchSHA256==digest && strings.EqualFold(r.TitleID,titleID) && strings.EqualFold(r.SourceCRC32,crc) {
            out=append(out,r)
        }
    }
    sort.SliceStable(out,func(i,j int)bool{return out[i].ReportedUTC>out[j].ReportedUTC})
    return out
}
func ConfidenceCachePath(root string)string{return filepath.Join(root,"confidence-reports.v1.json")}
func LoadConfidenceCache(root string)(ConfidenceDatabase,error){
    b,e:=os.ReadFile(ConfidenceCachePath(root))
    if errors.Is(e,os.ErrNotExist){return ConfidenceDatabase{Schema:ConfidenceSchema,Revision:1},nil}
    if e!=nil{return ConfidenceDatabase{},e}
    return ParseConfidenceDatabase(b)
}
// SyncConfidenceCache retrieves only the pinned public repository's reviewed
// JSON file. New reports are identified by stable IDs. Invalid remote data
// must not replace a previously valid local copy. User game files never leave.
func SyncConfidenceCache(ctx context.Context,root string,client *http.Client,endpoint string)(int,error){
    if endpoint==""{endpoint=ConfidenceURL}
    if client==nil{client=&http.Client{Timeout:15*time.Second}}
    req,e:=http.NewRequestWithContext(ctx,"GET",endpoint,nil)
    if e!=nil{return 0,e}
    req.Header.Set("User-Agent","MagicPatchDesk/"+AppVersion)
    resp,e:=client.Do(req)
    if e!=nil{return 0,e}
    defer resp.Body.Close()
    if resp.StatusCode!=http.StatusOK{return 0,fmt.Errorf("confidence index unavailable: HTTP %d",resp.StatusCode)}
    b,e:=io.ReadAll(io.LimitReader(resp.Body,(2<<20)+1))
    if e!=nil{return 0,e}
    db,e:=ParseConfidenceDatabase(b)
    if e!=nil{return 0,e}
    previous,e:=LoadConfidenceCache(root)
    if e!=nil{return 0,fmt.Errorf("retained existing confidence cache: %w",e)}
    if db.Revision<previous.Revision{return 0,errors.New("confidence index revision rolled back; cache retained")}
    old:=map[string]bool{}
    for _,report:=range previous.Reports{old[report.ID]=true}
    count:=0
    for _,report:=range db.Reports{if !old[report.ID]{count++}}
    if e=os.MkdirAll(root,0700);e!=nil{return 0,e}
    temp,e:=os.CreateTemp(root,".mpd-confidence-*.tmp")
    if e!=nil{return 0,e}
    defer os.Remove(temp.Name())
    if _,e=temp.Write(b);e!=nil{temp.Close();return 0,e}
    if e=temp.Sync();e!=nil{temp.Close();return 0,e}
    if e=temp.Close();e!=nil{return 0,e}
    if e=os.Rename(temp.Name(),ConfidenceCachePath(root));e!=nil{return 0,e}
    return count,nil
}

// ConfidenceIssueBody is a structured, opt-in submission template suitable for
// GitHub Issues. There is never an automatic upload of a success claim.
func ConfidenceIssueBody(r ConfidenceReport)(string,error){
    if e:=validateConfidenceReport(r);e!=nil{return "",e}
    body:=fmt.Sprintf("### Patch compatibility field report (awaiting review)\n\nPatch SHA256: %s\nTitle ID: %s\nOriginal XBE CRC32: %s\nEnvironment: %s\nResult: %s\nHardware: %s\nReported UTC: %s\n",r.PatchSHA256,r.TitleID,r.SourceCRC32,r.Environment,r.Result,r.Hardware,r.ReportedUTC)
    if r.Note!=""{body+="Notes: "+r.Note+"\n"}
    return body+"\nOnly reviewed reports are imported into confidence/reports.v1.json. No game bytes, local paths or patch payloads are included.\n",nil
}
