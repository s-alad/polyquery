# Weather Temperature Markets

## Overview

Daily "highest temperature" markets are among the most consistent **OOV2 (permissionless) markets** on Polymarket. T20 cricket markets also use OOV2 (v3 adapter) but are frequently resolved by competitor bots before the window opens. They appear daily for multiple cities, each with several temperature brackets. Only one bracket per city resolves "Yes" — the rest are "No."

These are the best path to building the 5-proposal track record needed for MOOV2 whitelist (sports/CDL markets).

## Data Source

Markets resolve based on **Weather Underground historical observations** for specific airport weather stations. The station is embedded in each market's `resolutionSource` field in the Gamma API.

Known city → station mappings:

| City | Station | IANA Timezone | Short |
|------|---------|---------------|-------|
| Ankara | LTAC (Esenboğa) | Europe/Istanbul | TRT |
| Dallas | KDAL or KDFW | America/Chicago | CST |
| Paris | LFPG (CDG) or LFPO (Orly) | Europe/Paris | CET |
| Atlanta | KATL | America/New_York | EST |
| Seoul | RKSI (Incheon) or RKSS (Gimpo) | Asia/Seoul | KST |

Weather Underground API (no auth needed, returns JSON):
```
https://api.weather.com/v1/location/{STATION}:9:{COUNTRY}/observations/historical.json?apiKey=e1f10a1e78da46f5b10a1e78da96f525&units=m&startDate=YYYYMMDD&endDate=YYYYMMDD
```
- `units=m` for °C, `units=e` for °F
- Each observation has `valid_time_gmt` (unix) + `temp`

## Market Structure

- Markets end at **12:00 UTC** (regardless of city timezone)
- Each city has ~5-9 bracket options (e.g. "11°C", "12°C", "13°C", "≥16°C")
- Each bracket is a separate Gamma market with its own UMA request
- Adapter: **v3** (`0x2F5e3684cb1F318ec51b00Edba38d79Ac2c0aA9d`) — OOV2, permissionless
- Bond: **$750 USDC** per proposal

## Proposing Strategy

### Propose Values
- `p1: 0` = No (bracket does NOT match the high)
- `p2: 1e18` = Yes (bracket MATCHES the high)
- `p3: 0.5e18` = Unknown/ambiguous

### "No (exceeded)" brackets
Once the station reads a temperature that exceeds the bracket's upper bound, propose "No" immediately — the daily high is already above it. Example: station reads 12°C → 11°C bracket is definitively "No."

### "Yes" and upper "No" brackets
Only resolvable after the station day ends (last observation ~20:00-21:00 UTC for Turkish stations). Wait for temps to clearly be declining before proposing.

## Competition Analysis (Feb 17-18, 2026)

Two active bots observed:
- `0xc9ffd41c7500d34606606246d8fb6242bfb6b04e`
- `0x53692dbf47fce6fb3d54fded6a4834be981f557c`

**Timing measured via `wxgap` tool:**

| Type | Resolvable | Bot proposes | Gap to beat |
|------|-----------|-------------|-------------|
| "No (exceeded)" | When station first reads above bracket | ~5 min after observation | < 5 min |
| "Yes / No (day end)" | Last station observation of day | ~19 min after last obs | < 19 min |

- `0x53692dbf` handles "exceeded" brackets (~5 min gap)
- `0xc9ffd41c` handles "day end" brackets (~19 min gap)
- Day-end gap was previously estimated at ~35 min; measured at 19 min on Feb 18 2026 Ankara data

## Tools

### `wxgap` — Timing analysis tool

Analyzes a weather event and shows when each bracket became resolvable vs when it was actually proposed:

```bash
cd pq/
go run ./cmd/wxgap/ 'https://polymarket.com/event/highest-temperature-in-ankara-on-february-17-2026'
```

Output columns: `BRACKET | OUTCOME | RESOLVABLE | PROPOSED | GAP | PROPOSER | UMA LINK`
- `RESOLVABLE` = first moment the bracket could be proposed (UTC / EST / local)
- `PROPOSED` = when it was actually proposed on UMA
- `GAP` = time the reward sat unclaimed

How it works:
1. Fetches all sub-markets from Gamma events API
2. Extracts station code from `resolutionSource`
3. Downloads observations from Weather Underground API
4. Determines resolvable time per bracket (exceeded = first obs above bracket; day-end = last obs)
5. Queries Goldsky subgraph for `proposalTimestamp` per market
6. Computes gap and renders table

### `scout` — Market discovery

```bash
# All permissionless (OOV2) proposable markets ending today (includes same-day markets past endDate but still open)
go run ./cmd/scout/ --permissionless --min-volume 0 -i=false

# All proposable markets ended in last 180 days (audit / catch stragglers)
go run ./cmd/scout/ --ended --min-volume 0 -i=false

# Same but show all including already-proposed (to audit competition)
go run ./cmd/scout/ --ended --min-volume 0 -i=false --proposable=false --count 50
```

**Note:** Scout floors `EndDateMin` to start-of-day (not now) so markets whose stated `endDate` has passed but are still `closed=false` remain visible. The authoritative "done" signal is `closed=true`, not `endDate`.

## Finding UMA Links

Each bracket's UMA link is found via the Goldsky subgraph using the Gamma market ID:
1. Fetch market: `curl 'https://gamma-api.polymarket.com/markets?slug=...'`
2. Get market ID
3. Hex-encode `"market_id: {ID}"` and query subgraph `ancillaryData_contains`
4. Build: `oracle.uma.xyz/propose?project=Polymarket&transactionHash={hash}&eventIndex={logIndex}`

Scout and `wxgap` fetch these automatically. You can also use the `linker` tool directly.

### `linker` — UMA link lookup

Given any Polymarket URL, prints a table of UMA propose links:

```bash
# Single market → one link
go run ./cmd/linker/ 'https://polymarket.com/market/highest-temperature-in-ankara-on-february-18-2026-9c'

# Event URL (any path depth) → all brackets
go run ./cmd/linker/ 'https://polymarket.com/event/highest-temperature-in-ankara-on-february-18-2026'
go run ./cmd/linker/ 'https://polymarket.com/event/highest-temperature-in-ankara-on-february-18-2026/highest-temperature-in-ankara-on-february-18-2026-9c'
```
