package core

import (
    "context"
    "fmt"
    "io"
    "net/http"
    "net/url"
    "strings"
    "testing"
)

const ProvidedPhantomCrashImage = "https://www.jayxbox.com/All-Patches/titles/Phantom%20Crash/USA_00000005_504C0001/Phantom%20Crash%20%7BWidescreen%7D%20%28NTSC%29%20%5BJay%5D/1.%20Before.jpg"
const ProvidedAMFBowlingImage = "https://www.jayxbox.com/All-Patches/titles/AMF%20Xtreme%20Bowling/USA_00000001_42530014/AMF%20Xtreme%20Bowling%20%7BWidescreen%7D%20%28NTSC%29%20%5BEsppiral%5D/1.%20Before.jpg"

func samplePhantomMediaGame() (Game,CatalogPatch) {
    g:=Game{Title:"Phantom Crash",Versions:[]HubVersion{
        {TitleID:"504C0001",Region:"USA",Version:"5"},
        {TitleID:"504C0001",Region:"PAL",Version:"4"},
    }}
    p:=CatalogPatch{Name:"Phantom Crash {Widescreen} (NTSC) [Jay]",TitleID:"504C0001",Region:"NTSC"}
    return g,p
}
func TestProvidedPhantomScreenshotAndNumberedImageNames(t *testing.T){
    shot,ok:=patchhubGalleryURL(ProvidedPhantomCrashImage)
    if !ok{t.Fatal("user-provided Phantom Crash photo was rejected")}
    if shot.Game!="Phantom Crash"||shot.Region!="USA"||shot.Version!="00000005"||
        shot.TitleID!="504C0001"||shot.Patch!="Phantom Crash {Widescreen} (NTSC) [Jay]"||
        shot.Kind!="before"||shot.Set!=1{
        t.Fatalf("incorrect URL identity or numbered photo parsing: %+v",shot)
    }
    folder:=strings.TrimSuffix(ProvidedPhantomCrashImage,"1.%20Before.jpg")
    samples:=[]struct{file,phase string;index int}{
        {"1. Before.jpg","before",1},
        {"2. After.png","after",2},
        {"BEFORE_1.PNG","before",1},
        {"after-11.jpeg","after",11},
        {"3-Before.JPG","before",3},
        {"4.After.gif","after",4},
        {"Alternate comparison.png","screenshot",0},
    }
    for _,c:=range samples{
        t.Run(c.file,func(t *testing.T){
            u:=folder+url.PathEscape(c.file)
            got,ok:=patchhubGalleryURL(u)
            if !ok||got.Kind!=c.phase||got.Set!=c.index{
                t.Fatalf("image name %s parsed incorrectly: %+v, ok=%t",c.file,got,ok)
            }
        })
    }
    if phase,_:=galleryPhase("Nightmare Before Christmas.png");phase!=""{
        t.Fatal("plain game title accidentally labelled as Before photo")
    }
}

func TestPatchHubPatchDirectoryMatchesRegionAndVersion(t *testing.T){
    g,p:=samplePhantomMediaGame()
    folder,err:=PatchHubPatchDirectory(g,p)
    if err!=nil{t.Fatal(err)}
    expected:=strings.TrimSuffix(ProvidedPhantomCrashImage,"1.%20Before.jpg")
    if folder!=expected{t.Fatalf("public folder does not match supplied URL\nactual: %q\nwanted: %q",folder,expected)}
    p.Region="PAL"
    other,err:=PatchHubPatchDirectory(g,p)
    if err!=nil||!strings.Contains(other,"PAL_00000004_504C0001"){
        t.Fatalf("PAL patch must remain separated from US images: %q %v",other,err)
    }
    p.Region="NTSC"
    g.Versions=append(g.Versions,HubVersion{TitleID:"504C0001",Region:"USA",Version:"6"})
    if _,err:=PatchHubPatchDirectory(g,p);err==nil{t.Fatal("ambiguous USA revisions were guessed")}
    p.TitleID="00000000"
    if _,err:=PatchHubPatchDirectory(g,p);err==nil{t.Fatal("missing title/version match accepted")}
}

func TestMediaManifestVerifiesExactPatchAssociation(t *testing.T){
    other:="https://www.jayxbox.com/All-Patches/titles/Other%20Game/USA_00000001_4A510002/Widescreen/Comparison.png"
    malicious:="https://bad.example/All-Patches/titles/Phantom%20Crash/USA_00000005_504C0001/Widescreen/Before.png"
    doc:=fmt.Sprintf(`{"schema":"%s","images":[%q,%q,%q,%q]}`,
        PatchHubMediaSchema,ProvidedPhantomCrashImage,ProvidedPhantomCrashImage,other,malicious)
    images,err:=ParsePatchHubMediaManifest([]byte(doc))
    if err!=nil||len(images)!=2{t.Fatalf("invalid/duplicate media not filtered: %+v %v",images,err)}
    if _,err:=ParsePatchHubMediaManifest([]byte(`{"schema":"old","images":[]}`));err==nil{
        t.Fatal("unexpected image-index schema accepted")
    }
    if _,err:=ParsePatchHubMediaManifest([]byte(strings.Repeat("x",8<<20+1)));err==nil{
        t.Fatal("oversized media index was accepted")
    }
    hub:=HubIndex{Games:[]HubGame{{Title:"Phantom Crash",Versions:[]HubVersion{
        {TitleID:"504C0001",Region:"USA",Version:"5"},
        {TitleID:"504C0001",Region:"PAL",Version:"4"},
    }}}}
    AppendPatchHubMedia(&hub,images)
    if len(hub.Games[0].Screenshots)!=1{
        t.Fatalf("media for another game slipped into Phantom Crash: %+v",hub.Games[0].Screenshots)
    }
    g,p:=samplePhantomMediaGame()
    _=g
    patches:=[]CatalogPatch{p,{Name:"Phantom Crash PAL widescreen",TitleID:"504C0001",Region:"PAL"}}
    AttachHubGallery(patches,hub.Games[0].Screenshots)
    if len(patches[0].Gallery)!=1||len(patches[1].Gallery)!=0{
        t.Fatalf("media attached to the wrong JMP patch: %+v",patches)
    }
}

func TestDirectoryListingExtractsArbitraryImageNamesSafely(t *testing.T){
    g,p:=samplePhantomMediaGame()
    folder,_:=PatchHubPatchDirectory(g,p)
    listing:=`<html><a href="1.%20Before.jpg">before</a>
       <a href="2.%20After.png">after</a>
       <a href="Unusual%20Photo.PNG">unknown photo</a>
       <a href="../../other.png">escape</a>
       <a href="https://hostile.invalid/evil.png">external</a>
       <a href="/All-Patches/titles/Some%20Other%20Game.jpg">wrong folder</a>
       <a href="example.txt">non-image</a>
       <a href="1.%20Before.jpg">duplicate</a></html>`
    shots:=MediaLinksFromDirectoryHTML(folder,[]byte(listing))
    if len(shots)!=3{t.Fatalf("expected exactly 3 trusted images, got %+v",shots)}
    if shots[0].Kind!="before"||shots[1].Kind!="after"||shots[2].Kind!="screenshot"{
        t.Fatalf("unexpected phase labels: %+v",shots)
    }
}
type patchMediaTransport struct{
    requests int
    listing string
    images map[string]bool
}
func (f *patchMediaTransport) RoundTrip(req *http.Request)(*http.Response,error){
    f.requests++
    status:=http.StatusNotFound
    mime:="text/plain"
    body:=""
    if req.Method=="GET"&&strings.HasSuffix(req.URL.EscapedPath(),"/"){
        if f.listing!=""{status=200;mime="text/html";body=f.listing}else{status=403}
    }
    if req.Method=="HEAD"&&f.images[req.URL.String()]{status=200;mime="image/jpeg"}
    return &http.Response{
        StatusCode:status,Status:fmt.Sprintf("%d",status),
        Header:http.Header{"Content-Type":[]string{mime}},
        ContentLength:int64(len(body)),
        Body:io.NopCloser(strings.NewReader(body)),
        Request:req,
    },nil
}
func TestBoundedPatchMediaDiscoveryFindsProvidedNumberedPatterns(t *testing.T){
    g,p:=samplePhantomMediaGame()
    folder,_:=PatchHubPatchDirectory(g,p)
    fake:=&patchMediaTransport{images:map[string]bool{
        ProvidedPhantomCrashImage:true,
        folder+"2.%20After.png":true,
    }}
    client:=&http.Client{Transport:fake}
    shots,err:=DiscoverPatchHubMedia(context.Background(),client,g,p)
    if err!=nil{t.Fatal(err)}
    if len(shots)!=2||shots[0].Kind!="before"||shots[1].Kind!="after"{
        t.Fatalf("bounded filename probe did not find both actual conventions: %+v",shots)
    }
    if fake.requests>2+len(CommonPatchHubMediaNames()){
        t.Fatalf("discovery became an unbounded server crawler: %d",fake.requests)
    }
    fake=&patchMediaTransport{listing:`<a href="Gallery%20Example.png">image</a>`}
    shots,err=DiscoverPatchHubMedia(context.Background(),&http.Client{Transport:fake},g,p)
    if err!=nil||len(shots)!=1||shots[0].Kind!="screenshot"||fake.requests!=2{
        t.Fatalf("directory listing should discover custom names without HEAD guessing: %+v %v %d",shots,err,fake.requests)
    }
}

func TestGalleryURLRejectsUnsafeFolders(t *testing.T){
    for _,raw:=range []string{
        "https://www.jayxbox.com/All-Patches/titles/Halo/USA_00000001_4D530064/Before.png",
        "/All-Patches/titles/Halo/USA_00000001_4D530064/Wide/../../Before.png",
        "/All-Patches/titles/Halo/USA_00000001_4D530064/Wide/After.png?x=1",
        "http://www.jayxbox.com/All-Patches/titles/Halo/USA_00000001_4D530064/Wide/After.png",
        "//evil.invalid/All-Patches/titles/Halo/USA_00000001_4D530064/Wide/After.png",
    }{
        if shot,ok:=patchhubGalleryURL(raw);ok{
            t.Fatalf("untrusted folder/media accepted: %s %+v",raw,shot)
        }
    }
}

func TestOwnerProvidedAMFBowlingMediaIdentity(t *testing.T) {
    shot,ok:=patchhubGalleryURL(ProvidedAMFBowlingImage)
    if !ok||shot.Game!="AMF Xtreme Bowling"||shot.Version!="00000001"||
        shot.TitleID!="42530014"||shot.Region!="USA"||
        shot.Patch!="AMF Xtreme Bowling {Widescreen} (NTSC) [Esppiral]"||
        shot.Kind!="before"||shot.Set!=1 {
        t.Fatalf("AMF image identity invalid: %+v (ok=%t)",shot,ok)
    }
    // Actual image filenames may be arbitrarily named. They are obtained
    // through the published manifest, not a special-case game table.
    manifest:=fmt.Sprintf(`{"schema":"patchhub-media-v1","images":[%q]}`,ProvidedAMFBowlingImage)
    shots,err:=ParsePatchHubMediaManifest([]byte(manifest))
    if err!=nil||len(shots)!=1||shots[0].URL!=ProvidedAMFBowlingImage {
        t.Fatalf("AMF image was not discoverable through the generic manifest: %+v %v",shots,err)
    }
}
