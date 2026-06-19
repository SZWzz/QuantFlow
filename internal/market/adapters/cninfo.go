package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Announcement represents a single stock exchange announcement.
type Announcement struct {
	Title string `json:"title"`
	Type  string `json:"type"` // 公告类型
	Date  string `json:"date"` // YYYY-MM-DD
	URL   string `json:"url"`  // Full announcement URL
}

// CninfoAdapter fetches stock exchange announcements from 巨潮资讯网.
// Based on a-stock-data SKILL §7.1 (cninfo.com.cn).
//
// Uses dynamic orgId mapping (official szse_stock.json, 6198 stocks) with
// hardcoded fallback for stocks not in the mapping.
type CninfoAdapter struct {
	client  *http.Client
	orgOnce sync.Once
	orgMap  map[string]string // code → orgId
	orgErr  error
}

// NewCninfoAdapter creates a new Cninfo announcements adapter.
func NewCninfoAdapter() *CninfoAdapter {
	return &CninfoAdapter{
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

func (a *CninfoAdapter) Name() string { return "cninfo" }

func (a *CninfoAdapter) IsAvailable(ctx context.Context) bool {
	_, err := a.FetchAnnouncements(ctx, "600519", 5)
	return err == nil
}

// FetchAnnouncements fetches announcements for a stock.
func (a *CninfoAdapter) FetchAnnouncements(ctx context.Context, code string, pageSize int) ([]Announcement, error) {
	if pageSize <= 0 {
		pageSize = 30
	}

	a.ensureOrgMap()

	orgID := a.lookupOrgID(code)

	// Build form-encoded POST request
	form := url.Values{}
	form.Set("stock", fmt.Sprintf("%s,%s", code, orgID))
	form.Set("tabName", "fulltext")
	form.Set("pageSize", fmt.Sprintf("%d", pageSize))
	form.Set("pageNum", "1")
	form.Set("column", "")
	form.Set("category", "")
	form.Set("plate", "")
	form.Set("seDate", "")
	form.Set("searchkey", "")
	form.Set("secid", "")
	form.Set("sortName", "")
	form.Set("sortType", "")
	form.Set("isHLtitle", "true")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://www.cninfo.com.cn/new/hisAnnouncement/query",
		strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("cninfo: %w", err)
	}
	req.Header.Set("User-Agent", emUA)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", "https://www.cninfo.com.cn/new/disclosure")
	req.Header.Set("Origin", "https://www.cninfo.com.cn")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cninfo: http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cninfo: http %d", resp.StatusCode)
	}

	var result struct {
		Announcements []struct {
			AnnouncementTitle string      `json:"announcementTitle"`
			AnnouncementType  interface{} `json:"announcementTypeName"`
			AnnouncementTime  interface{} `json:"announcementTime"`
			AnnouncementID    string      `json:"announcementId"`
		} `json:"announcements"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("cninfo: json: %w", err)
	}

	announcements := make([]Announcement, 0, len(result.Announcements))
	for _, item := range result.Announcements {
		date := cninfoTsToDate(item.AnnouncementTime)
		announcements = append(announcements, Announcement{
			Title: item.AnnouncementTitle,
			Type:  strval(item.AnnouncementType),
			Date:  date,
			URL: fmt.Sprintf(
				"https://www.cninfo.com.cn/new/disclosure/detail?annoId=%s",
				item.AnnouncementID,
			),
		})
	}

	slog.Debug("cninfo fetched", "code", code, "announcements", len(announcements))
	return announcements, nil
}

// ── orgId mapping ─────────────────────────────────────────────────────

func (a *CninfoAdapter) ensureOrgMap() {
	a.orgOnce.Do(func() {
		resp, err := a.client.Get("http://www.cninfo.com.cn/new/data/szse_stock.json")
		if err != nil {
			a.orgErr = fmt.Errorf("cninfo: orgId map fetch failed: %w", err)
			return
		}
		defer resp.Body.Close()

		var data struct {
			StockList []struct {
				Code  string `json:"code"`
				OrgID string `json:"orgId"`
			} `json:"stockList"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			a.orgErr = fmt.Errorf("cninfo: orgId map parse: %w", err)
			return
		}

		a.orgMap = make(map[string]string, len(data.StockList))
		for _, s := range data.StockList {
			a.orgMap[s.Code] = s.OrgID
		}
		slog.Debug("cninfo orgId map loaded", "stocks", len(a.orgMap))
	})
}

func (a *CninfoAdapter) lookupOrgID(code string) string {
	if a.orgMap != nil {
		if org, ok := a.orgMap[code]; ok {
			return org
		}
	}
	// Hardcoded fallback (works for some older stocks)
	if code[0] == '6' {
		return "gssh0" + code
	}
	if code[0] == '8' || code[0] == '4' {
		return "gsbj0" + code
	}
	return "gssz0" + code
}

// cninfoTsToDate converts cninfo's timestamp (Unix ms) to YYYY-MM-DD string.
func cninfoTsToDate(ts interface{}) string {
	switch v := ts.(type) {
	case float64:
		if v > 1e12 {
			return time.UnixMilli(int64(v)).Format("2006-01-02")
		}
		return ""
	case string:
		if len(v) >= 10 {
			return v[:10]
		}
	}
	return ""
}

