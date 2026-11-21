package core

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseCompareAndChannels(t *testing.T) {
	a, _ := ParseReleaseVersion("v0.11.0-test.2")
	b, _ := ParseReleaseVersion("v0.12.0-test.1")
	if a.Compare(b) >= 0 {
		t.Fatal("version comparison")
	}
	c, _ := ParseReleaseVersion("v0.12.0")
	if c.Compare(b) <= 0 {
		t.Fatal("stable release should outrank prerelease")
	}
	releases := []Release{{TagName: "v0.11.0", Prerelease: false}, {TagName: "v0.12.0-test.1", Prerelease: true, Assets: []ReleaseAsset{{Name: StableReleaseAsset, State: "uploaded", Size: 3, BrowserDownloadURL: "https://github.com/" + ReleaseRepo + "/releases/download/v0.12.0-test.1/" + StableReleaseAsset}, {Name: "SHA256SUMS.txt", State: "uploaded", Size: 80, BrowserDownloadURL: "https://github.com/" + ReleaseRepo + "/releases/download/v0.12.0-test.1/SHA256SUMS.txt"}}}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { json.NewEncoder(w).Encode(releases) }))
	defer srv.Close()
	_, new, err := CheckForUpdate(context.Background(), srv.Client(), srv.URL, "v0.11.0", false)
	if err != nil || new {
		t.Fatalf("stable should be current: %v %v", new, err)
	}
	rel, new, err := CheckForUpdate(context.Background(), srv.Client(), srv.URL, "v0.11.0", true)
	if err != nil || !new || rel.TagName != "v0.12.0-test.1" {
		t.Fatalf("beta update %v %v %+v", new, err, rel)
	}
}
func TestReleaseUpdateContract(t *testing.T) {
	// This test is a non-negotiable forward-compatibility guard for release CI.
	if UpdateProtocol != 1 || StableReleaseAsset != "MagicPatchDesk-Windows-x64.exe" || AppVersion == "" {
		t.Fatal("update protocol removed or changed")
	}
	if _, err := ParseReleaseVersion(AppVersion); err != nil {
		t.Fatal(err)
	}
}
func TestChecksumsAndSafePaths(t *testing.T) {
	data := []byte("fake PE test bytes")
	sum := sha256.Sum256(data)
	digest := hex.EncodeToString(sum[:])
	p := filepath.Join(t.TempDir(), "app.exe")
	if e := os.WriteFile(p, data, 0600); e != nil {
		t.Fatal(e)
	}
	if e := ValidateDownloadedDigest(p, digest); e != nil {
		t.Fatal(e)
	}
	if e := ValidateDownloadedDigest(p, strings.Repeat("0", 64)); e == nil {
		t.Fatal("tampered binary accepted")
	}
	want, e := ChecksumFromFile([]byte(digest+"  MagicPatchDesk-Windows-x64.exe\n"), StableReleaseAsset)
	if e != nil || want != digest {
		t.Fatalf("checksum %s %v", want, e)
	}
	if _, e := ChecksumFromFile([]byte(digest+"  x.exe\n"), StableReleaseAsset); e == nil {
		t.Fatal("missing checksum accepted")
	}
	t.Setenv("LOCALAPPDATA", t.TempDir())
	dir, e := InstalledApplicationFolder()
	if e != nil || filepath.Base(dir) != "MagicPatchDesk" {
		t.Fatal(dir, e)
	}
}
