package market

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"
)

// StockEntry represents a single stock in the search index (any market).
type StockEntry struct {
	Code   string `json:"code"`
	Name   string `json:"name"`
	Market string `json:"market"` // "SH"/"SZ"/"BJ" (CN), "HK", "US"
	Pinyin string `json:"pinyin"` // abbreviation, e.g. "gzmt" for 贵州茅台; empty for US
}

// SymbolSearchService provides fast code/name/pinyin search over CN A-shares
// (~5500), HK stocks (~2500), and US stocks (~13500). The index is built once
// at startup and held in memory.
type SymbolSearchService struct {
	mu      sync.RWMutex
	entries []StockEntry
}

// NewSymbolSearchService creates the search service and populates the index
// from EastMoney's stock list API for CN, HK, and US markets. Call once at startup.
func NewSymbolSearchService(ctx context.Context) (*SymbolSearchService, error) {
	var all []StockEntry

	// CN A-shares
	cn, err := fetchCNStockList(ctx)
	if err != nil {
		slog.Warn("symbol_search: CN stock list fetch failed", "error", err)
	} else {
		for i := range cn {
			cn[i].Pinyin = pinyinAbbr(cn[i].Name)
		}
		all = append(all, cn...)
	}

	// HK stocks
	hk, err := fetchHKStockList(ctx)
	if err != nil {
		slog.Warn("symbol_search: HK stock list fetch failed", "error", err)
	} else {
		for i := range hk {
			hk[i].Pinyin = pinyinAbbr(hk[i].Name)
		}
		all = append(all, hk...)
	}

	// US stocks
	us, err := fetchUSStockList(ctx)
	if err != nil {
		slog.Warn("symbol_search: US stock list fetch failed", "error", err)
	} else {
		all = append(all, us...)
	}

	if len(all) == 0 {
		return nil, fmt.Errorf("symbol_search: all market fetches failed")
	}

	slog.Info("symbol_search: index built",
		"cn", len(cn), "hk", len(hk), "us", len(us), "total", len(all))
	return &SymbolSearchService{entries: all}, nil
}

// Search finds stocks matching the query by code, name, or pinyin.
// Results are sorted by relevance: exact code > code prefix > name match > pinyin.
func (s *SymbolSearchService) Search(query string, limit int) []StockEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	q := strings.TrimSpace(query)
	if q == "" {
		return nil
	}
	if limit <= 0 {
		limit = 20
	}

	type scored struct {
		entry StockEntry
		score int
	}
	var results []scored

	qLower := strings.ToLower(q)

	for _, e := range s.entries {
		score := 0
		// Exact code match: highest priority
		if e.Code == q {
			score = 1000
		} else if strings.HasPrefix(e.Code, q) {
			// Code prefix match
			score = 800 + (6 - len(q)) // shorter query = more general
		} else if strings.Contains(e.Name, q) {
			// Name contains query (Chinese chars)
			score = 500
		} else if qLower == e.Pinyin {
			// Exact pinyin match
			score = 400
		} else if strings.HasPrefix(e.Pinyin, qLower) {
			// Pinyin prefix match
			score = 300
		} else if strings.Contains(strings.ToLower(e.Name), qLower) {
			// Case-insensitive name contains
			score = 200
		} else {
			continue // no match
		}
		results = append(results, scored{e, score})
	}

	// Sort by score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	// Apply limit
	if len(results) > limit {
		results = results[:limit]
	}

	entries := make([]StockEntry, len(results))
	for i, r := range results {
		entries[i] = r.entry
	}
	return entries
}

// Size returns the number of stocks in the index.
func (s *SymbolSearchService) Size() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.entries)
}

// ── Stock list fetcher ───────────────────────────────────────────────────

// fetchCNStockList fetches the full A-share stock list from EastMoney push2 API.
// Returns ~5500 entries (all SH + SZ + BJ stocks). Uses pagination (100 per page).
func fetchCNStockList(ctx context.Context) ([]StockEntry, error) {
	return fetchPaginatedStockList(ctx,
		"m:0+t:6,m:0+t:80,m:1+t:2,m:1+t:23",
		func(code, name string) (StockEntry, bool) {
			id, err := NormalizeCN(code)
			if err != nil {
				return StockEntry{}, false
			}
			return StockEntry{Code: id.Code, Name: name, Market: id.Market}, true
		},
		"CN",
	)
}

// fetchPaginatedStockList is a generic paginated fetcher for EastMoney push2 stock lists.
// fs is the market filter (e.g. "m:0+t:6,m:0+t:80"), mapFn converts raw entries.
func fetchPaginatedStockList(ctx context.Context, fs string, mapFn func(code, name string) (StockEntry, bool), label string) ([]StockEntry, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	const pageSize = 100
	page := 1
	var all []StockEntry

	for {
		url := fmt.Sprintf(
			"https://push2.eastmoney.com/api/qt/clist/get"+
				"?pn=%d&pz=%d&po=1&np=1&fltt=2&invt=2"+
				"&fs=%s&fields=f12,f14",
			page, pageSize, fs,
		)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("stock_list_%s: %w", label, err)
		}
		req.Header.Set("Host", "push2.eastmoney.com")
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
		req.Header.Set("Referer", "https://quote.eastmoney.com/")

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("stock_list_%s: http: %w", label, err)
		}

		var result struct {
			Data struct {
				Total int `json:"total"`
				Diff  []struct {
					F12 string `json:"f12"`
					F14 string `json:"f14"`
				} `json:"diff"`
			} `json:"data"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("stock_list_%s: json: %w", label, err)
		}
		resp.Body.Close()

		for _, d := range result.Data.Diff {
			if entry, ok := mapFn(d.F12, d.F14); ok {
				all = append(all, entry)
			}
		}

		if len(result.Data.Diff) < pageSize || len(all) >= result.Data.Total {
			break
		}
		page++
		// Small delay between pages to avoid rate limiting
		time.Sleep(50 * time.Millisecond)
	}

	slog.Info("stock_list: "+label+" fetched", "total", len(all))
	return all, nil
}

// fetchHKStockList fetches the full Hong Kong stock list (~2500 entries).
func fetchHKStockList(ctx context.Context) ([]StockEntry, error) {
	return fetchPaginatedStockList(ctx,
		"m:128+t:3",
		func(code, name string) (StockEntry, bool) {
			return StockEntry{Code: code, Name: name, Market: "HK"}, true
		},
		"HK",
	)
}

// fetchUSStockList fetches the full US stock list (~13500 entries).
func fetchUSStockList(ctx context.Context) ([]StockEntry, error) {
	return fetchPaginatedStockList(ctx,
		"m:105,m:106,m:107",
		func(code, name string) (StockEntry, bool) {
			return StockEntry{Code: code, Name: name, Market: "US"}, true
		},
		"US",
	)
}

// ── Pinyin helpers ──────────────────────────────────────────────────────

// pinyinAbbr generates a pinyin abbreviation for a Chinese stock name.
// "贵州茅台" → "gzmt", "平安银行" → "payx".
// Uses a minimal embedded table covering common financial characters.
func pinyinAbbr(name string) string {
	var b strings.Builder
	for _, r := range name {
		if p, ok := charPinyin[r]; ok {
			b.WriteByte(p[0]) // first letter of pinyin
		} else if unicode.Is(unicode.Han, r) {
			b.WriteByte('?') // unknown Chinese char
		}
		// non-Chinese chars (spaces, digits, A-Z) are skipped
	}
	return strings.ToLower(b.String())
}

// charPinyin maps common Chinese characters to their pinyin (first-letter lookup).
// Covers ~400 of the most frequent characters in A-share stock names.
// Generated from a minimal set; can be expanded on demand.
var charPinyin = map[rune]string{
	// A
	'阿': "a", '安': "an", '奥': "ao", '爱': "ai",
	// B
	'百': "bai", '邦': "bang", '宝': "bao", '北': "bei", '本': "ben", '博': "bo",
	'白': "bai", '保': "bao", '报': "bao", '贝': "bei", '波': "bo", '不': "bu",
	'冰': "bing", '步': "bu", '巴': "ba", '包': "bao", '比': "bi", '滨': "bin",
	'伯': "bo", '铂': "bo",
	// C
	'财': "cai", '长': "chang", '城': "cheng", '创': "chuang", '川': "chuan",
	'车': "che", '成': "cheng", '出': "chu", '传': "chuan", '春': "chun",
	'磁': "ci", '存': "cun", '材': "cai", '彩': "cai", '产': "chan", '昌': "chang",
	'潮': "chao", '晨': "chen", '诚': "cheng", '楚': "chu", '慈': "ci",
	// D
	'大': "da", '道': "dao", '德': "de", '地': "di", '电': "dian", '东': "dong",
	'动': "dong", '达': "da", '代': "dai", '打': "da", '导': "dao", '迪': "di", '鼎': "ding",
	'多': "duo", '端': "duan", '盾': "dun", '点': "dian", '第': "di", '董': "dong",
	// E
	'尔': "er", '二': "er",
	// F
	'发': "fa", '方': "fang", '飞': "fei", '丰': "feng", '风': "feng", '福': "fu",
	'富': "fu", '复': "fu", '服': "fu", '房': "fang", '分': "fen", '蜂': "feng",
	'辅': "fu", '孚': "fu",
	// G
	'高': "gao", '工': "gong", '公': "gong", '广': "guang", '国': "guo", '光': "guang",
	'贵': "gui", '股': "gu", '格': "ge", '港': "gang", '冠': "guan", '管': "guan",
	'钢': "gang", '古': "gu", '谷': "gu", '关': "guan", '硅': "gui",
	// H
	'海': "hai", '航': "hang", '好': "hao", '合': "he", '和': "he", '恒': "heng",
	'宏': "hong", '华': "hua", '化': "hua", '环': "huan", '汇': "hui", '惠': "hui",
	'沪': "hu", '湖': "hu", '互': "hu", '花': "hua", '汉': "han", '翰': "han",
	'杭': "hang", '豪': "hao", '浩': "hao", '禾': "he", '河': "he", '黑': "hei",
	'红': "hong", '鸿': "hong", '辉': "hui", '会': "hui",
	// J
	'基': "ji", '技': "ji", '建': "jian", '江': "jiang", '金': "jin", '京': "jing",
	'九': "jiu", '酒': "jiu", '巨': "ju", '机': "ji", '集': "ji", '健': "jian",
	'交': "jiao", '节': "jie", '景': "jing", '佳': "jia", '家': "jia", '嘉': "jia",
	'甲': "jia", '检': "jian", '见': "jian", '杰': "jie", '捷': "jie", '锦': "jin",
	'经': "jing", '晶': "jing", '敬': "jing", '军': "jun", '均': "jun",
	// K
	'开': "kai", '康': "kang", '科': "ke", '客': "ke", '空': "kong", '控': "kong",
	'凯': "kai", '克': "ke", '可': "ke", '快': "kuai", '矿': "kuang", '昆': "kun",
	// L
	'蓝': "lan", '乐': "le", '力': "li", '立': "li", '联': "lian", '良': "liang",
	'量': "liang", '林': "lin", '零': "ling", '领': "ling", '龙': "long", '隆': "long",
	'鲁': "lu", '陆': "lu", '路': "lu", '绿': "lv", '利': "li", '理': "li",
	'丽': "li", '兰': "lan", '朗': "lang", '浪': "lang", '老': "lao", '雷': "lei",
	'李': "li", '链': "lian", '辽': "liao", '六': "liu", '旅': "lv", '罗': "luo",
	// M
	'茅': "mao", '美': "mei", '明': "ming", '摩': "mo", '木': "mu", '目': "mu",
	'民': "min", '名': "ming", '麦': "mai", '贸': "mao", '煤': "mei", '梦': "meng",
	'米': "mi", '密': "mi", '面': "mian", '模': "mo", '墨': "mo",
	// N
	'能': "neng", '农': "nong", '南': "nan", '宁': "ning", '纳': "na", '牛': "niu",
	'诺': "nuo",
	// O
	'欧': "ou",
	// P
	'平': "ping", '普': "pu", '品': "pin", '浦': "pu", '鹏': "peng", '派': "pai",
	'磐': "pan", '配': "pei",
	// Q
	'千': "qian", '前': "qian", '强': "qiang", '青': "qing", '清': "qing",
	'全': "quan", '泉': "quan", '汽': "qi", '奇': "qi", '启': "qi", '确': "que",
	'齐': "qi", '旗': "qi", '企': "qi", '迁': "qian", '权': "quan",
	// R
	'人': "ren", '日': "ri", '融': "rong", '软': "ruan", '瑞': "rui", '润': "run",
	'荣': "rong", '燃': "ran", '热': "re",
	// S
	'三': "san", '山': "shan", '上': "shang", '深': "shen", '生': "sheng",
	'十': "shi", '世': "shi", '首': "shou", '数': "shu", '双': "shuang",
	'水': "shui", '顺': "shun", '四': "si", '速': "su", '苏': "su", '算': "suan",
	'神': "shen", '实': "shi", '石': "shi", '时': "shi", '食': "shi", '市': "shi",
	'事': "shi", '商': "shang", '沙': "sha", '纱': "sha", '厦': "sha", '尚': "shang",
	'绍': "shao", '社': "she", '申': "shen", '声': "sheng", '圣': "sheng", '省': "sheng",
	'胜': "sheng", '盛': "sheng", '视': "shi", '舒': "shu", '帅': "shuai", '司': "si",
	'松': "song", '搜': "sou", '塑': "su", '随': "sui", '索': "suo",
	// T
	'太': "tai", '唐': "tang", '天': "tian", '铁': "tie", '通': "tong", '同': "tong",
	'投': "tou", '土': "tu", '团': "tuan", '泰': "tai", '台': "tai", '拓': "tuo",
	'腾': "teng", '田': "tian", '特': "te", '图': "tu",
	// W
	'万': "wan", '网': "wang", '微': "wei", '为': "wei", '维': "wei", '伟': "wei",
	'文': "wen", '物': "wu", '无': "wu", '五': "wu", '武': "wu", '外': "wai",
	'完': "wan", '晚': "wan", '王': "wang", '旺': "wang", '望': "wang", '威': "wei",
	'味': "wei", '温': "wen", '乌': "wu", '芜': "wu", '吴': "wu", '务': "wu",
	// X
	'西': "xi", '希': "xi", '先': "xian", '现': "xian", '新': "xin", '信': "xin",
	'星': "xing", '行': "xing", '雄': "xiong", '学': "xue", '兴': "xing",
	'小': "xiao", '芯': "xin", '旭': "xu", '雪': "xue", '香': "xiang", '湘': "xiang",
	'祥': "xiang", '享': "xiang", '协': "xie", '心': "xin", '鑫': "xin", '许': "xu",
	'宣': "xuan", '旋': "xuan", '讯': "xun",
	// Y
	'亚': "ya", '研': "yan", '药': "yao", '业': "ye", '一': "yi", '医': "yi",
	'亿': "yi", '银': "yin", '英': "ying", '永': "yong", '友': "you", '有': "you",
	'玉': "yu", '元': "yuan", '远': "yuan", '云': "yun", '运': "yun", '扬': "yang",
	'阳': "yang", '洋': "yang", '养': "yang", '伊': "yi", '仪': "yi", '易': "yi",
	'益': "yi", '艺': "yi", '音': "yin", '盈': "ying", '影': "ying", '优': "you",
	'游': "you", '宇': "yu", '与': "yu", '雨': "yu", '裕': "yu", '豫': "yu",
	'园': "yuan", '原': "yuan", '源': "yuan", '院': "yuan", '越': "yue", '阅': "yue",
	'悦': "yue", '粤': "yue",
	// Z
	'招': "zhao", '浙': "zhe", '智': "zhi", '中': "zhong", '重': "zhong",
	'州': "zhou", '珠': "zhu", '资': "zi", '自': "zi", '总': "zong", '作': "zuo",
	'振': "zhen", '正': "zheng", '证': "zheng", '制': "zhi", '致': "zhi",
	'置': "zhi", '装': "zhuang", '卓': "zhuo", '紫': "zi", '棕': "zong", '组': "zu",
}
