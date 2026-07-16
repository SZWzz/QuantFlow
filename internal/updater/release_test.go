// internal/updater/release_test.go
package updater

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchLatestRelease(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/SZWzz/QuantFlow/releases/latest" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		rel := githubRelease{
			TagName: "v2026.7.20",
			Assets: []githubAsset{
				{Name: "quantflow-darwin-arm64.zip", BrowserDownloadURL: "https://example.com/q-darwin-arm64.zip", Size: 1000},
				{Name: "quantflow-darwin-arm64.zip.sha256", BrowserDownloadURL: "https://example.com/q-darwin-arm64.zip.sha256", Size: 64},
				{Name: "quantflow-linux-amd64.tar.gz", BrowserDownloadURL: "https://example.com/q-linux-amd64.tar.gz", Size: 2000},
				{Name: "quantflow-windows-amd64.zip", BrowserDownloadURL: "https://example.com/q-win-amd64.zip", Size: 3000},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(rel)
	}))
	defer ts.Close()

	origURL := githubAPIURL
	githubAPIURL = ts.URL + "/repos/%s/%s/releases/latest"
	defer func() { githubAPIURL = origURL }()

	rel, err := FetchLatestRelease("SZWzz", "QuantFlow")
	if err != nil {
		t.Fatalf("FetchLatestRelease failed: %v", err)
	}
	if rel.Version != "2026.7.20" {
		t.Errorf("expected version 2026.7.20, got %s", rel.Version)
	}
	if len(rel.Assets) != 4 {
		t.Errorf("expected 4 assets, got %d", len(rel.Assets))
	}
}

func TestMatchAsset(t *testing.T) {
	rel := &ReleaseInfo{
		Version: "2026.7.20",
		Assets: []AssetInfo{
			{Name: "quantflow-darwin-arm64.zip", URL: "https://example.com/q-darwin-arm64.zip"},
			{Name: "quantflow-darwin-arm64.zip.sha256", URL: "https://example.com/q-darwin-arm64.zip.sha256"},
		},
	}
	asset, err := MatchAsset(rel, "darwin", "arm64")
	if err != nil {
		t.Fatalf("MatchAsset failed: %v", err)
	}
	if asset.Name != "quantflow-darwin-arm64.zip" {
		t.Errorf("expected quantflow-darwin-arm64.zip, got %s", asset.Name)
	}

	_, err = MatchAsset(rel, "linux", "amd64")
	if err == nil {
		t.Error("expected error for missing asset")
	}
}
