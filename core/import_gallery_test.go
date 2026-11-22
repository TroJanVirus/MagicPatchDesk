package core

import (
    "errors"
    "fmt"
    "os"
    "path/filepath"
    "strings"
    "testing"
)

func writeImportFixture(t *testing.T,filename,body string)string{
    t.Helper()
    if e:=os.MkdirAll(filepath.Dir(filename),0755);e!=nil{t.Fatal(e)}
    if e:=os.WriteFile(filename,[]byte(body),0644);e!=nil{t.Fatal(e)}
    return filename
}
func TestImportJMPExactDuplicateAcrossLibraryAndNames(t *testing.T){
    root:=t.TempDir()
    src:=writeImportFixture(t,filepath.Join(t.TempDir(),"Halo Wide.jmp"),fixtureJMP(fakeXBE()))
    installed,e:=ImportJMP(root,src)
    if e!=nil{t.Fatal(e)}
    initial,_:=os.ReadFile(installed)
    if _,e=ImportJMP(root,src);!errors.Is(e,ErrDuplicatePatch){t.Fatalf("duplicate import not detected: %v",e)}
    alias:=writeImportFixture(t,filepath.Join(t.TempDir(),"Different Filename.JMP"),string(initial))
    if _,e=ImportJMP(root,alias);!errors.Is(e,ErrDuplicatePatch){t.Fatalf("same patch different name not detected: %v",e)}
    var d *DuplicatePatchError
    if !errors.As(e,&d)||d.Existing==""||!strings.Contains(e.Error(),"No need to add"){
        t.Fatalf("duplicate error does not identify existing file and friendly message: %v",e)
    }
    entries,e:=os.ReadDir(filepath.Join(root,"Custom"))
    if e!=nil||len(entries)!=1{t.Fatalf("duplicate created another patch: %d %v",len(entries),e)}
    after,_:=os.ReadFile(installed)
    if string(initial)!=string(after){t.Fatal("duplicate import modified original cached patch")}
}
func TestImportJMPDetectsGitHubDuplicateWithoutCustomWrites(t *testing.T){
    root:=t.TempDir()
    incoming:=fixtureJMP(fakeXBE())
    writeImportFixture(t,filepath.Join(root,"GitHub","Other author","Some Patch.jmp"),incoming)
    source:=writeImportFixture(t,filepath.Join(t.TempDir(),"Another Title.JMP"),incoming)
    if _,e:=ImportJMP(root,source);!errors.Is(e,ErrDuplicatePatch){t.Fatalf("existing GitHub patch not detected: %v",e)}
    if _,e:=os.Stat(filepath.Join(root,"Custom"));!errors.Is(e,os.ErrNotExist){
        t.Fatal("duplicate import created Custom folder unnecessarily")
    }
}
func TestImportJMPNameCollisionDoesNotOverwriteOtherPatch(t *testing.T){
    root:=t.TempDir()
    base:=fixtureJMP(fakeXBE())
    original:=writeImportFixture(t,filepath.Join(t.TempDir(),"Patch.jmp"),base)
    first,e:=ImportJMP(root,original)
    if e!=nil{t.Fatal(e)}
    revision:=strings.Replace(base,"notes=Sample","notes=Independent revision",1)
    if revision==base{t.Fatal("failed to build distinct test revision")}
    source2:=writeImportFixture(t,filepath.Join(t.TempDir(),"Patch.jmp"),revision)
    second,e:=ImportJMP(root,source2)
    if e!=nil{t.Fatal(e)}
    if second==first||!strings.Contains(second,"(import 2)") {
        t.Fatalf("distinct same-named patch overwrote original: %q / %q",first,second)
    }
    old,_:=os.ReadFile(first)
    changed,_:=os.ReadFile(second)
    if string(old)!=base || string(changed)!=revision{t.Fatal("file collision changed patch contents")}
    _,e=ImportJMP(root,source2)
    if !errors.Is(e,ErrDuplicatePatch){t.Fatal("second revision no longer detected as duplicate")}
}
func TestImportJMPRejectsNonPatchAndSymlink(t *testing.T){
    root:=t.TempDir()
    bad:=writeImportFixture(t,filepath.Join(t.TempDir(),"not-jmp.txt"),fixtureJMP(fakeXBE()))
    if _,e:=ImportJMP(root,bad);e==nil{t.Fatal("invalid file extension imported")}
    invalid:=writeImportFixture(t,filepath.Join(t.TempDir(),"broken.jmp"),"hello")
    if _,e:=ImportJMP(root,invalid);e==nil{t.Fatal("invalid JMP imported")}
    // Portable check: where symlinks are permitted, a symlink import must
    // be rejected even if it points to valid patch bytes.
    good:=writeImportFixture(t,filepath.Join(t.TempDir(),"real.jmp"),fixtureJMP(fakeXBE()))
    link:=filepath.Join(t.TempDir(),"link.jmp")
    if e:=os.Symlink(good,link);e==nil{
        if _,e=ImportJMP(root,link);e==nil{t.Fatal("symlink import accepted")}
    }
}
func TestPatchHubMultipleScreenshotExtraction(t *testing.T){
    fragment:=`<div class="game-card"><div class="game-title">Halo 2</div>
      <img data-src="/All-Patches/titles/Halo%202/4D530064.jpg">
      <a href="/All-Patches/titles/Halo%202/GLO_1_4D530064/720p%20Patch/Before_1.png">A</a>
      <a href="/All-Patches/titles/Halo%202/GLO_1_4D530064/720p%20Patch/After_1.png">B</a>
      <a href="/All-Patches/titles/Halo%202/GLO_1_4D530064/720p%20Patch/Before2.jpg">C</a>
      <a href="/All-Patches/titles/Halo%202/GLO_1_4D530064/720p%20Patch/After2.jpg">D</a>
      <img src="https://evil.example/All-Patches/titles/Halo%202/GLO_1_4D530064/720p%20Patch/Before3.jpg">
      <a href="/All-Patches/titles/Halo%202/GLO_1_4D530064/720p%20Patch/Before_1.png">duplicate</a>
      <div>Title ID: 4D530064 | Region: GLO | Version: 1</div></div>`
    var page strings.Builder
    page.WriteString(fragment)
    for i:=0;i<50;i++{
        fmt.Fprintf(&page,`<div class="game-card"><div class="game-title">Other Game %d</div><img data-src="/All-Patches/titles/Other/game.jpg"></div>`,i)
    }
    hub,e:=ParsePatchHubHTML([]byte(page.String()))
    if e!=nil{t.Fatal(e)}
    if len(hub.Games)!=51{t.Fatalf("wrong game count %d",len(hub.Games))}
    gallery:=hub.Games[0].Screenshots
    if len(gallery)!=4{t.Fatalf("wanted four unique trusted screenshots, got %+v",gallery)}
    patches:=[]CatalogPatch{
       {Name:"Halo 2 720p Patch",TitleID:"4D530064"},
       {Name:"Halo 2 480p Fix",TitleID:"4D530064"},
    }
    AttachHubGallery(patches,gallery)
    if len(patches[0].Gallery)!=4||len(patches[1].Gallery)!=0 {
        t.Fatalf("screenshot attached to wrong patch: %+v",patches)
    }
    for i:=0;i<4;i++{
        expected:="before"
        if i%2==1{expected="after"}
        if patches[0].Gallery[i].Kind!=expected {t.Fatalf("screenshots not paired in order: %+v",patches[0].Gallery)}
    }
}
func TestPatchHubGalleryRejectsUnsafeAndAmbiguousPhotos(t *testing.T){
    for _,bad:=range []string{
        "https://example.com/All-Patches/titles/Halo/GLO_1_4D530064/Wide/Before.png",
        "http://www.jayxbox.com/All-Patches/titles/Halo/GLO_1_4D530064/Wide/Before.png",
        "https://www.jayxbox.com/Other/titles/Halo/GLO_1_4D530064/Wide/Before.png",
        "/All-Patches/titles/Halo/GLO_1_4D530064/Wide/before.svg",
        "/All-Patches/titles/Halo/GLO_1_4D530064/Wide/After.png?redirect=1",
        "/All-Patches/titles/Halo/GLO_1_4D530064/Wide/Before.png#fragment",
        "/All-Patches/titles/Halo/GLO_1_4D530064/Wide/Normal.png",
    }{
        if shot,ok:=patchhubGalleryURL(bad);ok{t.Fatalf("accepted unsafe or unlabelled screenshot %s: %+v",bad,shot)}
    }
    good,ok:=patchhubGalleryURL("/All-Patches/titles/Halo/GLO_1_4D530064/Widescreen/Before.png")
    if !ok{t.Fatal("trusted screenshot ignored")}
    ambiguous:=[]CatalogPatch{
        {Name:"Widescreen patch A",TitleID:"4D530064"},
        {Name:"Widescreen patch B",TitleID:"4D530064"},
    }
    AttachHubGallery(ambiguous,[]HubScreenshot{good})
    if len(ambiguous[0].Gallery)+len(ambiguous[1].Gallery)!=0 {
        t.Fatal("ambiguous screenshot was guessed for a wrong patch")
    }
}
func TestLocalPatchHubGalleryMultiplePairs(t *testing.T){
    folder:=filepath.Join(t.TempDir(),"Custom","titles","Halo 2","GLO_1_4D530064","720p Patch")
    patch:=writeImportFixture(t,filepath.Join(folder,"Halo 2 720p Patch.jmp"),fixtureJMP(fakeXBE()))
    for _,name:=range []string{"Before1.png","After1.png","Before2.jpg","After2.jpg","cover.jpg"} {
        writeImportFixture(t,filepath.Join(folder,name),"not-needed-for-name-indexing")
    }
    photos:=LocalPatchScreenshots(patch)
    if len(photos)!=4{t.Fatalf("expected four local screenshots; got %+v",photos)}
    if photos[0].Kind!="before"||photos[1].Kind!="after"||photos[2].Set!=2 {
        t.Fatalf("local photos not arranged in paired order: %+v",photos)
    }
    other:=writeImportFixture(t,filepath.Join(t.TempDir(),"Custom","OtherPatch.jmp"),fixtureJMP(fakeXBE()))
    if shots:=LocalPatchScreenshots(other);len(shots)!=0{t.Fatal("non PatchHub folder guessed screenshot ownership")}
}
