# UMA Resolution Process

How Polymarket markets are resolved via UMA's Optimistic Oracle.

## The Optimistic Oracle Flow

1. **Market Ends** — Polymarket market reaches its end date
2. **Propose** — Anyone calls `proposePrice()` with a $750 USDC bond
3. **Liveness Period** — 2-hour challenge window begins
4. **Dispute (Optional)** — Anyone can dispute by posting their own bond
5. **Resolution** — If undisputed after 2hrs, proposal accepted. If disputed, goes to UMA DVM vote

## Oracle Versions

### OptimisticOracleV2 (OOV2)
- **Permissionless** — anyone can propose
- Fewer markets use this now
- Good for building a track record

### ManagedOptimisticOracleV2 (MOOV2)
- **Whitelisted** — ~177 approved proposer addresses (as of Nov 2025)
- Most Polymarket markets use this
- Disputes remain permissionless

## Becoming a Whitelisted Proposer

### Requirements (rolling 6-month window)
- 5+ proposals on OOV2 or MOOV2
- >95% proposal accuracy

### How to Qualify
1. Find non-whitelisted OOV2 requests at [oracle.uma.xyz/propose](https://oracle.uma.xyz/propose?oracleType=Optimistic+Oracle+V2&project=Polymarket)
2. Propose accurately (risking $750/proposal)
3. Build track record of 5+ accurate proposals
4. Get auto-added — whitelist snapshots taken on 2nd of each month

### Whitelist Schedule
- **Snapshot:** 2nd of each month (preceding 6 months)
- **Update:** within the following week
- **Removal:** addresses falling below criteria are removed

### Check Whitelist
[Polygonscan - MOOV2 Contract](https://polygonscan.com/address/0x9f35885ce8f67a942d7b2f4fbf937987da08c463#readContract#F1)

## ProposePrice Function

```solidity
function proposePrice(
    address requester,        // UmaCtfAdapter address
    bytes32 identifier,       // "YES_OR_NO_QUERY"
    uint256 timestamp,        // Request timestamp
    bytes ancillaryData,      // Question data
    int256 proposedPrice      // 1e18 for Yes, 0 for No
) external returns (uint256 totalBond);
```

## Key Contract Addresses (Polygon)

| Contract | Address |
|----------|---------|
| Finder | `0x09aea4b2242abC8bb4BB78D537A67a245A7bEC64` |
| OptimisticOracle V1 | `0xBb1A8db2D4350976a11cdfA60A1d43f97710Da49` |
| OptimisticOracle V2 | `0xee3afe347d5c74317041e2618c49534daf887c24` |
| OptimisticOracle V3 | `0x5953f2538F613E05bAED8A5AeFa8e6622467AD3D` |
| UMA CTF Adapter V2 | `0x6A9D222616C90FcA5754cd1333cFD9b7fb6a4F74` |
| UMA CTF Adapter V3 | `0x2f5e3684cb1f318ec51b00edba38d79ac2c0aa9d` |
| UMA CTF Adapter (sports) | `0x65070BE91477460D8A7AeEb94ef92fe056C2f2A7` |
| USDC | `0x2791Bca1f2de4661ED88A30C99A7a9449Aa84174` |
| ConditionalTokens | `0x4D97DCd97eC945f40cF65F87097ACe5EA0476045` |

## Economics

| | Amount |
|-|--------|
| Bond required | $750 USDC |
| Reward per correct proposal | $2–$5 |
| Risk if wrong | -$750 |
| Liveness period | 2 hours |

## Results Since MOOV2 Launch (Sept 2025)
- Incorrect proposals: **down 59%**
- Total disputes: **down 68%**

## Onboarding Path

```
Phase 0: Bootstrap
├── Fund wallet: $750+ USDC on Polygon + gas
├── Find OOV2 non-whitelisted Polymarket requests
├── Make 5+ accurate proposals (may take 2-6 months)
└── Wait for monthly whitelist snapshot

Phase 1: Whitelisted
├── Access MOOV2 markets (higher volume)
├── Scale to 50+ proposals/day
└── Target $100-500/day revenue
```
