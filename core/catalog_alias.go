package core

import (
	"sort"
	"strings"
	"unicode"
)

// catalogTitleWordKey treats punctuation and whitespace as *separators*,
// preserving word boundaries. "BeatDown: Fists" and "BeatDown - Fists"
// match, whereas "AB C" and "A BC" intentionally do not.
func catalogTitleWordKey(title string) string {
	return strings.Join(strings.FieldsFunc(strings.ToLower(strings.TrimSpace(title)), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}), " ")
}

// singleCatalogGameTitleID requires every attached patch to have exactly the
// same valid XBE Title ID; unknown, zero, or mixed IDs are not sufficient to
// merge catalogue entries. This is intentionally stricter than matching covers.
func singleCatalogGameTitleID(g *Game) string {
	if g == nil || len(g.Patches) == 0 { return "" }
	id := ""
	for _, p := range g.Patches {
		current := strings.ToUpper(strings.TrimSpace(p.TitleID))
		if !validCoverTitleID(current) { return "" }
		if id == "" { id = current } else if id != current { return "" }
	}
	return id
}

type catalogAliasEntry struct {
	key string
	game *Game
}

// mergeSafeCatalogTitleAliases groups only punctuation/spacing alternatives
// whose independent parsed JMP/PF records all agree on a single Title ID.
// It does not rewrite any record, original filename, author, CRC, or region.
// Because it operates before the Hub merge, site metadata can attach to one
// canonical game rather than arbitrarily choosing a split duplicate entry.
func mergeSafeCatalogTitleAliases(games map[string]*Game) {
	keys := make([]string, 0, len(games))
	for key := range games { keys = append(keys, key) }
	sort.Strings(keys)

	buckets := map[string][]catalogAliasEntry{}
	for _, key := range keys {
		g := games[key]
		wordKey := catalogTitleWordKey(g.Title)
		id := singleCatalogGameTitleID(g)
		if wordKey == "" || id == "" { continue }
		bucketKey := wordKey + "\x00" + id
		buckets[bucketKey] = append(buckets[bucketKey], catalogAliasEntry{key, g})
	}

	for _, entries := range buckets {
		if len(entries) < 2 { continue }
		// Prefer a group with local artwork; otherwise prefer the entry that
		// already contains the most patches. Ties use a stable lexical order.
		sort.SliceStable(entries, func(i, j int) bool {
			a,b := entries[i].game, entries[j].game
			if (a.Cover!="") != (b.Cover!="") { return a.Cover!="" }
			if len(a.Patches)!=len(b.Patches) { return len(a.Patches)>len(b.Patches) }
			return entries[i].key < entries[j].key
		})
		main := entries[0].game
		for _, alternate := range entries[1:] {
			g := alternate.game
			main.Patches = append(main.Patches, g.Patches...)
			main.TitleAliases = append(main.TitleAliases, g.Title)
			main.TitleAliases = append(main.TitleAliases, g.TitleAliases...)
			if main.Cover == "" { main.Cover = g.Cover }
			if main.CoverURL == "" { main.CoverURL = g.CoverURL }
			if main.CoverFolder == "" { main.CoverFolder = g.CoverFolder }
			if len(main.Versions)==0 { main.Versions = g.Versions }
			delete(games, alternate.key)
		}
	}
}

// When multiple distinct games claim the same Xbox Title ID, do not select
// one arbitrarily based on Go map iteration order. Hub title matching can
// still use exact names; ID fallback is allowed only for a unique owner.
func uniqueCatalogTitleIDOwners(games map[string]*Game) map[string]*Game {
	owners := make(map[string]*Game)
	ambiguous := make(map[string]bool)
	for _, g := range games {
		for _, p := range g.Patches {
			id := strings.ToUpper(strings.TrimSpace(p.TitleID))
			if !validCoverTitleID(id) { continue }
			if old := owners[id]; old != nil && old != g {
				ambiguous[id] = true
				continue
			}
			owners[id] = g
		}
	}
	for id := range ambiguous { delete(owners, id) }
	return owners
}
