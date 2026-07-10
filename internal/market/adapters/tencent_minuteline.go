package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"quantflow/internal/market"
)

const tencentMinuteURL = "http://ifzq.gtimg.cn/appstock/app/minute/query?_var=min_data&code=%s"

// tencentMinuteResponse is the top-level JSONP response from Tencent's minute API.
type tencentMinuteResponse struct {
	Code int                    `json:"code"`
	Msg  string                 `json:"msg"`
	Data map[string]interface{} `json:"data"`
}

// FetchMinuteLine returns today's intraday minute ticks via Tencent Finance.
// Implements market.MinuteLineProvider.
//
// Tencent endpoint: ifzq.gtimg.cn/appstock/app/minute/query
// Response: min_data={code:0, data:{sh000001:{data:{data:["0930 1234.56 7890",...]}}}}
func (a *TencentAdapter) FetchMinuteLine(symbol string) ([]market.MinuteTick, error) {
	code := toTencentCode(symbol)
	url := fmt.Sprintf(tencentMinuteURL, code)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("tencent minute: create request: %w", err)
	}
	req.Header.Set("Referer", "https://finance.qq.com/")
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, market.NewTransientErrorf("tencent minute: http error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tencent minute: HTTP %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 65536))

	bodyStr, err := decodeGBK(body)
	if err != nil {
		bodyStr = string(body)
	}

	// Strip JSONP wrapper: min_data={...}
	raw := strings.TrimSpace(bodyStr)
	if strings.HasPrefix(raw, "min_data=") {
		raw = raw[len("min_data="):]
	}

	var apiResp tencentMinuteResponse
	if err := json.Unmarshal([]byte(raw), &apiResp); err != nil {
		return nil, fmt.Errorf("tencent minute: parse error: %w", err)
	}

	if apiResp.Code != 0 {
		return nil, fmt.Errorf("tencent minute: API error code=%d msg=%s", apiResp.Code, apiResp.Msg)
	}

	// Navigate: data[code]["data"]["data"] -> []string
	outer, ok := apiResp.Data[code]
	if !ok {
		return nil, fmt.Errorf("tencent minute: no data for %s", symbol)
	}

	symbolMap, ok := outer.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("tencent minute: unexpected data format for %s", symbol)
	}

	innerRaw, ok := symbolMap["data"]
	if !ok {
		return nil, fmt.Errorf("tencent minute: no inner data for %s", symbol)
	}

	innerMap, ok := innerRaw.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("tencent minute: unexpected inner data format for %s", symbol)
	}

	minutesRaw, ok := innerMap["data"]
	if !ok {
		return nil, fmt.Errorf("tencent minute: no minute array for %s", symbol)
	}

	minuteStrings, ok := minutesRaw.([]interface{})
	if !ok || len(minuteStrings) == 0 {
		return nil, fmt.Errorf("tencent minute: empty or invalid minute data for %s", symbol)
	}

	ticks := make([]market.MinuteTick, 0, len(minuteStrings))
	cumVol := 0.0
	cumAmount := 0.0

	for _, item := range minuteStrings {
		line, ok := item.(string)
		if !ok {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		timeRaw := parts[0]
		if len(timeRaw) >= 4 {
			timeRaw = timeRaw[:2] + ":" + timeRaw[2:4]
		}

		price := parseFloatSafe(parts[1])
		vol := 0.0
		if len(parts) > 2 {
			vol = parseFloatSafe(parts[2])
		}

		cumVol += vol
		cumAmount += price * vol

		avgPrice := 0.0
		if cumVol > 0 {
			avgPrice = roundFloat(cumAmount/cumVol, 2)
		}

		ticks = append(ticks, market.MinuteTick{
			Time:     timeRaw,
			Price:    roundFloat(price, 2),
			Volume:   vol,
			AvgPrice: avgPrice,
			Amount:   roundFloat(price*vol, 2),
		})
	}

	return ticks, nil
}

// roundFloat rounds a float64 to the given number of decimal places.
func roundFloat(v float64, decimals int) float64 {
	pow := 1.0
	for i := 0; i < decimals; i++ {
		pow *= 10
	}
	return float64(int(v*pow+0.5)) / pow
}
