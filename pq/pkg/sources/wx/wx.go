// Package wx fetches Weather Underground historical observations.
package wx

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"sort"
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

// TypicalDayEnd fetches the past n days for the station and returns the median
// time-of-day (in UTC seconds since midnight) of the last daily observation.
// Use this to predict when today's final reading will arrive.
func TypicalDayEnd(station, country, unit string, targetDate time.Time, days int) (time.Duration, error) {
	ends := make([]time.Duration, 0, days)
	for i := 1; i <= days; i++ {
		date := targetDate.AddDate(0, 0, -i).Format("20060102")
		obs, err := Fetch(station, country, unit, date)
		if err != nil || len(obs) == 0 {
			continue
		}
		_, last := Stats(obs)
		// time-of-day in UTC
		sod := last.Truncate(24 * time.Hour)
		ends = append(ends, last.Sub(sod))
	}
	if len(ends) == 0 {
		return 0, fmt.Errorf("no historical data for %s", station)
	}
	sort.Slice(ends, func(i, j int) bool { return ends[i] < ends[j] })
	return ends[len(ends)/2], nil
}

// Interval returns the median gap between consecutive observations.
// Returns 0 if fewer than 2 observations.
func Interval(obs []Obs) time.Duration {
	if len(obs) < 2 {
		return 0
	}
	gaps := make([]time.Duration, len(obs)-1)
	for i := 1; i < len(obs); i++ {
		gaps[i-1] = time.Duration(obs[i].ValidTimeGMT-obs[i-1].ValidTimeGMT) * time.Second
	}
	sort.Slice(gaps, func(i, j int) bool { return gaps[i] < gaps[j] })
	return gaps[len(gaps)/2]
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
