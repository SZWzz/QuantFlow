# Release Pipeline (自动发布流水线)

## Motivation

当前 CI 只运行测试，不产生任何可分发产物。用户要使用 QuantFlow 需要自己 `go build`，这对非开发者用户不可行。需要自动化构建 → 签名 → 打包 → 发布到 GitHub Releases 的完整流水线。

## Design

### 发布工作流

```
git tag v2026.7.20 → push
  ↓
GitHub Actions Release Workflow
  ↓
并行构建 3 平台 × 2 架构 = 6 个 artifact:
  ├─ macOS arm64 (.app + .dmg)
  ├─ macOS amd64 (.app + .dmg)
  ├─ Linux amd64 (.tar.gz)
  ├─ Linux arm64 (.tar.gz)
  ├─ Windows amd64 (.exe + .msi)
  └─ Windows arm64 (.exe + .msi)
  ↓
代码签名 (若配置)
  ↓
生成 SHA256 checksums
  ↓
上传到 GitHub Releases
  ↓
(可选) 发布 Homebrew tap / winget PR
```

### 新增文件

| 文件 | 操作 |
|------|------|
| `.github/workflows/release.yml` | 新建 |

### release.yml 设计

```yaml
name: Release
on:
  push:
    tags: ['v*']

jobs:
  release:
    strategy:
      matrix:
        include:
          - os: ubuntu-latest
            goos: linux
            goarch: amd64
            ext: tar.gz
          - os: ubuntu-latest
            goos: linux
            goarch: arm64
            ext: tar.gz
          - os: macos-latest
            goos: darwin
            goarch: arm64
            ext: zip
          - os: macos-13
            goos: darwin
            goarch: amd64
            ext: zip
          - os: windows-latest
            goos: windows
            goarch: amd64
            ext: zip
          - os: windows-latest
            goos: windows
            goarch: arm64
            ext: zip

    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with: { go-version: '1.25' }

      - uses: actions/setup-node@v4
        with: { node-version: '20' }

      - name: Build frontend
        run: cd frontend && npm ci && npm run build -q

      - name: Build Go binary
        run: |
          GOOS=${{ matrix.goos }} GOARCH=${{ matrix.goarch }} \
          go build -a -ldflags="-s -w" -o build/quantflow${{ matrix.goos == 'windows' && '.exe' || '' }} .

      - name: Package
        run: |
          # 平台特定打包逻辑
          # macOS: .app bundle → .dmg
          # Linux: tar.gz binary + resources
          # Windows: .exe + .msi (wix)

      - name: Checksum
        run: sha256sum build/* > build/checksums.txt

      - name: Upload Release Assets
        uses: softprops/action-gh-release@v2
        with:
          files: build/*
          generate_release_notes: true
```

### macOS 打包细节

```bash
# 创建 .app bundle
mkdir -p build/QuantFlow.app/Contents/MacOS
mkdir -p build/QuantFlow.app/Contents/Resources
cp build/quantflow build/QuantFlow.app/Contents/MacOS/
cp resources/icon.icns build/QuantFlow.app/Contents/Resources/
cp -r frontend/dist build/QuantFlow.app/Contents/Resources/frontend
# Info.plist
cat > build/QuantFlow.app/Contents/Info.plist <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "...">
<plist version="1.0">
<dict>
    <key>CFBundleName</key><string>QuantFlow</string>
    <key>CFBundleDisplayName</key><string>QuantFlow Terminal</string>
    <key>CFBundleIdentifier</key><string>com.quantflow.terminal</string>
    <key>CFBundleVersion</key><string>${VERSION}</string>
    <key>CFBundleShortVersionString</key><string>${VERSION}</string>
    <key>CFBundleExecutable</key><string>quantflow</string>
    ...
</dict>
</plist>
EOF

# 创建 DMG
hdiutil create -volname QuantFlow -srcfolder build/QuantFlow.app \
  -ov -format UDZO build/QuantFlow-${VERSION}-${ARCH}.dmg
```

### Windows 打包

使用 WiX Toolset 生成 `.msi`，或简单的 `.exe` + 安装脚本。

### 版本配套

每次发布需同步更新：
1. `frontend/package.json` — `version` 字段
2. `README.md` — 版本 badge
3. `CHANGELOG.md` — 最新版本 header

已在 CLAUDE.md rule 3 规定，release pipeline 应自动检查或至少提醒。

### Homebrew Tap（可选）

```ruby
# Formula
class Quantflow < Formula
  desc "Dual-mode quantitative finance terminal"
  homepage "https://github.com/SZWzz/QuantFlow"
  version "2026.7.20"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/SZWzz/QuantFlow/releases/download/v2026.7.20/quantflow-darwin-arm64.zip"
      sha256 "..."
    end
  end

  # 不依赖 Python — Python sidecar 按需安装
  depends_on "go" => :optional

  def install
    app_path = "QuantFlow.app"
    prefix.install app_path
    bin.write_exec_script "#{prefix}/#{app_path}/Contents/MacOS/quantflow"
  end
end
```

## Acceptance Criteria

- [ ] `git tag v*` push 触发 release workflow
- [ ] 6 个平台架构组合全部构建成功
- [ ] macOS 产物为 .app bundle + .dmg
- [ ] Linux 产物为 .tar.gz（含二进制 + resources）
- [ ] Windows 产物为 .exe + .msi
- [ ] 构建产物包含 SHA256 checksums 文件
- [ ] Release notes 自动从 CHANGELOG 生成
- [ ] 产物上传到 GitHub Releases
- [ ] 构建产物可直接下载使用（无需额外编译）
- [ ] (可选) Homebrew tap 发布

## Risks / Trade-offs

- **风险**: GitHub Actions macOS runner 无法代码签名（无 Apple Developer 证书）。→ 签名需要自建 runner 或用户手动签名
- **风险**: DMG 创建需要 `hdiutil`（仅 macOS runner 可用）。→ 已在 macos-latest 上运行
- **风险**: WiX Toolset 学习成本高。→ 初期只发布 .exe zip，.msi 后续迭代
- **Trade-off**: 不发布 App Store（要求沙箱化，限制过多）
