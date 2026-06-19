package adapters

import (
	"context"
	"os"
	"testing"
	"time"
)

// hasIwencaiKey checks if IWENCAI_API_KEY is set for live tests.
func hasIwencaiKey() bool {
	return os.Getenv("IWENCAI_API_KEY") != ""
}

// ── Live tests (require API key) ──────────────────────────────────────

func TestIwencaiSearch_Live(t *testing.T) {
	if !hasIwencaiKey() {
		t.Skip("IWENCAI_API_KEY not set — skipping live test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	adapter := NewIwencaiAdapter()
	if !adapter.IsAvailable(ctx) {
		t.Fatal("iwencai should be available with valid API key")
	}

	// Search for research reports about 贵州茅台
	articles, err := adapter.Search(ctx, "贵州茅台 2026 年报", "report", 10)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	t.Logf("Search returned %d articles", len(articles))
	for i, a := range articles {
		t.Logf("  [%d] %s | %s | org=%s | date=%s | score=%s",
			i, a.Channel, a.Title[:min(len(a.Title), 60)], a.OrgName, a.PublishDate[:min(len(a.PublishDate), 10)], a.Score())
	}
}

func TestIwencaiSearch_MultiChannel(t *testing.T) {
	if !hasIwencaiKey() {
		t.Skip("IWENCAI_API_KEY not set — skipping live test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	adapter := NewIwencaiAdapter()

	channels := []string{"news", "announcement"}
	for _, ch := range channels {
		t.Run(ch, func(t *testing.T) {
			articles, err := adapter.Search(ctx, "新能源汽车", ch, 5)
			if err != nil {
				t.Fatalf("Search(%s) failed: %v", ch, err)
			}
			t.Logf("%s: %d results", ch, len(articles))
			for _, a := range articles {
				t.Logf("  %s | %s | date=%s", a.Channel, a.Title[:min(len(a.Title), 60)], a.PublishDate[:min(len(a.PublishDate), 10)])
			}
		})
	}
}

func TestIwencaiQuery_Live(t *testing.T) {
	if !hasIwencaiKey() {
		t.Skip("IWENCAI_API_KEY not set — skipping live test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	adapter := NewIwencaiAdapter()

	rows, err := adapter.Query(ctx, "贵州茅台 ROE 净利润 2025", 1, 5)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	t.Logf("Query returned %d rows", len(rows))
	for i, row := range rows {
		t.Logf("  [%d] fields=%d", i, len(row.Fields))
		for k, v := range row.Fields {
			if len(v) > 80 {
				v = v[:80] + "..."
			}
			t.Logf("    %s = %s", k, v)
		}
	}
}

func TestIwencaiTopicSearch_Live(t *testing.T) {
	if !hasIwencaiKey() {
		t.Skip("IWENCAI_API_KEY not set — skipping live test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	adapter := NewIwencaiAdapter()

	// Cross-topic search — unique iwencai capability
	articles, err := adapter.Search(ctx, "人形机器人 行星滚柱丝杠 2026", "report", 20)
	if err != nil {
		t.Fatalf("Topic search failed: %v", err)
	}

	t.Logf("Cross-topic search returned %d articles", len(articles))
	for i, a := range articles {
		t.Logf("  [%d] %s | org=%s | %s | score=%s",
			i, a.Title[:min(len(a.Title), 60)], a.OrgName, a.PublishDate[:min(len(a.PublishDate), 10)], a.Score())
	}

	// We expect at least some results for this hot topic
	if len(articles) == 0 {
		t.Error("Expected at least 1 article for cross-topic search '人形机器人 行星滚柱丝杠'")
	}
}

// ── Offline tests (no API key needed) ─────────────────────────────────

func TestIwencaiAdapter_NoKey(t *testing.T) {
	// Save and restore env
	oldKey := os.Getenv("IWENCAI_API_KEY")
	os.Unsetenv("IWENCAI_API_KEY")
	defer os.Setenv("IWENCAI_API_KEY", oldKey)

	adapter := NewIwencaiAdapter()

	if adapter.IsAvailable(context.Background()) {
		t.Error("adapter should not be available without API key")
	}

	if !adapter.RequiresAuth() {
		t.Error("iwencai should require auth")
	}

	// Search should return a clear error about missing key
	_, err := adapter.Search(context.Background(), "test", "report", 10)
	if err == nil {
		t.Error("Search without key should return error")
	}
	t.Logf("Expected error: %v", err)

	// Query should also return error
	_, err = adapter.Query(context.Background(), "test", 1, 10)
	if err == nil {
		t.Error("Query without key should return error")
	}
}

func TestDedupArticles(t *testing.T) {
	articles := []IwencaiArticle{
		{UID: "a1", Title: "Report A", ScoreRaw: "0.8", PublishDate: "2026-06-15"},
		{UID: "a1", Title: "Report A Dup", ScoreRaw: "0.9", PublishDate: "2026-06-15"},
		{UID: "a2", Title: "Report B", ScoreRaw: "0.7", PublishDate: "2026-06-10"},
		{UID: "a2", Title: "Report B Dup", ScoreRaw: "0.6", PublishDate: "2026-06-10"},
		{UID: "a3", Title: "Report C", ScoreRaw: "0.5", PublishDate: "2026-06-20"},
	}

	result := dedupArticles(articles)

	if len(result) != 3 {
		t.Fatalf("expected 3 deduplicated articles, got %d", len(result))
	}

	// a1: should keep the 0.9 score version
	for _, a := range result {
		switch a.UID {
		case "a1":
			if a.Score() != "0.9" {
				t.Errorf("a1: expected score 0.9, got %s", a.Score())
			}
		case "a2":
			if a.Score() != "0.7" {
				t.Errorf("a2: expected score 0.7, got %s", a.Score())
			}
		}
	}

	// Sort check: should be descending by date
	for i := 1; i < len(result); i++ {
		if result[i-1].PublishDate < result[i].PublishDate {
			t.Errorf("articles not sorted by date desc: %s < %s",
				result[i-1].PublishDate, result[i].PublishDate)
		}
	}
}

func TestClawHeaders(t *testing.T) {
	headers := clawHeaders()

	required := []string{
		"X-Claw-Call-Type",
		"X-Claw-Skill-Id",
		"X-Claw-Skill-Version",
		"X-Claw-Plugin-Id",
		"X-Claw-Plugin-Version",
		"X-Claw-Trace-Id",
	}
	for _, k := range required {
		if v, ok := headers[k]; !ok || v == "" {
			t.Errorf("missing or empty required header: %s", k)
		}
	}

	// Trace ID must be 64 hex chars (32 bytes)
	if len(headers["X-Claw-Trace-Id"]) != 64 {
		t.Errorf("trace ID should be 64 hex chars, got %d", len(headers["X-Claw-Trace-Id"]))
	}

	// Verify expected values
	if headers["X-Claw-Skill-Id"] != "report-search" {
		t.Errorf("unexpected skill id: %s", headers["X-Claw-Skill-Id"])
	}
	if headers["X-Claw-Skill-Version"] != "2.0.0" {
		t.Errorf("unexpected skill version: %s", headers["X-Claw-Skill-Version"])
	}
}

func TestIwencaiAdapter_Name(t *testing.T) {
	adapter := NewIwencaiAdapter()
	if adapter.Name() != "iwencai" {
		t.Errorf("expected name 'iwencai', got '%s'", adapter.Name())
	}
}

func TestParseExtra_Object(t *testing.T) {
	adapter := &IwencaiAdapter{}

	art := &IwencaiArticle{}
	extra := []byte(`{"organization":"中信证券","author":"张三","pdf_url":"https://example.com/report.pdf"}`)
	adapter.parseExtra(art, extra)

	if art.OrgName != "中信证券" {
		t.Errorf("expected org 中信证券, got %s", art.OrgName)
	}
	if art.Author != "张三" {
		t.Errorf("expected author 张三, got %s", art.Author)
	}
	if art.PDFURL != "https://example.com/report.pdf" {
		t.Errorf("expected pdf_url, got %s", art.PDFURL)
	}
}

func TestParseExtra_StringEncoded(t *testing.T) {
	adapter := &IwencaiAdapter{}

	art := &IwencaiArticle{}
	// Double-encoded: extra is a JSON string that contains JSON
	extra := []byte(`"{\"organization\":\"华泰证券\",\"author\":\"李四\"}"`)
	adapter.parseExtra(art, extra)

	if art.OrgName != "华泰证券" {
		t.Errorf("expected org 华泰证券, got %s", art.OrgName)
	}
	if art.Author != "李四" {
		t.Errorf("expected author 李四, got %s", art.Author)
	}
}
