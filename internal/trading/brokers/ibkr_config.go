package brokers

// IBKRConfig holds configuration for the Interactive Brokers Client Portal REST API.
type IBKRConfig struct {
	Host      string `json:"host"`       // IB Gateway host (default: localhost)
	Port      int    `json:"port"`       // IB Gateway port (default: 5000)
	AccountID string `json:"account_id"` // IBKR numeric account ID
}
