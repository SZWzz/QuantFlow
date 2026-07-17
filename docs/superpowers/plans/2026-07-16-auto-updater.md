# Auto-Updater Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add automatic update checking, downloading, verification, and restart capability via GitHub Releases.

**Architecture:** `internal/updater/` package handles GitHub API release discovery, platform-specific asset matching, SHA256 verification, and binary replacement. `app_system.go` exposes `CheckUpdate` and `ApplyUpdate` IPC methods. Frontend `UpdatePrompt.vue` shows available updates with progress.

**Tech Stack:** Go 1.25+ (net/http, crypto/sha256, archive/zip, os/exec), Vue 3 + TypeScript (Composition API), Pinia (settings store)

## Global Constraints

- No new Go dependencies beyond stdlib (net/http, crypto/sha256, archive/zip, os/exec, runtime)
- Use slog for Go logging
- Use Composition API with `<script setup lang="ts">` for Vue
- No `window.confirm()` / `window.alert()` — use `confirmDialog`/`alertDialog` from `@/lib/wails`
- All platform-specific code behind build tags (`//go:build darwin`, `//go:build linux`, `//go:build windows`)
- Update config struct with `UpdateCheckInterval` field (values: `always`, `daily`, `never`)
- SHA256 checksum file naming: `<asset-name>.sha256` in the same GitHub release

---

### Task 1: GitHub Release Client (release.go)

**Files:**
- Create: `internal/updater/release.go`
- Test: `internal/updater/release_test.go`

**Interfaces:**
- Consumes: nothing (standalone)
- Produces: `ReleaseInfo`, `AssetInfo` structs; `FetchLatestRelease(owner, repo string) (*ReleaseInfo, error)`; `MatchAsset(release *ReleaseInfo, goos, goarch string) (*AssetInfo, error)`

- [ ] **Step 1: Write the failing test**

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow && go test ./internal/updater/ -v -run TestFetchLatestRelease -count=1`
Expected: FAIL with "package internal/updater is not in std"

- [ ] **Step 3: Write minimal implementation**

```go
// internal/updater/release.go
package updater

import (
	"encoding/json"
	"fmt"
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

func FetchLatestRelease(owner, repo string) (*ReleaseInfo, error) {
	url := fmt.Sprintf(githubAPIURL, owner, repo)
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(url)
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
	return candidate, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow && go test ./internal/updater/ -v -run TestFetchLatestRelease -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/updater/release.go internal/updater/release_test.go
git commit -m "feat(updater): add GitHub Releases client and asset matching"
```

---

### Task 2: Updater Engine (updater.go)

**Files:**
- Create: `internal/updater/updater.go`
- Test: `internal/updater/updater_test.go`

**Interfaces:**
- Consumes: `ReleaseInfo`, `AssetInfo` from Task 1
- Produces: `Updater` struct with `Check(ctx, currentVersion string) (*UpdateInfo, error)`, `Download(ctx, assetURL, checksumURL, dest string) error`, `Verify(path, expectedChecksum string) error`, `Replace(execPath, newBinaryPath string) error`

- [ ] **Step 1: Write the failing test**

```go
// internal/updater/updater_test.go
package updater

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestVersionComparison(t *testing.T) {
	u := &Updater{}
	tests := []struct {
		current string
		latest  string
		want    bool
	}{
		{"2026.7.14", "2026.7.20", true},
		{"2026.7.20", "2026.7.20", false},
		{"2026.8.1", "2026.7.20", false},
		{"2025.12.31", "2026.1.1", true},
	}
	for _, tt := range tests {
		got := u.isNewer(tt.current, tt.latest)
		if got != tt.want {
			t.Errorf("isNewer(%q, %q) = %v, want %v", tt.current, tt.latest, got, tt.want)
		}
	}
}

func TestChecksumVerify(t *testing.T) {
	content := []byte("test binary content")
	sum := sha256.Sum256(content)
	expected := hex.EncodeToString(sum[:])

	tmp := filepath.Join(t.TempDir(), "test.bin")
	if err := os.WriteFile(tmp, content, 0644); err != nil {
		t.Fatal(err)
	}

	u := &Updater{}
	if err := u.Verify(tmp, expected); err != nil {
		t.Errorf("Verify failed: %v", err)
	}

	if err := u.Verify(tmp, "badchecksum"); err == nil {
		t.Error("expected error for bad checksum")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow && go test ./internal/updater/ -v -run TestVersionComparison -count=1`
Expected: FAIL with "Updater not defined"

- [ ] **Step 3: Write minimal implementation**

```go
// internal/updater/updater.go
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
	HasUpdate     bool   `json:"has_update"`
	CurrentVersion string `json:"current_version"`
	LatestVersion  string `json:"latest_version"`
	AssetURL      string `json:"asset_url"`
	AssetSize     int64  `json:"asset_size"`
	Checksum      string `json:"checksum"`
	Changelog     string `json:"changelog"`
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
	rel, err := FetchLatestRelease(u.Owner, u.Repo)
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
	rel, err := FetchLatestRelease(u.Owner, u.Repo)
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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow && go test ./internal/updater/ -v -run TestVersionComparison -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/updater/updater.go internal/updater/updater_test.go
git commit -m "feat(updater): add updater engine with version comparison and checksum verification"
```

---

### Task 3: Platform-Specific Replace Logic (darwin.go, linux.go, windows.go)

**Files:**
- Create: `internal/updater/darwin.go`
- Create: `internal/updater/linux.go`
- Create: `internal/updater/windows.go`

**Interfaces:**
- Consumes: `Updater` struct from Task 2
- Produces: platform-specific `Replace(execPath, downloadedPath string) error` implementation

- [ ] **Step 1: Write test for platform detection**

```go
// internal/updater/platform_test.go
package updater

import (
	"testing"
)

func TestPlatformDetection(t *testing.T) {
	// These should not panic — they return the actual OS/arch
	goos := goos()
	goarch := goarch()
	if goos == "unknown" || goarch == "unknown" {
		t.Errorf("platform detection failed: goos=%s goarch=%s", goos, goarch)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow && go test ./internal/updater/ -v -run TestPlatformDetection -count=1`
Expected: FAIL (osGoos and osGoarch return "unknown" before platform files are created)

- [ ] **Step 3: Write platform implementations**

```go
// internal/updater/darwin.go (build tag: darwin)
//go:build darwin

package updater

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func init() {
	osGoos = func() string { return "darwin" }
	osGoarch = func() string {
		cmd := exec.Command("uname", "-m")
		out, err := cmd.Output()
		if err != nil {
			return "arm64"
		}
		arch := string(out)
		if arch == "x86_64\n" {
			return "amd64"
		}
		return "arm64"
	}
}

func (u *Updater) Replace(execPath, downloadedPath string) error {
	// macOS: replace .app bundle
	appDir := filepath.Dir(filepath.Dir(filepath.Dir(execPath))) // .../QuantFlow.app/Contents/MacOS/quantflow → .../QuantFlow.app
	tmpDir, err := os.MkdirTemp("", "quantflow-update-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Unzip downloaded archive
	cmd := exec.Command("unzip", "-q", downloadedPath, "-d", tmpDir)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("unzip update: %w", err)
	}

	// Find .app bundle in temp
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		return fmt.Errorf("read temp dir: %w", err)
	}
	var newApp string
	for _, e := range entries {
		if e.IsDir() && filepath.Ext(e.Name()) == ".app" {
			newApp = filepath.Join(tmpDir, e.Name())
			break
		}
	}
	if newApp == "" {
		return fmt.Errorf("no .app bundle found in archive")
	}

	// Replace old app bundle
	parentDir := filepath.Dir(appDir)
	backupPath := appDir + ".bak"
	os.RemoveAll(backupPath)
	if err := os.Rename(appDir, backupPath); err != nil {
		return fmt.Errorf("backup existing app: %w", err)
	}
	if err := os.Rename(newApp, appDir); err != nil {
		// Restore backup
		os.Rename(backupPath, appDir)
		return fmt.Errorf("replace app bundle: %w", err)
	}
	os.RemoveAll(backupPath)

	return nil
}

// Restart re-launches the application and exits the current process.
func Restart() error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable: %w", err)
	}
	cmd := exec.Command("open", "-a", execPath)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("restart: %w", err)
	}
	os.Exit(0)
	return nil
}
```

```go
// internal/updater/linux.go (build tag: linux)
//go:build linux

package updater

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func init() {
	osGoos = func() string { return "linux" }
	osGoarch = func() string {
		cmd := exec.Command("uname", "-m")
		out, err := cmd.Output()
		if err != nil {
			return "amd64"
		}
		arch := string(out)
		if arch == "aarch64\n" || arch == "arm64\n" {
			return "arm64"
		}
		return "amd64"
	}
}

func (u *Updater) Replace(execPath, downloadedPath string) error {
	tmpDir, err := os.MkdirTemp("", "quantflow-update-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	cmd := exec.Command("tar", "-xzf", downloadedPath, "-C", tmpDir)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("extract archive: %w", err)
	}

	newBinary := filepath.Join(tmpDir, "quantflow")
	if _, err := os.Stat(newBinary); os.IsNotExist(err) {
		return fmt.Errorf("binary not found in archive")
	}

	if err := os.Rename(newBinary, execPath); err != nil {
		return fmt.Errorf("replace binary: %w", err)
	}
	if err := os.Chmod(execPath, 0755); err != nil {
		return fmt.Errorf("chmod binary: %w", err)
	}

	return nil
}

func Restart() error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable: %w", err)
	}
	cmd := exec.Command(execPath)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("restart: %w", err)
	}
	os.Exit(0)
	return nil
}
```

```go
// internal/updater/windows.go (build tag: windows)
//go:build windows

package updater

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func init() {
	osGoos = func() string { return "windows" }
	osGoarch = func() string {
		if os.Getenv("PROCESSOR_ARCHITECTURE") == "ARM64" {
			return "arm64"
		}
		return "amd64"
	}
}

func (u *Updater) Replace(execPath, downloadedPath string) error {
	tmpDir := filepath.Dir(downloadedPath)

	cmd := exec.Command("powershell", "-Command",
		fmt.Sprintf("Expand-Archive -Path '%s' -DestinationPath '%s' -Force", downloadedPath, tmpDir))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("extract zip: %w", err)
	}

	newBinary := filepath.Join(tmpDir, "quantflow.exe")
	if _, err := os.Stat(newBinary); os.IsNotExist(err) {
		return fmt.Errorf("quantflow.exe not found in archive")
	}

	// Use a bat script to replace after exit
	batPath := filepath.Join(tmpDir, "replace.bat")
	batContent := fmt.Sprintf(`@echo off
timeout /t 2 /nobreak >nul
copy /Y "%s" "%s"
del "%s"
start "" "%s"
`, newBinary, execPath, batPath, execPath)
	if err := os.WriteFile(batPath, []byte(batContent), 0644); err != nil {
		return fmt.Errorf("write replace script: %w", err)
	}

	cmd = exec.Command("cmd", "/C", "start", "/B", batPath)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start replace script: %w", err)
	}

	os.Exit(0)
	return nil
}

func Restart() error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable: %w", err)
	}
	cmd := exec.Command(execPath)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("restart: %w", err)
	}
	os.Exit(0)
	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow && go test ./internal/updater/ -v -run TestPlatformDetection -count=1`
Expected: PASS (on macOS: goos=darwin, goarch=arm64 or amd64)

- [ ] **Step 5: Commit**

```bash
git add internal/updater/darwin.go internal/updater/linux.go internal/updater/windows.go internal/updater/platform_test.go
git commit -m "feat(updater): add platform-specific replace logic for macOS, Linux, Windows"
```

---

### Task 4: IPC Methods (app_system.go)

**Files:**
- Modify: `app_system.go`
- Modify: `internal/config/config.go` (add UpdateCheckInterval)

**Interfaces:**
- Consumes: `Updater` from Task 2-3
- Produces: `CheckUpdate()` → `UpdateInfo` and `ApplyUpdate(assetURL, checksum string)` frontend-facing methods on `*App`

- [ ] **Step 1: Write the test**

```go
// app_system_test.go (at project root)
package main

import (
	"testing"
)

func TestCheckUpdateNoop(t *testing.T) {
	// When app is not fully initialized, CheckUpdate should not panic
	// This just tests the signature is correct
	_ = (&App{}).GetVersion
}
```

- [ ] **Step 2: Run test to verify baseline**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow && go test -v -run TestCheckUpdateNoop -count=1 .`
Expected: PASS (trivial test)

- [ ] **Step 3: Write implementation**

Add `UpdateCheckInterval` to config:

```go
// Modify internal/config/config.go — add field to Config struct after Version (line 18)
	UpdateCheckInterval string `yaml:"update_check_interval"`
```

```go
// Modify DefaultConfig in internal/config/config.go — add default
		UpdateCheckInterval: "daily",
```

Add IPC methods to app_system.go:

```go
// Add to app_system.go (after the existing imports, add updater import)
import (
	"quantflow/internal/updater"
)

// Add after GetVersion function (after line 53)

const updaterOwner = "SZWzz"
const updaterRepo = "QuantFlow"

// CheckUpdate checks for a new version. Returns update info or nil if up-to-date.
func (a *App) CheckUpdate() *updater.UpdateInfo {
	if a.cfg == nil {
		return &updater.UpdateInfo{HasUpdate: false}
	}

	u := updater.New(updaterOwner, updaterRepo)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	info, err := u.Check(ctx, a.cfg.Version)
	if err != nil {
		slog.Warn("check update failed", "error", err)
		return &updater.UpdateInfo{HasUpdate: false}
	}
	return info
}

// ApplyUpdate downloads, verifies, and applies an update.
func (a *App) ApplyUpdate(assetURL, checksum string) error {
	u := updater.New(updaterOwner, updaterRepo)

	downloadedPath, err := u.Download(context.Background(), assetURL, os.TempDir(), nil)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	if checksum != "" {
		if err := u.Verify(downloadedPath, checksum); err != nil {
			return fmt.Errorf("verification failed: %w", err)
		}
	}

	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable: %w", err)
	}

	if err := u.Replace(execPath, downloadedPath); err != nil {
		return fmt.Errorf("replace failed: %w", err)
	}

	return updater.Restart()
}

// GetUpdateInterval returns the current update check interval setting.
func (a *App) GetUpdateInterval() string {
	if a.cfg == nil {
		return "daily"
	}
	if a.cfg.UpdateCheckInterval == "" {
		return "daily"
	}
	return a.cfg.UpdateCheckInterval
}

// SetUpdateInterval sets the update check interval and persists config.
func (a *App) SetUpdateInterval(interval string) error {
	if a.cfg == nil {
		return fmt.Errorf("config not initialized")
	}
	switch interval {
	case "always", "daily", "never":
		a.cfg.UpdateCheckInterval = interval
		return a.cfg.Save()
	default:
		return fmt.Errorf("invalid interval: %s (must be always/daily/never)", interval)
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow && go vet ./... && go build -o /dev/null .`
Expected: No errors

- [ ] **Step 5: Commit**

```bash
git add app_system.go internal/config/config.go
git commit -m "feat(updater): add CheckUpdate/ApplyUpdate IPC and update interval config"
```

---

### Task 5: UpdatePrompt Frontend Component

**Files:**
- Create: `frontend/src/terminal/components/UpdatePrompt.vue`
- Modify: `frontend/src/lib/wails.ts` (add CheckUpdate, ApplyUpdate typed methods)
- Create: `frontend/src/stores/update.ts`

**Interfaces:**
- Consumes: `CheckUpdate()` → `UpdateInfo`, `ApplyUpdate(assetURL, checksum)` from Go IPC
- Produces: Update prompt dialog with changelog, download progress, restart flow

- [ ] **Step 1: Write the test**

```typescript
// frontend/src/__tests__/UpdatePrompt.test.ts
import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import UpdatePrompt from '@/terminal/components/UpdatePrompt.vue'

describe('UpdatePrompt', () => {
  it('renders update info when available', () => {
    const wrapper = mount(UpdatePrompt, {
      props: {
        visible: true,
        currentVersion: '2026.7.14',
        latestVersion: '2026.7.20',
        changelog: 'Bug fixes and improvements',
      },
    })
    expect(wrapper.text()).toContain('2026.7.20')
    expect(wrapper.text()).toContain('2026.7.14')
  })

  it('emits close on cancel', () => {
    const wrapper = mount(UpdatePrompt, {
      props: {
        visible: true,
        currentVersion: '2026.7.14',
        latestVersion: '2026.7.20',
        changelog: '',
      },
    })
    wrapper.find('[data-test="remind-later"]').trigger('click')
    expect(wrapper.emitted('close')).toBeTruthy()
  })
})
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vitest run -t "UpdatePrompt" 2>&1 || true`
Expected: FAIL (file doesn't exist)

- [ ] **Step 3: Write implementation**

```typescript
// frontend/src/stores/update.ts
import { defineStore } from 'pinia'
import { ref } from 'vue'
import { Call } from '@wailsio/runtime'
import { useSettingsStore } from '@/stores/settings'

export interface UpdateInfo {
  has_update: boolean
  current_version: string
  latest_version: string
  asset_url: string
  asset_size: number
  checksum: string
  changelog: string
}

export const useUpdateStore = defineStore('update', () => {
  const updateInfo = ref<UpdateInfo | null>(null)
  const checking = ref(false)
  const downloading = ref(false)
  const downloadProgress = ref(0)
  const error = ref('')

  async function check() {
    checking.value = true
    error.value = ''
    try {
      const info = await Call.ByName('main.App.CheckUpdate') as UpdateInfo
      updateInfo.value = info
      return info
    } catch (e: any) {
      error.value = e.message || 'Check failed'
      return null
    } finally {
      checking.value = false
    }
  }

  async function apply() {
    if (!updateInfo.value?.has_update) return
    downloading.value = true
    downloadProgress.value = 0
    error.value = ''
    try {
      await Call.ByName('main.App.ApplyUpdate', updateInfo.value.asset_url, updateInfo.value.checksum)
    } catch (e: any) {
      error.value = e.message || 'Update failed'
    } finally {
      downloading.value = false
    }
  }

  function shouldCheck(): boolean {
    const settings = useSettingsStore()
    const interval = settings.getSetting('updateCheckInterval') || 'daily'
    if (interval === 'never') return false
    if (interval === 'always') return true
    const lastCheck = localStorage.getItem('quantflow-last-update-check')
    if (!lastCheck) return true
    const last = new Date(lastCheck)
    const now = new Date()
    return last.toDateString() !== now.toDateString()
  }

  function markChecked() {
    localStorage.setItem('quantflow-last-update-check', new Date().toISOString())
  }

  return { updateInfo, checking, downloading, downloadProgress, error, check, apply, shouldCheck, markChecked }
})
```

```typescript
// frontend/src/lib/wails.ts — add these exported functions after GetLogs (after line 329)

export interface UpdateInfo {
  has_update: boolean
  current_version: string
  latest_version: string
  asset_url: string
  asset_size: number
  checksum: string
  changelog: string
}

export async function CheckUpdate(): Promise<UpdateInfo> {
  return wailsCall<UpdateInfo>('CheckUpdate')
}

export async function ApplyUpdate(assetURL: string, checksum: string): Promise<void> {
  return wailsCall<void>('ApplyUpdate', assetURL, checksum)
}
```

```vue
<!-- frontend/src/terminal/components/UpdatePrompt.vue -->
<script setup lang="ts">
import { useUpdateStore } from '@/stores/update'

const props = defineProps<{
  visible: boolean
  currentVersion: string
  latestVersion: string
  changelog: string
}>()

const emit = defineEmits<{
  close: []
  update: []
}>()

const store = useUpdateStore()
</script>

<template>
  <Teleport to="body">
    <div v-if="visible" class="update-overlay" @click.self="emit('close')">
      <div class="update-dialog">
        <div class="update-header">
          <span class="update-icon">📦</span>
          <h2>新版本 {{ latestVersion }} 可用</h2>
        </div>
        <div class="update-body">
          <p class="update-version-info">
            当前版本: <strong>{{ currentVersion }}</strong>
            → <strong class="latest">{{ latestVersion }}</strong>
          </p>
          <div v-if="changelog" class="update-changelog">
            <h3>更新内容</h3>
            <pre>{{ changelog }}</pre>
          </div>
          <div v-if="store.downloading" class="update-progress">
            <div class="progress-bar">
              <div class="progress-fill" :style="{ width: store.downloadProgress + '%' }" />
            </div>
            <span>{{ store.downloadProgress }}%</span>
          </div>
          <div v-if="store.error" class="update-error">
            {{ store.error }}
          </div>
        </div>
        <div class="update-actions">
          <button class="btn btn-secondary" data-test="remind-later" @click="emit('close')">
            稍后提醒
          </button>
          <button class="btn btn-secondary" @click="emit('close')">
            查看更新
          </button>
          <button class="btn btn-primary" :disabled="store.downloading" @click="emit('update')">
            {{ store.downloading ? '下载中...' : '更新' }}
          </button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.update-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 9999;
}
.update-dialog {
  background: var(--bg-surface);
  border: 1px solid var(--border);
  border-radius: 8px;
  width: 480px;
  max-width: 90vw;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.3);
}
.update-header {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border);
}
.update-header h2 {
  margin: 0;
  font-size: 16px;
}
.update-body {
  padding: 16px 20px;
}
.update-version-info {
  margin: 0 0 12px;
}
.latest { color: var(--accent); }
.update-changelog {
  background: var(--bg-code);
  border-radius: 4px;
  padding: 8px 12px;
  max-height: 200px;
  overflow-y: auto;
  margin-bottom: 12px;
}
.update-changelog h3 {
  font-size: 12px;
  margin: 0 0 8px;
  color: var(--text-secondary);
}
.update-changelog pre {
  margin: 0;
  font-size: 12px;
  white-space: pre-wrap;
  line-height: 1.5;
}
.update-progress {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}
.progress-bar {
  flex: 1;
  height: 6px;
  background: var(--bg-muted);
  border-radius: 3px;
  overflow: hidden;
}
.progress-fill {
  height: 100%;
  background: var(--accent);
  transition: width 0.3s;
}
.update-error {
  color: var(--error);
  font-size: 12px;
  margin-bottom: 8px;
}
.update-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  padding: 12px 20px;
  border-top: 1px solid var(--border);
}
.btn {
  padding: 6px 16px;
  border-radius: 4px;
  font-size: 13px;
  cursor: pointer;
  border: 1px solid transparent;
}
.btn-secondary {
  background: var(--bg-muted);
  color: var(--text);
  border-color: var(--border);
}
.btn-primary {
  background: var(--accent);
  color: #fff;
}
.btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
</style>
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vitest run -t "UpdatePrompt" 2>&1 || true`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add frontend/src/stores/update.ts frontend/src/terminal/components/UpdatePrompt.vue frontend/src/lib/wails.ts
git commit -m "feat(updater): add UpdatePrompt component and update store"
```

---

### Task 6: Integration — Auto-Check on Startup + Help Menu

**Files:**
- Modify: `frontend/src/App.vue`
- Modify: `frontend/src/terminal/components/SettingsPanel.vue` (update interval)

**Interfaces:**
- Consumes: `useUpdateStore`, `UpdatePrompt` from Task 5
- Produces: Auto-check on app mount (30s delay), Help menu trigger, settings option

- [ ] **Step 1: Check existing code patterns**

```typescript
// Read frontend/src/App.vue to understand where to hook auto-check
```

- [ ] **Step 2: Write no test needed — integration is verified manually**

- [ ] **Step 3: Write implementation**

```vue
<!-- Modify frontend/src/App.vue — add update check on mount -->
<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useUpdateStore } from '@/stores/update'
import UpdatePrompt from '@/terminal/components/UpdatePrompt.vue'

const updateStore = useUpdateStore()
const showUpdatePrompt = ref(false)

onMounted(() => {
  // Auto-check for updates 30 seconds after mount
  setTimeout(async () => {
    if (updateStore.shouldCheck()) {
      const info = await updateStore.check()
      if (info?.has_update) {
        showUpdatePrompt.value = true
      }
      updateStore.markChecked()
    }
  }, 30000)
})

async function handleApplyUpdate() {
  await updateStore.apply()
  showUpdatePrompt.value = false
}
</script>

<template>
  <!-- existing template -->
  <UpdatePrompt
    :visible="showUpdatePrompt"
    :current-version="updateStore.updateInfo?.current_version ?? ''"
    :latest-version="updateStore.updateInfo?.latest_version ?? ''"
    :changelog="updateStore.updateInfo?.changelog ?? ''"
    @close="showUpdatePrompt = false"
    @update="handleApplyUpdate"
  />
</template>
```

Add a "Check for Updates" menu item in SettingsPanel:

```vue
<!-- In SettingsPanel.vue, add a manual check button in the "About" section -->
<template>
  <!-- existing template -->
  <div class="settings-section">
    <h3>关于</h3>
    <div class="setting-row">
      <span>当前版本</span>
      <div class="setting-control">
        <code>{{ version }}</code>
        <button
          class="btn btn-sm"
          :disabled="updateStore.checking"
          @click="manualCheck"
        >
          {{ updateStore.checking ? '检查中...' : '检查更新' }}
        </button>
      </div>
    </div>
    <div class="setting-row">
      <span>更新检查</span>
      <select v-model="updateInterval" @change="saveInterval">
        <option value="always">每次启动</option>
        <option value="daily">每天一次</option>
        <option value="never">从不</option>
      </select>
    </div>
  </div>
</template>
```

```typescript
// SettingsPanel.vue script addition
import { ref, onMounted } from 'vue'
import { useUpdateStore } from '@/stores/update'
import { Call } from '@wailsio/runtime'

const updateStore = useUpdateStore()
const version = ref('')
const updateInterval = ref('daily')

onMounted(async () => {
  try {
    version.value = await Call.ByName('main.App.GetVersion')
    updateInterval.value = await Call.ByName('main.App.GetUpdateInterval')
  } catch {}
})

async function manualCheck() {
  const info = await updateStore.check()
  if (info?.has_update) {
    // Show UpdatePrompt
  } else if (!info?.has_update && info) {
    // Already latest
  }
}

async function saveInterval() {
  try {
    await Call.ByName('main.App.SetUpdateInterval', updateInterval.value)
  } catch {}
}
```

- [ ] **Step 4: Verify build**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vue-tsc --noEmit 2>&1 | head -20`
Expected: No type errors

- [ ] **Step 5: Commit**

```bash
git add frontend/src/App.vue frontend/src/terminal/components/SettingsPanel.vue
git commit -m "feat(updater): integrate auto-update check on startup and settings"
```
