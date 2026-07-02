package topics

type KlineUpdate struct {
	Symbol   string  `json:"symbol"`
	Interval string  `json:"interval"`
	Time     int64   `json:"time"`
	Open     float64 `json:"open"`
	High     float64 `json:"high"`
	Low      float64 `json:"low"`
	Close    float64 `json:"close"`
	Volume   float64 `json:"volume"`
	IsClosed bool    `json:"is_closed"`
}
