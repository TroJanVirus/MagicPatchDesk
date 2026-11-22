package core

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

const PatchHubURL = "https://www.jayxbox.com/Retail-Game-Modification/PatchHub.php"

// Patch Hub metadata updates must not race with one another. Source selection,
// Patch Library browsing and game-image inspection never need this lock.
var hubSyncMu sync.Mutex

// Invalid local metadata is a recoverable cache error, not an application crash.
var ErrHubCacheInvalid = errors.New("Patch Hub cached index rejected")

type HubVersion struct {
	TitleID string `json:"title_id"`
	Region  string `json:"region"`
	Version string `json:"version"`
}
type HubGame struct {
	Title    string       `json:"title"`
	Image    string       `json:"image"`
	Versions []HubVersion `json:"versions,omitempty"`
    Screenshots []HubScreenshot `json:"screenshots,omitempty"`
}
type HubIndex struct {
    GalleryScanVersion int `json:"gallery_scan_version,omitempty"`
	Updated time.Time `json:"updated"`
	Games   []HubGame `json:"games"`
}

var hubCard = regexp.MustCompile(`(?is)<div\s+class="game-card"[^>]*>`)
var hubImage = regexp.MustCompile(`(?is)<img[^>]*\bdata-src\s*=\s*["']([^"']+)["']`)
var hubTitle = regexp.MustCompile(`(?is)<div\s+class="game-title">([^<]+)</div>`)
var hubVersion = regexp.MustCompile(`(?is)Title ID:\s*([A-Fa-f0-9]{8})\s*\|\s*Region:\s*([^|<]+)\|\s*Version:\s*([^<]+)`)

func ParsePatchHubHTML(doc []byte) (HubIndex, error) {
	contents := string(doc)
	locations := hubCard.FindAllStringIndex(contents, -1)
	result := HubIndex{Updated: time.Now().UTC(),GalleryScanVersion:2}
	for idx, loc := range locations {
		end := len(contents)
		if idx+1 < len(locations) {
			end = locations[idx+1][0]
		}
		// Bounded search: game headers and version metadata follow their opening card.
		if end-loc[0] > 64<<10 {
			end = loc[0] + 64<<10
		}
		part := contents[loc[1]:end]
		name := hubTitle.FindStringSubmatch(part)
		if len(name) < 2 {
			continue
		}
		g := HubGame{Title: strings.TrimSpace(html.UnescapeString(name[1]))}
		if g.Title == "" {
			continue
		}
		image := hubImage.FindStringSubmatch(part)
		if len(image) > 1 {
			raw := html.UnescapeString(image[1])
			ref, e := url.Parse(raw)
			base, _ := url.Parse("https://www.jayxbox.com")
			if e == nil {
				abs := base.ResolveReference(ref)
				if trustedPatchHubURL(abs) && !strings.HasSuffix(strings.ToLower(abs.Path), "/noart.jpg") {
					g.Image = abs.String()
				}
			}
		}
        g.Screenshots=parseHubGameScreenshots(part)
		for _, v := range hubVersion.FindAllStringSubmatch(part, -1) {
			g.Versions = append(g.Versions, HubVersion{TitleID: strings.ToUpper(v[1]), Region: strings.TrimSpace(html.UnescapeString(v[2])), Version: strings.TrimSpace(html.UnescapeString(v[3]))})
		}
		result.Games = append(result.Games, g)
	}
	if len(result.Games) < 50 {
		return HubIndex{}, fmt.Errorf("Patch Hub returned only %d game cards; retaining existing index", len(result.Games))
	}
	return result, nil
}
// decodeHubCache guards the entire decoding boundary: a malformed or otherwise
// inconsistent cache must not bring down the GUI through an uncaught decoder
// panic. The caller can report ErrHubCacheInvalid without exposing game data.
func decodeHubCache(body []byte, decode func([]byte, any) error) (cache HubIndex, err error) {
    defer func() {
        if recover() != nil {
            cache = HubIndex{}
            err = fmt.Errorf("%w: JSON decoder panic", ErrHubCacheInvalid)
        }
    }()
    if e := decode(body, &cache); e != nil {
        return HubIndex{}, fmt.Errorf("%w: malformed JSON", ErrHubCacheInvalid)
    }
    if cache.Games == nil {
        return HubIndex{}, fmt.Errorf("%w: missing games array", ErrHubCacheInvalid)
    }
    return cache, nil
}

// ReadHubCacheChecked bounds disk reads and returns explicit cache errors. A
// missing cache is normal on first launch; a broken one is NOT silently
// rewritten by SyncPatchHub (the original remains available for diagnosis).
func ReadHubCacheChecked(root string) (HubIndex, error) {
    f, err := os.Open(filepath.Join(root, "patchhub-index.json"))
    if errors.Is(err, os.ErrNotExist) {
        return HubIndex{}, nil
    }
    if err != nil {
        return HubIndex{}, err
    }
    defer f.Close()
    st, err := f.Stat()
    if err != nil {
        return HubIndex{}, err
    }
    if !st.Mode().IsRegular() || st.Size() == 0 || st.Size() > 24<<20 {
        return HubIndex{}, fmt.Errorf("%w: invalid cache size or file type", ErrHubCacheInvalid)
    }
    body, err := io.ReadAll(io.LimitReader(f, (24<<20)+1))
    if err != nil {
        return HubIndex{}, err
    }
    if len(body) > 24<<20 {
        return HubIndex{}, fmt.Errorf("%w: oversized cache", ErrHubCacheInvalid)
    }
    return decodeHubCache(body, json.Unmarshal)
}

// Retain the historical read-only API used by catalogue painting. A bad cache
// results in no optional art; checked refresh paths report the actual failure.
func ReadHubCache(root string) HubIndex {
    cache, _ := ReadHubCacheChecked(root)
    return cache
}
// saveHubIndex atomically; keep existing valid metadata on failures.
func saveHubIndex(root string,hub HubIndex)error{
    b,e:=json.Marshal(hub)
    if e!=nil{return e}
    f,e:=os.CreateTemp(root,"hub-index-*.json")
    if e!=nil{return e}
    defer os.Remove(f.Name())
    if _,e=f.Write(b);e!=nil{f.Close();return e}
    if e=f.Sync();e!=nil{f.Close();return e}
    if e=f.Close();e!=nil{return e}
    return os.Rename(f.Name(),filepath.Join(root,"patchhub-index.json"))
}

// RefreshCachedPatchHubMedia fetches the lightweight JPG/PNG file inventory
// independently of the HTML game index. A fresh upload becomes visible on
// Check + sync without waiting for the 24-hour card-cache expiry. The HTTP
// client can be injected for deterministic offline regression tests.
func RefreshCachedPatchHubMedia(ctx context.Context, root string, client *http.Client)(int,error){
    hubSyncMu.Lock()
    defer hubSyncMu.Unlock()
    return refreshCachedPatchHubMedia(ctx,root,client)
}
func refreshCachedPatchHubMedia(ctx context.Context, root string, client *http.Client)(int,error){
    cache,e:=ReadHubCacheChecked(root)
    if e!=nil{return 0,e}
    if len(cache.Games)==0{return 0,errors.New("PatchHub game index is empty")}
    shots,e:=ReadPatchHubMediaManifest(ctx,client)
    if e!=nil{return 0,e}
    for i:=range cache.Games{cache.Games[i].Screenshots=nil}
    AppendPatchHubMedia(&cache,shots)
    count:=0
    for _,g:=range cache.Games{count+=len(g.Screenshots)}
    if e=saveHubIndex(root,cache);e!=nil{return 0,e}
    return count,nil
}

// Media refresh is independent of the 24-hour game-card cache. A published
// JPG/PNG index must become visible as soon as the user selects Check + sync.
func SyncPatchHub(ctx context.Context, root string) (int, error) {
    hubSyncMu.Lock()
    defer hubSyncMu.Unlock()
    cache,cacheErr:=ReadHubCacheChecked(root)
    if cacheErr!=nil{return 0,cacheErr}
    if len(cache.Games)>0 && cache.GalleryScanVersion>=2 &&
        time.Since(cache.Updated)<24*time.Hour {
        // Missing internet/media index must not make the offline game list
        // unusable. Preserve the previous photo cache on request failure.
        _,_ = refreshCachedPatchHubMedia(ctx,root,nil)
        // Offline mode still uses the previous game and image metadata.
        return len(cache.Games),nil
    }
	req, err := http.NewRequestWithContext(ctx, "GET", PatchHubURL, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", "MagicPatchDesk/1.0 (offline-friendly game browser)")
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return 0, fmt.Errorf("Patch Hub returned %s", resp.Status)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 24<<20))
	if err != nil {
		return 0, err
	}
	hub, err := ParsePatchHubHTML(b)
	if err != nil {
		return 0, err
	}
    // Every photo published by the server is matched against the actual
    // game+region+version+Title ID; optional media index absence must not
    // make the ordinary game cover/catalogue sync fail.
    if listed,e:=ReadPatchHubMediaManifest(ctx,nil);e==nil{
        AppendPatchHubMedia(&hub,listed)
    }
    if err=saveHubIndex(root,hub);err!=nil{return 0,err}
	return len(hub.Games), nil
}
func CoverCachePath(root, urlString string) string {
	digest := sha256.Sum256([]byte(urlString))
	return filepath.Join(root, "Covers", hex.EncodeToString(digest[:])[:24]+".jpg")
}

// A PatchHub URL can redirect between the apex domain and its www alias.
// Never follow arbitrary redirects: remote URLs come from a parsed public page.
func trustedPatchHubURL(u *url.URL) bool {
	return u != nil && u.Scheme == "https" && u.User == nil && (strings.EqualFold(u.Hostname(), "www.jayxbox.com") || strings.EqualFold(u.Hostname(), "jayxbox.com"))
}

// Validate the actual downloaded image, not just HTTP status and byte count.
// Otherwise an HTML error response can be cached forever and GDI+ cannot load it.
func validateCoverBytes(data []byte) error {
	if len(data) < 32 {
		return errors.New("cover is empty or too small")
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || (format != "jpeg" && format != "png" && format != "gif") {
		return errors.New("download was not a supported JPEG, PNG or GIF image")
	}
	if cfg.Width < 2 || cfg.Height < 2 || cfg.Width > 8192 || cfg.Height > 8192 {
		return errors.New("invalid cover dimensions")
	}
	return nil
}

// ValidCoverFile also detects corrupt pre-existing cache entries created by older
// builds (which did not verify the payload before writing).
func ValidCoverFile(path string) bool {
	data, err := os.ReadFile(path)
	return err == nil && validateCoverBytes(data) == nil
}

func DownloadCover(ctx context.Context, root, source string) (string, error) {
	u, err := url.Parse(source)
	if err != nil || !trustedPatchHubURL(u) {
		return "", errors.New("untrusted cover URL")
	}
	path := CoverCachePath(root, source)
	if ValidCoverFile(path) {
		return path, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, "GET", source, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "image/jpeg,image/png,image/gif;q=0.9")
	req.Header.Set("User-Agent", "MagicPatchDesk/1.0 (PatchHub artwork cache)")
	client := &http.Client{Timeout: 20 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) > 5 || !trustedPatchHubURL(req.URL) {
			return errors.New("untrusted cover redirect")
		}
		return nil
	}}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("cover download: %s", resp.Status)
	}
	if resp.ContentLength > 5<<20 {
		return "", errors.New("cover too large")
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, (5<<20)+1))
	if err != nil {
		return "", err
	}
	if len(data) > 5<<20 {
		return "", errors.New("cover exceeds 5 MiB")
	}
	if err = validateCoverBytes(data); err != nil {
		return "", err
	}
	f, err := os.CreateTemp(filepath.Dir(path), "cover-*.tmp")
	if err != nil {
		return "", err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(data); err != nil {
		f.Close()
		return "", err
	}
	if err = f.Close(); err != nil {
		return "", err
	}
	return path, os.Rename(f.Name(), path)
}
