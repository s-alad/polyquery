package resolvers

import (
	"context"
	"strings"

	"github.com/salad/polyquery/pq/pkg/resolve"
)

var nbaTeams = []string{
	"hawks", "celtics", "nets", "hornets", "bulls", "cavaliers",
	"mavericks", "nuggets", "pistons", "warriors", "rockets", "pacers",
	"clippers", "lakers", "grizzlies", "heat", "bucks", "timberwolves",
	"pelicans", "knicks", "thunder", "magic", "76ers", "suns",
	"trail blazers", "kings", "spurs", "raptors", "jazz", "wizards",
}

type NBA struct{}

func NewNBA() *NBA { return &NBA{} }

func (n *NBA) Name() string { return "nba" }

func (n *NBA) CanResolve(m resolve.Market) bool {
	q := strings.ToLower(m.Question)
	for _, team := range nbaTeams {
		if strings.Contains(q, team) {
			return true
		}
	}
	return strings.Contains(q, "nba") || strings.Contains(q, "basketball")
}

// Stub — will call a sports API when implemented.
func (n *NBA) Resolve(ctx context.Context, m resolve.Market) (resolve.Result, error) {
	return resolve.Result{Outcome: resolve.Unknown, Reasoning: "stub: not yet implemented"}, nil
}
