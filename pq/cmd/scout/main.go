package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/charmbracelet/log"
	"github.com/salad/polyquery/pq/pkg/gamma"
	"github.com/salad/polyquery/pq/pkg/utils"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	maxvol      float64
	minvol      float64
	limit       int
	interactive bool
)

var flags = []string{"max-volume", "min-volume", "limit"}

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
	markets, err := gamma.NewClient().GetMarkets(gamma.MarketQuery{
		VolumeMax: maxvol,
		VolumeMin: minvol,
		Closed:    &closed,
		Limit:     limit,
		Order:     "volumeNum",
	})
	if err != nil {
		log.Fatal("fetch failed", "err", err)
	}
	if len(markets) == 0 {
		log.Info("no markets found")
		return
	}

	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderRow(true).
		StyleFunc(func(_, _ int) lipgloss.Style {
			return lipgloss.NewStyle().Padding(0, 1)
		}).
		Headers("VOL", "LIQ", "END", "QUE", "LNK")

	t.Width(terminalWidth())

	for _, m := range markets {
		t.Row(
			fmt.Sprintf("$%.0f", m.VolumeNum),
			fmt.Sprintf("$%.0f", m.LiquidityNum),
			m.EndDate,
			utils.Truncate(m.Question, 70),
			utils.TerminalLink("@@@", fmt.Sprintf("https://polymarket.com/market/%s", strings.TrimSpace(m.Slug))),
		)
	}

	fmt.Println(t.Render())
	log.Info("markets summary", "count", len(markets), "volume_min", minvol, "volume_max", maxvol)
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
	var _maxvol, _minvol, _limit string
	form := huh.NewForm(huh.NewGroup(
		huh.NewInput().Title("max-volume ($)").Value(&_maxvol).Placeholder("50000"),
		huh.NewInput().Title("min-volume ($)").Value(&_minvol).Placeholder("0"),
		huh.NewInput().Title("result-limit").Value(&_limit).Placeholder("10"),
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
}
