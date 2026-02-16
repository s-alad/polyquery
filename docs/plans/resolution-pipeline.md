# Resolution Pipeline Architecture

> **Last updated:** 2026-02-15
> **Status:** In progress — scaffold + pipeline built, resolvers are stubs

## Context

Scout finds markets. Now we need the next piece: a system that takes a market type (e.g. NBA), automatically determines outcomes, presents them for human approval, then proposes on-chain via UMA. This is the revenue-generating core of polyquery.

## Decisions

Decisions made during planning. Update this as things change.

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Module structure | Single Go module in `pq/` | Go code isolated from docs and future non-Go services. Scout is `cmd/scout`, not standalone. |
| Storage | SQLite + SQLC | Type-safe generated Go from SQL. ACID for bond tracking. |
| Dashboard | Svelte (separate frontend) | Go serves JSON API, Svelte handles UI. Lightweight, compiles to vanilla JS. |
| Chain interaction | go-ethereum + abigen | Official Go Ethereum client. `abigen` generates typed contract bindings from ABI. No hand-rolling. |
| UMA language | Go (not TypeScript) | go-ethereum is the Ethereum reference implementation. `abigen` gives typed `ProposePrice()` etc. No need for ethers.js. |
| Docker | Not yet | Local-first tool. One binary + SQLite file. Dockerize when deploying remotely. |
| CLI framework | Cobra + Charm (huh, log) | Cobra for subcommands/flags, huh for interactive prompts, charm/log for logging. |
| HTTP framework | Fiber | Express-inspired, fast, less boilerplate than net/http. |
| Dev reload | Air | Live reload during development. File-watch → rebuild → restart. |

## Structure

All Go code in `pq/`. Docs and non-Go services at repo root.

```text
polyquery/
├── CLAUDE.md
├── docs/                      # project-wide knowledge base
├── pq/                        # Go monorepo (github.com/salad/polyquery/pq)
│   ├── go.mod
│   ├── sqlc.yaml
│   ├── cmd/
│   │   ├── scout/main.go      # market discovery CLI
│   │   └── resolver/main.go   # pipeline + API server
│   ├── pkg/
│   │   ├── gamma/             # Gamma API client (shared by scout + resolver)
│   │   ├── resolve/           # Resolver interface + registry
│   │   ├── store/             # SQLite via SQLC (schema.sql, queries.sql, generated code)
│   │   ├── chain/             # go-ethereum: UMA link lookup, future abigen bindings
│   │   ├── api/               # JSON API for dashboard
│   │   └── pipeline/          # Poller + resolver goroutines
│   └── resolvers/             # Concrete resolver implementations
│       └── nba.go
├── dash/                      # Svelte frontend (TODO)
└── .nixdev/
```

## Core Interface

```go
type Resolver interface {
    Name() string
    CanResolve(m Market) bool
    Resolve(ctx context.Context, m Market) (Result, error)
}

type Result struct {
    Outcome    Outcome   // Yes | No | Unknown
    Confidence float64   // 0.0–1.0
    Reasoning  string    // shown on dashboard
    Sources    []string  // evidence URLs
}
```

Adding a new market type = implement interface + one line in main.go.

**Routing:** Each resolver's `CanResolve()` inspects the market question directly. Registry tries resolvers in order, first match wins.

## Storage: SQLite + SQLC

SQLC generates type-safe Go from raw SQL. We write `.sql` files, SQLC produces Go code. No ORM, no runtime reflection.

Three tables:

- **markets** — status lifecycle: `watching → resolved → approved → proposed → settled | disputed`
- **approvals** — pending human approvals (the queue)
- **ledger** — bond/reward accounting ($750 bond_locked, bond_returned, reward, bond_lost)

## Pipeline (single process, goroutines)

```text
[Poller]  →  [Resolver]  →  [Dashboard/Approval]  →  [Proposer]  →  [Settler]
 5min         1min            human clicks             30s            5min
```

1. **Poller** — queries Gamma for ended/closed markets with no UMA resolution, upserts as `watching`
2. **Resolver** — scans `watching` markets, runs matched resolver, creates approval if confidence > threshold
3. **Dashboard** — user reviews pending approvals, clicks Approve/Reject
4. **Proposer** — takes approved markets, calls `proposePrice()` on OOV2, records tx + bond
5. **Settler** — after 2hr liveness, checks for disputes, calls `settle()` if clean, records reward

## Dashboard: Go API + Svelte

Go serves a JSON API. Svelte frontend consumes it.

**API routes (Go):**

| Route | Method | Purpose |
|-------|--------|---------|
| `/api/queue` | GET | Pending approvals |
| `/api/queue/{id}/approve` | POST | Approve a resolution |
| `/api/queue/{id}/reject` | POST | Reject a resolution |
| `/api/markets` | GET | All tracked markets by status |
| `/api/ledger` | GET | Bond/reward P&L |

**Svelte pages:**
- `/queue` — approval cards with outcome, confidence, reasoning, sources, Approve/Reject buttons
- `/markets` — tracked markets table
- `/ledger` — bond/reward history

## UMA Chain Client (go-ethereum)

```bash
# generate typed Go bindings from OOV2 ABI
abigen --abi OOV2.abi --pkg oov2 --out pkg/chain/oov2.go
```

```go
// typed contract calls, no hand-rolled ABI encoding
tx, err := oracle.ProposePrice(auth, requester, identifier, timestamp, ancillaryData, proposedPrice)
```

Four operations:

| Function | What it does |
|----------|-------------|
| `ApproveUSDC()` | One-time max approval for OOV2 to spend USDC |
| `ProposePrice()` | Post $750 bond + proposed outcome |
| `CheckDisputed()` | Monitor liveness period for disputes |
| `Settle()` | Claim bond + reward after liveness |

Source: [go-ethereum abigen docs](https://geth.ethereum.org/docs/developers/dapp-developer/native)

## Dependencies

| Dep | Why |
|-----|-----|
| `github.com/ethereum/go-ethereum` | ABI bindings via abigen, ethclient, tx signing |
| `modernc.org/sqlite` | Pure-Go SQLite driver (no CGO) |
| SQLC (build tool) | Generates Go from SQL at build time |
| `github.com/gofiber/fiber/v3` | HTTP framework for API server |
| `github.com/spf13/cobra` | CLI framework for scout + resolver commands |
| `github.com/charmbracelet/huh` | Interactive terminal forms (scout prompts) |
| `github.com/charmbracelet/log` | Structured terminal logging |
| `github.com/google/go-querystring` | Struct-to-URL query params |
| `github.com/air-verse/air` | Dev tool: live reload on file changes |

## Config

Resolver reads optional `config.json` (pass `--config path`). All fields have defaults:

```json
{
  "addr": ":8745",
  "db_path": "resolver.db",
  "confidence": 0.95,
  "volume_max": 10000,
  "volume_min": 0,
  "poll_interval": "5m"
}
```

Chain-related config (`rpc_url`, `private_key_path`) will be added in step 9.

## Build Order

1. ~~`pq/go.mod` + `cmd/resolver/main.go` scaffold with config~~ ✓
2. ~~`pkg/resolve/` — interface, types, registry~~ ✓
3. ~~`sql/` schema + queries, SQLC, `pkg/store/`~~ ✓
4. ~~`pkg/api/` — JSON API handlers~~ ✓
5. ~~`pkg/pipeline/` — poller + resolver goroutines~~ ✓
6. ~~`resolvers/nba.go` — first concrete resolver (stub)~~ ✓
7. ~~End-to-end dry run: poll → resolve → API → approve~~ ✓
8. `dash/` — Svelte frontend consuming the API
9. ~~`pkg/chain/` — UmaLink lookup via subgraph + RPC fallback~~ ✓ (abigen bindings for ProposePrice/Settle still TODO)
10. Wire chain into pipeline, production dry run with `dry_run=true`
11. Implement real NBA resolver (call sports API)

## Changelog

Track plan revisions here so stale decisions don't poison future sessions.

| Date | Change |
|------|--------|
| 2026-02-15 | Initial plan: separate Go modules, html/template dashboard, considered TypeScript for UMA |
| 2026-02-15 | Revised: single Go module, Svelte dashboard, go-ethereum for UMA, SQLC for storage |
| 2026-02-15 | Built steps 1-7. Restructured: Go code moved to `pq/`, scout merged into `cmd/scout`, docs stay at repo root |
| 2026-02-15 | Moved SQLC source (schema.sql, queries.sql) into `pkg/store/`, eliminated `sql/` dir. Single source of truth. |
| 2026-02-15 | Adopted Cobra (CLI), Charm libs (huh, log), Fiber (HTTP), Air (dev reload). Trimmed gamma types to used fields only, go-querystring for query params. |
| 2026-02-15 | Bloat pass: removed Tags/Description from resolve.Market, removed extractTagsFromQuestion, removed unused Registry.All(), embedded Config in Pipeline, removed premature config fields (rpc_url, dry_run, private_key_path), simplified scout to single root command. |
| 2026-02-15 | Built pkg/chain/uma.go: UmaLink lookup via QuestionInitialized event logs on Polygon. Chunks 10K blocks to respect free-tier RPC limits. Searches known adapter contracts (v2, v3, sports). Uses resolvedBy field from Gamma to target exact adapter. |
| 2026-02-16 | Rewrote pkg/chain/uma.go: dual-path lookup. Subgraph (instant) for v2/v3 adapters via UMA Goldsky subgraph, RPC log scan fallback for sports/MOOV2 adapter. Discovered Gamma `id` == ancillary `market_id` for cross-referencing. Added `--proposable` flag to scout (default true). See `docs/uma/uma-link-lookup.md`. |
