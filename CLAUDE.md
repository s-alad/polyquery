# CLAUDE.md

This file provides guidance to Claude Code when working with this repository.

## Code Style (MANDATORY)

You MUST read and follow `docs/style.md` before writing any code. It contains binding coding style rules for this entire repo. Always apply these rules — no exceptions.

You must always remain on the less bloated and succinct side when writing code or documentation as long as it does not impact functionality or robustness.

You must not make up anything, your code and docs must always be based in truth or real documentation. Make it a habbit to provide your supporting sources and evidence (e.g. documentation <https://docs.polymarket.com/llms.txt>)

You must not overly bloat, however you need to ensure you have everything written to pick up the project in the future if your context is reset, this is a fine balance that you must master.

## Project Overview

**Polyquery** is an automated UMA oracle proposer bot for resolving Polymarket prediction markets. The goal is to programmatically find one-off/isolated markets ready for resolution, propose outcomes via UMA's Optimistic Oracle, and earn the $2-$5 proposer rewards.

### Core Mission

Build an automated system that:

1. **Discovers** markets ready for resolution via Polymarket's API
2. **Evaluates** market outcomes (potentially with AI assistance for ambiguous cases)
3. **Proposes** resolutions via UMA's Optimistic Oracle
4. **Tracks** all proposals, bonds, rewards, and outcomes
5. **Manages** the $750 bond requirement efficiently

### Target: One-Off Markets

Focus on "isolated" or "one-off" markets that are:

- Clear-cut outcomes (sports scores, crypto prices, verifiable events)
- Less competitive (not the big political markets everyone watches)
- Quick to verify (NBA games, specific dated events)

## Docs Folder

`docs/` is a persistent knowledge base. Read from it to regain context. Write to it to preserve knowledge for future sessions.

```text
docs/
├── style.md                  # Coding style rules (mandatory, always follow)
├── poly/
│   ├── polymarket-apis.md    # Complete reference of all 7 Polymarket APIs
│   └── polymarket-llms.txt   # Full doc URL index (append .md for raw content)
├── uma/
│   ├── uma-resolution.md     # UMA oracle process, contracts, economics
│   └── uma-link-lookup.md    # How we find UMA propose URLs (subgraph + RPC fallback)
├── weather/
│   └── weather-markets.md    # Weather temp markets: data sources, strategy, competition, wxgap tool
└── plans/
    └── resolution-pipeline.md  # Architecture plan for the resolution pipeline
```

When you learn something new about an API, contract, or workflow, update the relevant doc. When starting fresh, read the docs to rebuild context before writing code.

## Code Structure

All Go code lives in `pq/`. Docs stay at repo root. Other languages (Rust, Svelte, etc.) get their own top-level folders.

```text
polyquery/
├── CLAUDE.md
├── docs/                      # project-wide knowledge base
├── pq/                        # Go monorepo
│   ├── go.mod                 # github.com/salad/polyquery/pq
│   ├── sqlc.yaml
│   ├── cmd/
│   │   ├── scout/main.go      # market discovery CLI
│   │   ├── resolver/main.go   # pipeline + API server
│   │   └── wxgap/main.go      # weather market timing analysis
│   ├── pkg/
│   │   ├── gamma/             # Gamma API client
│   │   ├── resolve/           # Resolver interface + registry
│   │   ├── store/             # SQLite via SQLC
│   │   ├── api/               # JSON API for dashboard
│   │   └── pipeline/          # Poller + resolver goroutines
│   └── resolvers/             # Concrete resolver implementations
│       └── nba.go
├── dash/                      # Svelte frontend (TODO)
└── .nixdev/                   # devenv config
```

Build commands (from `pq/`):
- `go build ./cmd/scout/` — market scanner
- `go build ./cmd/resolver/` — resolution pipeline
- `go build ./cmd/wxgap/` — weather timing analysis
- `sqlc generate` — regenerate store from SQL

Run commands (from `pq/`):
- `go run ./cmd/scout/` — scan markets; key flags:
  - `--permissionless` — OOV2 only (no whitelist needed)
  - `--ended` — markets past end date (floors at 180 days ago)
  - `--resolve-within 24h` — markets ending within duration (mutually exclusive with --ended)
  - `--proposable` (default true) — only show umaResolutionStatus==""
  - `--min-volume`, `--max-volume`, `--limit` — volume/count filters
  - `--rpc <url>` — Polygon RPC for MOOV2/sports UMA link lookup
  - `-i=false` — skip interactive prompt
- `go run ./cmd/scout/ -i` — interactive mode (prompts for filters)
- `go run ./cmd/resolver/` — start pipeline + API server
- `go run ./cmd/resolver/ --config config.json` — with custom config
- `go run ./cmd/wxgap/ <polymarket-event-url>` — analyze weather market timing gaps
- `go test ./cmd/wxgap/` — run integration test against real Ankara Feb 17 data
