package adapters

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// MarginTrading holds daily margin trading (融资融券) data for a stock.
type MarginTrading struct {
	Date   string  `json:"date"`
	RZYE   float64 `json:"rzye"`    // 融资余额(元)
	RZMRE  float64 `json:"rzmre"`   // 融资买入额(元)
	RZCHE  float64 `json:"rzche"`   // 融资偿还额(元)
	RQYE   float64 `json:"rqye"`    // 融券余额(元)
	RQMCL  float64 `json:"rqmcl"`   // 融券卖出量(股)
	RQCHL  float64 `json:"rqchl"`   // 融券偿还量(股)
	RZRQYE float64 `json:"rzrqye"`  // 融资融券余额合计(元)
}

// BlockTrade represents a single block trade (大宗交易) record.
type BlockTrade struct {
	Date       string  `json:"date"`
	DealPrice  float64 `json:"deal_price"`
	ClosePrice float64 `json:"close_price"`
	PremiumPct float64 `json:"premium_pct"` // 溢价率%
	Volume     float64 `json:"volume"`      // 成交量(股)
	Amount     float64 `json:"amount"`      // 成交额(元)
	Buyer      string  `json:"buyer"`
	Seller     string  `json:"seller"`
}

// HolderChange holds quarterly shareholder count change (股东户数变化) data.
type HolderChange struct {
	Date       string  `json:"date"`
	HolderNum  float64 `json:"holder_num"`   // 股东户数
	ChangeNum  float64 `json:"change_num"`   // 变化户数
	ChangePct  float64 `json:"change_pct"`   // 环比变化%
	AvgShares  float64 `json:"avg_shares"`   // 户均持股
}

// DividendRecord holds historical dividend (分红送转) data.
type DividendRecord struct {
	Date           string  `json:"date"`
	BonusRMB       float64 `json:"bonus_rmb"`       // 每股派息(税前)
	TransferRatio  float64 `json:"transfer_ratio"`  // 每10股转增
	BonusRatio     float64 `json:"bonus_ratio"`     // 每10股送股
	Plan           string  `json:"plan"`            // 进度
}

// EastMoneyCapitalAdapter fetches capital-side data: margin trading,
// block trades, shareholder changes, and dividend history.
// Based on a-stock-data SKILL §4.1-4.4 (EastMoney datacenter API).
type EastMoneyCapitalAdapter struct {
	client  *http.Client
	limiter *EastMoneyRateLimiter
	signals *EastMoneySignalsAdapter // reuse the datacenter query helper
}

// NewEastMoneyCapitalAdapter creates a new capital data adapter.
func NewEastMoneyCapitalAdapter() *EastMoneyCapitalAdapter {
	return &EastMoneyCapitalAdapter{
		client:  newEastMoneyHTTPClient(15 * time.Second),
		limiter: GlobalEMLimiter,
		signals: nil, // will use its own
	}
}

func (a *EastMoneyCapitalAdapter) Name() string { return "eastmoney_capital" }

func (a *EastMoneyCapitalAdapter) IsAvailable(ctx context.Context) bool {
	_, err := a.FetchMarginTrading(ctx, "600519", 5)
	return err == nil
}

// FetchMarginTrading fetches margin trading details for a stock.
func (a *EastMoneyCapitalAdapter) FetchMarginTrading(ctx context.Context, code string, pageSize int) ([]MarginTrading, error) {
	if a.signals == nil {
		a.signals = NewEastMoneySignalsAdapter()
	}

	filter := fmt.Sprintf(`(SCODE="%s")`, code)
	rows, err := a.signals.queryDatacenter("RPTA_WEB_RZRQ_GGMX", filter, pageSize, "DATE", "-1")
	if err != nil {
		return nil, fmt.Errorf("eastmoney_capital margin: %w", err)
	}

	results := make([]MarginTrading, 0, len(rows))
	for _, r := range rows {
		results = append(results, MarginTrading{
			Date:   strval(r["DATE"]),
			RZYE:   floatval(r["RZYE"]),
			RZMRE:  floatval(r["RZMRE"]),
			RZCHE:  floatval(r["RZCHE"]),
			RQYE:   floatval(r["RQYE"]),
			RQMCL:  floatval(r["RQMCL"]),
			RQCHL:  floatval(r["RQCHL"]),
			RZRQYE: floatval(r["RZRQYE"]),
		})
	}
	return results, nil
}

// FetchBlockTrades fetches block trade records for a stock.
func (a *EastMoneyCapitalAdapter) FetchBlockTrades(ctx context.Context, code string, pageSize int) ([]BlockTrade, error) {
	if a.signals == nil {
		a.signals = NewEastMoneySignalsAdapter()
	}

	filter := fmt.Sprintf(`(SECURITY_CODE="%s")`, code)
	rows, err := a.signals.queryDatacenter("RPT_DATA_BLOCKTRADE", filter, pageSize, "TRADE_DATE", "-1")
	if err != nil {
		return nil, fmt.Errorf("eastmoney_capital blocktrade: %w", err)
	}

	results := make([]BlockTrade, 0, len(rows))
	for _, r := range rows {
		closePrice := floatval(r["CLOSE_PRICE"])
		dealPrice := floatval(r["DEAL_PRICE"])
		premium := 0.0
		if closePrice > 0 {
			premium = (dealPrice/closePrice - 1) * 100
		}
		results = append(results, BlockTrade{
			Date:       strval(r["TRADE_DATE"]),
			DealPrice:  dealPrice,
			ClosePrice: closePrice,
			PremiumPct: premium,
			Volume:     floatval(r["DEAL_VOLUME"]),
			Amount:     floatval(r["DEAL_AMT"]),
			Buyer:      strval(r["BUYER_NAME"]),
			Seller:     strval(r["SELLER_NAME"]),
		})
	}
	return results, nil
}

// FetchHolderChanges fetches shareholder count change history.
func (a *EastMoneyCapitalAdapter) FetchHolderChanges(ctx context.Context, code string, pageSize int) ([]HolderChange, error) {
	if a.signals == nil {
		a.signals = NewEastMoneySignalsAdapter()
	}

	filter := fmt.Sprintf(`(SECURITY_CODE="%s")`, code)
	rows, err := a.signals.queryDatacenter("RPT_HOLDERNUMLATEST", filter, pageSize, "END_DATE", "-1")
	if err != nil {
		return nil, fmt.Errorf("eastmoney_capital holder: %w", err)
	}

	results := make([]HolderChange, 0, len(rows))
	for _, r := range rows {
		results = append(results, HolderChange{
			Date:      strval(r["END_DATE"]),
			HolderNum: floatval(r["HOLDER_NUM"]),
			ChangeNum: floatval(r["HOLDER_NUM_CHANGE"]),
			ChangePct: floatval(r["HOLDER_NUM_RATIO"]),
			AvgShares: floatval(r["AVG_FREE_SHARES"]),
		})
	}
	return results, nil
}

// FetchDividendHistory fetches dividend history for a stock.
func (a *EastMoneyCapitalAdapter) FetchDividendHistory(ctx context.Context, code string, pageSize int) ([]DividendRecord, error) {
	if a.signals == nil {
		a.signals = NewEastMoneySignalsAdapter()
	}

	filter := fmt.Sprintf(`(SECURITY_CODE="%s")`, code)
	rows, err := a.signals.queryDatacenter("RPT_SHAREBONUS_DET", filter, pageSize, "EX_DIVIDEND_DATE", "-1")
	if err != nil {
		return nil, fmt.Errorf("eastmoney_capital dividend: %w", err)
	}

	results := make([]DividendRecord, 0, len(rows))
	for _, r := range rows {
		results = append(results, DividendRecord{
			Date:          strval(r["EX_DIVIDEND_DATE"]),
			BonusRMB:      floatval(r["PRETAX_BONUS_RMB"]),
			TransferRatio: floatval(r["TRANSFER_RATIO"]),
			BonusRatio:    floatval(r["BONUS_RATIO"]),
			Plan:          strval(r["ASSIGN_PROGRESS"]),
		})
	}
	return results, nil
}

// ExDividendCalendarRecord represents a market-wide ex-dividend calendar entry.
type ExDividendCalendarRecord struct {
	Code          string  `json:"code"`
	Name          string  `json:"name"`
	ExDate        string  `json:"ex_date"`
	BonusRMB      float64 `json:"bonus_rmb"`
	TransferRatio float64 `json:"transfer_ratio"`
	BonusRatio    float64 `json:"bonus_ratio"`
	Plan          string  `json:"plan"`
	DividendYield float64 `json:"dividend_yield"`
	ClosePrice    float64 `json:"close_price"`
}

// FetchExDividendCalendar fetches market-wide ex-dividend calendar by date range.
func (a *EastMoneyCapitalAdapter) FetchExDividendCalendar(ctx context.Context, startDate, endDate string) ([]ExDividendCalendarRecord, error) {
	if a.signals == nil {
		a.signals = NewEastMoneySignalsAdapter()
	}

	filter := fmt.Sprintf(`(EX_DIVIDEND_DATE>='%s')(EX_DIVIDEND_DATE<='%s')`, startDate, endDate)
	rows, err := a.signals.queryDatacenter("RPT_SHAREBONUS_DET", filter, 500, "EX_DIVIDEND_DATE", "-1")
	if err != nil {
		return nil, fmt.Errorf("eastmoney_capital exdiv: %w", err)
	}

	results := make([]ExDividendCalendarRecord, 0, len(rows))
	for _, r := range rows {
		bonus := floatval(r["PRETAX_BONUS_RMB"])
		// DIVIDENT_RATIO from EastMoney = dividend_per_share / close_price (ratio, not %)
		divRatio := floatval(r["DIVIDENT_RATIO"])
		yield := divRatio * 100
		results = append(results, ExDividendCalendarRecord{
			Code:          strval(r["SECURITY_CODE"]),
			Name:          strval(r["SECURITY_NAME_ABBR"]),
			ExDate:        strval(r["EX_DIVIDEND_DATE"]),
			BonusRMB:      bonus,
			TransferRatio: floatval(r["IT_RATIO"]),
			BonusRatio:    floatval(r["BONUS_RATIO"]),
			Plan:          strval(r["ASSIGN_PROGRESS"]),
			DividendYield: yield,
			ClosePrice:    0, // not available from API
		})
	}
	return results, nil
}
