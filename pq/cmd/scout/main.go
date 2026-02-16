package main

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/log"
	"github.com/salad/polyquery/pq/pkg/gamma"
	"github.com/spf13/cobra"
)

var (
	maxVol      float64
	minVol      float64
	limit       int
	closed      bool
	interactive bool
)

var rootCmd = &cobra.Command{
	Use:   "scout",
	Short: "Discover Polymarket markets",
	Run:   run,
}

func init() {
	rootCmd.Flags().Float64Var(&maxVol, "max-volume", 10000, "max volume USD")
	rootCmd.Flags().Float64Var(&minVol, "min-volume", 0, "min volume USD")
	rootCmd.Flags().IntVar(&limit, "limit", 25, "result count")
	rootCmd.Flags().BoolVar(&closed, "closed", false, "include closed markets")
	rootCmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "prompt for filters")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) {
	if interactive {
		promptFilters()
	}

	markets, err := gamma.NewClient().GetMarkets(gamma.MarketQuery{
		VolumeMax: maxVol,
		VolumeMin: minVol,
		Closed:    &closed,
		Limit:     limit,
		Order:     "volumeNum",
	})
	if err != nil {
		log.Fatal("fetch failed", "err", err)
	}
	if len(markets) == 0 {
		fmt.Println("No markets found.")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "VOLUME\tLIQUIDITY\tEND DATE\tQUESTION\n")
	fmt.Fprintf(w, "------\t---------\t--------\t--------\n")
	for _, m := range markets {
		end := ""
		if len(m.EndDate) >= 10 {
			end = m.EndDate[:10]
		}
		fmt.Fprintf(w, "$%.0f\t$%.0f\t%s\t%s\n", m.VolumeNum, m.LiquidityNum, end, truncate(m.Question, 70))
	}
	w.Flush()
	fmt.Fprintf(os.Stderr, "\n%d markets (volume $%.0f–$%.0f)\n", len(markets), minVol, maxVol)
}

func promptFilters() {
	var maxVolStr, minVolStr, limitStr string
	err := huh.NewForm(huh.NewGroup(
		huh.NewInput().Title("Max volume ($)").Value(&maxVolStr).Placeholder("10000"),
		huh.NewInput().Title("Min volume ($)").Value(&minVolStr).Placeholder("0"),
		huh.NewInput().Title("Result limit").Value(&limitStr).Placeholder("25"),
		huh.NewConfirm().Title("Include closed markets?").Value(&closed),
	)).Run()
	if err != nil {
		log.Fatal("prompt failed", "err", err)
	}
	if maxVolStr != "" {
		fmt.Sscanf(maxVolStr, "%f", &maxVol)
	}
	if minVolStr != "" {
		fmt.Sscanf(minVolStr, "%f", &minVol)
	}
	if limitStr != "" {
		fmt.Sscanf(limitStr, "%d", &limit)
	}
}

func truncate(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}
