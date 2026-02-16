package pipeline

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/charmbracelet/log"
	"github.com/salad/polyquery/pq/pkg/gamma"
	"github.com/salad/polyquery/pq/pkg/resolve"
	"github.com/salad/polyquery/pq/pkg/store"
)

type Config struct {
	VolumeMax  float64
	VolumeMin  float64
	PollEvery  time.Duration
	Confidence float64
}

type Pipeline struct {
	gamma    *gamma.Client
	store    *store.Queries
	registry *resolve.Registry
	cfg      Config
}

func New(g *gamma.Client, s *store.Queries, r *resolve.Registry, cfg Config) *Pipeline {
	return &Pipeline{gamma: g, store: s, registry: r, cfg: cfg}
}

func (p *Pipeline) Run(ctx context.Context) {
	go p.pollLoop(ctx)
	go p.resolveLoop(ctx)
	<-ctx.Done()
}

func (p *Pipeline) pollLoop(ctx context.Context) {
	p.poll(ctx)
	ticker := time.NewTicker(p.cfg.PollEvery)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.poll(ctx)
		}
	}
}

func (p *Pipeline) poll(ctx context.Context) {
	closed := true
	markets, err := p.gamma.GetMarkets(gamma.MarketQuery{
		VolumeMax: p.cfg.VolumeMax,
		VolumeMin: p.cfg.VolumeMin,
		Closed:    &closed,
		Limit:     100,
		Order:     "endDate",
	})
	if err != nil {
		log.Error("poll", "err", err)
		return
	}
	for _, m := range markets {
		p.store.UpsertMarket(ctx, store.UpsertMarketParams{
			ID:          m.ID,
			Question:    m.Question,
			ConditionID: m.ConditionID,
			EndDate:     nullStr(m.EndDate),
			ResolutionSrc: nullStr(m.ResolutionSource),
		})
	}
	log.Info("poll", "upserted", len(markets))
}

func (p *Pipeline) resolveLoop(ctx context.Context) {
	time.Sleep(5 * time.Second)
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.resolveWatching(ctx)
		}
	}
}

func (p *Pipeline) resolveWatching(ctx context.Context) {
	markets, err := p.store.GetMarketsByStatus(ctx, "watching")
	if err != nil {
		log.Error("resolve", "err", err)
		return
	}
	for _, m := range markets {
		rm := resolve.Market{
			ID: m.ID, Question: m.Question, ConditionID: m.ConditionID,
			EndDate: m.EndDate.String, ResolutionSource: m.ResolutionSrc.String,
		}
		resolver, ok := p.registry.Match(rm)
		if !ok {
			continue
		}
		result, err := resolver.Resolve(ctx, rm)
		if err != nil {
			log.Error("resolve", "market", m.ID, "err", err)
			continue
		}
		if result.Outcome == resolve.Unknown || result.Confidence < p.cfg.Confidence {
			continue
		}

		srcJSON, _ := json.Marshal(result.Sources)
		p.store.UpdateMarketResolution(ctx, store.UpdateMarketResolutionParams{
			Status:       "resolved",
			ResolverName: sql.NullString{String: resolver.Name(), Valid: true},
			Outcome:      sql.NullInt64{Int64: int64(result.Outcome), Valid: true},
			Confidence:   sql.NullFloat64{Float64: result.Confidence, Valid: true},
			Reasoning:    sql.NullString{String: result.Reasoning, Valid: true},
			Sources:      sql.NullString{String: string(srcJSON), Valid: true},
			ID:           m.ID,
		})
		p.store.CreateApproval(ctx, store.CreateApprovalParams{
			MarketID:   m.ID,
			Outcome:    int64(result.Outcome),
			Confidence: result.Confidence,
			Reasoning:  sql.NullString{String: result.Reasoning, Valid: true},
			Sources:    sql.NullString{String: string(srcJSON), Valid: true},
		})
		log.Info("resolved", "market", m.ID, "outcome", result.Outcome,
			"confidence", result.Confidence, "resolver", resolver.Name())
	}
}

func nullStr(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}
