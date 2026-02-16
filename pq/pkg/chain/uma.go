package chain

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

const (
	subgraphURL = "https://api.goldsky.com/api/public/project_clus2fndawbcc01w31192938i/subgraphs/polygon-optimistic-oracle-v2/1.1.0/gn"
	umaBase     = "https://oracle.uma.xyz/propose?project=Polymarket&transactionHash=%s&eventIndex=%s"
	chunkSize   = 10000
)

var questionInitSig = crypto.Keccak256Hash(
	[]byte("QuestionInitialized(bytes32,uint256,address,bytes,address,uint256,uint256)"),
)

// OOV2 adapters — indexed by UMA's Goldsky subgraph.
var oov2Adapters = map[string]bool{
	"0x6a9d222616c90fca5754cd1333cfd9b7fb6a4f74": true, // v2
	"0x2f5e3684cb1f318ec51b00edba38d79ac2c0aa9d": true, // v3
}

type UmaClient struct {
	http   *http.Client
	eth    *ethclient.Client
	rpcURL string
}

func NewUmaClient(rpcURL string) *UmaClient {
	return &UmaClient{
		http:   &http.Client{Timeout: 15 * time.Second},
		rpcURL: rpcURL,
	}
}

func (u *UmaClient) Close() {
	if u.eth != nil {
		u.eth.Close()
	}
}

// UmaLink returns the oracle.uma.xyz propose URL for a market.
// marketID = gamma.Market.ID, adapter = gamma.Market.ResolvedBy, questionID = gamma.Market.QuestionID.
// Uses the subgraph for OOV2 adapters (instant), falls back to RPC log scan for MOOV2/sports.
func (u *UmaClient) UmaLink(ctx context.Context, marketID, adapter, questionID string) (string, error) {
	if adapter == "" {
		return "", fmt.Errorf("adapter address required")
	}

	if oov2Adapters[strings.ToLower(adapter)] {
		return u.subgraphLookup(ctx, marketID)
	}
	return u.rpcLookup(ctx, questionID, adapter)
}

// subgraphLookup queries UMA's Goldsky subgraph by market_id encoded in ancillaryData.
func (u *UmaClient) subgraphLookup(ctx context.Context, marketID string) (string, error) {
	// ancillaryData is hex-encoded text containing "market_id: XXXXXX"
	marker := hex.EncodeToString([]byte("market_id: " + marketID))

	query := fmt.Sprintf(`{
		optimisticPriceRequests(
			where: { ancillaryData_contains: "%s" }
			first: 1
		) { requestHash requestLogIndex }
	}`, marker)

	body, _ := json.Marshal(map[string]string{"query": query})
	req, _ := http.NewRequestWithContext(ctx, "POST", subgraphURL, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := u.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("subgraph: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	var result struct {
		Data struct {
			Requests []struct {
				Hash     string `json:"requestHash"`
				LogIndex string `json:"requestLogIndex"`
			} `json:"optimisticPriceRequests"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", fmt.Errorf("subgraph decode: %w", err)
	}
	if len(result.Data.Requests) == 0 {
		return "", fmt.Errorf("market %s not found in subgraph", marketID)
	}

	r := result.Data.Requests[0]
	return fmt.Sprintf(umaBase, r.Hash, r.LogIndex), nil
}

// rpcLookup scans Polygon logs for QuestionInitialized events (MOOV2/sports fallback).
func (u *UmaClient) rpcLookup(ctx context.Context, questionID, adapter string) (string, error) {
	if err := u.ensureEth(); err != nil {
		return "", err
	}

	head, err := u.eth.BlockNumber(ctx)
	if err != nil {
		return "", fmt.Errorf("get block number: %w", err)
	}

	floor := head - 30*43200
	addr := common.HexToAddress(adapter)
	qID := common.HexToHash(questionID)

	for hi := head; hi > floor; {
		lo := hi - chunkSize
		if lo < floor {
			lo = floor
		}
		logs, err := u.eth.FilterLogs(ctx, ethereum.FilterQuery{
			Addresses: []common.Address{addr},
			Topics:    [][]common.Hash{{questionInitSig}, {qID}},
			FromBlock: new(big.Int).SetUint64(lo),
			ToBlock:   new(big.Int).SetUint64(hi),
		})
		if err != nil {
			return "", fmt.Errorf("filter logs [%d-%d]: %w", lo, hi, err)
		}
		if len(logs) > 0 {
			l := logs[0]
			return fmt.Sprintf(umaBase, l.TxHash.Hex(), fmt.Sprintf("%d", l.Index)), nil
		}
		hi = lo
	}
	return "", fmt.Errorf("no QuestionInitialized event found for %s", questionID)
}

func (u *UmaClient) ensureEth() error {
	if u.eth != nil {
		return nil
	}
	if u.rpcURL == "" {
		return fmt.Errorf("rpc_url required for MOOV2/sports market lookup")
	}
	eth, err := ethclient.Dial(u.rpcURL)
	if err != nil {
		return fmt.Errorf("rpc dial: %w", err)
	}
	u.eth = eth
	return nil
}
