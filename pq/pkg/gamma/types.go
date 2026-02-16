package gamma

// https://docs.polymarket.com/api-reference/markets/list-markets.md
type Market struct {
	ID                  string  `json:"id"`
	Question            string  `json:"question"`
	ConditionID         string  `json:"conditionId"`
	Slug                string  `json:"slug"`
	VolumeNum           float64 `json:"volumeNum"`
	LiquidityNum        float64 `json:"liquidityNum"`
	EndDate             string  `json:"endDate"`
	ResolutionSource    string  `json:"resolutionSource"`
	Active              bool    `json:"active"`
	Closed              bool    `json:"closed"`
	UmaResolutionStatus string  `json:"umaResolutionStatus"`
	UmaBond             string  `json:"umaBond"`
	UmaReward           string  `json:"umaReward"`
}

// https://docs.polymarket.com/api-reference/markets/list-markets.md
type MarketQuery struct {
	VolumeMax float64 `url:"volume_num_max,omitempty"`
	VolumeMin float64 `url:"volume_num_min,omitempty"`
	Closed    *bool   `url:"closed,omitempty"`
	Active    *bool   `url:"active,omitempty"`
	EndDateMin string `url:"end_date_min,omitempty"`
	EndDateMax string `url:"end_date_max,omitempty"`
	Limit     int     `url:"limit,omitempty"`
	Offset    int     `url:"offset,omitempty"`
	Order     string  `url:"order,omitempty"`
	Ascending bool    `url:"ascending,omitempty"`
}
