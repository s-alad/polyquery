package main

import (
	"fmt"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/charmbracelet/log"
	"github.com/salad/polyquery/pq/pkg/gamma"
	"github.com/salad/polyquery/pq/pkg/utils"
	"github.com/spf13/cobra"
)

var (
	maxvol      float64
	minvol      float64
	limit       int
	interactive bool
)

var rcmd = &cobra.Command{
	Use:   "scout",
	Short: "scout polymarket",
	Run:   run,
}

func init() {
	rcmd.Flags().Float64Var(&maxvol, "max-volume", 10000, "max volume USD")
	rcmd.Flags().Float64Var(&minvol, "min-volume", 0, "min volume USD")
	rcmd.Flags().IntVar(&limit, "limit", 25, "result count")
	rcmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "prompt for filters")
}

func main() {
	if err := rcmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) {
	if interactive {
		prompt()
	}

	markets, err := gamma.NewClient().GetMarkets(gamma.MarketQuery{
		VolumeMax: maxvol,
		VolumeMin: minvol,
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
		Headers("VOLUME", "LIQUIDITY", "END DATE", "QUESTION")

	for _, m := range markets {
		end := ""
		if len(m.EndDate) >= 10 {
			end = m.EndDate[:10]
		}
		t.Row(
			fmt.Sprintf("$%.0f", m.VolumeNum),
			fmt.Sprintf("$%.0f", m.LiquidityNum),
			end,
			utils.Truncate(m.Question, 70),
		)
	}

	fmt.Println(t.Render())
	log.Info("markets summary", "count", len(markets), "volume_min", minvol, "volume_max", maxvol)
}

func prompt() {
	var maxvolStr, minvolStr, limitStr string
	err := huh.NewForm(huh.NewGroup(
		huh.NewInput().Title("Max volume ($)").Value(&maxvolStr).Placeholder("10000"),
		huh.NewInput().Title("Min volume ($)").Value(&minvolStr).Placeholder("0"),
		huh.NewInput().Title("Result limit").Value(&limitStr).Placeholder("25"),
	)).Run()

	if err != nil {
		log.Fatal("prompt failed", "err", err)
	}
	if maxvolStr != "" {
		fmt.Sscanf(maxvolStr, "%f", &maxvol)
	}
	if minvolStr != "" {
		fmt.Sscanf(minvolStr, "%f", &minvol)
	}
	if limitStr != "" {
		fmt.Sscanf(limitStr, "%d", &limit)
	}
}
