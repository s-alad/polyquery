// wxwatch polls a Weather Underground station until the day's observations stop
// advancing, then prints each market's outcome and UMA propose link.
//
// Usage:
//
//	wxwatch <polymarket-event-url>
//
// It polls every 60s. When the last observation timestamp hasn't changed for
// two consecutive polls (station day ended), it prints results and exits.
package main

import (
	"context"
	"fmt"
	"math"
	"net/url"
	"os"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/salad/polyquery/pq/pkg/chain"
	"github.com/salad/polyquery/pq/pkg/gamma"
	"github.com/salad/polyquery/pq/pkg/sources/wx"
	"github.com/salad/polyquery/pq/pkg/utils"
)

const pollInterval = 60 * time.Second

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: wxwatch <polymarket-event-url>")
		os.Exit(1)
	}
	if err := run(os.Args[1]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var (
	belowRe  = regexp.MustCompile(`be (-?\d+)°[CF] or below`)
	higherRe = regexp.MustCompile(`be (-?\d+)°[CF] or higher`)
	rangeRe  = regexp.MustCompile(`between (-?\d+)-(-?\d+)°[CF]`)
	exactRe  = regexp.MustCompile(`be (-?\d+)°[CF]`)
	unitRe   = regexp.MustCompile(`(-?\d+)°([CF])`)
)

type bracket struct {
	market            gamma.Market
	low, high         int
	orBelow, orHigher bool
}

func parseBracket(m gamma.Market) (bracket, bool) {
	b, q := bracket{market: m}, m.Question
	if match := belowRe.FindStringSubmatch(q); match != nil {
		v, _ := strconv.Atoi(match[1])
		b.low, b.high, b.orBelow = math.MinInt32, v, true
		return b, true
	}
	if match := higherRe.FindStringSubmatch(q); match != nil {
		v, _ := strconv.Atoi(match[1])
		b.low, b.high, b.orHigher = v, math.MaxInt32, true
		return b, true
	}
	if match := rangeRe.FindStringSubmatch(q); match != nil {
		b.low, _ = strconv.Atoi(match[1])
		b.high, _ = strconv.Atoi(match[2])
		return b, true
	}
	if match := exactRe.FindStringSubmatch(q); match != nil {
		v, _ := strconv.Atoi(match[1])
		b.low, b.high = v, v
		return b, true
	}
	return b, false
}

func (b bracket) label() string {
	switch {
	case b.orBelow:
		return fmt.Sprintf("≤%d°", b.high)
	case b.orHigher:
		return fmt.Sprintf("≥%d°", b.low)
	case b.low == b.high:
		return fmt.Sprintf("%d°", b.low)
	default:
		return fmt.Sprintf("%d-%d°", b.low, b.high)
	}
}

func (b bracket) outcome(maxTemp int) string {
	if b.orHigher {
		if maxTemp >= b.low {
			return "Yes"
		}
		return "No"
	}
	if maxTemp >= b.low && maxTemp <= b.high {
		return "Yes"
	}
	return "No"
}

func run(arg string) error {
	slug := extractSlug(arg)
	gc := gamma.NewClient()
	event, err := gc.GetEvent(slug)
	if err != nil {
		return err
	}
	log.Info("event", "title", event.Title, "end", event.EndDate, "markets", len(event.Markets))

	endDate, _ := time.Parse(time.RFC3339, event.EndDate)

	var brackets []bracket
	unit := "m"
	for _, m := range event.Markets {
		if match := unitRe.FindStringSubmatch(m.Question); match != nil && match[2] == "F" {
			unit = "e"
		}
		if b, ok := parseBracket(m); ok {
			brackets = append(brackets, b)
		}
	}
	sort.Slice(brackets, func(i, j int) bool { return brackets[i].low < brackets[j].low })

	station, country := extractStation(event.Markets[0].ResolutionSource)
	if station == "" {
		return fmt.Errorf("could not extract station from: %s", event.Markets[0].ResolutionSource)
	}
	date := endDate.Format("20060102")

	log.Info("polling", "station", station, "interval", pollInterval)

	var prevLast time.Time
	for {
		obs, err := wx.Fetch(station, country, unit, date)
		if err != nil {
			log.Warn("fetch failed", "err", err)
			time.Sleep(pollInterval)
			continue
		}
		maxTemp, lastObs := wx.Stats(obs)
		log.Info("poll", "obs", len(obs), "max", maxTemp, "last", lastObs.UTC().Format("15:04 UTC"))

		if !lastObs.IsZero() && lastObs.Equal(prevLast) {
			log.Info("station day ended — resolving")
			return resolve(brackets, maxTemp, lastObs)
		}
		prevLast = lastObs
		time.Sleep(pollInterval)
	}
}

func resolve(brackets []bracket, maxTemp int, lastObs time.Time) error {
	uma := chain.NewUmaClient("")
	ctx := context.Background()

	fmt.Printf("\n=== OUTCOMES (max=%d° at %s) ===\n\n", maxTemp, lastObs.UTC().Format("15:04 UTC"))
	for _, b := range brackets {
		outcome := b.outcome(maxTemp)
		m := b.market
		link := "-"
		if l, err := uma.UmaLink(ctx, m.ID, m.ResolvedBy, m.QuestionID); err == nil {
			link = utils.TL(l, l)
		}
		fmt.Printf("  %-8s  %-4s  %s\n", b.label(), outcome, link)
	}
	fmt.Println()
	return nil
}

func extractSlug(input string) string {
	u, err := url.Parse(input)
	if err != nil || u.Host == "" {
		return input
	}
	// /event/{event-slug} or /event/{event-slug}/{market-slug}
	if strings.Contains(u.Path, "/event/") {
		parts := strings.SplitN(strings.TrimPrefix(u.Path, "/event/"), "/", 2)
		return parts[0]
	}
	return path.Base(u.Path)
}

func extractStation(src string) (string, string) {
	u, err := url.Parse(src)
	if err != nil {
		return "", ""
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) >= 5 {
		return parts[len(parts)-1], strings.ToUpper(parts[2])
	}
	return "", ""
}
