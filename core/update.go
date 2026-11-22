package core

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Keep this as an invariant in every future release. Both the portable EXE and
// installed program use exactly the same update protocol.
const AppVersion = "v0.14.0-test.6"
const UpdateProtocol = 1
const StableReleaseAsset = "MagicPatchDesk-Windows-x64.exe"
const ReleaseRepo = "TroJanVirus/MagicPatchDesk"
const GitHubReleaseAPI = "https://api.github.com/repos/" + ReleaseRepo + "/releases?per_page=100"
const updaterMaxSize int64 = 80 << 20

var versionPattern = regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)(?:-([a-z0-9][a-z0-9.-]*))?$`)
var checksumPattern = regexp.MustCompile(`^[a-fA-F0-9]{64}$`)

type Version struct {
	Major, Minor, Patch int
	Pre                 string
}

func ParseReleaseVersion(s string) (Version, error) {
	m := versionPattern.FindStringSubmatch(strings.ToLower(s))
	if m == nil {
		return Version{}, fmt.Errorf("unsupported release version %q", s)
	}
	v := Version{Pre: m[4]}
	var err error
	v.Major, err = strconv.Atoi(m[1])
	if err != nil {
		return v, err
	}
	v.Minor, err = strconv.Atoi(m[2])
	if err != nil {
		return v, err
	}
	v.Patch, err = strconv.Atoi(m[3])
	return v, err
}
func comparePrerelease(a, b string) int {
	if a == b {
		return 0
	}
	if a == "" {
		return 1
	}
	if b == "" {
		return -1
	}
	aa, bb := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < len(aa) && i < len(bb); i++ {
		ai, ae := strconv.ParseUint(aa[i], 10, 64)
		bi, be := strconv.ParseUint(bb[i], 10, 64)
		switch {
		case ae == nil && be == nil:
			if ai < bi {
				return -1
			}
			if ai > bi {
				return 1
			}
		case ae == nil && be != nil:
			return -1
		case ae != nil && be == nil:
			return 1
		default:
			if aa[i] < bb[i] {
				return -1
			}
			if aa[i] > bb[i] {
				return 1
			}
		}
	}
	if len(aa) < len(bb) {
		return -1
	}
	return 1
}
func (v Version) Compare(w Version) int {
	for _, p := range [][2]int{{v.Major, w.Major}, {v.Minor, w.Minor}, {v.Patch, w.Patch}} {
		if p[0] < p[1] {
			return -1
		}
		if p[0] > p[1] {
			return 1
		}
	}
	return comparePrerelease(v.Pre, w.Pre)
}

type ReleaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Digest             string `json:"digest"`
	Size               int64  `json:"size"`
	State              string `json:"state"`
}
type Release struct {
	TagName    string         `json:"tag_name"`
	Name       string         `json:"name"`
	Draft      bool           `json:"draft"`
	Prerelease bool           `json:"prerelease"`
	HTMLURL    string         `json:"html_url"`
	Assets     []ReleaseAsset `json:"assets"`
	Version    Version        `json:"-"`
	Executable ReleaseAsset   `json:"-"`
	Checksums  ReleaseAsset   `json:"-"`
}

func isCanonicalReleaseURL(value, tag, asset string) bool {
	u, err := url.Parse(value)
	if err != nil || u.Scheme != "https" || u.Host != "github.com" || u.RawQuery != "" || u.Fragment != "" {
		return false
	}
	return u.EscapedPath() == "/"+ReleaseRepo+"/releases/download/"+url.PathEscape(tag)+"/"+url.PathEscape(asset)
}
func resolveUpdateAssets(r *Release) error {
	for _, a := range r.Assets {
		if a.State != "uploaded" || !isCanonicalReleaseURL(a.BrowserDownloadURL, r.TagName, a.Name) {
			continue
		}
		if a.Name == StableReleaseAsset {
			r.Executable = a
		}
		if a.Name == "SHA256SUMS.txt" {
			r.Checksums = a
		}
	}
	if r.Executable.Name == "" { // Backward-compatible with historical v0.5-v0.11 build names.
		for _, a := range r.Assets {
			if a.State != "uploaded" || !isCanonicalReleaseURL(a.BrowserDownloadURL, r.TagName, a.Name) {
				continue
			}
			if strings.HasPrefix(a.Name, "MagicPatchDesk-v") && strings.HasSuffix(a.Name, "-Windows-x64.exe") {
				r.Executable = a
				break
			}
		}
	}
	if r.Executable.Name == "" || r.Checksums.Name == "" {
		return errors.New("latest release is missing its Windows EXE or SHA256SUMS.txt; do not update")
	}
	if r.Executable.Size <= 0 || r.Executable.Size > updaterMaxSize {
		return errors.New("release executable size is outside safety limit")
	}
	if r.Checksums.Size <= 0 || r.Checksums.Size > 16<<10 {
		return errors.New("release checksum file is outside safety limit")
	}
	if r.Executable.Digest != "" && (!strings.HasPrefix(r.Executable.Digest, "sha256:") || !checksumPattern.MatchString(strings.TrimPrefix(r.Executable.Digest, "sha256:"))) {
		return errors.New("invalid GitHub release asset digest")
	}
	return nil
}
func CheckForUpdate(ctx context.Context, client *http.Client, baseURL, current string, allowPrerelease bool) (Release, bool, error) {
	if client == nil {
		client = &http.Client{Timeout: 25 * time.Second}
	}
	if baseURL == "" {
		baseURL = GitHubReleaseAPI
	}
	cur, err := ParseReleaseVersion(current)
	if err != nil {
		return Release{}, false, err
	}
	req, err := http.NewRequestWithContext(ctx, "GET", baseURL, nil)
	if err != nil {
		return Release{}, false, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "MagicPatchDesk/"+AppVersion)
	resp, err := client.Do(req)
	if err != nil {
		return Release{}, false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return Release{}, false, fmt.Errorf("GitHub releases HTTP %d", resp.StatusCode)
	}
	var releases []Release
	if err = json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&releases); err != nil {
		return Release{}, false, err
	}
	var latest Release
	have := false
	for _, r := range releases {
		if r.Draft || (r.Prerelease && !allowPrerelease) {
			continue
		}
		v, e := ParseReleaseVersion(r.TagName)
		if e != nil {
			continue
		}
		r.Version = v
		if !have || v.Compare(latest.Version) > 0 {
			latest = r
			have = true
		}
	}
	if !have {
		return Release{}, false, errors.New("no compatible GitHub release found in selected channel")
	}
	if latest.Version.Compare(cur) <= 0 {
		return latest, false, nil
	}
	if err = resolveUpdateAssets(&latest); err != nil {
		return latest, false, err
	}
	return latest, true, nil
}
func ChecksumFromFile(data []byte, name string) (string, error) {
	if len(data) > 16<<10 {
		return "", errors.New("checksum file too large")
	}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	seen := false
	match := ""
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		if !checksumPattern.MatchString(fields[0]) {
			continue
		}
		if strings.TrimPrefix(fields[1], "*") == name {
			if seen {
				return "", errors.New("duplicate checksum entry")
			}
			seen = true
			match = strings.ToLower(fields[0])
		}
	}
	if scanner.Err() != nil {
		return "", scanner.Err()
	}
	if !seen {
		return "", fmt.Errorf("no checksum listed for %s", name)
	}
	return match, nil
}
func ValidateDownloadedDigest(path, want string) error {
	if !checksumPattern.MatchString(want) {
		return errors.New("invalid SHA256 checksum")
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, io.LimitReader(f, updaterMaxSize+1))
	if err != nil {
		return err
	}
	if n > updaterMaxSize {
		return errors.New("download exceeds file safety limit")
	}
	if !strings.EqualFold(hex.EncodeToString(h.Sum(nil)), want) {
		return errors.New("downloaded executable failed SHA256 verification")
	}
	return nil
}
func downloadLimited(ctx context.Context, c *http.Client, location string, max int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", location, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "MagicPatchDesk/"+AppVersion)
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("download HTTP %d", resp.StatusCode)
	}
	if resp.ContentLength > max {
		return nil, errors.New("download too large")
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, max+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > max {
		return nil, errors.New("download too large")
	}
	return b, nil
}
func checkedDownloadClient() *http.Client {
	return &http.Client{Timeout: 3 * time.Minute, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) > 6 {
			return errors.New("too many release redirects")
		}
		host := strings.ToLower(req.URL.Hostname())
		if req.URL.Scheme != "https" || (host != "github.com" && host != "release-assets.githubusercontent.com" && host != "objects.githubusercontent.com") {
			return errors.New("untrusted update redirect host")
		}
		return nil
	}}
}

// FetchVerifiedUpdate stages the new portable EXE without touching the running
// program. It requires both the release SHA256SUMS asset and the GitHub API
// digest (if GitHub provides one) to agree with the downloaded file.
func FetchVerifiedUpdate(ctx context.Context, r Release, stagingDir string) (string, string, error) {
	if err := resolveUpdateAssets(&r); err != nil {
		return "", "", err
	}
	c := checkedDownloadClient()
	sums, err := downloadLimited(ctx, c, r.Checksums.BrowserDownloadURL, 16<<10)
	if err != nil {
		return "", "", err
	}
	want, err := ChecksumFromFile(sums, r.Executable.Name)
	if err != nil {
		return "", "", err
	}
	if r.Executable.Digest != "" && strings.TrimPrefix(r.Executable.Digest, "sha256:") != want {
		return "", "", errors.New("GitHub asset digest and release checksums disagree")
	}
	if err = os.MkdirAll(stagingDir, 0700); err != nil {
		return "", "", err
	}
	f, err := os.CreateTemp(stagingDir, ".mpd-update-*.exe")
	if err != nil {
		return "", "", err
	}
	path := f.Name()
	okay := false
	defer func() {
		f.Close()
		if !okay {
			os.Remove(path)
		}
	}()
	req, err := http.NewRequestWithContext(ctx, "GET", r.Executable.BrowserDownloadURL, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("User-Agent", "MagicPatchDesk/"+AppVersion)
	resp, err := c.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", "", fmt.Errorf("download HTTP %d", resp.StatusCode)
	}
	if resp.ContentLength > updaterMaxSize {
		return "", "", errors.New("download too large")
	}
	h := sha256.New()
	written, err := io.Copy(io.MultiWriter(f, h), io.LimitReader(resp.Body, updaterMaxSize+1))
	if err != nil {
		return "", "", err
	}
	if written > updaterMaxSize {
		return "", "", errors.New("download too large")
	}
	if hex.EncodeToString(h.Sum(nil)) != want {
		return "", "", errors.New("download SHA256 mismatch")
	}
	if err = f.Sync(); err != nil {
		return "", "", err
	}
	if err = f.Close(); err != nil {
		return "", "", err
	}
	okay = true
	return path, want, nil
}
func InstalledApplicationFolder() (string, error) {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		return "", errors.New("LOCALAPPDATA unavailable; cannot determine private install location")
	}
	return filepath.Join(base, "Programs", "MagicPatchDesk"), nil
}
func IsInstalledPath(path string) bool {
	folder, err := InstalledApplicationFolder()
	if err != nil {
		return false
	}
	return strings.EqualFold(filepath.Clean(path), filepath.Join(folder, "MagicPatchDesk.exe"))
}
