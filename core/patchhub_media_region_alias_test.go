package core

import (
    "context"
    "fmt"
    "net/http"
    "strings"
    "testing"
)

// The owner supplied these exact, differently labelled sources. No upstream
// filenames or PatchHub media index entries need to change.
const badBoysRegionalMediaURL = "https://www.jayxbox.com/All-Patches/titles/Bad%20Boys%20-%20Miami%20Takedown/USA_00000001_454D0016/Bad%20Boys%20-%20Miami%20Takedown%20%7BWidescreen%7D%20%28NTSC%2BPAL%29%20%5BEsppiral%5D/1.%20Before.jpg"
const badBoysGithubPatchName = "Bad Boys - Miami Takedown {Widescreen} (NTSC) [Esppiral]"

func badBoysArtworkFixture(t *testing.T)(Game,CatalogPatch,HubScreenshot){
    t.Helper()
    shot,ok:=patchhubGalleryURL(badBoysRegionalMediaURL)
    if !ok{t.Fatal("provided website artwork URL rejected")}
    p:=CatalogPatch{Name:badBoysGithubPatchName,Game:"Bad Boys - Miami Takedown",
        Region:"NTSC",TitleID:"454D0016",Author:"Esppiral"}
    g:=Game{Title:p.Game,Patches:[]CatalogPatch{p}}
    return g,p,shot
}

func TestBadBoysRegionAliasWorksThroughEntireGalleryPipeline(t *testing.T){
    g,p,shot:=badBoysArtworkFixture(t)
    if shot.Game!="Bad Boys - Miami Takedown"||shot.TitleID!="454D0016"||
        shot.Patch!="Bad Boys - Miami Takedown {Widescreen} (NTSC+PAL) [Esppiral]"||
        shot.Kind!="before"{t.Fatalf("incorrect website identity: %+v",shot)}
    if !patchFolderRegionalAlias(p,shot){t.Fatal("NTSC+PAL must include NTSC")}
    manifest:=fmt.Sprintf(`{"schema":"patchhub-media-v1","images":[%q]}`,badBoysRegionalMediaURL)
    photos,err:=ParsePatchHubMediaManifest([]byte(manifest))
    if err!=nil||len(photos)!=1{t.Fatalf("site index rejected supplied photo: %+v %v",photos,err)}
    cached:=HubIndex{Games:[]HubGame{{Title:g.Title}}}
    AppendPatchHubMedia(&cached,photos)
    if len(cached.Games[0].Screenshots)!=1{t.Fatal("index not attributed to game")}
    AttachHubGallery(g.Patches,cached.Games[0].Screenshots)
    if len(g.Patches[0].Gallery)!=1||g.Patches[0].Gallery[0].Kind!="before"{
        t.Fatalf("catalogue failed to show valid comparison artwork: %+v",g.Patches[0].Gallery)
    }
    matched:=MatchPatchHubMedia(g,p,photos)
    if len(matched)!=1||matched[0].URL!=badBoysRegionalMediaURL{
        t.Fatalf("direct lookup failed with same valid index: %+v",matched)
    }
    fake:=&patchMediaTransport{manifest:manifest}
    found,e:=DiscoverPatchHubMedia(context.Background(),&http.Client{Transport:fake},g,p)
    if e!=nil||len(found)!=1||fake.requests!=1{
        t.Fatalf("manifest-first discovery should be one GET: %+v %v requests=%d",found,e,fake.requests)
    }
}

func TestRegionalAliasSafetyAndBroadRegionVariations(t *testing.T){
    _,p,shot:=badBoysArtworkFixture(t)
    cases:=[]struct{name string;want bool}{
        {"Bad Boys - Miami Takedown {Widescreen} (PAL) [Esppiral]",true},
        {"Bad Boys - Miami Takedown {Widescreen} (NTSC-U) [Esppiral]",true},
        {"Bad Boys - Miami Takedown {Widescreen} (JPN) [Esppiral]",false},
        {"Bad Boys - Miami Takedown {720p} (NTSC) [Esppiral]",false},
        {"Bad Boys - Miami Takedown {Widescreen} (NTSC) [Jay]",false},
        {"Other Game {Widescreen} (NTSC) [Esppiral]",false},
        {"Bad Boys - Miami Takedown {Widescreen} (NTSC+JPN) [Esppiral]",false},
        {"Bad Boys - Miami Takedown {Widescreen} (ALL REGIONS) [Esppiral]",false},
    }
    for _,tc:=range cases{
        candidate:=p
        candidate.Name=tc.name
        if got:=patchFolderRegionalAlias(candidate,shot);got!=tc.want{
            t.Errorf("%q compatible=%v want=%v",tc.name,got,tc.want)
        }
    }
    palOnly:=shot
    palOnly.Patch=strings.Replace(shot.Patch,"(NTSC+PAL)","(PAL)",1)
    if patchFolderRegionalAlias(p,palOnly){t.Fatal("PAL-only photos attached to NTSC")}
    jpnPAL:=shot
    jpnPAL.Patch=strings.Replace(shot.Patch,"(NTSC+PAL)","(JPN+PAL)",1)
    if patchFolderRegionalAlias(p,jpnPAL){t.Fatal("non-NTSC regional coverage attached to NTSC")}
    wrongID:=shot
    wrongID.TitleID="AAAAAAAA"
    if patchFolderRegionalAlias(p,wrongID){t.Fatal("Title ID conflict ignored")}
    if got:=MatchPatchHubMedia(Game{Title:"Other Game"},p,[]HubScreenshot{shot});len(got)!=0{
        t.Fatal("cross-game media accepted")
    }
}

func TestGalleryRegionAliasOwnershipIsDeterministic(t *testing.T){
    g,ntsc,shot:=badBoysArtworkFixture(t)
    pal:=ntsc
    pal.Name=strings.Replace(ntsc.Name,"(NTSC)","(PAL)",1)
    pal.Region="PAL"
    g.Patches=[]CatalogPatch{ntsc,pal}
    AttachHubGallery(g.Patches,[]HubScreenshot{shot})
    if len(g.Patches[0].Gallery)!=1||len(g.Patches[1].Gallery)!=1{
        t.Fatalf("explicit website dual-region photos should cover both variants: %+v",g.Patches)
    }
    if matched:=MatchPatchHubMedia(g,pal,[]HubScreenshot{shot});len(matched)!=1{
        t.Fatal("PAL variant not covered by website-declared NTSC+PAL")
    }
    exact:=ntsc
    exact.Name=shot.Patch
    exact.Region="NTSC+PAL"
    g.Patches=[]CatalogPatch{ntsc,pal,exact}
    owners:=galleryFolderOwners(g.Patches,shot,true)
    if len(owners)!=1||owners[0]!=2{t.Fatalf("exact folder did not take priority: %+v",owners)}
    if got:=MatchPatchHubMedia(g,ntsc,[]HubScreenshot{shot});len(got)!=0{
        t.Fatal("narrow regional alias stole an existing exact patch folder")
    }
    dup:=ntsc
    dup.File="distinct-patch.jmp"
    g.Patches=[]CatalogPatch{ntsc,dup}
    if owners:=galleryFolderOwners(g.Patches,shot,true);len(owners)!=0{
        t.Fatalf("duplicate regional candidates guessed ownership: %+v",owners)
    }
    mismatched:=ntsc
    mismatched.TitleID="AAAAAAAA"
    g.Patches=[]CatalogPatch{mismatched}
    AttachHubGallery(g.Patches,[]HubScreenshot{shot})
    if len(g.Patches[0].Gallery)!=0{t.Fatal("wrong Title ID got regional image")}
}
