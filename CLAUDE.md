# CLAUDE.md

This file provides guidance to Claude Code when working with this repository.

## Project Overview

**Polyquery** is a Bloomberg Terminal / Axiom-style interface for prediction markets, starting with Polymarket and expanding to Kalshi.

### Core Mission

Build a powerful terminal/dashboard that enables users to:
- Query Polymarket for whales, addresses, and unique events
- Track and analyze market activity in real-time
- Create custom detection rules via drag-and-drop editor
- Set up automated buy/sell triggers based on custom conditions

### Target Platforms

1. **Polymarket** (primary, initial focus)
2. **Kalshi** (planned expansion)

## Current Development Phase

**Phase 1: Go CLOB Client** - Building a native Go client for Polymarket's CLOB API by reverse-engineering the official Python client. This is foundational for low-latency operations.

### Why Go?

- Low latency and high performance for real-time market data
- Excellent concurrency model for handling multiple streams
- Single binary deployment
- Strong typing catches errors at compile time

## Architecture

```
polyquery/
├── pkg/
│   └── clob/              # Go CLOB client (reverse-engineered from Python)
│       ├── client.go      # Main client struct and methods
│       ├── types.go       # Data structures and enums
│       ├── endpoints.go   # API endpoint constants
│       ├── auth.go        # Authentication (L1/L2)
│       ├── orders.go      # Order creation/management
│       ├── markets.go     # Market data queries
│       └── signer.go      # EIP-712 signing
├── internal/
│   └── ...                # Internal application logic
├── cmd/
│   └── polyquery/         # Main application entry point
└── web/                   # Future: UI layer
```

## Polymarket CLOB API Reference

### Base URLs
- **Production**: `https://clob.polymarket.com`
- **Chain**: Polygon (chain_id: 137)

### Authentication Levels

| Level | Requirements | Capabilities |
|-------|--------------|--------------|
| L0 | None | Public market data (orderbooks, prices, markets) |
| L1 | Private key + chain_id | API key creation/derivation |
| L2 | API credentials (key, secret, passphrase) | Trading, orders, account management |

### API Endpoints

#### Public (No Auth)
```
GET  /time                          # Server time
GET  /markets                       # All markets
GET  /simplified-markets            # Simplified market list
GET  /markets/{condition_id}        # Single market
GET  /sampling-markets              # Sampling pool markets
GET  /book?token_id=X               # Order book
GET  /books                         # Multiple order books
GET  /midpoint?token_id=X           # Mid price
GET  /midpoints                     # Multiple mid prices
GET  /price?token_id=X&side=Y       # Price for side
GET  /prices                        # Multiple prices
GET  /spread?token_id=X             # Bid-ask spread
GET  /spreads                       # Multiple spreads
GET  /last-trade-price?token_id=X   # Last trade
GET  /last-trades-prices            # Multiple last trades
GET  /tick-size?token_id=X          # Min price increment
GET  /neg-risk?token_id=X           # Negative risk flag
GET  /fee-rate?token_id=X           # Fee in basis points
```

#### L1 Auth (Private Key Signing)
```
GET  /auth/api-key                  # Create API key
GET  /auth/derive-api-key           # Derive existing key
GET  /auth/api-keys                 # List API keys
DELETE /auth/api-key                # Delete API key
POST /auth/readonly-api-key         # Create readonly key
GET  /auth/readonly-api-keys        # List readonly keys
DELETE /auth/readonly-api-key       # Delete readonly key
GET  /auth/validate-readonly-api-key # Validate readonly key (public)
```

#### L2 Auth (API Credentials)
```
POST /order                         # Submit order
POST /orders                        # Batch submit orders
DELETE /order/{order_id}            # Cancel order
DELETE /orders                      # Cancel multiple orders
DELETE /cancel-all                  # Cancel all orders
DELETE /cancel-market-orders        # Cancel by market
GET  /data/order/{order_id}         # Get order details
GET  /data/orders                   # List open orders
GET  /data/trades                   # Trade history
GET  /order-scoring                 # Single order scoring
GET  /orders-scoring                # Multiple orders scoring
GET  /balance-allowance             # Check balance/allowance
POST /balance-allowance/update      # Update allowance
GET  /notifications                 # Get notifications
DELETE /notifications               # Clear notifications
POST /v1/heartbeats                 # Keepalive (10s timeout)
GET  /auth/ban-status/closed-only   # Check close-only mode
GET  /live-activity/events/{id}     # Market trade events
```

#### Builder Endpoints
```
GET  /builder/trades                # Builder-attributed trades
```

#### RFQ (Request for Quote)
```
POST /rfq/request                   # Create RFQ
GET  /rfq/data/requests             # List requests
POST /rfq/quote                     # Submit quote
GET  /rfq/data/requester/quotes     # Requester's quotes
GET  /rfq/data/quoter/quotes        # Quoter's quotes
GET  /rfq/data/best-quote           # Best quote
POST /rfq/request/accept            # Accept RFQ
POST /rfq/quote/approve             # Approve quote
GET  /rfq/config                    # RFQ configuration
```

### Core Types (To Implement in Go)

```go
// Order types
type OrderType string
const (
    GTC OrderType = "GTC"  // Good till cancelled
    FOK OrderType = "FOK"  // Fill or kill
    GTD OrderType = "GTD"  // Good till date
    FAK OrderType = "FAK"  // Fill and kill
)

// Asset types
type AssetType string
const (
    Collateral  AssetType = "COLLATERAL"
    Conditional AssetType = "CONDITIONAL"
)

// Side
type Side string
const (
    Buy  Side = "BUY"
    Sell Side = "SELL"
)

// Tick sizes (price precision)
type TickSize string  // "0.1", "0.01", "0.001", "0.0001"

// API credentials
type ApiCreds struct {
    ApiKey        string
    ApiSecret     string
    ApiPassphrase string
}

// Order arguments
type OrderArgs struct {
    TokenID     string
    Price       float64
    Size        float64
    Side        Side
    FeeRateBps  int
    Nonce       int64
    Expiration  int64
    Taker       string  // optional
}

// Market order arguments
type MarketOrderArgs struct {
    TokenID   string
    Amount    float64
    Side      Side
    Price     float64    // optional
    OrderType OrderType  // optional
}

// Order book
type OrderBookSummary struct {
    Market    string
    AssetID   string
    Bids      []OrderSummary
    Asks      []OrderSummary
    Timestamp int64
}

type OrderSummary struct {
    Price string
    Size  string
}
```

### Signature Types

The client supports three wallet types:
- **Type 0 (EOA)**: Direct private key (MetaMask, hardware wallets)
- **Type 1 (Delegated)**: Email/Magic wallet proxy
- **Type 2 (Proxy)**: Browser wallet proxy contracts

### Contract Addresses (Polygon)

```go
const (
    USDC              = "0x2791Bca1f2de4661ED88A30C99A7a9449Aa84174"
    ConditionalTokens = "0x4D97DCd97eC945f40cF65F87097ACe5EA0476045"
)
```

## Key Features (Planned)

### Phase 1: Go CLOB Client ← Current
- [ ] HTTP client with proper auth headers
- [ ] L0 public endpoints (markets, orderbooks, prices)
- [ ] L1 auth (EIP-712 signing for API key management)
- [ ] L2 auth (HMAC signing for trading)
- [ ] Order creation and submission
- [ ] Order cancellation
- [ ] Trade history queries

### Phase 2: Query & Analytics UI
- Whale tracking and alerts
- Address lookup and portfolio analysis
- Event discovery and filtering
- Historical data analysis
- Market movement detection
- Filter orders by size (e.g., $2000-$3000 range)

### Phase 3: Automation
- Drag-and-drop rule builder for detection logic
- Custom alert conditions
- Automated trading triggers
- User-defined strategies

## Development Guidelines

### Code Style
- Follow standard Go conventions (gofmt, golint)
- Use meaningful variable/function names
- Group related functionality into separate files
- Write table-driven tests

### Architecture Principles
- Design for extensibility (new prediction markets)
- Keep the API client modular and reusable
- Separate HTTP transport from business logic
- Use interfaces for testability

## API References

### Polymarket
- Docs: https://docs.polymarket.com/
- Python client (reference): https://github.com/Polymarket/py-clob-client
- TypeScript client: https://github.com/Polymarket/clob-client
- Rust client: https://github.com/Polymarket/rust-clob-client

### Kalshi (Future)
- API documentation: https://trading-api.readme.io/reference/getting-started

## Common Commands

```bash
# Run tests
go test ./...

# Build
go build ./cmd/polyquery

# Run
./polyquery

# Format code
go fmt ./...

# Lint
golangci-lint run
```

## Project Status

Phase 1: Building Go CLOB client by reverse-engineering Python implementation.
