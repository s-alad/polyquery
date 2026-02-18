// ---
// infinitesum.xyz
// chain/links.go provides functions to get UMA proposal links
// ---
package chain

import (
	"context"
	"fmt"
	"math/big"
	"strconv"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

// rpcLink scans Polygon logs for QuestionInitialized events — MOOV2/sports fallback
func (u *UmaClient) rpcLink(ctx context.Context, questionID, adapter string) (string, error) {
	if u.eth == nil {
		if u.rpcURL == "" {
			return "", fmt.Errorf("rpc_url required for MOOV2/sports market lookup")
		}
		eth, err := ethclient.Dial(u.rpcURL)
		if err != nil {
			return "", fmt.Errorf("rpc dial: %w", err)
		}
		u.eth = eth
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
			return fmt.Sprintf(umaBase, l.TxHash.Hex(), strconv.FormatUint(uint64(l.Index), 10)), nil
		}
		hi = lo
	}
	return "", fmt.Errorf("no QuestionInitialized event found for %s", questionID)
}

// subgraphLink queries the Goldsky subgraph for a market's request tx hash + log index
func (u *UmaClient) subgraphLink(ctx context.Context, marketID string) (string, error) {
	q := marketQuery(`{ optimisticPriceRequests(where:{ancillaryData_contains:"%s"} first:1) { requestHash requestLogIndex } }`, marketID)
	var result struct {
		Data struct {
			Requests []struct {
				Hash     string `json:"requestHash"`
				LogIndex string `json:"requestLogIndex"`
			} `json:"optimisticPriceRequests"`
		} `json:"data"`
	}
	if err := u.subgraphQuery(ctx, q, &result); err != nil {
		return "", fmt.Errorf("subgraph: %w", err)
	}
	if len(result.Data.Requests) == 0 {
		return "", fmt.Errorf("market %s not found in subgraph", marketID)
	}
	r := result.Data.Requests[0]
	return fmt.Sprintf(umaBase, r.Hash, r.LogIndex), nil
}
