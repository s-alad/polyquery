package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/charmbracelet/log"
	"github.com/salad/polyquery/pq/pkg/chain"
	"github.com/salad/polyquery/pq/pkg/gamma"
	"github.com/salad/polyquery/pq/pkg/utils"
	"github.com/spf13/cobra"
)

const wxAPI = "https://api.weather.com/v1/location/%s:9:%s/observations/historical.json?apiKey=e1f10a1e78da46f5b10a1e78da96f525&units=%s&startDate=%s&endDate=%s"

var hc = &http.Client{Timeout: 15 * time.Second}

var rcmd = &cobra.Command{
	Use:   "wxgap <polymarket-event-url-or-slug>",
	Short: "analyze weather market proposal timing gaps",
	Args:  cobra.ExactArgs(1),
	RunE:  func(_ *cobra.Command, args []string) error { return run(args[0]) },
}

func main() {
	if err := rcmd.Execute(); err != nil {
		os.Exit(1)
	}
}

type wxObs struct {
	ValidTimeGMT int64 `json:"valid_time_gmt"`
	Temp         int   `json:"temp"`
}

type bracket struct {
	market            gamma.Market
	low, high         int
	orBelow, orHigher bool
}

var (
	belowRe  = regexp.MustCompile(`be (-?\d+)°[CF] or below`)
	higherRe = regexp.MustCompile(`be (-?\d+)°[CF] or higher`)
	rangeRe  = regexp.MustCompile(`between (-?\d+)-(-?\d+)°[CF]`)
	exactRe  = regexp.MustCompile(`be (-?\d+)°[CF]`)
	unitRe   = regexp.MustCompile(`(-?\d+)°([CF])`)
)

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

func (b bracket) contains(t int) bool { return t >= b.low && t <= b.high }

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

// resolvableAt returns when this bracket first became resolvable and an outcome label.
// Exceeded brackets resolve at the first obs where temp > bracket.high.
// Day-end brackets (Yes or above daily max) resolve at the last observation.
func (b bracket) resolvableAt(obs []wxObs, maxTemp int, lastObs time.Time) (time.Time, string) {
	if b.contains(maxTemp) {
		return lastObs, "Yes (day end)"
	}
	if b.orHigher {
		return lastObs, "No (day end)"
	}
	for _, o := range obs {
		if o.Temp > b.high {
			return time.Unix(o.ValidTimeGMT, 0).UTC(), "No (exceeded)"
		}
	}
	return lastObs, "No (day end)"
}

func run(arg string) error {
	gc := gamma.NewClient()
	event, err := gc.GetEvent(extractSlug(arg))
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
	obs, err := fetchWeather(station, country, unit, endDate.Format("20060102"))
	if err != nil {
		return err
	}

	localTZ, localLabel := stationLocal(station)
	maxTemp, lastObs := stats(obs)
	log.Info("weather", "station", station, "obs", len(obs), "max", maxTemp, "last", fmtTime(lastObs, localTZ, localLabel))

	ids := make([]string, len(brackets))
	for i, b := range brackets {
		ids[i] = b.market.ID
	}
	uma := chain.NewUmaClient("")
	proposals := uma.FetchProposals(context.Background(), ids)

	tbl := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderRow(true).
		StyleFunc(func(_, _ int) lipgloss.Style { return lipgloss.NewStyle().Padding(0, 1) }).
		Headers("BRACKET", "OUTCOME", "RESOLVABLE", "PROPOSED", "GAP", "PROPOSER", "UMA LINK")
	if w := utils.TerminalWidth(); w > 0 {
		tbl.Width(w)
	}

	for i, b := range brackets {
		resolvable, outcome := b.resolvableAt(obs, maxTemp, lastObs)
		proposedAt, gap, proposer := "-", "-", "-"
		if p, ok := proposals[i]; ok {
			pt := time.Unix(p.Timestamp, 0).UTC()
			proposedAt = fmtTime(pt, localTZ, localLabel)
			d := pt.Sub(resolvable)
			if d < 0 {
				gap = "-" + fmtDur(-d)
			} else {
				gap = fmtDur(d)
			}
			proposer = p.Proposer[:10] + "…"
		}
		m := b.market
		umaLink := "-"
		if link, err := uma.UmaLink(context.Background(), m.ID, m.ResolvedBy, m.QuestionID); err == nil {
			umaLink = utils.TL(link, link)
		}
		tbl.Row(b.label(), outcome, fmtTime(resolvable, localTZ, localLabel), proposedAt, gap, proposer, umaLink)
	}
	fmt.Println(tbl.Render())
	return nil
}

func stats(obs []wxObs) (maxTemp int, lastObs time.Time) {
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

var est = time.FixedZone("EST", -5*3600)

var stationTZ = map[string][2]string{
	"LTAC": {"Europe/Istanbul", "TRT"},
	"KDAL": {"America/Chicago", "CST"},
	"KDFW": {"America/Chicago", "CST"},
	"KATL": {"America/New_York", "EST"},
	"RKSI": {"Asia/Seoul", "KST"},
	"RKSS": {"Asia/Seoul", "KST"},
	"LFPG": {"Europe/Paris", "CET"},
	"LFPO": {"Europe/Paris", "CET"},
}

func stationLocal(station string) (*time.Location, string) {
	if entry, ok := stationTZ[station]; ok {
		if tz, err := time.LoadLocation(entry[0]); err == nil {
			return tz, entry[1]
		}
	}
	return nil, ""
}

func fmtTime(t time.Time, localTZ *time.Location, localLabel string) string {
	s := fmt.Sprintf("%s / %s", t.UTC().Format("15:04 UTC"), t.In(est).Format("15:04 EST"))
	if localTZ != nil {
		s += " / " + t.In(localTZ).Format("15:04 ") + localLabel
	}
	return s
}

func fmtDur(d time.Duration) string {
	if h := int(d.Hours()); h > 0 {
		return fmt.Sprintf("%dh%dm", h, int(d.Minutes())%60)
	}
	return fmt.Sprintf("%dm", int(d.Minutes()))
}

func extractSlug(input string) string {
	if u, err := url.Parse(input); err == nil && u.Host != "" {
		return path.Base(u.Path)
	}
	return input
}

// extractStation parses station and country from a wunderground URL.
// .../daily/tr/çubuk/LTAC → ("LTAC","TR"), .../daily/us/tx/dallas/KDAL → ("KDAL","US")
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

func fetchWeather(station, country, unit, date string) ([]wxObs, error) {
	resp, err := hc.Get(fmt.Sprintf(wxAPI, station, country, unit, date, date))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var wr struct {
		Observations []wxObs `json:"observations"`
	}
	if err := json.Unmarshal(raw, &wr); err != nil {
		return nil, fmt.Errorf("weather decode: %w", err)
	}
	return wr.Observations, nil
}
