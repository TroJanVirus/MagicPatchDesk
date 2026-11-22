package core

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"
)

const GithubRepo = "JayYardley/Xbox-Magic-Patches-by-Jay"
const GitTreeAPI = "https://api.github.com/repos/" + GithubRepo + "/git/trees/main?recursive=1"
const GitZipURL = "https://github.com/" + GithubRepo + "/archive/refs/heads/main.zip"

type CatalogPatch struct {
	File    string
	Name    string
	Game    string
	Type    string
	Author  string
	Region  string
	Version string
	TitleID string
	Notes   string
	Groups  int
	Records int
	CRC     string
	Source  string
	Art     string
	RAM     string
	Issue   string
    Gallery []PatchGalleryImage
}
type Game struct {
	Title    string
	Patches  []CatalogPatch
	Cover    string
	CoverURL string
	Versions []HubVersion
}
type Catalog struct {
	Games      []Game
	PatchCount int
	GameCount  int
	Issues     int
}

var idPattern = regexp.MustCompile(`(?i)^\s*(?:0x)?([0-9A-F]{8})\b`)

// CoverMatchKey tolerates punctuation, spaces and common separators that
// differ between the GitHub JMP title and the PatchHub game directory.
// Ambiguous normalized matches are *never* merged automatically.
func CoverMatchKey(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(name)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func TitleIDFromVersion(v string) string {
	a := idPattern.FindStringSubmatch(v)
	if len(a) > 1 {
		return strings.ToUpper(a[1])
	}
	return ""
}
func GetLibraryDir() (string, error) {
	d, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	p := filepath.Join(d, "MagicPatchDesk")
	return p, os.MkdirAll(p, 0755)
}
func LoadCatalog(root string, workers int) (Catalog, error) {
	c := Catalog{}
	if workers < 1 {
		workers = 1
	}
	if workers > 16 {
		workers = 16
	}
	paths := []string{}
	for _, sub := range []string{"GitHub", "Custom"} {
		d := filepath.Join(root, sub)
		_ = filepath.WalkDir(d, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if !entry.IsDir() && (strings.EqualFold(filepath.Ext(path), ".jmp") || strings.EqualFold(filepath.Ext(path), ".pf")) {
				paths = append(paths, path)
			}
			return nil
		})
	}
	type result struct {
		p   CatalogPatch
		err error
	}
	jobs := make(chan string, workers*4)
	results := make(chan result, workers*4)
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range jobs {
				b, e := os.ReadFile(path)
				if e != nil {
					results <- result{err: e}
					continue
				}
				var p *Patch
				var pf *PF
				var parseErr error
				if strings.EqualFold(filepath.Ext(path), ".pf") {
					pf, parseErr = ParsePF(filepath.Base(path), b)
				} else {
					p, parseErr = ParseJMP(filepath.Base(path), b)
				}
				name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
				title := strings.Split(name, " {")[0]
				parts := strings.Split(filepath.ToSlash(path), "/")
				for i, seg := range parts {
					if strings.EqualFold(seg, "titles") && i+2 < len(parts) {
						title = parts[i+1]
						break
					}
				}
				source := "Custom"
				if strings.Contains(filepath.ToSlash(path), "/GitHub/") {
					source = "GitHub"
				}
				cp := CatalogPatch{File: path, Name: name, Game: strings.TrimSpace(title), Type: PatchTypeFromFilename(name), Author: "Unknown", Source: source}
				// Match PatchHub's titles/<game>/<region_version_TitleID>/<patch>/ layout.
				for pi, segment := range parts {
					if !strings.EqualFold(segment, "titles") || pi+2 >= len(parts) {
						continue
					}
					cp.Game = parts[pi+1]
					versionParts := strings.Split(parts[pi+2], "_")
					if len(versionParts) >= 3 {
						cp.Region = versionParts[0]
						cp.Version = versionParts[1]
						cp.TitleID = strings.ToUpper(versionParts[2])
					}
					if ramBytes, ramErr := os.ReadFile(filepath.Join(filepath.Dir(path), "RAM.txt")); ramErr == nil {
						switch strings.ToLower(strings.TrimSpace(string(ramBytes))) {
						case "y", "yes":
							cp.RAM = "64MB compatible"
						case "n", "no":
							cp.RAM = "128MB required"
						default:
							cp.RAM = "RAM unknown"
						}
					}
					break
				}

				if parseErr == nil && pf != nil {
					if v := pf.Header["title"]; v != "" {
						cp.Game = v
					}
					cp.Type = pf.Header["type"]
					if cp.Type == "" {
						cp.Type = "XBMC4Gamers PF"
					}
					cp.Author = pf.Header["author"]
					if cp.Author == "" {
						cp.Author = "Unknown"
					}
					cp.Region = pf.Header["region"]
					cp.TitleID = strings.ToUpper(pf.Header["titleid"])
					if cp.TitleID == "MULTI" {
						cp.TitleID = ""
					}
					cp.Notes = pf.Header["info"]
					cp.Groups = 0
					cp.Records = len(pf.Operations)
					cp.CRC = "PF TitleID check"
				} else if parseErr == nil {
					if v := p.Headers["game-title"]; v != "" && !strings.Contains(filepath.ToSlash(path), "/titles/") {
						cp.Game = v
					}
					cp.Author = p.Headers["author"]
					if cp.Author == "" {
						cp.Author = "Unknown"
					}
					if cp.Region == "" {
						cp.Region = p.Headers["region"]
					}
					if cp.Version == "" {
						cp.Version = p.Headers["version"]
					}
					if cp.TitleID == "" {
						cp.TitleID = TitleIDFromVersion(p.Headers["version"])
					}
					cp.Notes = p.Headers["notes"]
					cp.Groups = len(p.Groups)
					cp.Records = len(p.Records)
					cp.CRC = p.ExpectedCRC()
				} else {
					cp.Issue = parseErr.Error()
					cp.Notes = "Patch must be reviewed: " + parseErr.Error()
				}
                cp.Gallery=LocalPatchScreenshots(path)
				results <- result{p: cp, err: parseErr}
			}
		}()
	}

	go func() {
		for _, f := range paths {
			jobs <- f
		}
		close(jobs)
		wg.Wait()
		close(results)
	}()
	games := map[string]*Game{}
	for r := range results {
		if r.err != nil {
			c.Issues++
			if r.p.File == "" {
				continue
			}
		}
		p := r.p
		key := strings.ToLower(strings.TrimSpace(p.Game))
		if key == "" {
			key = "unknown"
		}
		g := games[key]
		if g == nil {
			g = &Game{Title: p.Game}
			games[key] = g
		}
		g.Patches = append(g.Patches, p)
		c.PatchCount++
	}
	// Include every game title in a locally imported PatchHub 'titles' tree,
	// even games for which no JMP currently exists. Cache local cover art paths.
	_ = filepath.WalkDir(filepath.Join(root, "Custom"), func(dir string, entry os.DirEntry, err error) error {
		if err != nil || !entry.IsDir() || !strings.EqualFold(entry.Name(), "titles") {
			return nil
		}
		folderList, e := os.ReadDir(dir)
		if e != nil {
			return nil
		}
		for _, folder := range folderList {
			if !folder.IsDir() {
				continue
			}
			title := folder.Name()
			key := strings.ToLower(strings.TrimSpace(title))
			g := games[key]
			if g == nil {
				g = &Game{Title: title}
				games[key] = g
			}
			for _, ext := range []string{".jpg", ".jpeg", ".png", ".bmp"} {
				b, er := filepath.Glob(filepath.Join(dir, title, "*"+ext))
				if er == nil && len(b) > 0 {
					g.Cover = b[0]
					break
				}
			}
		}
		return filepath.SkipDir
	})
	// Merge the public Patch Hub game index. This supplies title artwork and
	// games that do not yet have a downloadable JMP. GitHub JMPs remain the
	// only authoritative patch payloads; Hub metadata never executes anything.
	hub := ReadHubCache(root)
	byID := make(map[string]*Game)
	for _, g := range games {
		for _, p := range g.Patches {
			if p.TitleID != "" && p.TitleID != "00000000" {
				byID[strings.ToUpper(p.TitleID)] = g
			}
		}
	}
	// Unique normalized titles allow matching "Halo: Combat Evolved" against
	// "Halo - Combat Evolved" without guessing between near-identical games.
	byNormalizedName := make(map[string]*Game)
	ambiguousNames := make(map[string]bool)
	for _, existing := range games {
		key := CoverMatchKey(existing.Title)
		if key == "" {
			continue
		}
		if prior, ok := byNormalizedName[key]; ok && prior != existing {
			ambiguousNames[key] = true
		} else {
			byNormalizedName[key] = existing
		}
	}
	for _, hg := range hub.Games {
		key := strings.ToLower(strings.TrimSpace(hg.Title))
		if key == "" {
			continue
		}
		g := games[key]
		if g == nil {
			keyNormalized := CoverMatchKey(hg.Title)
			if !ambiguousNames[keyNormalized] {
				g = byNormalizedName[keyNormalized]
			}
		}
		if g == nil {
			for _, v := range hg.Versions {
				if v.TitleID != "00000000" {
					if matched := byID[strings.ToUpper(v.TitleID)]; matched != nil {
						g = matched
						break
					}
				}
			}
		}
		if g == nil {
			g = &Game{Title: hg.Title}
			games[key] = g
		}
		if g.CoverURL == "" {
			g.CoverURL = hg.Image
		}
        if len(hg.Screenshots)>0 {AttachHubGallery(g.Patches,hg.Screenshots)}
		if len(g.Versions) == 0 {
			g.Versions = hg.Versions
		}
	}
	for _, g := range games {
		if g.Cover == "" {
			for _, ext := range []string{".jpg", ".png", ".jpeg"} {
				for _, base := range []string{filepath.Join(root, "Custom", "Art"), filepath.Join(root, "Art")} {
					path := filepath.Join(base, g.Title+ext)
					if _, err := os.Stat(path); err == nil {
						g.Cover = path
						break
					}
				}
				if g.Cover != "" {
					break
				}
			}
		}
	}
	for _, g := range games {
		sort.SliceStable(g.Patches, func(i, j int) bool {
			if g.Patches[i].Type == g.Patches[j].Type {
				return g.Patches[i].Name < g.Patches[j].Name
			}
			return g.Patches[i].Type < g.Patches[j].Type
		})
		c.Games = append(c.Games, *g)
	}
	sort.Slice(c.Games, func(i, j int) bool { return strings.ToLower(c.Games[i].Title) < strings.ToLower(c.Games[j].Title) })
	c.GameCount = len(c.Games)
	return c, nil
}

type gitTreeResponse struct {
	Sha       string `json:"sha"`
	Truncated bool   `json:"truncated"`
	Tree      []struct {
		Path string `json:"path"`
		Type string `json:"type"`
	} `json:"tree"`
}
type SyncResult struct {
	Updated     bool
	RemoteCount int
	CachedCount int
	Offline     bool
	SHA         string
	Message     string
}

func SyncGithub(ctx context.Context, root string, onStatus func(string)) (SyncResult, error) {
	res := SyncResult{}
	tell := func(s string) {
		if onStatus != nil {
			onStatus(s)
		}
	}
	client := &http.Client{Timeout: 25 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", GitTreeAPI, nil)
	if err != nil {
		return res, err
	}
	req.Header.Set("User-Agent", "MagicPatchDesk/1.0")
	req.Header.Set("Accept", "application/vnd.github+json")
	response, err := client.Do(req)
	if err != nil {
		return res, fmt.Errorf("GitHub unavailable (offline cache kept): %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return res, fmt.Errorf("GitHub responded %s; offline cache kept", response.Status)
	}
	var tree gitTreeResponse
	if err := json.NewDecoder(io.LimitReader(response.Body, 16<<20)).Decode(&tree); err != nil {
		return res, err
	}
	if tree.Truncated {
		return res, errors.New("GitHub tree truncated; refusing incomplete update")
	}
	for _, f := range tree.Tree {
		if f.Type == "blob" && strings.EqualFold(filepath.Ext(f.Path), ".jmp") {
			res.RemoteCount++
		}
	}
	res.SHA = tree.Sha
	manifest := filepath.Join(root, "github-sha.txt")
	previous, _ := os.ReadFile(manifest)
	cache := filepath.Join(root, "GitHub")
	existing := 0
	_ = filepath.WalkDir(cache, func(path string, entry os.DirEntry, err error) error {
		if err == nil && !entry.IsDir() && strings.EqualFold(filepath.Ext(path), ".jmp") {
			existing++
		}
		return nil
	})
	res.CachedCount = existing
	if tree.Sha == strings.TrimSpace(string(previous)) && existing == res.RemoteCount && existing > 0 {
		res.Message = "Up to date"
		return res, nil
	}
	tell(fmt.Sprintf("GitHub has %d patches — updating cached collection…", res.RemoteCount))
	req2, err := http.NewRequestWithContext(ctx, "GET", GitZipURL, nil)
	if err != nil {
		return res, err
	}
	req2.Header.Set("User-Agent", "MagicPatchDesk/1.0")
	response2, err := client.Do(req2)
	if err != nil {
		return res, err
	}
	defer response2.Body.Close()
	if response2.StatusCode != 200 {
		return res, fmt.Errorf("archive returned %s", response2.Status)
	}
	if response2.ContentLength > 150<<20 {
		return res, errors.New("GitHub archive unexpectedly large")
	}
	zipFile, err := os.CreateTemp(root, "download-*.zip")
	if err != nil {
		return res, err
	}
	zipPath := zipFile.Name()
	defer os.Remove(zipPath)
	if _, err = io.Copy(zipFile, io.LimitReader(response2.Body, 150<<20)); err != nil {
		zipFile.Close()
		return res, err
	}
	if err = zipFile.Close(); err != nil {
		return res, err
	}
	z, err := zip.OpenReader(zipPath)
	if err != nil {
		return res, err
	}
	defer z.Close()
	staging := filepath.Join(root, "GitHub.new")
	os.RemoveAll(staging)
	if err = os.MkdirAll(staging, 0755); err != nil {
		return res, err
	}
	ready := false
	defer func() {
		if !ready {
			os.RemoveAll(staging)
		}
	}()
	count := 0
	for _, file := range z.File {
		if file.FileInfo().IsDir() || !strings.EqualFold(filepath.Ext(file.Name), ".jmp") {
			continue
		}
		parts := strings.SplitN(filepath.ToSlash(file.Name), "/", 2)
		if len(parts) != 2 {
			continue
		}
		rel := filepath.Clean(filepath.FromSlash(parts[1]))
		if filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return res, errors.New("unsafe archive path")
		}
		if file.UncompressedSize64 > 32<<20 {
			return res, errors.New("oversized JMP in archive")
		}
		dest := filepath.Join(staging, rel)
		if err = os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return res, err
		}
		src, err := file.Open()
		if err != nil {
			return res, err
		}
		b, err := io.ReadAll(io.LimitReader(src, 32<<20+1))
		src.Close()
		if err != nil {
			return res, err
		}
		if err = os.WriteFile(dest, b, 0644); err != nil {
			return res, err
		}
		count++
	}
	if count != res.RemoteCount || count == 0 {
		return res, fmt.Errorf("download incomplete: %d of %d JMP files", count, res.RemoteCount)
	}
	old := filepath.Join(root, "GitHub.old")
	_ = os.RemoveAll(old)
	_ = os.Rename(cache, old)
	if err = os.Rename(staging, cache); err != nil {
		_ = os.Rename(old, cache)
		return res, err
	}
	_ = os.RemoveAll(old)
	ready = true
	_ = os.WriteFile(manifest, []byte(tree.Sha), 0644)
	res.Updated = true
	res.CachedCount = count
	res.Message = fmt.Sprintf("Updated to %d GitHub patches", count)
	return res, nil
}
// ImportJMP validates an input patch, then leaves the library unchanged if
// identical bytes already exist in Custom or the GitHub cache. Different patch
// revisions with the same filename get an independent, uniquely named copy.
func ImportJMP(root, source string) (string, error) {
    ext:=strings.ToLower(filepath.Ext(source))
    if ext!=".jmp"&&ext!=".pf"{return "",errors.New("choose a .jmp or .pf patch file")}
    stat,err:=os.Lstat(source)
    if err!=nil{return "",err}
    if !stat.Mode().IsRegular()||stat.Size()<1||stat.Size()>maxImportedPatchBytes{
        return "",errors.New("patch import requires a regular file of at most 16 MiB")
    }
    b,err:=os.ReadFile(source)
    if err!=nil{return "",err}
    if int64(len(b))!=stat.Size(){return "",errors.New("source patch changed while reading; retry import")}
    if ext==".jmp"{
        if _,err=ParseJMP(filepath.Base(source),b);err!=nil{return "",err}
    }else{
        if _,err=ParsePF(filepath.Base(source),b);err!=nil{return "",err}
    }
    existing,err:=existingIdenticalPatch(root,ext,b)
    if err!=nil{return "",err}
    if existing!=""{return "",&DuplicatePatchError{Existing:existing}}
    destDir:=filepath.Join(root,"Custom")
    if err=os.MkdirAll(destDir,0755);err!=nil{return "",err}
    return writeImportedPatchExclusive(destDir,filepath.Base(source),b)
}

// ImportFolder accepts a local PatchHub titles directory or a folder containing JMP files.
// It preserves the folder hierarchy and supported artwork, with no network access.
func ImportFolder(root, folder string) (int, error) {
	st, err := os.Stat(folder)
	if err != nil {
		return 0, err
	}
	if !st.IsDir() {
		return 0, errors.New("choose a directory")
	}
	base := filepath.Base(filepath.Clean(folder))
	destination := filepath.Join(root, "Custom", base)
	if filepath.Clean(folder) == filepath.Clean(destination) {
		return 0, errors.New("source already inside the patch library")
	}
	if _, err := os.Stat(destination); err == nil {
		destination += fmt.Sprintf("-%d", time.Now().Unix())
	}
	count := 0
	err = filepath.WalkDir(folder, func(path string, entry os.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		switch ext {
		case ".jmp", ".pf", ".jpg", ".jpeg", ".png", ".bmp", ".txt", ".earl":
		default:
			return nil
		}
		rel, e := filepath.Rel(folder, path)
		if e != nil {
			return e
		}
		dest := filepath.Join(destination, rel)
		if e = os.MkdirAll(filepath.Dir(dest), 0755); e != nil {
			return e
		}
		src, e := os.Open(path)
		if e != nil {
			return e
		}
		defer src.Close()
		output, e := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if e != nil {
			return e
		}
		_, e = io.Copy(output, io.LimitReader(src, 40<<20))
		closeErr := output.Close()
		if e == nil {
			e = closeErr
		}
		if e != nil {
			return e
		}
		if ext == ".jmp" || ext == ".pf" {
			count++
		}
		return nil
	})
	return count, err
}
