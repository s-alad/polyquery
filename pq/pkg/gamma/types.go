package gamma

// https://docs.polymarket.com/api-reference/markets/list-markets.md
type Market struct {
	ID               string  `json:"id"`
	Question         string  `json:"question"`
	ConditionID      string  `json:"conditionId"`
	VolumeNum        float64 `json:"volumeNum"`
	LiquidityNum     float64 `json:"liquidityNum"`
	EndDate          string  `json:"endDate"`
	ResolutionSource string  `json:"resolutionSource"`
}

// https://docs.polymarket.com/api-reference/markets/list-markets.md
type MarketQuery struct {
	VolumeMax float64 `url:"volume_num_max,omitempty"`
	VolumeMin float64 `url:"volume_num_min,omitempty"`
	Closed    *bool   `url:"closed,omitempty"`
	Active    *bool   `url:"active,omitempty"`
	Limit     int     `url:"limit,omitempty"`
	Offset    int     `url:"offset,omitempty"`
	Order     string  `url:"order,omitempty"`
	Ascending bool    `url:"ascending,omitempty"`
}
