package topics

type DepthLevel struct {
	Price  float64 `json:"price"`
	Volume float64 `json:"volume"`
}

type DepthUpdate struct {
	Symbol string        `json:"symbol"`
	Bids   []DepthLevel  `json:"bids"`
	Asks   []DepthLevel  `json:"asks"`
}
