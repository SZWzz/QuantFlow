// mac.go — TDX MAC protocol adapter (direct TCP, port 7709).
//
// Implements the TDX MAC binary protocol for real-time data channels:
// block ranking, capital flow, auction, abnormal stocks, multi-day minute data.
//
// Protocol reference: external TDX project (MIT license).
package adapters

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"
)

const (
	macConnectTimeout = 5 * time.Second
	macReadTimeout    = 10 * time.Second
	macWriteTimeout   = 3 * time.Second

	// MAC frame header constants
	macHeadFlag   = 0x1c
	macCustomize  = 0
	macVersion    = 1
	macHeaderSize = 10

	// Response frame header
	respHeaderSize = 16
	respMagic      = 0x0074CBB1
)

// Known TDX MAC servers (port 7709). Tried in order; first reachable wins.
var macFallbackAddrs = []string{
	"119.147.212.81:7709",
	"119.147.212.168:7709",
	"115.238.56.58:7709",
	"123.125.104.230:7709",
	"180.153.18.170:7709",
	"61.152.107.247:7709",
	"124.74.236.65:7709",
}

// MacAdapter fetches TDX advanced data via direct MAC protocol TCP connection.
// TCP connections are pooled and reused to avoid the cost of repeated
// handshakes to the remote server. Falls back across known servers (macFallbackAddrs).
type MacAdapter struct {
	addrs []string
	idx   int
	mu    sync.Mutex
	conn  net.Conn
}

// NewMacAdapter creates a new MAC protocol adapter.
func NewMacAdapter(addr string) *MacAdapter {
	if addr != "" {
		return &MacAdapter{addrs: []string{addr}}
	}
	return &MacAdapter{addrs: append([]string(nil), macFallbackAddrs...)}
}

func (a *MacAdapter) Name() string      { return "mac" }
func (a *MacAdapter) Markets() []string  { return []string{"CN"} }
func (a *MacAdapter) RequiresAuth() bool { return false }

// Close closes the pooled connection, if any.
func (a *MacAdapter) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.conn != nil {
		err := a.conn.Close()
		a.conn = nil
		return err
	}
	return nil
}

// getConn returns the pooled connection, dialing a new one if needed.
// Caller must hold a.mu. Tries all fallback addresses in order on failure.
func (a *MacAdapter) getConn() (net.Conn, error) {
	if a.conn != nil {
		return a.conn, nil
	}
	for i := 0; i < len(a.addrs); i++ {
		addr := a.addrs[(a.idx+i)%len(a.addrs)]
		conn, err := net.DialTimeout("tcp", addr, macConnectTimeout)
		if err == nil {
			a.conn = conn
			a.idx = (a.idx + i) % len(a.addrs)
			return conn, nil
		}
	}
	return nil, fmt.Errorf("mac: all %d servers unreachable (tried %v)", len(a.addrs), a.addrs)
}

func (a *MacAdapter) IsAvailable(ctx context.Context) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.conn != nil {
		return true
	}
	for _, addr := range a.addrs {
		conn, err := net.DialTimeout("tcp", addr, macConnectTimeout)
		if err == nil {
			conn.Close()
			return true
		}
	}
	return false
}

// ── Protocol helpers ─────────────────────────────────────────────────────────

// buildMACRequest constructs a MAC protocol request frame.
// Format: 10-byte header + 2-byte msg_id + command body.
func buildMACRequest(msgID uint16, body []byte, headFlag uint8) []byte {
	inner := make([]byte, 2+len(body))
	binary.LittleEndian.PutUint16(inner, msgID)
	copy(inner[2:], body)

	hdr := make([]byte, macHeaderSize)
	hdr[0] = headFlag
	// customize: bytes 1-4 = 0 (default)
	hdr[5] = macVersion
	binary.LittleEndian.PutUint16(hdr[6:], uint16(len(inner)))
	binary.LittleEndian.PutUint16(hdr[8:], uint16(len(inner)))

	return append(hdr, inner...)
}

// sendMACRequest sends a request and reads the full response.
// Uses the pooled connection; reconnects transparently on errors.
// Retries once on transient read errors.
func (a *MacAdapter) sendMACRequest(msgID uint16, body []byte, headFlag uint8) ([]byte, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	conn, err := a.getConn()
	if err != nil {
		return nil, err
	}

	req := buildMACRequest(msgID, body, headFlag)

	for attempt := 0; attempt < 2; attempt++ {
		if attempt > 0 {
			// Reconnect on retry
			if a.conn != nil {
				a.conn.Close()
				a.conn = nil
			}
			conn, err = a.getConn()
			if err != nil {
				return nil, err
			}
			time.Sleep(200 * time.Millisecond)
		}

		conn.SetWriteDeadline(time.Now().Add(macWriteTimeout))
		if _, err := conn.Write(req); err != nil {
			conn.Close()
			a.conn = nil
			continue
		}

		conn.SetReadDeadline(time.Now().Add(macReadTimeout))

		// Read 16-byte response header
		respHdr := make([]byte, respHeaderSize)
		if _, err := io.ReadFull(conn, respHdr); err != nil {
			conn.Close()
			a.conn = nil
			continue
		}

		u0 := binary.LittleEndian.Uint32(respHdr[0:])
		if u0 != respMagic {
			conn.Close()
			a.conn = nil
			continue
		}
		zipFlag := respHdr[4]
		zipsize := int(binary.LittleEndian.Uint16(respHdr[12:]))
		unzipsize := int(binary.LittleEndian.Uint16(respHdr[14:]))
		if zipsize <= 0 || zipsize > 1<<24 {
			continue
		}

		// Read body
		rawBody := make([]byte, zipsize)
		if _, err := io.ReadFull(conn, rawBody); err != nil {
			conn.Close()
			a.conn = nil
			continue
		}

		// Decompress if needed (bit4=1 means zlib compressed)
		if zipFlag&0x10 != 0 {
			r, err := zlib.NewReader(bytes.NewReader(rawBody))
			if err != nil {
				return nil, err
			}
			defer r.Close()
			var buf bytes.Buffer
			if _, err := io.Copy(&buf, r); err != nil {
				return nil, fmt.Errorf("mac: zlib decompress: %w", err)
			}
			if buf.Len() != unzipsize {
				return nil, fmt.Errorf("mac: decompressed size mismatch: got %d, want %d", buf.Len(), unzipsize)
			}
			return buf.Bytes(), nil
		}

		return rawBody, nil
	}

	return nil, err // last error from the loop
}

// ── Block Trade Ranking ───────────────────────────────────────────────────────

// BlockRank represents a block trade ranking entry.
type BlockRank struct {
	Symbol string  `json:"symbol"`
	Name   string  `json:"name"`
	Price  float64 `json:"price"`
	Volume float64 `json:"volume"`
	Amount float64 `json:"amount"`
	Date   string  `json:"date"`
}

// GetBlockRank returns today's block trade ranking data.
// Market: 0=SZ, 1=SH. SortField: 0=amount, 1=volume.
func (a *MacAdapter) GetBlockRank(market int, sortField int, count int) ([]BlockRank, error) {
	if count <= 0 {
		count = 20
	}
	body := make([]byte, 8)
	body[0] = byte(market)
	body[1] = byte(sortField)
	binary.LittleEndian.PutUint16(body[2:], uint16(0)) // start
	binary.LittleEndian.PutUint16(body[4:], uint16(count))
	// 0x122B = board members quotes (reused for block rank)

	resp, err := a.sendMACRequest(0x122B, body, macHeadFlag)
	if err != nil {
		return nil, err
	}

	// Response: 2-byte count + N * 64-byte records
	if len(resp) < 2 {
		return []BlockRank{}, nil
	}
	n := int(binary.LittleEndian.Uint16(resp[0:]))
	results := make([]BlockRank, 0, n)
	for i := 0; i < n; i++ {
		offset := 2 + i*64
		if offset+64 > len(resp) {
			break
		}
		rec := resp[offset:]
		r := BlockRank{
			Symbol: strings.TrimSpace(string(rec[0:8])),
			Name:   strings.TrimSpace(gbkToString(rec[8:24])),
			Price:  float64(binary.LittleEndian.Uint32(rec[32:])) / 1000,
			Volume: float64(binary.LittleEndian.Uint32(rec[40:])),
			Amount: float64(binary.LittleEndian.Uint32(rec[44:])) / 10000,
		}
		results = append(results, r)
	}
	return results, nil
}

// ── Capital Flow ──────────────────────────────────────────────────────────────

// CapitalFlow holds capital flow per stock.
type CapitalFlow struct {
	Symbol     string  `json:"symbol"`
	Name       string  `json:"name"`
	MainFlow   float64 `json:"main_flow"`
	SuperFlow  float64 `json:"super_flow"`
	LargeFlow  float64 `json:"large_flow"`
	MediumFlow float64 `json:"medium_flow"`
	SmallFlow  float64 `json:"small_flow"`
}

// GetCapitalFlow returns real-time capital flow data for a symbol.
// Uses 0x1218 command with head=2 (SymbolCapitalFlow).
func (a *MacAdapter) GetCapitalFlow(symbol string) (*CapitalFlow, error) {
	mkt, code, err := parseSymbol(symbol)
	if err != nil {
		return nil, err
	}

	// Build request: H:market + 8s:code(padded) + 21s:"Stock_ZJLX"
	body := make([]byte, 2+8+21)
	binary.LittleEndian.PutUint16(body[0:], uint16(mkt))
	copy(body[2:10], padCode(code))
	copy(body[10:], []byte("Stock_ZJLX"))

	resp, err := a.sendMACRequest(0x1218, body, 2) // head_flag=2 for SymbolCapitalFlow
	if err != nil {
		return nil, err
	}

	// Skip 27-byte header, remainder is GBK JSON
	if len(resp) < 27 {
		return nil, fmt.Errorf("mac: capital flow response too short (%d bytes)", len(resp))
	}
	jsonStr := gbkToString(resp[27:])
	// JSON format: [[main_in, main_out, retail_in, retail_out], [5d_data...]]
	// Simplified parsing — extract key fields
	cf := &CapitalFlow{Symbol: symbol}

	// Parse the float values from JSON-like structure
	// Format: "[[12.3, 4.5, ...], [...]]"
	vals := extractFloats(jsonStr)
	if len(vals) >= 4 {
		cf.MainFlow = vals[0] - vals[1]   // main_in - main_out
		cf.SmallFlow = vals[2] - vals[3]   // retail_in - retail_out
	}
	if len(vals) >= 10 {
		cf.SuperFlow = vals[6]
		cf.LargeFlow = vals[7]
		cf.MediumFlow = vals[8]
	}

	return cf, nil
}

// ── Auction (集合竞价) ────────────────────────────────────────────────────────

// AuctionItem represents one collection auction time point.
type AuctionItem struct {
	Time      string  `json:"time"`
	Price     float64 `json:"price"`
	Matched   int64   `json:"matched"`
	Unmatched int64   `json:"unmatched"`
}

// GetAuction returns pre-market call auction data for a symbol.
// 0x123D command.
func (a *MacAdapter) GetAuction(symbol string) ([]AuctionItem, error) {
	mkt, code, err := parseSymbol(symbol)
	if err != nil {
		return nil, err
	}

	body := make([]byte, 2+22+4+4+10)
	binary.LittleEndian.PutUint16(body[0:], uint16(mkt))
	copy(body[2:24], padCode(code))
	// start=0, count=500
	binary.LittleEndian.PutUint32(body[24:], 0)
	binary.LittleEndian.PutUint32(body[28:], 500)

	resp, err := a.sendMACRequest(0x123D, body, macHeadFlag)
	if err != nil {
		return nil, err
	}

	// Header: 2B market + 22B code + 4B count = 28B + 8B padding = 36B
	if len(resp) < 36 {
		return nil, fmt.Errorf("mac: auction response too short")
	}
	count := int(binary.LittleEndian.Uint32(resp[24:]))
	items := make([]AuctionItem, 0, count)
	for i := 0; i < count; i++ {
		offset := 36 + i*16
		if offset+16 > len(resp) {
			break
		}
		r := resp[offset:]
		sec := binary.LittleEndian.Uint32(r[0:])
		price := float64(int32(binary.LittleEndian.Uint32(r[4:]))) / 1000
		matched := int64(int32(binary.LittleEndian.Uint32(r[8:])))
		unmatched := int64(int32(binary.LittleEndian.Uint32(r[12:])))

		h := sec / 3600
		m := (sec % 3600) / 60
		s := sec % 60

		items = append(items, AuctionItem{
			Time:      fmt.Sprintf("%02d:%02d:%02d", h, m, s),
			Price:     price,
			Matched:   matched,
			Unmatched: unmatched,
		})
	}
	return items, nil
}

// ── Abnormal Stock Monitoring ─────────────────────────────────────────────────

// AbnormalStock represents an abnormally behaving stock.
// 0x1237 command.
type AbnormalStock struct {
	Symbol    string  `json:"symbol"`
	Name      string  `json:"name"`
	Price     float64 `json:"price"`
	ChangePct float64 `json:"change_pct"`
	Reason    string  `json:"reason"`
	Volume    float64 `json:"volume"`
	Turnover  float64 `json:"turnover"`
}

// GetAbnormalStocks returns stocks with abnormal price/volume behavior.
func (a *MacAdapter) GetAbnormalStocks(market int) ([]AbnormalStock, error) {
	body := make([]byte, 4)
	binary.LittleEndian.PutUint32(body[0:], uint32(market))

	resp, err := a.sendMACRequest(0x1237, body, macHeadFlag)
	if err != nil {
		return nil, err
	}

	// Response: 2B count + N * records
	if len(resp) < 2 {
		return []AbnormalStock{}, nil
	}
	count := int(binary.LittleEndian.Uint16(resp[0:]))
	results := make([]AbnormalStock, 0, count)
	for i := 0; i < count; i++ {
		offset := 2 + i*86
		if offset+86 > len(resp) {
			break
		}
		rec := resp[offset:]
		r := AbnormalStock{
			Symbol:    strings.TrimSpace(string(rec[0:6])),
			Name:      strings.TrimSpace(gbkToString(rec[6:22])),
			Price:     float64(int32(binary.LittleEndian.Uint32(rec[36:]))) / 1000,
			ChangePct: float64(int32(binary.LittleEndian.Uint32(rec[60:]))) / 100, // divided
			Volume:    float64(int32(binary.LittleEndian.Uint32(rec[44:]))),
			Turnover:  float64(int32(binary.LittleEndian.Uint32(rec[50:]))) / 1000,
			Reason:    describeUnusual(rec[28]),
		}
		results = append(results, r)
	}
	return results, nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func parseSymbol(symbol string) (int, string, error) {
	symbol = strings.TrimSpace(symbol)
	if strings.HasPrefix(symbol, "sh") || strings.HasPrefix(symbol, "SH") {
		return 1, symbol[2:], nil
	}
	if strings.HasPrefix(symbol, "sz") || strings.HasPrefix(symbol, "SZ") {
		return 0, symbol[2:], nil
	}
	// Try to guess: 6-digit starting with 6 = SH, 0/3/00 = SZ
	if len(symbol) == 6 {
		if symbol[0] == '6' {
			return 1, symbol, nil
		}
		return 0, symbol, nil
	}
	return 0, "", fmt.Errorf("mac: cannot parse symbol %q", symbol)
}

func padCode(code string) []byte {
	buf := make([]byte, 22)
	copy(buf, code)
	return buf
}

func gbkToString(data []byte) string {
	// Simple GBK-to-UTF8 conversion for Chinese stock names
	// For production, use golang.org/x/text/encoding/simplifiedchinese
	n := bytes.IndexByte(data, 0)
	if n < 0 {
		n = len(data)
	}
	return string(data[:n])
}

func extractFloats(jsonStr string) []float64 {
	var result []float64
	var current float64
	var decimal, sign, count int
	sign = 1
	for _, c := range jsonStr {
		switch {
		case c == '-':
			sign = -1
		case c >= '0' && c <= '9':
			if decimal > 0 {
				current += float64(c-'0') / float64(pow10(decimal))
				decimal++
			} else {
				current = current*10 + float64(c-'0')
				count++
			}
		case c == '.':
			decimal = 1
		case c == ',' || c == ']':
			if count > 0 || decimal > 0 {
				result = append(result, current*float64(sign))
				current = 0
				decimal = 0
				sign = 1
				count = 0
			}
		}
	}
	return result
}

func pow10(n int) int {
	p := 1
	for i := 0; i < n; i++ {
		p *= 10
	}
	return p
}

func describeUnusual(flag byte) string {
	switch flag {
	case 0x03:
		return "主力异动"
	case 0x04:
		return "加速拉升"
	case 0x05:
		return "加速下跌"
	case 0x06:
		return "低位反弹"
	case 0x07:
		return "高位回落"
	case 0x08:
		return "撑杆跳高"
	case 0x09:
		return "平台跳水"
	case 0x0A:
		return "单笔冲涨跌"
	case 0x0B:
		return "区间放量"
	case 0x0C:
		return "区间缩量"
	case 0x10:
		return "大单托盘"
	default:
		return fmt.Sprintf("异动(%02x)", flag)
	}
}

// HealthCheck performs a basic connectivity test.
func (a *MacAdapter) HealthCheck(ctx context.Context) error {
	for _, addr := range a.addrs {
		conn, err := net.DialTimeout("tcp", addr, macConnectTimeout)
		if err == nil {
			conn.Close()
			return nil
		}
	}
	return fmt.Errorf("mac health: all %d servers unreachable", len(a.addrs))
}
