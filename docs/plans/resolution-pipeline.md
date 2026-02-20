# Resolution Pipeline Architecture

> **Last updated:** 2026-02-19
> **Status:** Bottom-up tooling complete (scout, wxgap, UMA lookup). Top-down scaffold stripped to skeletons. Pipeline/store/resolvers to be built bottom-up as real resolvers are implemented.

## Context

Scout finds markets. The next piece is a system that takes a market type, automatically determines outcomes, presents them for human approval, then proposes on-chain via UMA. This is the revenue-generating core of polyquery.

## Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Module structure | Single Go module in `pq/` | Go code isolated from docs and future non-Go services |
| Storage | SQLite + SQLC | Type-safe generated Go from SQL. ACID for bond tracking. |
| Dashboard | Svelte (separate frontend) | Go serves JSON API, Svelte handles UI |
| Chain interaction | go-ethereum + abigen | Official Go Ethereum client. `abigen` generates typed contract bindings from ABI |
| UMA language | Go (not TypeScript) | go-ethereum is the Ethereum reference implementation |
| Docker | Not yet | Local-first tool. One binary + SQLite file |
| CLI framework | Cobra + Charm (huh, log) | Cobra for subcommands/flags, huh for interactive prompts |
| HTTP framework | Fiber | Express-inspired, fast, less boilerplate than net/http |
| Dev reload | Air | Live reload during development |
| Build strategy | Bottom-up | Early top-down scaffold stripped to skeletons (2026-02-18). Add pipeline/store/resolvers only as real implementations are built. |

## Current Structure

```text
polyquery/
├── pq/
│   ├── cmd/
│   │   ├── scout/         # ✅ fully working market discovery CLI
│   │   ├── resolver/      # 🔲 skeleton — health check only
│   │   ├── wxgap/         # ✅ fully working weather gap analyzer + integration test
│   │   ├── wxwatch/       # ✅ polls WU station, fires outcomes + UMA links at day end
│   │   └── linker/        # ✅ given any Polymarket URL, prints UMA propose links
│   ├── pkg/
│   │   ├── gamma/         # ✅ Gamma API client
│   │   ├── chain/         # ✅ UMA link lookup (subgraph + RPC), proposal fetching
│   │   ├── kv/            # ✅ Badger KV cache
│   │   ├── sources/wx/    # ✅ Weather Underground fetch/stats/interval/TypicalDayEnd
│   │   ├── store/         # 🔲 skeleton — schema/queries placeholders
│   │   ├── api/           # 🔲 skeleton — health route only
│   │   └── utils/         # ✅ terminal helpers
│   └── resolvers/         # 🔲 empty — concrete impls go here
└── dash/                  # 🔲 TODO Svelte frontend
```

## Core Interface (to be re-added when first real resolver is built)

```go
type Resolver interface {
    Name() string
    CanResolve(m Market) bool
    Resolve(ctx context.Context, m Market) (Result, error)
}

type Result struct {
    Outcome    Outcome   // Yes | No | Unknown
    Confidence float64   // 0.0–1.0
    Reasoning  string
    Sources    []string
}
```

Adding a new market type = implement interface + register in resolver cmd.

## Storage: SQLite + SQLC

SQLC generates type-safe Go from raw SQL. Schema and queries are placeholders now.

Planned tables (add when pipeline is built):

- **markets** — status lifecycle: `watching → resolved → approved → proposed → settled | disputed`
- **approvals** — pending human approvals (the queue)
- **ledger** — bond/reward accounting ($750 bond_locked, bond_returned, reward, bond_lost)

## Pipeline (planned — not yet implemented)

```text
[Poller]  →  [Resolver]  →  [Dashboard/Approval]  →  [Proposer]  →  [Settler]
 5min         1min            human clicks             30s            5min
```

1. **Poller** — queries Gamma for closed markets with no UMA resolution, upserts as `watching`
2. **Resolver** — scans `watching` markets, runs matched resolver, creates approval if confidence > threshold
3. **Dashboard** — user reviews pending approvals, clicks Approve/Reject
4. **Proposer** — takes approved markets, calls `proposePrice()` on OOV2, records tx + bond
5. **Settler** — after 2hr liveness, checks for disputes, calls `settle()` if clean, records reward

## API Routes (planned)

| Route | Method | Purpose |
|-------|--------|---------|
| `/api/health` | GET | Health check (implemented) |
| `/api/queue` | GET | Pending approvals |
| `/api/queue/{id}/approve` | POST | Approve a resolution |
| `/api/queue/{id}/reject` | POST | Reject a resolution |
| `/api/markets` | GET | All tracked markets by status |
| `/api/ledger` | GET | Bond/reward P&L |

## UMA Chain Client (go-ethereum)

```bash
# generate typed Go bindings from OOV2 ABI
abigen --abi OOV2.abi --pkg oov2 --out pkg/chain/oov2.go
```

Four operations needed:

| Function | What it does |
|----------|-------------|
| `ApproveUSDC()` | One-time max approval for OOV2 to spend USDC |
| `ProposePrice()` | Post $750 bond + proposed outcome |
| `CheckDisputed()` | Monitor liveness period for disputes |
| `Settle()` | Claim bond + reward after liveness |

## Build Order

1. ~~scaffold + config~~ ✓ (then stripped to skeleton 2026-02-18)
2. ~~`pkg/resolve/` — interface, types, registry~~ ✓ (stripped 2026-02-18, will re-add with first real resolver)
3. ~~`pkg/store/` — SQLC schema + generated code~~ ✓ (stripped to placeholder, will rebuild with real schema)
4. ~~`pkg/api/` — JSON API handlers~~ ✓ (stripped to health-only, will expand with real routes)
5. ~~`pkg/pipeline/` — poller + resolver goroutines~~ ✓ (stripped, will rebuild with real pipeline)
6. ~~`resolvers/nba.go` — first resolver stub~~ ✓ (deleted stub, real resolver comes next)
7. ~~End-to-end dry run~~ ✓
8. `dash/` — Svelte frontend
9. ~~`pkg/chain/` — UmaLink via subgraph + RPC fallback~~ ✅ complete
10. Wire chain into pipeline — ProposePrice via abigen
11. First real resolver (weather or CDL)

## Changelog

| Date | Change |
|------|--------|
| 2026-02-15 | Initial plan. Single Go module, Svelte dashboard, go-ethereum for UMA, SQLC for storage |
| 2026-02-15 | Built steps 1-7. Scout, resolver scaffold, pipeline, NBA stub, API, store |
| 2026-02-16 | Built `pkg/chain/`: dual-path UMA link lookup (subgraph for OOV2, RPC fallback for MOOV2) |
| 2026-02-18 | Deep work on scout + wxgap. Top-down boilerplate stripped to skeletons. pipeline, resolve, nba stub deleted. store/api/resolver cmd reduced to minimal placeholders. Bottom-up strategy going forward. |
| 2026-02-19 | Built `wxwatch` (station poller, fires at `dayStart + TypicalDayEnd`). Built `linker` (UMA link table from any Polymarket URL). Extracted shared `pkg/sources/wx` (Fetch, Stats, Interval, TypicalDayEnd). Scout: fixed EndDateMin floor, added pagination, `--limit` flag (stops paginating early when set), semaphore-capped UMA link fetching (20 workers). HTTP client timeout bumped to 60s. |
