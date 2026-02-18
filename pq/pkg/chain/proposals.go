package chain

import (
	"context"
	"strconv"
)

// FetchProposals queries the subgraph for proposal data for each market ID.
// Returns index → Proposal; skips markets with no proposal yet.
func (u *UmaClient) FetchProposals(ctx context.Context, marketIDs []string) map[int]Proposal {
	results := make(map[int]Proposal)
	for i, id := range marketIDs {
		q := marketQuery(`{ optimisticPriceRequests(where:{ancillaryData_contains:"%s"} first:1) { proposalTimestamp proposer } }`, id)
		var out struct {
			Data struct {
				Requests []struct {
					ProposalTimestamp string `json:"proposalTimestamp"`
					Proposer          string `json:"proposer"`
				} `json:"optimisticPriceRequests"`
			} `json:"data"`
		}
		if err := u.subgraphQuery(ctx, q, &out); err != nil || len(out.Data.Requests) == 0 {
			continue
		}
		r := out.Data.Requests[0]
		if ts, _ := strconv.ParseInt(r.ProposalTimestamp, 10, 64); ts != 0 {
			results[i] = Proposal{ts, r.Proposer}
		}
	}
	return results
}
