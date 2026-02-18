# Weather Temperature Markets

## Overview

Daily "highest temperature" markets are currently the **only reliable OOV2 (permissionless) markets** on Polymarket. They appear daily for multiple cities, each with several temperature brackets. Only one bracket per city resolves "Yes" — the rest are "No."

These are the best path to building the 5-proposal track record needed for MOOV2 whitelist (sports/CDL markets).

## Data Source

Markets resolve based on **Weather Underground historical observations** for specific airport weather stations.

Example: Ankara uses **LTAC** (Esenboğa Airport)
- URL: `https://www.wunderground.com/history/daily/tr/çubuk/LTAC`
- Observations are in **local time** — convert to UTC for UMA timing

Known city → station mappings (discover more by checking market resolution sources):

| City | Station | Timezone | UTC Offset |
|------|---------|----------|------------|
| Ankara | LTAC (Esenboğa) | Turkey Time | UTC+3 |
| Dallas | KDFW or KDAL | Central | UTC-6 |
| Paris | LFPG (CDG) or LFPO (Orly) | CET | UTC+1 |
| Atlanta | KATL | Eastern | UTC-5 |
| Seoul | RKSI (Incheon) or RKSS (Gimpo) | KST | UTC+9 |

## Market Structure

- Markets end at **12:00 UTC** (regardless of city timezone)
- Each city has ~5-7 bracket options (e.g. "11°C", "12°C", "13°C", etc.)
- Each bracket is a separate Gamma market with its own UMA request
- Adapter: **v3** (`0x2F5e3684cb1F318ec51b00Edba38d79Ac2c0aA9d`) — OOV2, permissionless
- Bond: **$750 USDC** per proposal

## Proposing Strategy

### Propose Values
- `p1: 0` = No (bracket does NOT match the high)
- `p2: 1e18` = Yes (bracket MATCHES the high)
- `p3: 0.5e18` = Unknown/ambiguous

### "No" Proposals (Low Risk)
Once the temperature **exceeds** a bracket, you can propose "No" immediately — the high will be above that value. Example: if the station reads 12°C at 10:50 AM, the 11°C bracket is definitively "No."

### "Yes" Proposals (Wait for Day End)
Only propose "Yes" on the bracket matching the actual high **after the day fully ends** at the station. Temperatures can still change.

## Competition Analysis (Feb 17, 2026)

Observed one active bot: `0xc9ffd41c7500d34606606246d8fb6242bfb6b04e`

- Polls weather stations on a **~65 minute interval**
- Proposes "No" on brackets that have been clearly exceeded
- Does NOT propose on uncertain brackets (waits for day to end)
- Pattern: exactly 65 min delay from station reading to proposal, 1 hour between successive proposals

### Competitive Edge
- Poll station every 5-10 min → ~55 min head start on "No" proposals
- "Yes" proposals are less competitive — bot waits for certainty
- Multiple cities × multiple brackets = many opportunities per day

## Finding Markets via Scout

```bash
# All OOV2 proposable markets ending today
go run ./cmd/scout/ --open --resolve-within 24h -i=false

# Weather markets specifically (most OOV2 markets are weather)
go run ./cmd/scout/ --open --resolve-within 24h --min-volume 0 -i=false
```

## Finding UMA Links

Each bracket's UMA link is found via the Goldsky subgraph using the Gamma market ID:
1. Search Gamma: `curl 'https://gamma-api.polymarket.com/markets?slug=highest-temperature-in-ankara-on-february-17-2026-13c'`
2. Get market ID from response
3. Subgraph query with `ancillaryData_contains` hex-encoded `"market_id: {ID}"`
4. Build URL: `oracle.uma.xyz/propose?project=Polymarket&transactionHash={hash}&eventIndex={logIndex}`

Scout does this automatically when the UMA column is enabled.
