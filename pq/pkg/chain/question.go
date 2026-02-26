// ---
// infinitesum.xyz
// chain/question.go reads QuestionData from UmaCtfAdapter contracts via getQuestion(bytes32)
// ---
package chain

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

// QuestionData mirrors the on-chain QuestionData struct returned by getQuestion(bytes32)
type QuestionData struct {
	RequestTimestamp *big.Int
	Reward           *big.Int
	ProposalBond     *big.Int
	Resolved         bool
	Paused           bool
	AncillaryData    []byte
	Creator          common.Address
	RewardToken      common.Address
}

var getQuestionABI, _ = abi.JSON(strings.NewReader(`[{
	"name":"getQuestion",
	"type":"function",
	"inputs":[{"name":"questionID","type":"bytes32"}],
	"outputs":[{"name":"","type":"tuple","components":[
		{"name":"requestTimestamp","type":"uint256"},
		{"name":"reward","type":"uint256"},
		{"name":"proposalBond","type":"uint256"},
		{"name":"liveness","type":"uint256"},
		{"name":"manualResolutionTimestamp","type":"uint256"},
		{"name":"resolved","type":"bool"},
		{"name":"paused","type":"bool"},
		{"name":"reset","type":"bool"},
		{"name":"refund","type":"bool"},
		{"name":"rewardToken","type":"address"},
		{"name":"creator","type":"address"},
		{"name":"ancillaryData","type":"bytes"}
	]}]
}]`))

// GetQuestion reads QuestionData from the adapter contract.
// For negRisk markets, pass negRiskRequestID; for non-negRisk, pass questionID.
func GetQuestion(ctx context.Context, eth *ethclient.Client, adapter string, questionKey string) (*QuestionData, error) {
	addr := common.HexToAddress(adapter)
	key := common.HexToHash(questionKey)

	calldata, err := getQuestionABI.Pack("getQuestion", key)
	if err != nil {
		return nil, fmt.Errorf("pack: %w", err)
	}

	result, err := eth.CallContract(ctx, ethereum.CallMsg{To: &addr, Data: calldata}, nil)
	if err != nil {
		return nil, fmt.Errorf("call: %w", err)
	}

	out, err := getQuestionABI.Unpack("getQuestion", result)
	if err != nil {
		return nil, fmt.Errorf("unpack: %w", err)
	}

	// go-ethereum ABI unpacks tuples as anonymous structs
	raw := out[0].(struct {
		RequestTimestamp          *big.Int       `json:"requestTimestamp"`
		Reward                   *big.Int       `json:"reward"`
		ProposalBond             *big.Int       `json:"proposalBond"`
		Liveness                 *big.Int       `json:"liveness"`
		ManualResolutionTimestamp *big.Int       `json:"manualResolutionTimestamp"`
		Resolved                 bool           `json:"resolved"`
		Paused                   bool           `json:"paused"`
		Reset                    bool           `json:"reset"`
		Refund                   bool           `json:"refund"`
		RewardToken              common.Address `json:"rewardToken"`
		Creator                  common.Address `json:"creator"`
		AncillaryData            []byte         `json:"ancillaryData"`
	})

	if raw.RequestTimestamp.Sign() == 0 {
		return nil, fmt.Errorf("question not initialized (timestamp=0)")
	}

	return &QuestionData{
		RequestTimestamp: raw.RequestTimestamp,
		Reward:          raw.Reward,
		ProposalBond:    raw.ProposalBond,
		Resolved:        raw.Resolved,
		Paused:          raw.Paused,
		AncillaryData:   raw.AncillaryData,
		Creator:         raw.Creator,
		RewardToken:     raw.RewardToken,
	}, nil
}

// QuestionKey returns the correct bytes32 key for getQuestion() based on market type.
// negRisk markets use NegRiskRequestID; non-negRisk use QuestionID.
func QuestionKey(negRisk bool, questionID, negRiskRequestID string) string {
	if negRisk && negRiskRequestID != "" {
		return negRiskRequestID
	}
	return questionID
}
