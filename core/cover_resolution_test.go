package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCoverCandidatesPrototypeAndFallbacks(t *testing.T) {
	g := Game{Title: "American Idol (Prototype)",
		Patches: []CatalogPatch{{TitleID: "434d0027"}},
		Versions: []HubVersion{{TitleID: "434D0027"}}}
	got := CoverCandidates(g)
	want := "https://www.jayxbox.com/All-Patches/titles/American%20Idol%20%28Prototype%29/434D0027.jpg"
	if len(got) != 2 || got[0] != want || !strings.HasSuffix(got[1], "/434D0027.png") {
		t.Fatalf("prototype fallback mismatch: %#v", got)
	}
}

func TestCoverCandidatesPreferOfficialURLAndHubFolder(t *testing.T) {
	g := Game{Title: "Amped 2", CoverFolder: "Amped 2 (Tenku 2)",
		CoverURL: "https://www.jayxbox.com/exact.jpg",
		Versions: []HubVersion{{TitleID: "4D530041"}}}
	got := CoverCandidates(g)
	if len(got) != 3 || got[0] != g.CoverURL || !strings.Contains(got[1], "Amped%202%20%28Tenku%202%29/4D530041.jpg") {
		t.Fatalf("expected official URL then Hub's exact filename: %#v", got)
	}
}

func TestCoverCandidatesRejectAmbiguousOrUnsafeData(t *testing.T) {
	for _, g := range []Game{
		{Title: "../wrong", Versions: []HubVersion{{TitleID: "434D0027"}}},
		{Title: "some/game", Versions: []HubVersion{{TitleID: "434D0027"}}},
		{Title: "Game", Versions: []HubVersion{{TitleID: "00000000"}}},
		{Title: "Game", Versions: []HubVersion{{TitleID: "434D002Z"}}},
		{Title: "Game", Versions: []HubVersion{{TitleID: "00000001"}, {TitleID: "00000002"}, {TitleID: "00000003"}, {TitleID: "00000004"}}},
	} {
		if got := CoverCandidates(g); len(got) != 0 {
			t.Fatalf("unsafe fallback: %#v for %+v", got, g)
		}
	}
}

func TestCoverCandidatesPreserveOfficialURLWithoutID(t *testing.T) {
	g := Game{Title: "Game", CoverURL: "https://www.jayxbox.com/All-Patches/known.jpg"}
	got := CoverCandidates(g)
	if len(got) != 1 || got[0] != g.CoverURL {
		t.Fatalf("lost official image: %#v", got)
	}
}

func TestCoverCandidatesFromCachedHubWithoutCardArtwork(t *testing.T) {
	root := t.TempDir()
	hub := HubIndex{Updated: time.Now().UTC(), Games: []HubGame{
		{Title: "American Idol (Prototype)", Versions: []HubVersion{{TitleID: "434D0027", Region: "NTSC", Version: "1200"}}},
	}}
	body, err := json.Marshal(hub)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(root, "patchhub-index.json"), body, 0600); err != nil {
		t.Fatal(err)
	}
	cat, err := LoadCatalog(root, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(cat.Games) != 1 {
		t.Fatalf("expected one cached Hub game: %+v", cat)
	}
	got := CoverCandidates(cat.Games[0])
	if len(got) != 2 || !strings.HasSuffix(got[0], "/434D0027.jpg") {
		t.Fatalf("cached catalogue did not infer website cover: %#v", got)
	}
}


func TestCoverCandidatesActualAmericanIdolDiagnosticMismatch(t *testing.T) {
	// Owner's test.22 report: game name from JMP, folder from website's
	// "Miscellaneous-Prototype [American Idol]" card, same XBE Title ID.
	g := Game{
		Title: "American Idol (Prototype)",
		CoverFolder: "Miscellaneous-Prototype [American Idol]",
		Patches: []CatalogPatch{{TitleID: "434D0027"}},
		Versions: []HubVersion{{TitleID: "434D0027", Region: "NTSC", Version: "1200"}},
	}
	got := CoverCandidates(g)
	canonical := "https://www.jayxbox.com/All-Patches/titles/American%20Idol%20%28Prototype%29/434D0027.jpg"
	hubAlias := "https://www.jayxbox.com/All-Patches/titles/Miscellaneous-Prototype%20%5BAmerican%20Idol%5D/434D0027.jpg"
	if len(got) != 4 || got[0] != canonical || got[2] != hubAlias {
		t.Fatalf("must try actual canonical art BEFORE alternate Hub title: %#v", got)
	}
	if got[1] != strings.TrimSuffix(canonical, ".jpg")+".png" {
		t.Fatalf("missing canonical PNG fallback: %#v", got)
	}
}

func TestCoverCandidatesDoNotGuessUnverifiedAlias(t *testing.T) {
	g := Game{
		Title: "Different Local Name",
		CoverFolder: "Distinct Patch Hub Title",
		Patches: []CatalogPatch{{TitleID: "434D0027"}},
		Versions: []HubVersion{{TitleID: "DEADBEEF"}},
	}
	got := CoverCandidates(g)
	if len(got) != 4 {
		t.Fatalf("should only try known Hub folder for unverified alias: %#v", got)
	}
	for _, u := range got {
		if strings.Contains(u,"Different%20Local%20Name") {
			t.Fatalf("guessed unverified alias folder: %#v", got)
		}
	}
}

func TestCoverCandidatesImportedJMPAndMisnamedHubIndex(t *testing.T) {
	root := t.TempDir()
	patchFolder := filepath.Join(root,"GitHub","SPPV")
	if err:=os.MkdirAll(patchFolder,0755); err!=nil { t.Fatal(err) }
	jmp := fixtureJMP(fakeXBE())
	jmp = strings.Replace(jmp,"game-title=Halo 2","game-title=American Idol (Prototype)",1)
	jmp = strings.Replace(jmp,"version=4D530064 (MS-100)","version=434D0027 (CM-039)",1)
	if err:=os.WriteFile(filepath.Join(patchFolder,"American Idol {720p} (PROTOTYPE) [SPPV].JMP"),[]byte(jmp),0644);err!=nil{t.Fatal(err)}
	hub := HubIndex{Updated:time.Now().UTC(),Games:[]HubGame{{
		Title:"Miscellaneous-Prototype [American Idol]",
		Versions:[]HubVersion{{TitleID:"434D0027",Region:"NTSC",Version:"1200"}},
	}}}
	data,err:=json.Marshal(hub);if err!=nil{t.Fatal(err)}
	if err=os.WriteFile(filepath.Join(root,"patchhub-index.json"),data,0600);err!=nil{t.Fatal(err)}
	cat,err:=LoadCatalog(root,2);if err!=nil{t.Fatal(err)}
	if len(cat.Games)!=1 { t.Fatalf("expected one game, got %+v",cat.Games) }
	g:=cat.Games[0]
	if g.Title!="American Idol (Prototype)" || g.CoverFolder!="Miscellaneous-Prototype [American Idol]" || len(g.Patches)!=1 {
		t.Fatalf("incorrect GitHub/Hub alias merge: %+v",g)
	}
	got:=CoverCandidates(g)
	if len(got)!=4 || !strings.HasSuffix(got[0],"/American%20Idol%20%28Prototype%29/434D0027.jpg") {
		t.Fatalf("real diagnostic alias failure reproduced, canonical fallback still absent: %#v",got)
	}
}


func TestVerifiedPunctuationAliasCoverFallback(t *testing.T) {
	g:=Game{
		Title:"BeatDown - Fists of Vengeance",
		TitleAliases:[]string{"BeatDown: Fists of Vengeance"},
		Patches:[]CatalogPatch{{TitleID:"43430015"}},
	}
	got:=CoverCandidates(g)
	want:="https://www.jayxbox.com/All-Patches/titles/BeatDown%20-%20Fists%20of%20Vengeance/43430015.jpg"
	if len(got)!=4 || got[0]!=want ||
		!strings.HasSuffix(got[2],"/BeatDown:%20Fists%20of%20Vengeance/43430015.jpg"){
		t.Fatalf("title alias cover resolution is not retained: %#v",got)
	}
}
