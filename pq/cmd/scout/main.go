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
	"golang.org/x/term"
)

var (
	maxvol        float64
	minvol        float64
	limit         int
	interactive   bool
	resolveWithin time.Duration
	proposable    bool
	rpcURL        string
	openOnly      bool
	ended         bool
)

var flags = []string{"max-volume", "min-volume", "limit", "resolve-within", "proposable"}

var rcmd = &cobra.Command{
	Use:   "scout",
	Short: "scout polymarket",
	Run:   run,
}

func init() {
	rcmd.Flags().Float64Var(&maxvol, "max-volume", 10000, "max volume USD")
	rcmd.Flags().Float64Var(&minvol, "min-volume", 0, "min volume USD")
	rcmd.Flags().IntVar(&limit, "limit", 25, "result count")
	rcmd.Flags().BoolVarP(&interactive, "interactive", "i", true, "prompt for filters")
	rcmd.Flags().DurationVar(&resolveWithin, "resolve-within", 0, "only markets ending within this duration (e.g., 24h)")
	rcmd.Flags().BoolVar(&proposable, "proposable", true, "only show markets with no UMA proposal yet")
	rcmd.Flags().StringVar(&rpcURL, "rpc", "", "Polygon RPC URL (needed for sports/MOOV2 UMA links)")
	rcmd.Flags().BoolVar(&openOnly, "open", false, "only show OOV2 markets (no whitelist needed)")
	rcmd.Flags().BoolVar(&ended, "ended", false, "only show markets past their end date")
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

	closed := false
	now := time.Now().UTC()
	var endDateMin, endDateMax string
	if ended {
		endDateMax = now.Format(time.RFC3339)
	} else if resolveWithin > 0 {
		endDateMin = now.Format(time.RFC3339)
		endDateMax = now.Add(resolveWithin).Format(time.RFC3339)
	}
	markets, err := gamma.NewClient().GetMarkets(gamma.MarketQuery{
		VolumeMax:  maxvol,
		VolumeMin:  minvol,
		Closed:     &closed,
		EndDateMin: endDateMin,
		EndDateMax: endDateMax,
		Limit:      limit,
		Order:      "endDate",
		Ascending:  true,
	})
	if err != nil {
		log.Fatal("fetch failed", "err", err)
	}
	if proposable {
		markets = lo.Filter(markets, func(m gamma.Market, _ int) bool {
			return m.UmaResolutionStatus == ""
		})
	}
	if openOnly {
		markets = lo.Filter(markets, func(m gamma.Market, _ int) bool {
			return chain.IsOOV2(m.ResolvedBy)
		})
	}

	if len(markets) == 0 {
		log.Info("no markets found")
		return
	}

	// Fetch UMA links concurrently
	umaLinks := fetchUmaLinks(markets)

	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderRow(true).
		StyleFunc(func(_, _ int) lipgloss.Style {
			return lipgloss.NewStyle().Padding(0, 1)
		}).
		Headers("VOL", "LIQ", "END", "QUE", "LNK", "UMA")

	t.Width(terminalWidth())

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
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	for i, m := range markets {
		if m.ResolvedBy == "" {
			continue
		}
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

func terminalWidth() int {
	fd := int(os.Stdout.Fd())
	width, _, err := term.GetSize(fd)
	if err != nil || !term.IsTerminal(fd) || width <= 20 {
		return 0
	}
	return width
}

func prompt() {
	var _maxvol, _minvol, _limit, _resolveWithin string
	var _proposable bool
	form := huh.NewForm(huh.NewGroup(
		huh.NewInput().Title("max-volume ($)").Value(&_maxvol).Placeholder("50000"),
		huh.NewInput().Title("min-volume ($)").Value(&_minvol).Placeholder("0"),
		huh.NewInput().Title("result-limit").Value(&_limit).Placeholder("10"),
		huh.NewInput().Title("resolve-within (e.g., 24h)").Value(&_resolveWithin).Placeholder(""),
		huh.NewConfirm().Title("proposable only?").Value(&_proposable),
	)).WithTheme(huh.ThemeBase())
	err := form.Run()

	if err != nil {
		log.Fatal("prompt failed", "err", err)
	}

	if _maxvol != "" {
		fmt.Sscanf(_maxvol, "%f", &maxvol)
	}
	if _minvol != "" {
		fmt.Sscanf(_minvol, "%f", &minvol)
	}
	if _limit != "" {
		fmt.Sscanf(_limit, "%d", &limit)
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
