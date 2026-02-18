package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/salad/polyquery/pq/pkg/chain"
	"github.com/salad/polyquery/pq/pkg/gamma"
	"github.com/salad/polyquery/pq/pkg/utils"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: linker <polymarket-market-or-event-url>")
		os.Exit(1)
	}
	if err := run(os.Args[1]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(arg string) error {
	u, err := url.Parse(arg)
	if err != nil || u.Host == "" {
		return fmt.Errorf("invalid URL: %s", arg)
	}
	gc := gamma.NewClient()
	uma := chain.NewUmaClient("")
	ctx := context.Background()

	var markets []gamma.Market
	switch {
	case strings.Contains(u.Path, "/event/"):
		// /event/{event-slug} or /event/{event-slug}/{market-slug}
		parts := strings.SplitN(strings.TrimPrefix(u.Path, "/event/"), "/", 2)
		event, err := gc.GetEvent(parts[0])
		if err != nil {
			return err
		}
		markets = event.Markets
	case strings.Contains(u.Path, "/market/"):
		m, err := gc.GetMarket(path.Base(u.Path))
		if err != nil {
			return err
		}
		markets = []gamma.Market{*m}
	default:
		return fmt.Errorf("unrecognised URL (expected /market/ or /event/ path): %s", arg)
	}

	tbl := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderRow(true).
		StyleFunc(func(_, _ int) lipgloss.Style { return lipgloss.NewStyle().Padding(0, 1) }).
		Headers("MARKET", "UMA LINK")
	if w := utils.TerminalWidth(); w > 0 {
		tbl.Width(w)
	}

	for _, m := range markets {
		link := "-"
		if l, err := uma.UmaLink(ctx, m.ID, m.ResolvedBy, m.QuestionID); err == nil {
			link = utils.TL(l, l)
		}
		tbl.Row(m.Question, link)
	}
	fmt.Println(tbl.Render())
	return nil
}
