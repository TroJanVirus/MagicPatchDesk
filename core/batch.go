package core

import (
    "context"
    "crypto/sha256"
    "errors"
    "fmt"
    "os"
    "path/filepath"
    "strings"
)

type BatchDecision string
const (
    BatchAuto BatchDecision="auto"
    BatchReview BatchDecision="review"
    BatchSkip BatchDecision="skip"
)
type BatchJob struct {
    Game LibraryGame
    Decision BatchDecision
    PatchFile string
    PatchSHA256 string
    OutputMask uint32
}
type BatchSummary struct {
    Completed int
    Skipped int
    Manual int
    Failed int
    Outputs []string
}
// VerifyBatchJob refuses to treat an ambiguous game as unattended-safe.
// A path can never be inferred from only a similar game name.
func VerifyBatchJob(job BatchJob)error{
    if job.Decision==BatchSkip || job.Decision==BatchReview{return nil}
    if job.Decision!=BatchAuto{return errors.New("choose auto, review or skip")}
    if job.Game.Status!="ready" || job.Game.TitleID=="" || job.Game.CRC32=="" {
        return errors.New("game is not verified for unattended patching")
    }
    eligible:=0
    for _,p:=range job.Game.Patches{
        if p.Status=="exact" && p.File==job.PatchFile && p.SHA256==job.PatchSHA256 && job.PatchSHA256!=""{eligible++}
    }
    if eligible!=1{return errors.New("choose exactly one original-CRC-verified JMP")}
    if job.OutputMask==0 || job.OutputMask&^GameOutputAll!=0{return errors.New("invalid batch output format")}
    if job.Game.Kind=="cci"{return errors.New("CCI games require source identity review before unattended patching")}
    if job.OutputMask&GameOutputISO!=0&&job.Game.Kind!="fulliso"{return errors.New("full-disc ISO output needs an original full-disc ISO")}
    return nil
}
func batchOriginalXBE(g LibraryGame)([]byte,error){
    if g.Kind=="folder"{return ReadGameMember(g.Path,"default.xbe")}
    im,e:=OpenXISO(g.Path)
    if e!=nil{return nil,e}
    member,e:=im.DefaultXBE()
    if e!=nil{return nil,e}
    if member.Size>64<<20{return nil,errors.New("batch XBE exceeds 64MiB limit")}
    return im.ReadFile(member)
}
func safeBatchName(g LibraryGame,order int)string{
    s:=strings.TrimSuffix(filepath.Base(g.Path),filepath.Ext(g.Path))
    if g.Kind=="folder"{s=filepath.Base(g.Path)}
    var b strings.Builder
    for _,r:=range s{
        if r>='a'&&r<='z'||r>='A'&&r<='Z'||r>='0'&&r<='9'||r=='_'||r=='-' {b.WriteRune(r)}
        if b.Len()>=64{break}
    }
    if b.Len()==0{s="Game"}else{s=b.String()}
    return fmt.Sprintf("%04d_%s_patched",order+1,s)
}
// RunBatchJobs makes distinct output packages. Previous packages and every
// source file remain untouched. Complex, ambiguous and skipped games receive
// no writes; failures do not abort later independently safe jobs.
func RunBatchJobs(ctx context.Context,sourceRoot,outRoot string,jobs []BatchJob,log *VerificationLog)(BatchSummary,error){
    var summary BatchSummary
    if len(jobs)>10000{return summary,errors.New("batch exceeds 10,000 games")}
    a,e:=filepath.Abs(sourceRoot);if e!=nil{return summary,e}
    b,e:=filepath.Abs(outRoot);if e!=nil{return summary,e}
    if strings.EqualFold(a,b)||strings.HasPrefix(strings.ToLower(b),strings.ToLower(a)+string(os.PathSeparator)){
        return summary,errors.New("batch output must be outside the scanned game library")
    }
    if e=os.MkdirAll(outRoot,0755);e!=nil{return summary,e}
    for i,job:=range jobs{
        if ctx!=nil&&ctx.Err()!=nil{return summary,ctx.Err()}
        label:=job.Game.Title
        if label==""{label=filepath.Base(job.Game.Path)}
        if job.Decision==BatchSkip{summary.Skipped++;log.Add("info","Skip",label);continue}
        if job.Decision==BatchReview{summary.Manual++;log.Add("warning","Manual approval",label+": "+job.Game.Reason);continue}
        if e:=VerifyBatchJob(job);e!=nil{
            summary.Failed++;log.Add("error","Preflight",label+": "+e.Error());continue
        }
        original,e:=batchOriginalXBE(job.Game)
        if e==nil{
            var info XBEInfo
            info,e=InspectXBE(original)
            if e==nil && (!strings.EqualFold(info.TitleID,job.Game.TitleID)||!strings.EqualFold(info.CRC32,job.Game.CRC32)){
                e=errors.New("source changed after library scan; re-scan before patching")
            }
        }
        if e!=nil{summary.Failed++;log.Add("error","Identity",label+": "+e.Error());continue}
        b,e:=os.ReadFile(job.PatchFile)
        if e!=nil{summary.Failed++;log.Add("error","Patch",label+": "+e.Error());continue}
        digest:=sha256.Sum256(b)
        if job.PatchSHA256!=fmt.Sprintf("%x",digest){
            summary.Failed++
            log.Add("error","Patch",label+": JMP changed since library scan; rescan required")
            continue
        }
        patch,e:=ParseJMP(filepath.Base(job.PatchFile),b)
        if e==nil{e=patch.CheckSingleFileOnly()}
        if e==nil{
            var output []byte
            output,e=patch.Apply(original,false)
            if e==nil {
                out:=filepath.Join(outRoot,safeBatchName(job.Game,i))
                log.Add("info","Export",label+": creating independent copy")
                e=ExportGameOutputs(job.Game.Path,out,job.OutputMask,map[string][]byte{"default.xbe":output})
                if e==nil{
                    summary.Completed++
                    summary.Outputs=append(summary.Outputs,out)
                    log.Add("pass","Verified",label+": all selected outputs created and verified")
                }
            }
        }
        if e!=nil{summary.Failed++;log.Add("error","Patch",label+": "+e.Error())}
    }
    return summary,nil
}
