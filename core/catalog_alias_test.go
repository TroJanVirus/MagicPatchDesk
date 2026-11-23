package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func testBeatDownJMP(t *testing.T,root,dir,name,game,group string) {
	t.Helper()
	folder := filepath.Join(root,"GitHub",dir)
	if err := os.MkdirAll(folder,0755); err!=nil { t.Fatal(err) }
	src := fixtureJMP(fakeXBE())
	src = strings.Replace(src,"game-title=Halo 2","game-title="+game,1)
	src = strings.Replace(src,"version=4D530064 (MS-100)","version=43430015 (CC-021)",1)
	// Different operational groups/records are intentional: these are three
	// distinct patches, NOT fingerprints to collapse as duplicates.
	src = strings.ReplaceAll(src,"#Widescreen","#"+group)
	src = strings.Replace(src,"region=Global","region=NTSC",1)
	author:=dir
	if dir=="icup321" { author="FrankyBuster" }
	src = strings.Replace(src,"author=Tester","author="+author,1)
	if err:=os.WriteFile(filepath.Join(folder,name),[]byte(src),0644);err!=nil{ t.Fatal(err) }
}

// Actual upstream headers in Nemesis2000, Silverrock, and icup321 JMP files
// as inspected 2026-11-23: two hyphens and one colon, all 43430015 (CC-021).
func TestBeatDownThreeJMPVariantsMergeAndKeepTheirPatches(t *testing.T) {
	root:=t.TempDir()
	testBeatDownJMP(t,root,"Nemesis2000","BeatDown - Fists of Vengeance {Widescreen} (NTSC) [Nemesis2000].JMP","BeatDown - Fists of Vengeance","Nemesis-Widescreen")
	testBeatDownJMP(t,root,"Silverrock","BeatDown - Fists of Vengeance {720p} (NTSC) [Silverrock].JMP","BeatDown: Fists of Vengeance","Silverrock-720p")
	testBeatDownJMP(t,root,"icup321","BeatDown - Fists of Vengeance {Widescreen} (NTSC) [FrankyBuster].JMP","BeatDown - Fists of Vengeance","Franky-Widescreen")

	// The Hub's actual game-card title is the hyphen spelling. It should
	// attach all versions/art to the single canonical game.
	hub:=HubIndex{Updated:time.Now().UTC(),Games:[]HubGame{{
		Title:"BeatDown - Fists of Vengeance",
		Image:"https://www.jayxbox.com/All-Patches/titles/BeatDown%20-%20Fists%20of%20Vengeance/43430015.jpg",
		Versions:[]HubVersion{{TitleID:"43430015",Region:"USA",Version:"2"}},
	}}}
	raw,err:=json.Marshal(hub);if err!=nil{t.Fatal(err)}
	if err=os.WriteFile(filepath.Join(root,"patchhub-index.json"),raw,0600);err!=nil{t.Fatal(err)}

	cat,err:=LoadCatalog(root,3);if err!=nil{t.Fatal(err)}
	if len(cat.Games)!=1 || cat.GameCount!=1 || cat.PatchCount!=3 || len(cat.Duplicates)!=0 {
		t.Fatalf("expected one catalog game with 3 distinct patches: %+v",cat)
	}
	game:=cat.Games[0]
	if game.Title!="BeatDown - Fists of Vengeance" || game.CoverFolder!="BeatDown - Fists of Vengeance" {
		t.Fatalf("wrong display/catalog cover association: %+v",game)
	}
	if len(game.Patches)!=3 { t.Fatalf("lost source patches: %+v",game.Patches) }
	authors:=map[string]bool{}
	for _,p:=range game.Patches {
		if p.TitleID!="43430015" || !strings.Contains(p.Name,"BeatDown - Fists of Vengeance") {
			t.Fatalf("altered source patch identity: %+v",p)
		}
		authors[p.Author]=true
	}
	for _,a:=range []string{"Nemesis2000","Silverrock","FrankyBuster"} {
		if !authors[a] {t.Fatalf("lost patch author %s: %+v",a,game.Patches)}
	}
	if len(game.TitleAliases)!=1 || game.TitleAliases[0]!="BeatDown: Fists of Vengeance" {
		t.Fatalf("lost verified source-title alias: %+v",game.TitleAliases)
	}
	urls:=CoverCandidates(game)
	if len(urls)!=4 || urls[0]!=hub.Games[0].Image ||
		!strings.Contains(urls[2],"BeatDown:%20Fists%20of%20Vengeance/43430015.jpg") {
		t.Fatalf("missing canonical cover and safe alias fallback: %#v",urls)
	}
}

func TestTitleAliasMergeFailsClosedForUnrelatedGames(t *testing.T) {
	table:=[]struct{name string;games []*Game;want int}{
		{"different IDs despite equivalent punctuation",[]*Game{
			{Title:"A - B",Patches:[]CatalogPatch{{TitleID:"43430015"}}},
			{Title:"A: B",Patches:[]CatalogPatch{{TitleID:"43430016"}}},
		},2},
		{"same ID but unrelated game names",[]*Game{
			{Title:"BeatDown - Fists of Vengeance",Patches:[]CatalogPatch{{TitleID:"43430015"}}},
			{Title:"Street Fighter - Anniversary Collection",Patches:[]CatalogPatch{{TitleID:"43430015"}}},
		},2},
		{"same normalized characters but different token boundaries",[]*Game{
			{Title:"AB C",Patches:[]CatalogPatch{{TitleID:"43430015"}}},
			{Title:"A BC",Patches:[]CatalogPatch{{TitleID:"43430015"}}},
		},2},
		{"one missing identifier",[]*Game{
			{Title:"A - B",Patches:[]CatalogPatch{{TitleID:"43430015"}}},
			{Title:"A: B",Patches:[]CatalogPatch{{TitleID:""}}},
		},2},
		{"mixed IDs in one group",[]*Game{
			{Title:"A - B",Patches:[]CatalogPatch{{TitleID:"43430015"},{TitleID:"43430016"}}},
			{Title:"A: B",Patches:[]CatalogPatch{{TitleID:"43430015"}}},
		},2},
		{"same ID same token sequence",[]*Game{
			{Title:"A - B",Patches:[]CatalogPatch{{TitleID:"43430015"}}},
			{Title:"A: B",Patches:[]CatalogPatch{{TitleID:"43430015"}}},
		},1},
	}
	for _,tc:=range table {
		t.Run(tc.name,func(t *testing.T){
			games:=map[string]*Game{}
			for _,g:=range tc.games{games[strings.ToLower(g.Title)]=g}
			mergeSafeCatalogTitleAliases(games)
			if len(games)!=tc.want {t.Fatalf("unexpected merge count %d want %d: %+v",len(games),tc.want,games)}
		})
	}
}

func TestAmbiguousTitleIDCannotStealHubMetadata(t *testing.T) {
	g1:=&Game{Title:"Alpha",Patches:[]CatalogPatch{{TitleID:"43430015"}}}
	g2:=&Game{Title:"Another Game",Patches:[]CatalogPatch{{TitleID:"43430015"}}}
	owners:=uniqueCatalogTitleIDOwners(map[string]*Game{"alpha":g1,"another":g2})
	if owners["43430015"]!=nil {t.Fatal("ambiguous ID assigned to arbitrary game")}
	only:=uniqueCatalogTitleIDOwners(map[string]*Game{"alpha":g1})
	if only["43430015"]!=g1{t.Fatal("unambiguous ID no longer attached")}
}
