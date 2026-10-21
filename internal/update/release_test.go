package update

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseReleaseAndSelectAssets(t *testing.T) {
	payload := `{
		"tag_name": "v0.2.1",
		"draft": false,
		"prerelease": false,
		"assets": [
			{"name":"better-rpt_0.2.1-1_amd64.deb","size":10,"browser_download_url":"https://github.com/a.deb"},
			{"name":"BetterRPT-0.2.1-x86_64.AppImage","size":20,"browser_download_url":"https://github.com/a.AppImage"},
			{"name":"SHA256SUMS","size":30,"browser_download_url":"https://github.com/SHA256SUMS"},
			{"name":"BetterRPT-0.2.1-aarch64.AppImage","size":40,"browser_download_url":"https://github.com/arm.AppImage"}
		]
	}`
	release, err := parseRelease([]byte(payload))
	if err != nil {
		t.Fatal(err)
	}
	if release.Tag != "v0.2.1" || release.Version.String() != "0.2.1" {
		t.Fatalf("release = %+v", release)
	}
	appImage, err := selectAsset(release, KindAppImage, "amd64")
	if err != nil {
		t.Fatal(err)
	}
	if appImage.Name != "BetterRPT-0.2.1-x86_64.AppImage" {
		t.Fatalf("AppImage = %q", appImage.Name)
	}
	deb, err := selectAsset(release, KindDeb, "amd64")
	if err != nil {
		t.Fatal(err)
	}
	if deb.Name != "better-rpt_0.2.1-1_amd64.deb" {
		t.Fatalf("deb = %q", deb.Name)
	}
	checksums, err := selectChecksums(release)
	if err != nil {
		t.Fatal(err)
	}
	if checksums.Name != checksumName {
		t.Fatalf("checksums = %q", checksums.Name)
	}
	if _, err := selectAsset(release, KindAppImage, "arm64"); !errors.Is(err, ErrNoAsset) {
		t.Fatalf("arm64 error = %v, want ErrNoAsset", err)
	}
}

func TestSelectAssetRejectsMissingAndAmbiguous(t *testing.T) {
	release := Release{Assets: []Asset{
		{Name: "one-x86_64.AppImage", URL: "https://example.test/one"},
		{Name: "two-x86_64.AppImage", URL: "https://example.test/two"},
	}}
	if _, err := selectAsset(release, KindAppImage, "amd64"); !errors.Is(err, ErrNoAsset) {
		t.Fatalf("ambiguous error = %v, want ErrNoAsset", err)
	}
	if _, err := selectAsset(Release{}, KindDeb, "amd64"); !errors.Is(err, ErrNoAsset) {
		t.Fatalf("missing error = %v, want ErrNoAsset", err)
	}
	if _, err := selectAsset(release, KindDevelopment, "amd64"); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("development error = %v, want ErrUnsupported", err)
	}
}

func TestParseChecksums(t *testing.T) {
	payload := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef  file.AppImage\n" +
		"abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789 *other.deb\n"
	sums, err := parseChecksums([]byte(payload))
	if err != nil {
		t.Fatal(err)
	}
	if len(sums) != 2 {
		t.Fatalf("sums = %v", sums)
	}
	if sums["other.deb"] != "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789" {
		t.Fatalf("other = %q", sums["other.deb"])
	}
	for _, malformed := range []string{"", "abc file", "zzzz file", "abc file extra"} {
		if _, err := parseChecksums([]byte(malformed)); err == nil {
			t.Errorf("parseChecksums(%q) succeeded", malformed)
		}
	}
}

func TestVerifyFileDetectsMismatch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "asset")
	if err := os.WriteFile(path, []byte("new content"), 0o600); err != nil {
		t.Fatal(err)
	}
	expected := sha256.Sum256([]byte("old content"))
	if err := verifyFile(path, hex.EncodeToString(expected[:])); !errors.Is(err, ErrChecksum) {
		t.Fatalf("verifyFile error = %v, want ErrChecksum", err)
	}
	matching := sha256.Sum256([]byte("new content"))
	if err := verifyFile(path, strings.ToUpper(hex.EncodeToString(matching[:]))); err != nil {
		t.Fatalf("verifyFile matching error = %v", err)
	}
}
