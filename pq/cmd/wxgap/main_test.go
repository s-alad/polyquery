package main

import (
	"context"
	"testing"
	"time"

	"github.com/salad/polyquery/pq/pkg/chain"
	"github.com/salad/polyquery/pq/pkg/gamma"
	"github.com/salad/polyquery/pq/pkg/sources/wx"
)

// TestAnkaraFeb17 validates against the real Ankara Feb 17 2026 market.
// Expected values verified against oracle.uma.xyz proposals and LTAC wunderground observations.
func TestAnkaraFeb17(t *testing.T) {
	event, err := gamma.NewClient().GetEvent("highest-temperature-in-ankara-on-february-17-2026")
	if err != nil {
		t.Fatal("GetEvent:", err)
	}

	endDate, _ := time.Parse(time.RFC3339, event.EndDate)
	obs, err := wx.Fetch("LTAC", "TR", "m", endDate.Format("20060102"))
	if err != nil {
		t.Fatal("wx.Fetch:", err)
	}

	var brackets []bracket
	for _, m := range event.Markets {
		if b, ok := parseBracket(m); ok {
			brackets = append(brackets, b)
		}
	}
	sortBrackets(brackets)

	maxTemp, lastObs := wx.Stats(obs)

	ids := make([]string, len(brackets))
	for i, b := range brackets {
		ids[i] = b.market.ID
	}
	proposals := chain.NewUmaClient("").FetchProposals(context.Background(), ids)

	if maxTemp != 13 {
		t.Errorf("daily max: got %d want 13", maxTemp)
	}
	wantLastObs := time.Date(2026, 2, 17, 20, 50, 0, 0, time.UTC)
	if !lastObs.Equal(wantLastObs) {
		t.Errorf("lastObs: got %v want %v", lastObs, wantLastObs)
	}

	trunc := func(t time.Time) time.Time { return t.Truncate(time.Minute) }
	d16 := func(h, m int) time.Time { return time.Date(2026, 2, 16, h, m, 0, 0, time.UTC) }
	d17 := func(h, m int) time.Time { return time.Date(2026, 2, 17, h, m, 0, 0, time.UTC) }

	type row struct {
		label    string
		outcome  string
		res      time.Time
		proposed time.Time
		proposer string
	}

	// ≤8°, 9°, 10° exceeded by first LTAC obs (11°C at 21:20 UTC Feb 16 = 00:20 TRT Feb 17)
	want := []row{
		{"≤8°", "No (exceeded)", d16(21, 20), d16(21, 25), "0xc9ffd41c"},
		{"9°", "No (exceeded)", d16(21, 20), d16(21, 25), "0x53692dbf"},
		{"10°", "No (exceeded)", d16(21, 20), d16(21, 25), "0xc9ffd41c"},
		{"11°", "No (exceeded)", d17(8, 50), d17(8, 55), "0xc9ffd41c"},
		{"12°", "No (exceeded)", d17(9, 50), d17(9, 55), "0xc9ffd41c"},
		{"13°", "Yes (day end)", d17(20, 50), d17(21, 25), "0xc9ffd41c"},
		{"14°", "No (day end)", d17(20, 50), d17(21, 25), "0x53692dbf"},
		{"15°", "No (day end)", d17(20, 50), d17(21, 25), "0x53692dbf"},
		{"≥16°", "No (day end)", d17(20, 50), d17(21, 25), "0xc9ffd41c"},
	}

	if len(brackets) != len(want) {
		t.Fatalf("bracket count: got %d want %d", len(brackets), len(want))
	}

	for i, b := range brackets {
		w := want[i]
		if b.label() != w.label {
			t.Errorf("[%d] label: got %q want %q", i, b.label(), w.label)
		}
		resolvable, outcome := b.resolvableAt(obs, maxTemp, lastObs)
		if outcome != w.outcome {
			t.Errorf("[%d] %s outcome: got %q want %q", i, w.label, outcome, w.outcome)
		}
		if !trunc(resolvable).Equal(w.res) {
			t.Errorf("[%d] %s resolvable: got %v want %v", i, w.label, trunc(resolvable), w.res)
		}
		p, ok := proposals[i]
		if !ok {
			t.Errorf("[%d] %s: no proposal found", i, w.label)
			continue
		}
		pt := trunc(time.Unix(p.Timestamp, 0).UTC())
		if !pt.Equal(w.proposed) {
			t.Errorf("[%d] %s proposed: got %v want %v", i, w.label, pt, w.proposed)
		}
		if len(p.Proposer) < len(w.proposer) || p.Proposer[:len(w.proposer)] != w.proposer {
			t.Errorf("[%d] %s proposer: got %q want prefix %q", i, w.label, p.Proposer, w.proposer)
		}
	}
}

func sortBrackets(bs []bracket) {
	for i := 1; i < len(bs); i++ {
		for j := i; j > 0 && bs[j].low < bs[j-1].low; j-- {
			bs[j], bs[j-1] = bs[j-1], bs[j]
		}
	}
}
