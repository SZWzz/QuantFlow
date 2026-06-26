// mac.go — TDX MAC protocol adapter (direct TCP, port 7709).
//
// The TDX MAC protocol provides block/capital-flow/auction/abnormal-stocks
// data that is not available through the standard quote/OHLCV interface.
// This adapter reuses TDX connection patterns from mootdx with direct TCP.
package adapters

import (
	"bufio"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"quantflow/internal/market"
)

const (
	macDefaultAddr     = "119.147.212.81:7709"
	macConnectTimeout  = 5 * time.Second
	macReadTimeout     = 10 * time.Second
	macWriteTimeout    = 3 * time.Second
)

// MacAdapter fetches TDX advanced data via direct MAC protocol TCP connection.
type MacAdapter struct {
	addr string
}

// NewMacAdapter creates a new MAC protocol adapter.
func NewMacAdapter(addr string) *MacAdapter {
	if addr == "" {
		addr = macDefaultAddr
	}
	return &MacAdapter{addr: addr}
}

func (a *MacAdapter) Name() string      { return "mac" }
func (a *MacAdapter) Markets() []string  { return []string{"CN"} }
func (a *MacAdapter) RequiresAuth() bool { return false }

func (a *MacAdapter) IsAvailable(ctx context.Context) bool {
	conn, err := net.DialTimeout("tcp", a.addr, macConnectTimeout)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// ── Connection helpers ────────────────────────────────────────────────────────

// dial connects to the MAC server with timeout.
func (a *MacAdapter) dial() (net.Conn, error) {
	conn, err := net.DialTimeout("tcp", a.addr, macConnectTimeout)
	if err != nil {
		return nil, fmt.Errorf("mac: dial %s: %w", a.addr, err)
	}
	return conn, nil
}

// sendAndRecv sends a raw request frame and reads the response.
// TDX MAC uses a simple binary frame: 2-byte length prefix + payload.
func (a *MacAdapter) sendAndRecv(cmd byte, params []string) ([]byte, error) {
	conn, err := a.dial()
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	conn.SetWriteDeadline(time.Now().Add(macWriteTimeout))

	// Build payload: command byte + param count + params
	payload := make([]byte, 1, 256)
	payload[0] = cmd
	payload = append(payload, byte(len(params)))
	for _, p := range params {
		pb := []byte(p)
		payload = append(payload, byte(len(pb)))
		payload = append(payload, pb...)
	}

	// Write length-prefixed frame
	frameLen := make([]byte, 2)
	binary.BigEndian.PutUint16(frameLen, uint16(len(payload)))
	if _, err := conn.Write(frameLen); err != nil {
		return nil, fmt.Errorf("mac: write frame header: %w", err)
	}
	if _, err := conn.Write(payload); err != nil {
		return nil, fmt.Errorf("mac: write payload: %w", err)
	}

	// Read response
	conn.SetReadDeadline(time.Now().Add(macReadTimeout))
	reader := bufio.NewReader(conn)

	respHeader := make([]byte, 2)
	if _, err := io.ReadFull(reader, respHeader); err != nil {
		return nil, fmt.Errorf("mac: read response header: %w", err)
	}
	respLen := binary.BigEndian.Uint16(respHeader)
	respPayload := make([]byte, respLen)
	if _, err := io.ReadFull(reader, respPayload); err != nil {
		return nil, fmt.Errorf("mac: read response payload: %w", err)
	}

	_ = json.Marshal // placeholder for future structured output
	return respPayload, nil
}

// ── MAC Protocol Commands ─────────────────────────────────────────────────────

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
// Market: 0=SZ, 1=SH. SortField: 0=amount, 1=volume, 2=count.
func (a *MacAdapter) GetBlockRank(market int, sortField int, count int) ([]BlockRank, error) {
	_ = market
	_ = sortField
	_ = count
	return nil, fmt.Errorf("mac: GetBlockRank not yet implemented (protocol pending)")
}

// CapitalFlow holds capital flow per stock.
type CapitalFlow struct {
	Symbol       string  `json:"symbol"`
	Name         string  `json:"name"`
	MainFlow     float64 `json:"main_flow"`     // 主力净流入(万元)
	SuperFlow    float64 `json:"super_flow"`    // 超大单净流入(万元)
	LargeFlow    float64 `json:"large_flow"`    // 大单净流入(万元)
	MediumFlow   float64 `json:"medium_flow"`   // 中单净流入(万元)
	SmallFlow    float64 `json:"small_flow"`    // 小单净流入(万元)
}

// GetCapitalFlow returns real-time capital flow data.
func (a *MacAdapter) GetCapitalFlow(market int) ([]CapitalFlow, error) {
	_ = market
	return nil, fmt.Errorf("mac: GetCapitalFlow not yet implemented (protocol pending)")
}

// AuctionEntry represents a call auction (集合竞价) data point.
type AuctionEntry struct {
	Symbol    string  `json:"symbol"`
	OpenPrice float64 `json:"open_price"`
	OpenVol   float64 `json:"open_vol"`
	PreClose  float64 `json:"pre_close"`
	BidPrice  float64 `json:"bid_price"`
	BidVol    float64 `json:"bid_vol"`
	AskPrice  float64 `json:"ask_price"`
	AskVol    float64 `json:"ask_vol"`
}

// GetAuction returns pre-market call auction data.
func (a *MacAdapter) GetAuction(market int) ([]AuctionEntry, error) {
	_ = market
	return nil, fmt.Errorf("mac: GetAuction not yet implemented (protocol pending)")
}

// AbnormalStock represents an abnormally behaving stock (涨跌异常).
type AbnormalStock struct {
	Symbol     string  `json:"symbol"`
	Name       string  `json:"name"`
	Price      float64 `json:"price"`
	ChangePct  float64 `json:"change_pct"`
	Reason     string  `json:"reason"`
	Volume     float64 `json:"volume"`
	Turnover   float64 `json:"turnover"`
}

// GetAbnormalStocks returns a list of stocks with abnormal price/volume behavior.
func (a *MacAdapter) GetAbnormalStocks(market int) ([]AbnormalStock, error) {
	_ = market
	return nil, fmt.Errorf("mac: GetAbnormalStocks not yet implemented (protocol pending)")
}

// MultiDayMinute holds multi-day minute-level data for a symbol.
type MultiDayMinute struct {
	Symbol string          `json:"symbol"`
	Days   []MultiDayData `json:"days"`
}

// MultiDayData is one day of minute-level OHLCV data.
type MultiDayData struct {
	Date  string          `json:"date"`
	Ticks []market.MinuteTick `json:"ticks"`
}

// GetMultiDayMinute fetches multi-day minute-level data for a symbol.
// days: number of recent trading days to fetch (max ~5).
func (a *MacAdapter) GetMultiDayMinute(symbol string, days int) (*MultiDayMinute, error) {
	if days <= 0 {
		days = 1
	}
	if days > 5 {
		days = 5
	}

	// Normalize to TDX format: sh600519 or sz000001
	symbol = strings.ToLower(strings.TrimSpace(symbol))
	if !strings.HasPrefix(symbol, "sh") && !strings.HasPrefix(symbol, "sz") {
		return nil, fmt.Errorf("mac: symbol must be shXXXXXX or szXXXXXX format, got %q", symbol)
	}

	_ = days
	return nil, fmt.Errorf("mac: GetMultiDayMinute not yet implemented (protocol pending)")
}

// HealthCheck performs a basic connectivity test to the MAC server.
func (a *MacAdapter) HealthCheck(ctx context.Context) error {
	conn, err := a.dial()
	if err != nil {
		return fmt.Errorf("mac health: %w", err)
	}
	conn.Close()
	return nil
}
