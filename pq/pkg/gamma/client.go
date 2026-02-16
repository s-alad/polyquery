package gamma

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/go-querystring/query"
)

const BaseURL = "https://gamma-api.polymarket.com"

type Client struct {
	http    *http.Client
	baseURL string
}

func NewClient() *Client {
	return &Client{
		http:    &http.Client{Timeout: 15 * time.Second},
		baseURL: BaseURL,
	}
}

func (c *Client) GetMarkets(q MarketQuery) ([]Market, error) {
	params, _ := query.Values(q)

	resp, err := c.http.Get(fmt.Sprintf("%s/markets?%s", c.baseURL, params.Encode()))
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("gamma returned %d: %s", resp.StatusCode, body)
	}

	var markets []Market
	if err := json.NewDecoder(resp.Body).Decode(&markets); err != nil {
		return nil, fmt.Errorf("decode failed: %w", err)
	}
	return markets, nil
}
