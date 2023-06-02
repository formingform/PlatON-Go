package monitor

import (
	"github.com/PlatONnetwork/PlatON-Go/common"
	"github.com/PlatONnetwork/PlatON-Go/log"
	"github.com/PlatONnetwork/PlatON-Go/rpc"
	"github.com/PlatONnetwork/PlatON-Go/x/gov"
	"github.com/PlatONnetwork/PlatON-Go/x/plugin"
	"github.com/PlatONnetwork/PlatON-Go/x/restricting"
	"github.com/PlatONnetwork/PlatON-Go/x/staking"
	"math/big"
)

// API defines an exposed API function interface.
type API interface {
	GetHistoryVerifierList(blockNumber *big.Int) (staking.ValidatorExQueue, error)
	GetHistoryValidatorList(blockNumber *big.Int) (staking.ValidatorExQueue, error)
	GetHistoryReward(blockNumber *big.Int) (staking.RewardReturn, error)
	GetHistoryLowRateSlashList(blockNumber *big.Int) (staking.SlashNodeQueue, error)
	GetNodeVersion(blockHash common.Hash) (staking.CandidateVersionQueue, error)
	GetRestrictingBalance(addresses []common.Address, blockHash common.Hash, blockNumber *big.Int) []restricting.BalanceResult
	GetProposalParticipants(proposalID, blockHash common.Hash) (accuVerifierAccount uint64, yeas uint64, nays uint64, abstentions uint64, err error)
	GetPPosInvokeInfo(blockNumber *big.Int) (staking.TransBlockReturnQueue, error)
}

type MonitorAPI struct {
	monitor API
}

func NewMonitorAPI(monitor API) *MonitorAPI {
	return &MonitorAPI{monitor: monitor}
}

// APIs returns a list of APIs provided by the consensus engine.
func (monitor *Monitor) APIs() []rpc.API {
	return []rpc.API{
		{
			Namespace: "monitor",
			Version:   "1.0",
			Service:   NewMonitorAPI(monitor),
			Public:    true,
		},
	}
}

func (monitor *Monitor) GetHistoryVerifierList(blockNumber *big.Int) (staking.ValidatorExQueue, error) {
	return plugin.StakingInstance().GetHistoryVerifierList(blockNumber.Uint64())
}

func (monitor *Monitor) GetHistoryValidatorList(blockNumber *big.Int) (staking.ValidatorExQueue, error) {
	return plugin.StakingInstance().GetHistoryValidatorList(blockNumber.Uint64())
}

func (monitor *Monitor) GetHistoryReward(blockNumber *big.Int) (staking.RewardReturn, error) {
	return plugin.StakingInstance().GetHistoryReward(blockNumber.Uint64())
}

func (monitor *Monitor) GetHistoryLowRateSlashList(blockNumber *big.Int) (staking.SlashNodeQueue, error) {
	return plugin.StakingInstance().GetSlashData(blockNumber.Uint64())
}

func (monitor *Monitor) GetNodeVersion(blockHash common.Hash) (staking.CandidateVersionQueue, error) {
	return plugin.StakingInstance().GetNodeVersion(blockHash)
}

func (monitor *Monitor) GetRestrictingBalance(accounts []common.Address, blockHash common.Hash, blockNumber *big.Int) []restricting.BalanceResult {
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

func (monitor *Monitor) GetProposalParticipants(proposalID, blockHash common.Hash) (accuVerifierAccount uint64, yeas uint64, nays uint64, abstentions uint64, err error) {
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

func (monitor *Monitor) GetPPosInvokeInfo(blockNumber *big.Int) (staking.TransBlockReturnQueue, error) {
	return plugin.StakingInstance().GetTransData(blockNumber.Uint64())
}
