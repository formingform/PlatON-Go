package monitor

import (
	"context"
	"github.com/PlatONnetwork/PlatON-Go/common"
	"github.com/PlatONnetwork/PlatON-Go/core/types"
	"github.com/PlatONnetwork/PlatON-Go/log"
	"github.com/PlatONnetwork/PlatON-Go/rpc"
	"github.com/PlatONnetwork/PlatON-Go/x/gov"
	"github.com/PlatONnetwork/PlatON-Go/x/plugin"
	"github.com/PlatONnetwork/PlatON-Go/x/restricting"
	"github.com/PlatONnetwork/PlatON-Go/x/staking"
	"math/big"
)

// 这个方式暂时未启用，需要scan-agent的改造，用rpc-call代替合约方法调用
type Backend interface {
	CurrentHeader() *types.Header
	CurrentBlock() *types.Block
	HeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Header, error)
	BlockByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Block, error)
	GetReceipts(ctx context.Context, blockHash common.Hash) (types.Receipts, error)
}

type MonitorAPI struct {
	b Backend
}

// APIs returns a list of APIs provided by the consensus engine.
func NewMonitorAPIs(b Backend) []rpc.API {
	return []rpc.API{
		{
			Namespace: "monitor",
			Version:   "1.0",
			Service:   &MonitorAPI{b},
			Public:    true,
		},
	}
}

func (api *MonitorAPI) GetHistoryVerifierList(blockNumber *big.Int) (staking.ValidatorExQueue, error) {
	return plugin.StakingInstance().GetHistoryVerifierList(blockNumber.Uint64())
}

func (api *MonitorAPI) GetHistoryValidatorList(blockNumber *big.Int) (staking.ValidatorExQueue, error) {
	return plugin.StakingInstance().GetHistoryValidatorList(blockNumber.Uint64())
}

func (api *MonitorAPI) GetHistoryReward(blockNumber *big.Int) (staking.RewardReturn, error) {
	return plugin.StakingInstance().GetHistoryReward(blockNumber.Uint64())
}

func (api *MonitorAPI) GetHistoryLowRateSlashList(blockNumber *big.Int) (staking.SlashNodeQueue, error) {
	return plugin.StakingInstance().GetSlashData(blockNumber.Uint64())
}

func (api *MonitorAPI) GetNodeVersion(blockHash common.Hash) (staking.CandidateVersionQueue, error) {
	return plugin.StakingInstance().GetNodeVersion(blockHash)
}

func (api *MonitorAPI) GetRestrictingBalance(accounts []common.Address, blockHash common.Hash, blockNumber *big.Int) []restricting.BalanceResult {
	resposne := make([]restricting.BalanceResult, len(accounts))

	for idx, address := range accounts {
		result, err := plugin.RestrictingInstance().GetRestrictingBalance(address, monitor.statedb, blockHash, blockNumber.Uint64())
		if err != nil {
			log.Error("getRestrictingBalance err", "account:", address, "err", err)
			rb := restricting.BalanceResult{
				Account: address,
			}
			resposne[idx] = rb

		} else {
			resposne[idx] = result
		}
	}
	return resposne
}

func (api *MonitorAPI) GetProposalParticipants(proposalID, blockHash common.Hash) (accuVerifierAccount uint64, yeas uint64, nays uint64, abstentions uint64, err error) {
	proposal, err := gov.GetProposal(proposalID, monitor.statedb)
	if err != nil {
		return 0, 0, 0, 0, err
	} else if proposal == nil {
		return 0, 0, 0, 0, gov.ProposalNotFound
	}

	list, err := gov.ListAccuVerifier(blockHash, proposalID)
	if err != nil {
		return 0, 0, 0, 0, err
	}

	yeas, nays, abstentions, err = gov.TallyVoteValue(proposalID, blockHash)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	return uint64(len(list)), yeas, nays, abstentions, nil
}

func (api *MonitorAPI) GetPPosInvokeInfo(blockNumber *big.Int) (staking.TransBlockReturnQueue, error) {
	return plugin.StakingInstance().GetTransData(blockNumber.Uint64())
}
