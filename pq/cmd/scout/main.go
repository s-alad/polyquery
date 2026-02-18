package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/charmbracelet/log"
	"github.com/salad/polyquery/pq/pkg/chain"
	"github.com/salad/polyquery/pq/pkg/gamma"
	"github.com/salad/polyquery/pq/pkg/utils"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
)

var (
	maxvol         float64
	minvol         float64
	count          int
	interactive    bool
	resolveWithin  time.Duration
	proposable     bool
	rpcURL         string
	permissionless bool
	ended          bool
)

// flags that suppress the interactive prompt when explicitly set
var flags = []string{"max-volume", "min-volume", "count", "resolve-within", "proposable", "permissionless", "ended", "rpc"}

var rcmd = &cobra.Command{
	Use:   "scout",
	Short: "scout polymarket",
	Run:   run,
}

func init() {
	rcmd.Flags().Float64Var(&maxvol, "max-volume", 10000, "max volume USD")
	rcmd.Flags().Float64Var(&minvol, "min-volume", 0, "min volume USD")
	rcmd.Flags().IntVar(&count, "count", 0, "max results to display (0 = all)")
	rcmd.Flags().BoolVarP(&interactive, "interactive", "i", true, "prompt for filters")
	rcmd.Flags().DurationVar(&resolveWithin, "resolve-within", 7*24*time.Hour, "markets ending within this duration from now")
	rcmd.Flags().BoolVar(&proposable, "proposable", true, "only show markets with no UMA proposal yet")
	rcmd.Flags().StringVar(&rpcURL, "rpc", "", "Polygon RPC URL (needed for MOOV2/sports UMA links)")
	rcmd.Flags().BoolVar(&permissionless, "permissionless", false, "only show OOV2 markets (no whitelist needed)")
	rcmd.Flags().BoolVar(&ended, "ended", false, "only show markets past their end date (floors at 180 days ago)")
	rcmd.MarkFlagsMutuallyExclusive("ended", "resolve-within")
}

func main() {
	if err := rcmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) {
	if interactive && !lo.SomeBy(flags, func(n string) bool { return cmd.Flags().Changed(n) }) {
		prompt()
	}

	now := time.Now().UTC()
	floor := now.Add(-180 * 24 * time.Hour).Format(time.RFC3339)

	// ended: closed=true, floor endDate to 180 days ago, descending (most recent first)
	// future: closed=false, endDate window from now → now+resolveWithin, ascending
	yes, no := true, false
	q := gamma.MarketQuery{
		VolumeMax: maxvol,
		VolumeMin: minvol,
		Limit:     500,
		Order:     "endDate",
	}
	if ended {
		q.Closed = &yes
		q.EndDateMin = floor
		q.Ascending = false
	} else {
		q.Closed = &no
		q.EndDateMin = now.Format(time.RFC3339)
		q.EndDateMax = now.Add(resolveWithin).Format(time.RFC3339)
		q.Ascending = true
	}

	// fetch 500 (Gamma max) — date windows fill up fast with 5-min crypto markets
	markets, err := gamma.NewClient().GetMarkets(q)
	if err != nil {
		log.Fatal("fetch failed", "err", err)
	}

	// drop pre-UMA markets (no oracle address)
	markets = lo.Filter(markets, func(m gamma.Market, _ int) bool {
		return m.ResolvedBy != ""
	})
	if proposable {
		markets = lo.Filter(markets, func(m gamma.Market, _ int) bool {
			// singular absent/null → proposable for standard markets
			// plural "[]" → proposable for negRisk markets
			return m.UmaResolutionStatus == "" && (m.UmaResolutionStatuses == "" || m.UmaResolutionStatuses == "[]")
		})
	}
	if permissionless {
		markets = lo.Filter(markets, func(m gamma.Market, _ int) bool {
			return chain.IsOOV2(m.ResolvedBy)
		})
	}
	if count > 0 && len(markets) > count {
		markets = markets[:count]
	}

	if len(markets) == 0 {
		log.Info("no markets found")
		return
	}

	umaLinks := fetchUmaLinks(markets)

	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderRow(true).
		StyleFunc(func(_, _ int) lipgloss.Style {
			return lipgloss.NewStyle().Padding(0, 1)
		}).
		Headers("VOL", "LIQ", "END", "QUE", "LNK", "UMA")

	t.Width(utils.TerminalWidth())

	for i, m := range markets {
		umaLnk := "-"
		if umaLinks[i] != "" {
			umaLnk = utils.TL("@@@", umaLinks[i])
		}
		t.Row(
			fmt.Sprintf("$%.0f", m.VolumeNum),
			fmt.Sprintf("$%.0f", m.LiquidityNum),
			m.EndDate,
			utils.Truncate(m.Question, 70),
			utils.TL("@@@", fmt.Sprintf("https://polymarket.com/market/%s", strings.TrimSpace(m.Slug))),
			umaLnk,
		)
	}

	fmt.Println(t.Render())
	log.Info("markets summary", "count", len(markets), "volume_min", minvol, "volume_max", maxvol)
}

func fetchUmaLinks(markets []gamma.Market) []string {
	uma := chain.NewUmaClient(rpcURL)
	defer uma.Close()

	links := make([]string, len(markets))
	var wg sync.WaitGroup
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	for i, m := range markets {
		wg.Add(1)
		go func(idx int, m gamma.Market) {
			defer wg.Done()
			link, err := uma.UmaLink(ctx, m.ID, m.ResolvedBy, m.QuestionID)
			if err != nil {
				log.Debug("uma link failed", "market", m.ID, "err", err)
				return
			}
			links[idx] = link
		}(i, m)
	}
	wg.Wait()
	return links
}

func prompt() {
	var _maxvol, _minvol, _resolveWithin string
	var _proposable bool
	form := huh.NewForm(huh.NewGroup(
		huh.NewInput().Title("max-volume ($)").Value(&_maxvol).Placeholder("50000"),
		huh.NewInput().Title("min-volume ($)").Value(&_minvol).Placeholder("0"),
		huh.NewInput().Title("resolve-within (e.g., 24h)").Value(&_resolveWithin).Placeholder(""),
		huh.NewConfirm().Title("proposable only?").Value(&_proposable),
	)).WithTheme(huh.ThemeBase())
	if err := form.Run(); err != nil {
		log.Fatal("prompt failed", "err", err)
	}

	if _maxvol != "" {
		fmt.Sscanf(_maxvol, "%f", &maxvol)
	}
	if _minvol != "" {
		fmt.Sscanf(_minvol, "%f", &minvol)
	}
	if _resolveWithin != "" {
		dur, err := time.ParseDuration(_resolveWithin)
		if err != nil {
			log.Fatal("resolve-within invalid", "err", err)
		}
		resolveWithin = dur
	}
	proposable = _proposable
}
