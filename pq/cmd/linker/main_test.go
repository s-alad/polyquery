package main

import (
	"context"
	"testing"

	"github.com/salad/polyquery/pq/pkg/chain"
	"github.com/salad/polyquery/pq/pkg/gamma"
)

// TestFluWeek6Links validates UMA link resolution against the flu hospitalization week 6 event.
// Expected links verified against oracle.uma.xyz.
func TestFluWeek6Links(t *testing.T) {
	gc := gamma.NewClient()
	uma := chain.NewUmaClient("")
	ctx := context.Background()

	event, err := gc.GetEvent("flu-hospitalization-rate-week-6-2026")
	if err != nil {
		t.Fatal("GetEvent:", err)
	}

	// question → expected UMA link
	want := map[string]string{
		"Will the U.S. flu hospitalization rate per 100,000 in Week 6 be less than 50?":      "https://oracle.uma.xyz/propose?project=Polymarket&transactionHash=0xebd750e9bf9297e3fb628115a87c7da0f09d3afb83c23ae8f84991758c83e8c0&eventIndex=141",
		"Will the U.S. flu hospitalization rate per 100,000 in Week 6 be between 60 and 70?": "https://oracle.uma.xyz/propose?project=Polymarket&transactionHash=0x1b6faaf9cd261f974b607475513cf0b040beb2f083c83567d6cf5db86c07eb57&eventIndex=314",
		"Will the U.S. flu hospitalization rate per 100,000 in Week 6 be between 80 and 90?": "https://oracle.uma.xyz/propose?project=Polymarket&transactionHash=0x919ca397d30ab7028da538610c9d3348c7a1a96da9f3e2853ccbe1b20a33fd3a&eventIndex=226",
		"Will the U.S. flu hospitalization rate per 100,000 in Week 6 be between 50 and 60?": "https://oracle.uma.xyz/propose?project=Polymarket&transactionHash=0xd3cb1977b357513bc4ef82d2790143e5ad2869fb138058cbb36ce611928df6b9&eventIndex=228",
		"Will the U.S. flu hospitalization rate per 100,000 in Week 6 be between 70 and 80?": "https://oracle.uma.xyz/propose?project=Polymarket&transactionHash=0xec2190fdc9f536cee8148f085bf7c27d3c2e3cfcbba6a9aa55e5d817d0cc8d91&eventIndex=183",
		"Will the U.S. flu hospitalization rate per 100,000 in Week 6 be 90 or more?":        "https://oracle.uma.xyz/propose?project=Polymarket&transactionHash=0xae72e07699af286ae4140e799b53f71306e0a0d2d482851ad5a9106545f909ad&eventIndex=254",
	}

	if len(event.Markets) != len(want) {
		t.Fatalf("market count: got %d want %d", len(event.Markets), len(want))
	}

	for _, m := range event.Markets {
		expected, ok := want[m.Question]
		if !ok {
			t.Errorf("unexpected market: %q", m.Question)
			continue
		}
		link, err := uma.UmaLink(ctx, m.ID, m.ResolvedBy, m.QuestionID)
		if err != nil {
			t.Errorf("%q: UmaLink error: %v", m.Question, err)
			continue
		}
		if link != expected {
			t.Errorf("%q:\n  got  %s\n  want %s", m.Question, link, expected)
		}
	}
}
