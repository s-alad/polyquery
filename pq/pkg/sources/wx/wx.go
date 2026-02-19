// Package wx fetches Weather Underground historical observations.
package wx

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"
)

const api = "https://api.weather.com/v1/location/%s:9:%s/observations/historical.json?apiKey=e1f10a1e78da46f5b10a1e78da96f525&units=%s&startDate=%s&endDate=%s"

var hc = &http.Client{Timeout: 15 * time.Second}

type Obs struct {
	ValidTimeGMT int64 `json:"valid_time_gmt"`
	Temp         int   `json:"temp"`
}

// Fetch returns all observations for the given station/country/unit on date (YYYYMMDD).
// unit: "m" for °C, "e" for °F.
func Fetch(station, country, unit, date string) ([]Obs, error) {
	resp, err := hc.Get(fmt.Sprintf(api, station, country, unit, date, date))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var wr struct {
		Observations []Obs `json:"observations"`
	}
	if err := json.Unmarshal(raw, &wr); err != nil {
		return nil, fmt.Errorf("weather decode: %w", err)
	}
	return wr.Observations, nil
}

// Stats returns the max temperature and time of the last observation.
func Stats(obs []Obs) (maxTemp int, lastObs time.Time) {
	maxTemp = math.MinInt32
	for _, o := range obs {
		if o.Temp > maxTemp {
			maxTemp = o.Temp
		}
		if t := time.Unix(o.ValidTimeGMT, 0).UTC(); t.After(lastObs) {
			lastObs = t
		}
	}
	return
}
