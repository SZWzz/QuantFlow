package topics

type Tick struct {
	Symbol string  `json:"symbol"`
	Price  float64 `json:"price"`
	Volume float64 `json:"volume"`
	Time   int64   `json:"time"`
	Side   string  `json:"side"`
}
