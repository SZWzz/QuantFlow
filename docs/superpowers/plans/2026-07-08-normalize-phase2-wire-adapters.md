# 正规化二期：适配器接入 NormalizeVolume — 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 6 个 A 股适配器中全部硬编码 `* 100` 替换为 `normalize.NormalizeVolume()`，缺失归一化的添加归一化，统一成交量单位。

**Architecture:** normalize.NormalizeVolume(source string, volume float64) → volumeMultiplier[source] → ×100 for A-share sources

**Tech Stack:** Go 1.25+

## Global Constraints

- 仅修改 CN A-share 相关代码路径
- HK/US stocks 的成交量不归一化（港股/美股原生单位是 shares）
- 不修改 `NormalizeVolume` 函数签名
- 所有测试使用 `go test` 运行

---

### Task 1: eastmoney.go — 替换 OHLCV + 添加 Quote 归一化

**Files:**
- Modify: `internal/market/adapters/eastmoney.go`

- [ ] **Step 1: 添加 normalize import**

```go
import (
    // ... existing imports ...
    "quantflow/internal/normalize"
)
```

- [ ] **Step 2: Quote Volume 添加 NormalizeVolume（line 113）**

Before:
```go
Volume:    d.F47,           // f47: 成交量 (手)
```

After:
```go
Volume:    normalize.NormalizeVolume(a.Name(), d.F47), // f47: 成交量 (手→股)
```

- [ ] **Step 3: OHLCV Volume 替换 `* 100`（line 213）**

Before:
```go
Volume: parseFloatSafe(fields[5]) * 100,
```

After:
```go
Volume: normalize.NormalizeVolume(a.Name(), parseFloatSafe(fields[5])),
```

- [ ] **Step 4: 删除过时注释（line 204-205）**

删除两行：
```
		// Volume from EastMoney is in 手 (lots, 1 lot = 100 shares). Normalize to
		// shares for consistency with other CN adapters (TuShare, Sina, Tencent).
```

- [ ] **Step 5: 编译验证**

Run: `go build ./...`

---

### Task 2: mootdx.go — 替换 OHLCV + 添加 Quote 归一化

**Files:**
- Modify: `internal/market/adapters/mootdx.go`

- [ ] **Step 1: 添加 normalize import**

- [ ] **Step 2: Quote Volume 添加 NormalizeVolume（line 89）**

Before:
```go
Volume:        q.Volume,
```

After:
```go
Volume:        normalize.NormalizeVolume(a.Name(), q.Volume),
```

- [ ] **Step 3: OHLCV Volume 替换 `* 100`（line 150）**

Before:
```go
Volume: b.Volume * 100, // 手→shares, consistent with EastMoney adapter
```

After:
```go
Volume: normalize.NormalizeVolume(a.Name(), b.Volume),
```

- [ ] **Step 4: 编译验证**

---

### Task 3: tushare.go — 替换 Quote + OHLCV 的 `* 100`

**Files:**
- Modify: `internal/market/adapters/tushare.go`

- [ ] **Step 1: 添加 normalize import**

- [ ] **Step 2: Quote Volume 替换（line 73）**

Before:
```go
Volume:    rowFloat(row, "vol") * 100, // 手→股
```

After:
```go
Volume:    normalize.NormalizeVolume(a.Name(), rowFloat(row, "vol")),
```

- [ ] **Step 3: OHLCV Volume 替换（line 106）**

Before:
```go
Volume: rowFloat(row, "vol") * 100,
```

After:
```go
Volume: normalize.NormalizeVolume(a.Name(), rowFloat(row, "vol")),
```

- [ ] **Step 4: 编译验证**

---

### Task 4: parsers.go — 替换 parseSinaQuote 中的 `* 100`

**Files:**
- Modify: `internal/market/adapters/parsers.go`

- [ ] **Step 1: 添加 normalize import**

- [ ] **Step 2: 替换 line 113**

Before:
```go
volume := parseFloatSafe(fields[8]) * 100 // 手→股
```

After:
```go
volume := normalize.NormalizeVolume("sina", parseFloatSafe(fields[8])) // 手→股
```

> Note: 此函数仅用于 sina CN Quote，sina US/HK 有独立解析函数不经过此路径。

- [ ] **Step 3: 编译验证**

---

### Task 5: akshare.go — 添加 tencent Quote Volume 归一化

**Files:**
- Modify: `internal/market/adapters/akshare.go`

- [ ] **Step 1: 添加 normalize import**

- [ ] **Step 2: 添加 NormalizeVolume 到 volume 字段**

找到 `Volume: result.Volume` 附近的行。tencent Quote 返回 volume(手)，目前未归一化。

After the volume is extracted from `fields[6]`, add NormalizeVolume:

```go
// [6]=volume(手)
volume := normalize.NormalizeVolume("tencent", parseFloatSafe(fields[6]))
```

然后将 `Volume: ...` 改为使用 `volume` 变量。

> 注意：parseTencentQuote 中 volume 在原始格式字段 [6] 和 JSON fallback 格式 `result.Volume` 两处。只需归一化原始格式路径（fields 路径），JSON fallback 不经过适配器无法知道单位。

- [ ] **Step 3: 编译验证**

---

### Task 6: tencent.go — 添加 OHLCV Volume 归一化 + 替换 Depth `* 100`

**Files:**
- Modify: `internal/market/adapters/tencent.go`

- [ ] **Step 1: 添加 normalize import**

- [ ] **Step 2: OHLCV Volume 添加 NormalizeVolume（line 181）**

Before:
```go
Volume: toFloatVal(r[5]),
```

After:
```go
Volume: normalize.NormalizeVolume(a.Name(), toFloatVal(r[5])),
```

- [ ] **Step 3: Depth bid Size 替换 `* 100`（line 355）**

Before:
```go
Size:  parseFloatSafe(fields[bidVolIdx]) * 100,
```

After:
```go
Size:  normalize.NormalizeVolume(a.Name(), parseFloatSafe(fields[bidVolIdx])),
```

- [ ] **Step 4: Depth ask Size 替换 `* 100`（line 359）**

Same pattern.

- [ ] **Step 5: 编译验证**

---

### Task 7: baidu.go — 添加 Quote + OHLCV Volume 归一化

**Files:**
- Modify: `internal/market/adapters/baidu.go`

- [ ] **Step 1: 添加 normalize import**

- [ ] **Step 2: Quote Volume 添加 NormalizeVolume（line 75）**

Before:
```go
Volume:    d.Volume,
```

After:
```go
Volume:    normalize.NormalizeVolume(a.Name(), d.Volume),
```

- [ ] **Step 3: OHLCV Volume 添加 NormalizeVolume（line 163）**

Before:
```go
Volume: colFloat(values, colIdx, "volume"),
```

After:
```go
Volume: normalize.NormalizeVolume(a.Name(), colFloat(values, colIdx, "volume")),
```

- [ ] **Step 4: 编译验证**

---

### Task 8: sina.go — 替换 Depth `* 100`

**Files:**
- Modify: `internal/market/adapters/sina.go`

- [ ] **Step 1: 添加 normalize import**

- [ ] **Step 2: Depth bid/ask Size 替换 `* 100`（lines 144, 145, 151, 158）**

四处替换：

Before (line 144):
```go
bids[0] = market.DepthLevel{Price: parseFloatSafe(fields[6]), Size: parseFloatSafe(fields[10]) * 100}
```

After:
```go
bids[0] = market.DepthLevel{Price: parseFloatSafe(fields[6]), Size: normalize.NormalizeVolume(a.Name(), parseFloatSafe(fields[10]))}
```

同样替换 lines 145, 151, 158。

> Note: sina.go FetchOHLCV 是 no-op（line 211-212：返回错误 "OHLCV not supported"），无需修改。sina US/HK Quote 不在本次范围（不是 A 股，单位是 shares）。

- [ ] **Step 3: 编译验证**

---

### Task 9: normalize/volume.go — 添加 VolumeMultiplier 公开函数

**Files:**
- Modify: `internal/normalize/volume.go`

- [ ] **Step 1: 添加公共函数**

After `VolumeSources()`:
```go
// VolumeMultiplier returns the volume multiplier for a source, or 1 for unknown sources.
func VolumeMultiplier(source string) float64 {
	if mult, ok := volumeMultiplier[source]; ok {
		return mult
	}
	return 1
}
```

---

### Task 10: 测试 + 完整验证

**Files:**
- Create: `internal/market/adapters/volume_normalize_test.go`
- Modify: `internal/normalize/volume_test.go`

- [ ] **Step 1: 创建适配器集成测试**

```go
package adapters

import (
	"testing"
	"quantflow/internal/normalize"
)

// TestAllCNAdaptersNormalizeVolume 验证所有 CN A-share 适配器对 NormalizeVolume 的一致性。
func TestAllCNAdaptersNormalizeVolume(t *testing.T) {
	tests := []struct {
		adapterName string
		rawVolume   float64
		expected    float64
	}{
		{"eastmoney", 1000, 100000},
		{"sina", 1000, 100000},
		{"tencent", 1000, 100000},
		{"tushare", 1000, 100000},
		{"mootdx", 1000, 100000},
		{"baidu", 1000, 100000},
	}

	for _, tt := range tests {
		t.Run(tt.adapterName, func(t *testing.T) {
			got := normalize.NormalizeVolume(tt.adapterName, tt.rawVolume)
			if got != tt.expected {
				t.Errorf("NormalizeVolume(%q, %v) = %v, want %v",
					tt.adapterName, tt.rawVolume, got, tt.expected)
			}
		})
	}
}

// TestNonCNAdaptersNotNormalized 验证非 A 股适配器不会被归一化。
func TestNonCNAdaptersNotNormalized(t *testing.T) {
	tests := []string{"binance", "yahoo", "finnhub", "polygon", "alpaca", "gateio", "okx", "coingecko", "unknown"}
	for _, name := range tests {
		t.Run(name, func(t *testing.T) {
			got := normalize.NormalizeVolume(name, 1000)
			if got != 1000 {
				t.Errorf("NormalizeVolume(%q, 1000) = %v, want 1000 (no multiplier)", name, got)
			}
		})
	}
}

// TestVolumeMultiplier一致性 验证 VolumeMultiplier 和 NormalizeVolume 一致。
func TestVolumeMultiplier_Consistency(t *testing.T) {
	for _, source := range normalize.VolumeSources() {
		mult := normalize.VolumeMultiplier(source)
		got := normalize.NormalizeVolume(source, 1)
		if got != mult {
			t.Errorf("mismatch for %q: NormalizeVolume(1)=%v, VolumeMultiplier=%v", source, got, mult)
		}
	}
}
```

- [ ] **Step 2: 扩展 normalize/volume_test.go**

添加测试用例覆盖 VolumeMultiplier 和非 CN 源。

- [ ] **Step 3: 运行全部测试**

```bash
go test ./internal/normalize/... -count=1
go test ./internal/market/adapters/... -count=1
go vet ./internal/market/adapters/...
go vet ./internal/normalize/...
```

- [ ] **Step 4: 运行适配器测试验证无回归**

```bash
go test ./internal/market/adapters/... -v -count=1
```

---

### Task 11: CHANGELOG + 提交

**Files:**
- Modify: `CHANGELOG.md`

- [ ] **Step 1: 更新 CHANGELOG**

Under `[2026.7.8]`:
```markdown
- [MarketData] Normalize Phase 2 — all 6 A-share adapters now route volume through
  normalize.NormalizeVolume(), eliminating hardcoded *100; added missing normalization
  to tencent/baidu/eastmoney quote volumes
```

- [ ] **Step 2: 完整检查 + 提交**

```bash
go vet ./... && go test ./internal/market/adapters/... ./internal/normalize/... -count=1
git add -A
git commit -m "refactor(normalize): wire all 6 A-share adapters to use NormalizeVolume"
```
