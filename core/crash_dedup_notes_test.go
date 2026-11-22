package core

import (
    "errors"
    "os"
    "path/filepath"
    "strings"
    "testing"
)

func TestJMPNotesMarkupFullyReadable(t *testing.T){
    raw:=`Translation Mod.<br><br>1. Apply patch to default.xbe<br>2. Merge entire &quot;ThousandLand_Translation&quot; folder<br/>3. Enjoy!`
    got:=FormatJMPNotes(raw)
    want:="Translation Mod.\n\n1. Apply patch to default.xbe\n2. Merge entire \"ThousandLand_Translation\" folder\n3. Enjoy!"
    if got!=want{t.Fatalf("JMP instructions must preserve line breaks and not truncate:\n got: %q\nwant: %q",got,want)}
    htmlNotes:=`<p>Important <strong>steps</strong></p><ul><li>Backup files</li><li>Use <b>original</b> XBE</li></ul>`
    result:=FormatJMPNotes(htmlNotes)
    if !strings.Contains(result,"• Backup files")||!strings.Contains(result,"• Use original XBE")||
        strings.Contains(result,"<") {t.Fatalf("HTML list rendering not readable: %q",result)}
    if FormatJMPNotes(strings.Repeat("x",200000))==""{t.Fatal("long notes lost")}
    if strings.Contains(FormatJMPNotes("abc\x00def"),"\x00"){t.Fatal("embedded NUL will panic UTF16 writer")}
}
func TestSemanticJMPEquivalenceAndIdentities(t *testing.T){
    base:=fixtureJMP(fakeXBE())
    change:=strings.ReplaceAll(base,"notes=Sample","notes=New instructions<br>also changed")
    change=strings.ReplaceAll(change,"author=Tester","author=Another author")
    change="\ufeff"+strings.ReplaceAll(change,"\n","\r\n")
    if !EquivalentJMP([]byte(base),[]byte(change)){t.Fatal("equivalent patch edits not matched across author, notes, BOM and CRLF")}
    changedRecord:=strings.Replace(base,"0x391:11","0x391:22",1)
    if EquivalentJMP([]byte(base),[]byte(changedRecord)){t.Fatal("different patch bytes must not be identified as a duplicate")}
    changedRegion:=strings.Replace(base,"region=Global","region=Japan",1)
    if EquivalentJMP([]byte(base),[]byte(changedRegion)){t.Fatal("different game region wrongly treated as duplicate")}
}
func TestImportJMPRejectsSemanticDuplicateAndFolderDuplicates(t *testing.T){
    root:=t.TempDir()
    original:=writeImportFixture(t,filepath.Join(t.TempDir(),"patch.jmp"),fixtureJMP(fakeXBE()))
    installed,e:=ImportJMP(root,original)
    if e!=nil{t.Fatal(e)}
    differentMetadata:=strings.ReplaceAll(fixtureJMP(fakeXBE()),"notes=Sample","notes=Translation steps<br>Use original files")
    differentMetadata=strings.ReplaceAll(differentMetadata,"author=Tester","author=Extra credit")
    updated:=writeImportFixture(t,filepath.Join(t.TempDir(),"renamed.jmp"),differentMetadata)
    if _,e=ImportJMP(root,updated);!errors.Is(e,ErrDuplicatePatch){t.Fatalf("semantic duplicate was added: %v",e)}
    folder:=filepath.Join(t.TempDir(),"MyPatchHubPackage")
    writeImportFixture(t,filepath.Join(folder,"duplicate.jmp"),differentMetadata)
    writeImportFixture(t,filepath.Join(folder,"Before1.jpg"),"local screenshot remains importable")
    count,e:=ImportFolder(root,folder)
    if e!=nil{t.Fatal(e)}
    if count!=0{t.Fatalf("duplicate folder import added %d duplicate JMP files",count)}
    if _,e=os.Stat(filepath.Join(root,"Custom","MyPatchHubPackage","duplicate.jmp"));!errors.Is(e,os.ErrNotExist){
        t.Fatal("duplicate patch was copied by folder import")
    }
    if _,e=os.Stat(filepath.Join(root,"Custom","MyPatchHubPackage","Before1.jpg"));e!=nil{
        t.Fatal("related screenshot should not be dropped:",e)
    }
    before,_:=os.ReadFile(installed)
    if string(before)!=fixtureJMP(fakeXBE()){t.Fatal("folder import overwrote earlier JMP")}
}
func TestDurableCrashReportAfterForcedExit(t *testing.T){
    root:=t.TempDir()
    s,e:=BeginCrashSession(root,"v0.14.0-test.4",12345678)
    if e!=nil{t.Fatal(e)}
    if e=s.SetStage("read_xbe");e!=nil{t.Fatal(e)}
    if recovered,e:=RecoverAbandonedCrashSessions(root,func(pid int)bool{return true});e!=nil||recovered!=0{
        t.Fatalf("active session mistakenly considered a crash: %d %v",recovered,e)
    }
    recovered,e:=RecoverAbandonedCrashSessions(root,func(pid int)bool{return false})
    if e!=nil||recovered!=1{t.Fatalf("abandoned session not recorded: %d %v",recovered,e)}
    reports,e:=RecentFailureReports(root,10)
    if e!=nil||len(reports)!=1{t.Fatalf("no recovered report: %+v %v",reports,e)}
    report:=reports[0]
    if report.ErrorCode!="unexpected_exit"||!strings.Contains(report.Operation,"read_xbe"){
        t.Fatalf("report does not preserve failure category and safe stage: %+v",report)
    }
    if report.CrashEvidenceID!=s.ID(){t.Fatal("report not linked to local crash evidence")}
    private,e:=os.ReadFile(CrashEvidencePath(root,s.ID()))
    if e!=nil||!strings.Contains(string(private),"read_xbe"){t.Fatalf("human-readable crash log not generated: %v",e)}
    notices,e:=PendingCrashNotices(root)
    if e!=nil||notices!=1{t.Fatal("crash not acknowledged by app:",notices,e)}
    again,e:=RecoverAbandonedCrashSessions(root,func(int)bool{return false})
    if e!=nil||again!=0{t.Fatal("same crash reported twice")}
    if e=ClearCrashNotices(root);e!=nil{t.Fatal(e)}
    notices,_=PendingCrashNotices(root)
    if notices!=0{t.Fatal("crash notices not cleared on visit")}
}
func TestCaughtPanicProducesRedactedShareableReportAndPrivateTrace(t *testing.T){
    root:=t.TempDir()
    session,e:=BeginCrashSession(root,"v0.14.0-test.4",123)
    if e!=nil{t.Fatal(e)}
    if e=session.SetStage("parse_patch");e!=nil{t.Fatal(e)}
    secret:=`C:UsersPrivatePersonOriginalGame.xbe`
    if e=session.SaveCaughtCrash("panic from "+secret,[]byte("stack contains "+secret));e!=nil{t.Fatal(e)}
    b,e:=os.ReadFile(CrashEvidencePath(root,session.ID()))
    if e!=nil||!strings.Contains(string(b),secret){t.Fatal("private trace not preserved")}
    results,e:=RecentFailureReports(root,10)
    if e!=nil||len(results)!=1{t.Fatal("no diagnostic for caught panic",e)}
    report:=results[0]
    if report.ErrorCode!="unhandled_panic"||report.CrashEvidenceID!=session.ID(){t.Fatalf("bad panic metadata: %+v",report)}
    sanitized:=sanitizeReportForIssue(report)
    if strings.Contains(sanitized.Summary(),secret)||sanitized.CrashEvidenceID!=""{
        t.Fatal("private stack or raw-log reference leaked to shareable issue")
    }
    notices,_:=PendingCrashNotices(root)
    if notices!=1{t.Fatal("caught panic not flagged for next-launch notice")}
    recovered,e:=RecoverAbandonedCrashSessions(root,func(int)bool{return false})
    if e!=nil||recovered!=0{t.Fatal("caught panic also generated duplicate unclean-exit event")}
}
func TestCleanCrashSessionDoesNotCreateReport(t *testing.T){
    root:=t.TempDir()
    s,e:=BeginCrashSession(root,"v0.14.0-test.4",81234)
    if e!=nil{t.Fatal(e)}
    if e=s.End();e!=nil{t.Fatal(e)}
    n,e:=RecoverAbandonedCrashSessions(root,func(int)bool{return false})
    if e!=nil||n!=0{t.Fatal("clean shutdown was recorded as crash")}
}


func TestCatalogCollapsesPreExistingSemanticDuplicates(t *testing.T){
    root:=t.TempDir()
    upstream:=filepath.Join(root,"GitHub","titles","Halo 2","GLO_1_4D530064","Widescreen","Original.jmp")
    writeImportFixture(t,upstream,fixtureJMP(fakeXBE()))
    custom:=strings.ReplaceAll(fixtureJMP(fakeXBE()),"author=Tester","author=Translator")
    custom=strings.ReplaceAll(custom,"notes=Sample","notes=Use original game files<br><br>Then apply the patch")
    local:=filepath.Join(root,"Custom","titles","Halo 2","GLO_1_4D530064","Widescreen","Local version.jmp")
    writeImportFixture(t,local,custom)
    writeImportFixture(t,filepath.Join(filepath.Dir(local),"Before1.jpg"),"synthetic sample")
    writeImportFixture(t,filepath.Join(filepath.Dir(local),"After1.jpg"),"synthetic sample")
    catalog,e:=LoadCatalog(root,2)
    if e!=nil{t.Fatal(e)}
    if catalog.PatchCount!=1{t.Fatalf("duplicate JMP was still displayed: %d",catalog.PatchCount)}
    if len(catalog.Games)!=1||len(catalog.Games[0].Patches)!=1{
        t.Fatalf("catalog includes redundant entries: %+v",catalog.Games)
    }
    p:=catalog.Games[0].Patches[0]
    if p.Source!="GitHub"{t.Fatalf("unpredictably replaced upstream patch: %q",p.Source)}
    if len(p.Gallery)!=2{t.Fatalf("locally imported screenshots lost while deduplicating: %+v",p.Gallery)}
    if _,e=os.Stat(local);e!=nil{t.Fatal("preexisting duplicate was deleted:",e)}
}
