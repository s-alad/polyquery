package chain

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/salad/polyquery/pq/pkg/kv"
)

const (
	subgraphURL = "https://api.goldsky.com/api/public/project_clus2fndawbcc01w31192938i/subgraphs/polygon-optimistic-oracle-v2/1.1.0/gn"
	umaBase     = "https://oracle.uma.xyz/propose?project=Polymarket&transactionHash=%s&eventIndex=%s"
	chunkSize   = 10000
	scanDays    = 180
)

var questionInitSig = crypto.Keccak256Hash(
	[]byte("QuestionInitialized(bytes32,uint256,address,bytes,address,uint256,uint256)"),
)

// OOV2 adapters — indexed by UMA's Goldsky subgraph. Permissionless (no whitelist)
var oov2Adapters = map[string]bool{
	"0x6a9d222616c90fca5754cd1333cfd9b7fb6a4f74": true, // v2
	"0x2f5e3684cb1f318ec51b00edba38d79ac2c0aa9d": true, // v3
	"0x157ce2d672854c848c9b79c49a8cc6cc89176a49": true, // legacy (soccer draw markets etc.)
}

// IsOOV2 returns true if the adapter uses OOV2 (permissionless, no whitelist needed)
func IsOOV2(adapter string) bool { return oov2Adapters[strings.ToLower(adapter)] }

// Proposal holds on-chain proposal data for a market
type Proposal struct {
	Timestamp int64
	Proposer  string
}

type UmaClient struct {
	http   *http.Client
	eth    *ethclient.Client
	rpcURL string
	cache  *kv.DB
}

func NewUmaClient(rpcURL string) *UmaClient {
	cache, _ := kv.Open("pkg/kv/cache")
	return &UmaClient{
		http:   &http.Client{Timeout: 60 * time.Second},
		rpcURL: rpcURL,
		cache:  cache,
	}
}

func (u *UmaClient) Close() {
	if u.eth != nil {
		u.eth.Close()
	}
	if u.cache != nil {
		u.cache.Close()
	}
}

// UmaLink returns the oracle.uma.xyz propose URL for a market
// Uses the subgraph for OOV2 adapters; falls back to RPC log scan for MOOV2
func (u *UmaClient) UmaLink(ctx context.Context, marketID, adapter, questionID string) (string, error) {
	if adapter == "" {
		return "", fmt.Errorf("adapter address required")
	}
	if IsOOV2(adapter) {
		return u.subgraphLink(ctx, marketID)
	}
	return u.rpcLink(ctx, questionID, adapter)
}

// subgraphQuery sends a GraphQL query to the Goldsky subgraph and decodes the response into out
func (u *UmaClient) subgraphQuery(ctx context.Context, query string, out any) error {
	req, _ := http.NewRequestWithContext(ctx, "POST", subgraphURL, strings.NewReader(query))
	req.Header.Set("Content-Type", "application/json")
	resp, err := u.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(out)
}

// marketQuery builds a JSON GraphQL request body, hex-encoding the market ID into the query.
func marketQuery(queryFmt, marketID string) string {
	marker := hex.EncodeToString([]byte("market_id: " + marketID))
	b, _ := json.Marshal(map[string]string{"query": fmt.Sprintf(queryFmt, marker)})
	return string(b)
}
