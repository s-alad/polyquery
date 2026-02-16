# Polymarket APIs

Complete reference of all Polymarket APIs, their purposes, and key details.

## 1. Gamma API — Market Discovery & Metadata

**Base URL:** `https://gamma-api.polymarket.com`
**Auth:** None (public, read-only)

Hosted indexing service for market data, categorization, and indexed volume.

### Structure

- **Events** group related markets (single-market or group-market)
- **Markets** are the atomic unit — each has two CLOB token IDs, a market address, question ID, and condition ID

### Key Endpoints

| Endpoint | Description |
|----------|-------------|
| `GET /events` | List/filter events |
| `GET /events/{id}` | Single event by ID |
| `GET /events/{slug}` | Single event by slug |
| `GET /markets` | List/filter markets |
| `GET /markets/{id}` | Single market by ID |
| `GET /markets/{slug}` | Single market by slug |
| `GET /series` | Market series |
| `GET /series/{id}` | Series by ID |
| `GET /tags` | List tags |
| `GET /tags/{id}` | Tag by ID |
| `GET /tags/{slug}` | Tag by slug |

### Market Query Parameters

**Pagination & Sorting:**
- `limit` (int) — result count
- `offset` (int) — pagination offset
- `order` (string) — comma-separated field names
- `ascending` (bool) — sort direction

**Volume & Liquidity Filtering:**
- `volume_num_min` / `volume_num_max` — volume range in USD
- `liquidity_num_min` / `liquidity_num_max` — liquidity range

**Status Filters:**
- `closed` (bool)
- `active` (bool)
- `archived` (bool)
- `featured` (bool)
- `cyom` (bool) — "Create Your Own Market"

**Date Filters:**
- `start_date_min` / `start_date_max`
- `end_date_min` / `end_date_max`

**Other:**
- `id` (int[]) — filter by IDs
- `slug` (string[])
- `tag_id` (int) / `tag_slug` (string)
- `condition_ids` (string[])
- `clob_token_ids` (string[])
- `uma_resolution_status` (string)
- `game_id`, `sports_market_types` (string[])
- `rewards_min_size` (number)

### Event Query Parameters

Same pagination/sorting plus:
- `liquidity_min` / `liquidity_max`
- `volume_min` / `volume_max`
- `recurrence` (string)
- `include_chat` / `include_template` (bool)

### Market Response Fields

**Core:** `id`, `question`, `conditionId`, `slug`, `description`
**Pricing:** `outcomePrices`, `lastTradePrice`, `bestBid`, `bestAsk`
**Volume:** `volume`, `volumeNum`, `volume24hr`, `volume1wk`, `volume1mo`, `volume1yr`
**Liquidity:** `liquidity`, `liquidityNum`, `liquidityAmm`, `liquidityClob`
**Dates:** `startDate`, `endDate`, `createdAt`, `updatedAt`, `closedTime`
**Status:** `active`, `closed`, `archived`
**Resolution:** `resolutionSource`, `umaResolutionStatus`
**Relations:** `events[]`, `categories[]`, `tags[]`

### Event Response Fields

**Core:** `id`, `ticker`, `slug`, `title`, `subtitle`, `description`, `image`, `icon`
**Status:** `active`, `closed`, `archived`, `featured`, `restricted`, `new`
**Financial:** `liquidity`, `volume`, `openInterest`, `liquidityAmm`, `liquidityClob`
**Dates:** `startDate`, `endDate`, `creationDate`, `closedTime`
**Sports:** `score`, `elapsed`, `period`, `live`, `ended`, `gameStatus`
**Relations:** `markets[]`, `series[]`, `tags[]`, `categories[]`

---

## 2. CLOB API — Trading & Order Books

**Base URL:** `https://clob.polymarket.com`

Central Limit Order Book for real-time pricing and trade execution.

### Method Tiers

| Tier | Auth Required | Purpose |
|------|---------------|---------|
| Public | None | Prices, order books, spreads, market info |
| L1 | Private key (signer) | API key management, local order signing |
| L2 | API key + signer | Place/cancel orders, positions, trades |

### Public Methods (No Auth)

| Method | Description |
|--------|-------------|
| `getOk()` | Health check |
| `getServerTime()` | Server unix timestamp |
| `getMarket(conditionId)` | Market details |
| `getMarkets()` | Paginated market list |
| `getSimplifiedMarkets()` | Faster simplified market data |
| `getSamplingMarkets()` | Market sampling |
| `getOrderBook(tokenID)` | Bids/asks for a token |
| `getOrderBooks(BookParams[])` | Batch order books |
| `getPrice(tokenID, side)` | Best bid/ask |
| `getPrices(BookParams[])` | Batch prices |
| `getMidpoint(tokenID)` | Mid price |
| `getMidpoints(BookParams[])` | Batch midpoints |
| `getSpread(tokenID)` | Bid-ask spread |
| `getSpreads(BookParams[])` | Batch spreads |
| `calculateMarketPrice(tokenID, side, amount, orderType)` | Price for trade size |
| `getPricesHistory(params)` | Historical prices (intervals: max, 1w, 1d, 6h, 1h) |
| `getLastTradePrice(tokenID)` | Last trade price + side |
| `getLastTradesPrices(BookParams[])` | Batch last trades |
| `getMarketTradesEvents(conditionID)` | Trade event stream |
| `getFeeRateBps(tokenID)` | Fee rate in bps |
| `getTickSize(tokenID)` | Min price increment (0.1, 0.01, 0.001, 0.0001) |
| `getNegRisk(tokenID)` | Negative risk status |

### L1 Methods (Signer Required)

| Method | Description |
|--------|-------------|
| `createApiKey(nonce?)` | Generate new API credentials |
| `deriveApiKey(nonce?)` | Retrieve existing credentials |
| `createOrDeriveApiKey(nonce?)` | Create or retrieve (recommended) |
| `createOrder(userOrder, options?)` | Sign limit order locally |
| `createMarketOrder(userMarketOrder, options?)` | Sign market order locally |

**API Key Response:** `{ apiKey, secret, passphrase }`

### L2 Methods (Full Auth)

| Method | Description |
|--------|-------------|
| `createAndPostOrder(params)` | Sign + submit limit order |
| `createAndPostMarketOrder(params)` | Sign + submit market order |
| `postOrder(signedOrder, orderType?, postOnly?)` | Submit pre-signed order |
| `postOrders(orders[])` | Batch submit (up to 15) |
| `cancelOrder(orderID)` | Cancel single order |
| `cancelOrders(orderIDs[])` | Cancel multiple |
| `cancelAll()` | Cancel all open orders |
| `cancelMarketOrders(market?, asset_id?)` | Cancel by market |
| `getOrder(orderID)` | Order details |
| `getOpenOrders(params?)` | All open orders |
| `getTrades(params?)` | Trade history |
| `getTradesPaginated(params)` | Paginated trade history |
| `getBalanceAllowance(assetType, tokenId?)` | Balance + allowance |
| `updateBalanceAllowance(assetType, tokenId?)` | Refresh cached balance |
| `getApiKeys()` | List API keys |
| `deleteApiKey()` | Revoke current key |
| `getNotifications()` | Event notifications (48hr TTL) |
| `dropNotifications(ids[])` | Dismiss notifications |

**Order Types:** GTC (Good Till Cancelled), GTD (Good Till Date), FOK (Fill or Kill), FAK (Fill and Kill)
**Sides:** BUY, SELL

### SDKs

| Language | Package |
|----------|---------|
| TypeScript | `@polymarket/clob-client` |
| Python | `py-clob-client` |
| Rust | `polymarket-client-sdk` |

---

## 3. Data API — User Positions & History

**Base URL:** `https://data-api.polymarket.com`

User-specific portfolio and activity data.

### Endpoints

| Endpoint | Description |
|----------|-------------|
| `GET /` | Health check |
| `GET /positions` | User positions (current) |
| `GET /positions/closed` | Closed positions |
| `GET /activity` | User activity logs |
| `GET /trades` | Trade history |
| Top holders for markets | |
| Trader leaderboard rankings | |
| Total value of positions | |
| Open interest | |
| Live volume for events | |
| Builder leaderboard | |
| Daily builder volume time-series | |

---

## 4. WebSocket (CLOB) — Real-Time Updates

**URL:** `wss://ws-subscriptions-clob.polymarket.com/ws/`

### Channels

| Channel | Subscribe By | Data |
|---------|-------------|------|
| MARKET | `assets_ids` (token IDs) | Orderbook & price updates |
| USER | `markets` (condition IDs) | Order fills, cancellations, status |

### Subscription Message

```json
{
  "auth": {},
  "markets": ["condition_id_1"],
  "assets_ids": ["token_id_1"],
  "type": "MARKET",
  "custom_feature_enabled": false
}
```

Modify subscriptions after connecting:
```json
{
  "assets_ids": ["token_id_2"],
  "operation": "subscribe"
}
```

---

## 5. RTDS — Real-Time Data Socket

**URL:** `wss://ws-live-data.polymarket.com`

Lightweight WebSocket streaming for live data.

### Channels

- **Crypto Prices** — real-time cryptocurrency price feeds
- **Comments** — live comment stream with reactions

### Message Format

```json
{
  "topic": "string",
  "type": "string",
  "timestamp": 1234567890,
  "payload": {}
}
```

### Notes

- Send PING every ~5 seconds to maintain connection
- Some streams require `gamma_auth` with wallet address
- Can add/remove subscriptions without reconnecting
- Official TS client: `real-time-data-client` on GitHub

---

## 6. CTF — Conditional Token Framework

On-chain operations for outcome tokens (Gnosis ERC1155 on Polygon).

### Operations

| Operation | Description |
|-----------|-------------|
| Split | Convert USDC into YES/NO outcome token pairs |
| Merge | Combine YES+NO tokens back into USDC |
| Redeem | Cash out winning tokens after resolution |

### Token Hierarchy

```
ConditionId = getConditionId(oracle, questionId, outcomeSlotCount=2)
CollectionId = getCollectionId(parentCollectionId, conditionId, indexSet)
PositionId = getPositionId(collateralToken, collectionId)
```

- `IndexSet`: 1 (0b01) = first outcome, 2 (0b10) = second outcome
- Tokens are ERC1155 standard

---

## 7. Subgraph — On-Chain GraphQL

**Repo:** https://github.com/Polymarket/polymarket-subgraph

Self-hostable GraphQL indexer for on-chain Polygon data.

### Indexed Data

- Volume
- User positions
- Market information
- Liquidity
- Activity history

Can be hosted by anyone. Schema in `schema.graphql` in the repo.

---

## Quick Reference: Which API for What?

| Task | API |
|------|-----|
| Find markets by volume/liquidity | Gamma |
| Get market metadata & resolution info | Gamma |
| Get real-time prices & order books | CLOB (public) |
| Place/cancel trades | CLOB (L2) |
| Check user positions & history | Data API |
| Stream live price updates | WebSocket |
| Stream crypto prices & comments | RTDS |
| Split/merge/redeem tokens on-chain | CTF |
| Query on-chain state via GraphQL | Subgraph |
