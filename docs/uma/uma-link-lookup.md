# UMA Link Lookup

How we find the oracle.uma.xyz propose page for a Polymarket market.

## The Problem

Polymarket markets are resolved via UMA's Optimistic Oracle, but Polymarket's API doesn't expose a direct link to the UMA propose page. To propose a resolution, you need a URL like:

```
https://oracle.uma.xyz/propose?project=Polymarket&transactionHash=0x...&eventIndex=132
```

Where `transactionHash` is the on-chain `initialize()` transaction and `eventIndex` is the block-level log index. Neither field exists in any Polymarket API.

## Data Sources

### Gamma API (Polymarket)

Gives us the market metadata but NOT the UMA tx hash.

```
GET https://gamma-api.polymarket.com/markets?id={marketID}
```

Relevant fields:
- `id` — numeric market ID (e.g. `"1384032"`)
- `questionID` — keccak256 hash used in on-chain events
- `resolvedBy` — UmaCtfAdapter contract address
- `umaResolutionStatus` — `""` (proposable), `"proposed"`, `"resolved"`

### UMA Goldsky Subgraph (free, no auth)

```
POST https://api.goldsky.com/api/public/project_clus2fndawbcc01w31192938i/subgraphs/polygon-optimistic-oracle-v2/1.1.0/gn
```

Indexes all `OptimisticPriceRequest` events on the OOV2 contract (`0xee3afe...`). Contains `requestHash`, `requestLogIndex`, `ancillaryData`, `state`, `proposer`, etc.

**Coverage:** v2 (`0x6A9D...`), v3 (`0x2f5e...`), and legacy (`0x157Ce2d6...`) adapters — all OOV2, permissionless. ~57% of markets.

**Does NOT cover:** sports adapter (`0x6507...`) which uses MOOV2 (`0x2C0367...`), a separate contract with no deployed subgraph.

**Indexing gap:** Very old OOV2 markets (pre-mid-2024 approx.) predate the subgraph's start block and will return no results. These show as `-` in scout — not a bug.

### Polygon RPC (FilterLogs)

Direct log scanning on Polygon. Works for any contract but limited to 10K blocks per query on free-tier RPCs (~5.5 hours of blocks). Must chunk and iterate.

## The Cross-Reference

Gamma's `id` field maps to the `market_id` embedded in UMA's `ancillaryData`. The ancillary data is hex-encoded text:

```
q: title: Will Chelsea FC win on 2026-03-01?, description: ...
market_id: 1384032 res_data: p1: 0, p2: 1, p3: 0.5. ...
```

To find a market in the subgraph: hex-encode `"market_id: {gammaID}"` and use `ancillaryData_contains`.

```graphql
{
  optimisticPriceRequests(
    where: { ancillaryData_contains: "6d61726b65745f69643a2031333834303332" }
    first: 1
  ) { requestHash requestLogIndex state }
}
```

## Our Solution (`pkg/chain/uma.go`)

Two-path lookup based on which oracle contract the adapter uses:

```
Gamma market.ResolvedBy
        │
        ├── v2, v3, or legacy ─→ Subgraph lookup (instant, ~200ms)
        │                         Query ancillaryData_contains with hex(market_id)
        │                         Returns requestHash + requestLogIndex
        │
        └── sports adapter ───→ RPC log scan (slow, ~30-40s)
                                  FilterLogs for QuestionInitialized event
                                  Chunked 10K blocks backward from head
                                  Returns txHash + log.Index
```

Both paths build the same URL format:
```
oracle.uma.xyz/propose?project=Polymarket&transactionHash={hash}&eventIndex={logIndex}
```

### Why two paths?

The OOV2 subgraph only indexes the `OptimisticOracleV2` contract. Sports markets route through `ManagedOptimisticOracleV2` (MOOV2) which is a separate contract. UMA has the subgraph code (`UMAprotocol/subgraphs/packages/managed-oracle-v2`) but it's not deployed on Goldsky for Polygon.

### Adapter → Oracle mapping

| Adapter | Oracle | Subgraph? | Market share |
|---------|--------|-----------|-------------|
| v2 (`0x6A9D...`) | OOV2 | Yes | ~5% |
| v3 (`0x2f5e...`) | OOV2 | Yes | ~51% |
| sports (`0x6507...`) | MOOV2 | No | ~43% |
| `0x157Ce2d6...` | OOV2 | Yes | ~1% |

### eventIndex

UMA's oracle page expects the **block-level** log index (the position of the log within the entire block, not within the transaction). This is:
- `log.Index` from go-ethereum's `FilterLogs`
- `requestLogIndex` from the subgraph

Both are block-level. Confirmed working with eventIndex=132 for the Stony Brook market.

## API (`pkg/chain/uma.go`)

```go
client := chain.NewUmaClient("https://polygon-rpc.example.com")
defer client.Close()

// UmaLink — single market propose URL (fields from gamma.Market)
link, err := client.UmaLink(ctx, market.ID, market.ResolvedBy, market.QuestionID)

// FetchProposals — batch proposal data for a slice of market IDs (OOV2 subgraph only)
proposals := client.FetchProposals(ctx, []string{market.ID, ...})
// returns map[int]chain.Proposal{Timestamp int64, Proposer string}
```

RPC URL is only needed for MOOV2/sports markets. OOV2 markets use the subgraph (no RPC).

### Internal structure

```
subgraphQuery(ctx, query, &out)   ← core engine: HTTP POST + JSON decode
       ↑                  ↑
subgraphLink          FetchProposals   ← each builds query via marketQuery(), decodes own struct
```

- `subgraphQuery` — generic: sends any GraphQL body, streams response into `out`
- `marketQuery(queryFmt, marketID)` — builds JSON body with hex-encoded `market_id:` in the where clause
- `subgraphLink` — OOV2 path: fetches `requestHash + requestLogIndex`, returns propose URL
- `rpcLink` — MOOV2 fallback: chunked `FilterLogs` scan for `QuestionInitialized`

## Alternative URL Format

UMA also supports a parameter-based URL (no tx hash needed):

```
oracle.uma.xyz/propose?chainId=137&oracleType=Optimistic+Oracle+V2
  &requester={adapter}&timestamp={requestTimestamp}
  &identifier=YES_OR_NO_QUERY&ancillaryData={hexData}
```

All params available from the subgraph. Not currently implemented but useful as a fallback.

## Ancillary Data Structure

The `res_data` field in ancillary data maps outcomes to propose values:

```
res_data: p1: 0, p2: 1, p3: 0.5
Where p1 corresponds to No, p2 to Yes, p3 to unknown/50-50
```

For `proposePrice()`: propose `1e18` for Yes (p2=1), `0` for No (p1=0), `0.5e18` for unknown.
