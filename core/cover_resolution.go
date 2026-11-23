package core

import (
	"net/url"
	"strings"
	"unicode"
)

// CoverCandidates returns trusted, bounded URL choices for a game's cover.
// A site's explicit cover URL remains authoritative. PatchHub's public
// catalogue title is not necessarily its /All-Patches/titles/ directory name:
// a JMP may say "American Idol (Prototype)" while the Hub displays
// "Miscellaneous-Prototype [American Idol]" for the same XBE Title ID.
// A title alias is eligible only when the JMP and Hub share a valid Title ID.
// This changes artwork lookup ONLY, never patch compatibility.
func CoverCandidates(g Game) []string {
	candidates := make([]string, 0, 25)
	seen := make(map[string]bool)
	add := func(raw string) {
		if raw != "" && !seen[raw] {
			seen[raw] = true
			candidates = append(candidates, raw)
		}
	}
	add(g.CoverURL)

	ids := make([]string, 0, 3)
	idSeen := map[string]bool{}
	addID := func(raw string) {
		id := strings.ToUpper(strings.TrimSpace(raw))
		if !validCoverTitleID(id) || idSeen[id] { return }
		idSeen[id] = true
		ids = append(ids, id)
	}
	for _, p := range g.Patches { addID(p.TitleID) }
	for _, v := range g.Versions { addID(v.TitleID) }
	// Mixed, unrelated IDs are ambiguous. Never generate an unbounded crawl.
	if len(ids) == 0 || len(ids) > 3 { return candidates }

	// The actual JMP/XBE and Hub index can name one game differently.
	// Only a corroborating Title ID allows the JMP's distinct game title to
	// be tried as a folder. This works even with a cached or oddly named Hub
	// index and avoids guessing based on fuzzy title similarities.
	verifiedAlias := false
	for _, p := range g.Patches {
		id := strings.ToUpper(strings.TrimSpace(p.TitleID))
		if !validCoverTitleID(id) { continue }
		for _, v := range g.Versions {
			if strings.EqualFold(id, strings.TrimSpace(v.TitleID)) {
				verifiedAlias = true
				break
			}
		}
		if verifiedAlias { break }
	}

	folders := []string{}
	folderSeen := map[string]bool{}
	addFolder := func(name string) {
		if len(folders) >= 4 || !safeCoverFolder(name) { return }
		key := strings.ToLower(name)
		if folderSeen[key] { return }
		folderSeen[key] = true
		folders = append(folders, name)
	}

	if g.CoverFolder == "" ||
		CoverMatchKey(g.Title) == CoverMatchKey(g.CoverFolder) ||
		verifiedAlias {
		// The JMP's canonical title gets priority over a Hub display alias.
		addFolder(g.Title)
	}
	addFolder(g.CoverFolder)
	// A safely merged punctuation-only JMP alias can have the actual image
	// directory even when its spelling is not the chosen display title.
	// These aliases were admitted only after a corroborating Xbox Title ID.
	for _, alias := range g.TitleAliases { addFolder(alias) }
	// In the normal one-title case, only the canonical directory is tried.
	for _, folder := range folders {
		base := "https://www.jayxbox.com/All-Patches/titles/" + url.PathEscape(folder) + "/"
		for _, id := range ids {
			add(base + id + ".jpg")
			add(base + id + ".png")
		}
	}
	return candidates
}

func validCoverTitleID(id string) bool {
	if len(id) != 8 || id == "00000000" { return false }
	for _, r := range id {
		if !((r >= '0' && r <= '9') || (r >= 'A' && r <= 'F')) { return false }
	}
	return true
}

func safeCoverFolder(folder string) bool {
	if folder == "" || folder != strings.TrimSpace(folder) || folder == "." || folder == ".." || len(folder) > 200 { return false }
	for _, r := range folder {
		if r == '/' || r == '\\' || unicode.IsControl(r) { return false }
	}
	return true
}
