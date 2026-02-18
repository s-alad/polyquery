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
	"golang.org/x/sync/errgroup"
)

const workers = 10

// rpcLink scans Polygon logs for QuestionInitialized events — MOOV2/sports fallback.
// Scans forward from 30 days ago in parallel chunks of 10K blocks.
func (u *UmaClient) rpcLink(ctx context.Context, questionID, adapter string) (string, error) {
	if u.cache != nil {
		if val, err := u.cache.Get(questionID); val != nil && err == nil {
			return string(val), nil
		}
	}

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
	floor := head - scanDays*43200
	addr := common.HexToAddress(adapter)
	qID := common.HexToHash(questionID)

	// build chunk list forward (oldest → newest): QuestionInitialized fires at market
	// creation, so older chunks are more likely to contain the event
	type chunk struct{ lo, hi uint64 }
	var chunks []chunk
	for lo := floor; lo < head; lo += chunkSize {
		hi := lo + chunkSize
		if hi > head {
			hi = head
		}
		chunks = append(chunks, chunk{lo, hi})
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	found := make(chan string, 1)
	work := make(chan chunk, len(chunks))
	for _, c := range chunks {
		work <- c
	}
	close(work)

	g, gctx := errgroup.WithContext(ctx)
	for range workers {
		g.Go(func() error {
			for c := range work {
				select {
				case <-gctx.Done():
					return nil
				default:
				}
				logs, err := u.eth.FilterLogs(gctx, ethereum.FilterQuery{
					Addresses: []common.Address{addr},
					Topics:    [][]common.Hash{{questionInitSig}, {qID}},
					FromBlock: new(big.Int).SetUint64(c.lo),
					ToBlock:   new(big.Int).SetUint64(c.hi),
				})
				if err != nil {
					return fmt.Errorf("filter logs [%d-%d]: %w", c.lo, c.hi, err)
				}
				if len(logs) > 0 {
					l := logs[0]
					link := fmt.Sprintf(umaBase, l.TxHash.Hex(), strconv.FormatUint(uint64(l.Index), 10))
					select {
					case found <- link:
						cancel()
					default:
					}
					return nil
				}
			}
			return nil
		})
	}

	err = g.Wait()
	select {
	case link := <-found:
		if u.cache != nil {
			u.cache.Set(questionID, []byte(link))
		}
		return link, nil
	default:
		if err != nil {
			return "", err
		}
		return "", fmt.Errorf("no QuestionInitialized event found for %s", questionID)
	}
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
