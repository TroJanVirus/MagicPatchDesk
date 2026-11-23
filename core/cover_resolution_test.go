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
