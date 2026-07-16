package updater

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type UpdateInfo struct {
	HasUpdate      bool   `json:"has_update"`
	CurrentVersion string `json:"current_version"`
	LatestVersion  string `json:"latest_version"`
	AssetURL       string `json:"asset_url"`
	AssetSize      int64  `json:"asset_size"`
	Checksum       string `json:"checksum"`
	Changelog      string `json:"changelog"`
}

type UpdateProgress struct {
	Downloaded int64
	Total      int64
}

type Updater struct {
	Owner string
	Repo  string
}

func New(owner, repo string) *Updater {
	return &Updater{Owner: owner, Repo: repo}
}

func (u *Updater) Check(ctx context.Context, currentVersion string) (*UpdateInfo, error) {
	rel, err := FetchLatestRelease(ctx, u.Owner, u.Repo)
	if err != nil {
		return nil, fmt.Errorf("fetch latest release: %w", err)
	}

	info := &UpdateInfo{
		CurrentVersion: currentVersion,
		LatestVersion:  rel.Version,
		Changelog:      rel.Changelog,
	}

	if !u.isNewer(currentVersion, rel.Version) {
		info.HasUpdate = false
		return info, nil
	}

	goos := goos()
	goarch := goarch()
	asset, err := MatchAsset(rel, goos, goarch)
	if err != nil {
		return nil, fmt.Errorf("match asset for %s/%s: %w", goos, goarch, err)
	}

	info.HasUpdate = true
	info.AssetURL = asset.URL
	info.AssetSize = asset.Size

	checksum, err := u.fetchChecksum(ctx, asset.Name+".sha256")
	if err != nil {
		slog.Warn("failed to fetch checksum, continuing without verification", "asset", asset.Name, "error", err)
	} else {
		info.Checksum = checksum
	}

	return info, nil
}

func (u *Updater) fetchChecksum(ctx context.Context, filename string) (string, error) {
	rel, err := FetchLatestRelease(ctx, u.Owner, u.Repo)
	if err != nil {
		return "", err
	}
	for _, a := range rel.Assets {
		if a.Name == filename {
			client := &http.Client{Timeout: 10 * time.Second}
			resp, err := client.Get(a.URL)
			if err != nil {
				return "", fmt.Errorf("fetch checksum: %w", err)
			}
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return "", fmt.Errorf("read checksum: %w", err)
			}
			parts := strings.Fields(string(body))
			if len(parts) > 0 {
				return parts[0], nil
			}
			return "", fmt.Errorf("empty checksum file")
		}
	}
	return "", fmt.Errorf("checksum file %s not found", filename)
}

func (u *Updater) Download(ctx context.Context, url, destDir string, progressCh chan<- int64) (string, error) {
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", fmt.Errorf("create download dir: %w", err)
	}

	client := &http.Client{Timeout: 30 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("download request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("download status %d", resp.StatusCode)
	}

	filename := filepath.Base(url)
	destPath := filepath.Join(destDir, filename)
	f, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	total := resp.ContentLength
	downloaded := int64(0)
	buf := make([]byte, 32*1024)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := f.Write(buf[:n]); writeErr != nil {
				return "", fmt.Errorf("write download: %w", writeErr)
			}
			downloaded += int64(n)
			if progressCh != nil && total > 0 {
				progressCh <- downloaded * 100 / total
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return "", fmt.Errorf("read download: %w", readErr)
		}
	}
	if progressCh != nil {
		close(progressCh)
	}
	return destPath, nil
}

func (u *Updater) Verify(path, expectedChecksum string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open for checksum: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("checksum read: %w", err)
	}
	got := hex.EncodeToString(h.Sum(nil))
	if got != expectedChecksum {
		os.Remove(path)
		return fmt.Errorf("checksum mismatch: got %s, expected %s", got, expectedChecksum)
	}
	return nil
}

func (u *Updater) isNewer(current, latest string) bool {
	cur := parseVersion(current)
	lat := parseVersion(latest)
	if cur.year != lat.year {
		return lat.year > cur.year
	}
	if cur.month != lat.month {
		return lat.month > cur.month
	}
	return lat.day > cur.day
}

type version struct {
	year, month, day int
}

func parseVersion(v string) version {
	parts := strings.Split(v, ".")
	if len(parts) < 3 {
		return version{}
	}
	year, _ := strconv.Atoi(parts[0])
	month, _ := strconv.Atoi(parts[1])
	day, _ := strconv.Atoi(parts[2])
	return version{year: year, month: month, day: day}
}

func goos() string {
	return map[string]string{
		"darwin":  "darwin",
		"linux":   "linux",
		"windows": "windows",
	}[osGoos()]
}

func goarch() string {
	return map[string]string{
		"arm64": "arm64",
		"amd64": "amd64",
	}[osGoarch()]
}

// osGoos and osGoarch are overridden in platform-specific files
var osGoos = func() string { return "unknown" }
var osGoarch = func() string { return "unknown" }
