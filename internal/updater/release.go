// internal/updater/release.go
package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

var githubAPIURL = "https://api.github.com/repos/%s/%s/releases/latest"

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

type githubRelease struct {
	TagName string        `json:"tag_name"`
	Assets  []githubAsset `json:"assets"`
	Body    string        `json:"body"`
}

type AssetInfo struct {
	Name string
	URL  string
	Size int64
}

type ReleaseInfo struct {
	Version   string
	Assets    []AssetInfo
	Changelog string
}

func FetchLatestRelease(ctx context.Context, owner, repo string) (*ReleaseInfo, error) {
	url := fmt.Sprintf(githubAPIURL, owner, repo)
	slog.Info("fetching latest release", "owner", owner, "repo", repo, "url", url)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create github api request: %w", err)
	}
	req.Header.Set("User-Agent", "QuantFlow-Updater/1.0")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("github api request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("github api status %d", resp.StatusCode)
	}

	var gh githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&gh); err != nil {
		return nil, fmt.Errorf("decode github release: %w", err)
	}

	rel := &ReleaseInfo{
		Version:   strings.TrimPrefix(gh.TagName, "v"),
		Changelog: gh.Body,
		Assets:    make([]AssetInfo, 0, len(gh.Assets)),
	}
	for _, a := range gh.Assets {
		rel.Assets = append(rel.Assets, AssetInfo{Name: a.Name, URL: a.BrowserDownloadURL, Size: a.Size})
	}
	slog.Info("release fetched successfully", "version", rel.Version, "assets", len(rel.Assets))
	return rel, nil
}

func MatchAsset(rel *ReleaseInfo, goos, goarch string) (*AssetInfo, error) {
	suffix := fmt.Sprintf("%s-%s.", goos, goarch)
	var candidate *AssetInfo
	for i, a := range rel.Assets {
		if strings.Contains(a.Name, suffix) && !strings.HasSuffix(a.Name, ".sha256") {
			candidate = &rel.Assets[i]
			break
		}
	}
	if candidate == nil {
		return nil, fmt.Errorf("no asset matching %s/%s in release %s", goos, goarch, rel.Version)
	}
	slog.Info("matched asset for platform", "goos", goos, "goarch", goarch, "asset", candidate.Name)
	return candidate, nil
}
