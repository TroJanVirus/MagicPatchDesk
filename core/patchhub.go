package core

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const PatchHubURL = "https://www.jayxbox.com/Retail-Game-Modification/PatchHub.php"

type HubVersion struct {
	TitleID string `json:"title_id"`
	Region  string `json:"region"`
	Version string `json:"version"`
}
type HubGame struct {
	Title    string       `json:"title"`
	Image    string       `json:"image"`
	Versions []HubVersion `json:"versions,omitempty"`
}
type HubIndex struct {
	Updated time.Time `json:"updated"`
	Games   []HubGame `json:"games"`
}

var hubCard = regexp.MustCompile(`(?is)<div\s+class="game-card"[^>]*>`)
var hubImage = regexp.MustCompile(`(?is)<img[^>]*\bdata-src="([^"]+)"`)
var hubTitle = regexp.MustCompile(`(?is)<div\s+class="game-title">([^<]+)</div>`)
var hubVersion = regexp.MustCompile(`(?is)Title ID:\s*([A-Fa-f0-9]{8})\s*\|\s*Region:\s*([^|<]+)\|\s*Version:\s*([^<]+)`)

func ParsePatchHubHTML(doc []byte) (HubIndex, error) {
	contents := string(doc)
	locations := hubCard.FindAllStringIndex(contents, -1)
	result := HubIndex{Updated: time.Now().UTC()}
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
				if strings.EqualFold(abs.Hostname(), "www.jayxbox.com") && abs.Scheme == "https" {
					g.Image = abs.String()
				}
			}
		}
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
func ReadHubCache(root string) HubIndex {
	var c HubIndex
	b, err := os.ReadFile(filepath.Join(root, "patchhub-index.json"))
	if err == nil {
		_ = json.Unmarshal(b, &c)
	}
	return c
}
func SyncPatchHub(ctx context.Context, root string) (int, error) {
	cache := ReadHubCache(root)
	if len(cache.Games) > 0 && time.Since(cache.Updated) < 24*time.Hour {
		return len(cache.Games), nil
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
	j, err := json.Marshal(hub)
	if err != nil {
		return 0, err
	}
	tmp, err := os.CreateTemp(root, "hub-index-*.json")
	if err != nil {
		return 0, err
	}
	defer os.Remove(tmp.Name())
	if _, err = tmp.Write(j); err != nil {
		tmp.Close()
		return 0, err
	}
	if err = tmp.Close(); err != nil {
		return 0, err
	}
	if err = os.Rename(tmp.Name(), filepath.Join(root, "patchhub-index.json")); err != nil {
		return 0, err
	}
	return len(hub.Games), nil
}
func CoverCachePath(root, urlString string) string {
	digest := sha256.Sum256([]byte(urlString))
	return filepath.Join(root, "Covers", hex.EncodeToString(digest[:])[:24]+".img")
}
func DownloadCover(ctx context.Context, root, source string) (string, error) {
	u, e := url.Parse(source)
	if e != nil || u.Scheme != "https" || !strings.EqualFold(u.Hostname(), "www.jayxbox.com") {
		return "", errors.New("untrusted cover URL")
	}
	path := CoverCachePath(root, source)
	if st, err := os.Stat(path); err == nil && st.Size() > 0 {
		return path, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, "GET", source, nil)
	if err != nil {
		return "", err
	}
	client := &http.Client{Timeout: 20 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if !strings.EqualFold(req.URL.Hostname(), "www.jayxbox.com") {
			return errors.New("redirect to another host")
		}
		return nil
	}}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("cover download: %s", resp.Status)
	}
	if resp.ContentLength > 5<<20 {
		return "", errors.New("cover too large")
	}
	f, err := os.CreateTemp(filepath.Dir(path), "cover-*.tmp")
	if err != nil {
		return "", err
	}
	defer os.Remove(f.Name())
	n, err := io.Copy(f, io.LimitReader(resp.Body, (5<<20)+1))
	if err != nil {
		f.Close()
		return "", err
	}
	if err = f.Close(); err != nil {
		return "", err
	}
	if n > (5<<20) || n < 16 {
		return "", errors.New("invalid cover size")
	}
	return path, os.Rename(f.Name(), path)
}
