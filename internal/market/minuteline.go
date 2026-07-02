// minuteline.go — minute-line (intraday) data types and provider interface.
package market

// MinuteTick represents one minute's trade data during the day.
type MinuteTick struct {
	Time     string  `json:"time"`      // "09:35"
	Price    float64 `json:"price"`     // 该分钟均价
	Volume   float64 `json:"volume"`    // 该分钟成交量
	AvgPrice float64 `json:"avg_price"` // 日内累计均价
	Amount   float64 `json:"amount"`    // 该分钟成交额(元)
}

// MinuteLineProvider is implemented by adapters that can fetch intraday
// minute-line data (primarily mootdx via TDX TCP protocol).
type MinuteLineProvider interface {
	FetchMinuteLine(symbol string) ([]MinuteTick, error)
}
