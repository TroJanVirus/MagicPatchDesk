package core

import (
	"net/url"
	"strings"
	"unicode"
)

// CoverCandidates returns prioritized URLs. The page's explicit cover URL is
// authoritative. Fall back only to IDs from the same catalogue game/version;
// no cross-title fuzzy image matching or networking happens here.
func CoverCandidates(g Game) []string {
	candidates := make([]string, 0, 7)
	seen := make(map[string]bool)
	add := func(raw string) {
		if raw != "" && !seen[raw] {
			seen[raw] = true
			candidates = append(candidates, raw)
		}
	}
	add(g.CoverURL)
	folder := g.CoverFolder
	if folder == "" {
		folder = g.Title
	}
	if !safeCoverFolder(folder) {
		return candidates
	}
	ids := make([]string, 0, 3)
	idSeen := map[string]bool{}
	addID := func(raw string) {
		id := strings.ToUpper(strings.TrimSpace(raw))
		if !validCoverTitleID(id) || idSeen[id] {
			return
		}
		idSeen[id] = true
		ids = append(ids, id)
	}
	for _, p := range g.Patches {
		addID(p.TitleID)
	}
	for _, v := range g.Versions {
		addID(v.TitleID)
	}
	// Excessive mixed IDs indicate ambiguous metadata, never choose at random.
	if len(ids) > 3 {
		return candidates
	}
	directory := "https://www.jayxbox.com/All-Patches/titles/" + url.PathEscape(folder) + "/"
	for _, id := range ids {
		add(directory + id + ".jpg")
		add(directory + id + ".png")
	}
	return candidates
}

func validCoverTitleID(id string) bool {
	if len(id) != 8 || id == "00000000" {
		return false
	}
	for _, r := range id {
		if !((r >= '0' && r <= '9') || (r >= 'A' && r <= 'F')) {
			return false
		}
	}
	return true
}

func safeCoverFolder(folder string) bool {
	if folder == "" || folder != strings.TrimSpace(folder) || folder == "." || folder == ".." || len(folder) > 200 {
		return false
	}
	for _, r := range folder {
		if r == '/' || r == '\\' || unicode.IsControl(r) {
			return false
		}
	}
	return true
}
