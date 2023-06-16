package monitor

import (
	"context"
	"github.com/PlatONnetwork/AppChain-Go/common"
	"github.com/PlatONnetwork/AppChain-Go/common/hexutil"
	"github.com/PlatONnetwork/AppChain-Go/core/types"
	"github.com/PlatONnetwork/AppChain-Go/log"
	"github.com/PlatONnetwork/AppChain-Go/rlp"
	"github.com/PlatONnetwork/AppChain-Go/rpc"
	"github.com/PlatONnetwork/AppChain-Go/x/gov"
	"github.com/PlatONnetwork/AppChain-Go/x/staking"
	"github.com/PlatONnetwork/AppChain-Go/x/xcom"
	"github.com/PlatONnetwork/AppChain-Go/x/xutil"
	"math/big"
	"strconv"
)

// API defines an exposed API function interface.
/*type API interface {
	GetExtReceipts(blockNumber *big.Int) ([]map[string]interface{}, error)
	GetHistoryVerifierList(blockNumber *big.Int) (staking.ValidatorExQueue, error)
	GetHistoryValidatorList(blockNumber *big.Int) (staking.ValidatorExQueue, error)
	GetHistoryReward(blockNumber *big.Int) (staking.RewardReturn, error)
	GetHistoryLowRateSlashList(blockNumber *big.Int) (staking.SlashNodeQueue, error)
	GetNodeVersion(blockHash common.Hash) (staking.CandidateVersionQueue, error)
	GetRestrictingBalance(addresses []common.Address, blockHash common.Hash, blockNumber *big.Int) []restricting.BalanceResult
	GetProposalParticipants(proposalID, blockHash common.Hash) (accuVerifierAccount, yeas, nays, abstentions uint64, err error)
	GetImplicitPPOSTx(blockNumber *big.Int) (*ImplicitPPOSTx, error)
}*/

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

// GetReceiptExtsByBlockNumber returns the transaction receipt for the given block number.
func (api *MonitorAPI) GetReceiptExtsByBlockNumber(blockNumber uint64) ([]map[string]interface{}, error) {
	blockNr := rpc.BlockNumber(blockNumber)
	block, err := api.b.BlockByNumber(nil, blockNr)
	if block == nil || err != nil {
		return nil, err
	}

	queue := make([]map[string]interface{}, len(block.Transactions()))

	receipts, err := api.b.GetReceipts(nil, block.Hash())
	if err != nil {
		log.Error("GetExtReceipts, get receipt error", "receipts:", receipts)
		return nil, err
	}

	for idx, tx := range block.Transactions() {
		//tx, blockHash, blockNumber, index := rawdb.ReadTransaction(s.b.ChainDb(), value.Hash())
		//if tx == nil {
		//	log.Error("rpcGetTransactionByBlock, get tx error","blockHash:",blockHash,"blockNumber:",blockNumber,"index:",index)
		//	continue
		//}
		if len(receipts) <= int(idx) {
			log.Error("fail to GetReceiptExtsByBlockNumber", "receipts:", receipts, "index:", idx)
			continue
		}
		receipt := receipts[idx]

		fields := map[string]interface{}{
			//"blockHash":         blockHash,
			//"blockNumber":       hexutil.Uint64(blockNumber),
			"transactionHash":  tx.Hash(),
			"transactionIndex": hexutil.Uint64(idx),
			//"from":              from,
			//"to":                tx.To(),
			"gasUsed": hexutil.Uint64(receipt.GasUsed),
			//"cumulativeGasUsed": hexutil.Uint64(receipt.CumulativeGasUsed),
			"contractAddress": nil,
			"logs":            receipt.Logs,
			//"logsBloom":         receipt.Bloom,
		}

		// Assign receipt status or post state.
		if len(receipt.PostState) > 0 {
			fields["root"] = hexutil.Bytes(receipt.PostState)
		} else {
			fields["status"] = hexutil.Uint(receipt.Status)
		}
		if receipt.Logs == nil {
			fields["logs"] = [][]*types.Log{}
		}
		// If the ContractAddress is 20 0x0 bytes, assume it is not a contract creation
		if receipt.ContractAddress != (common.Address{}) {
			fields["contractAddress"] = receipt.ContractAddress
		}

		// 把tx.to==nil/opCreate/opCreate2操作码3种方式建的合约地址拿出来
		createdContractInfoList := MonitorInstance().GetCreatedContracts(tx.Hash())
		if nil == createdContractInfoList {
			fields["contractCreated"] = []*ContractInfo{}
		} else {
			fields["contractCreated"] = createdContractInfoList
		}

		// 把opSuicide操作码销毁的合约地址拿出来，并放入fields["contractSuicided"]
		suicidedContractInfoList := MonitorInstance().GetSuicidedContracts(block.NumberU64(), tx.Hash())
		if nil == suicidedContractInfoList {
			fields["contractSuicided"] = []*ContractInfo{}
		} else {
			fields["contractSuicided"] = suicidedContractInfoList
		}

		// 把本交易发现的代理关系拿出来，放入proxyContract
		proxyPatternList := MonitorInstance().GetProxyPatterns(block.NumberU64(), tx.Hash())
		if nil == proxyPatternList {
			fields["proxyPatterns"] = []*ProxyPattern{}
		} else {
			fields["proxyPatterns"] = proxyPatternList
		}

		// 把交易中产生的非常规原生代币转账交易返回（原始交易是合约调用，才会产生非常规转账）
		unusualTransferList := MonitorInstance().GetUnusualTransfer(blockNumber, tx.Hash())
		if unusualTransferList == nil {
			fields["unusualTransfer"] = []*UnusualTransfer{}
		} else {
			fields["unusualTransfer"] = unusualTransferList
		}
		queue[idx] = fields
	}
	return queue, nil
}

func (api *MonitorAPI) GetVerifiersByBlockNumber(blockNumber uint64) (*staking.ValidatorExQueue, error) {
	// epoch starts from 1
	epoch := xutil.CalculateEpoch(blockNumber)
	dbKey := VerifiersOfEpochKey.String() + strconv.FormatUint(epoch, 10)

	data, err := MonitorInstance().monitordb.Get([]byte(dbKey))
	if nil != err {
		log.Error("fail to GetVerifiersByBlockNumber", "blockNumber", blockNumber, "err", err)
		return nil, err
	}

	if len(data) == 0 { //len(nil)==0
		return nil, err
	}
	var validatorExQueue staking.ValidatorExQueue
	ParseJson(data, &validatorExQueue)

	return &validatorExQueue, nil
}

func (api *MonitorAPI) GetValidatorsByBlockNumber(blockNumber uint64) (*staking.ValidatorExQueue, error) {
	// epoch starts from 1
	epoch := xutil.CalculateEpoch(blockNumber)
	dbKey := ValidatorsOfEpochKey.String() + strconv.FormatUint(epoch, 10)

	data, err := MonitorInstance().monitordb.Get([]byte(dbKey))
	if nil != err {
		log.Error("fail to GetValidatorsByBlockNumber", "blockNumber", blockNumber, "err", err)
		return nil, err
	}
	if len(data) == 0 { //len(nil)==0
		return nil, nil
	}
	var validators staking.ValidatorExQueue
	ParseJson(data, &validators)
	return &validators, nil
}

func (api *MonitorAPI) GetEpochInfoByBlockNumber(blockNumber uint64) (*EpochView, error) {
	epoch := xutil.CalculateEpoch(blockNumber)
	dbKey := EpochInfoKey.String() + "_" + strconv.FormatUint(epoch, 10)
	data, err := MonitorInstance().monitordb.Get([]byte(dbKey))
	if nil != err {
		log.Error("fail to GetEpochInfoByBlockNumber", "blockNumber", blockNumber, "epoch", epoch, "err", err)
		return nil, err
	}
	if len(data) == 0 { //len(nil)==0
		return nil, nil
	}
	var view EpochView
	ParseJson(data, &view)

	if &view == nil {
		return nil, nil
	}

	view.NextPackageReward = common.Big0
	view.NextStakingReward = common.Big0

	nextDbKey := EpochInfoKey.String() + "_" + strconv.FormatUint(epoch+1, 10)
	nextData, nextErr := MonitorInstance().monitordb.Get([]byte(nextDbKey))
	if nil != nextErr {
		log.Error("fail to GetEpochInfoByBlockNumber", "blockNumber", blockNumber, "epoch", epoch+1, "err", err)
		return &view, nil
	}
	if len(nextData) > 0 { //len(nil)==0
		var nextView *EpochView
		ParseJson(data, nextView)

		view.NextPackageReward = nextView.PackageReward
		view.NextStakingReward = nextView.StakingReward
	}

	return &view, nil
}

func (api *MonitorAPI) GetSlashInfoByBlockNumber(electionBlockNumber uint64) (*staking.SlashQueue, error) {
	dbKey := SlashKey.String() + "_" + strconv.FormatUint(electionBlockNumber, 10)
	data, err := MonitorInstance().monitordb.Get([]byte(dbKey))
	if nil != err {
		return nil, err
	}
	var slashQueue staking.SlashQueue
	err = rlp.DecodeBytes(data, &slashQueue)
	if nil != err {
		return nil, err
	}
	return &slashQueue, nil
}

// GetNodeVersion 链上获取当前的所有质押节点版本
func (api *MonitorAPI) GetNodeVersion() (staking.ValidatorExQueue, error) {
	return MonitorInstance().stakingPlugin.GetNodeVersion(api.b.CurrentHeader().Hash())
}

// GetAccountView 链上获取帐号的当前信息，包括：余额，锁仓，委托等
func (api *MonitorAPI) GetAccountView(accounts []common.Address) []*AccountView {
	response := make([]*AccountView, len(accounts))
	header, _ := api.b.HeaderByNumber(context.Background(), rpc.LatestBlockNumber) // latest header should always be available

	for idx, address := range accounts {
		accountView, err := getAccountView(address, monitor.statedb, header.Hash(), header.Number.Uint64())
		if err != nil {
			log.Error("fail to GetAccountView", "account:", address, "err", err)
			rb := &AccountView{
				Account: address,
			}
			response[idx] = rb

		} else {
			response[idx] = accountView
		}
	}
	return response
}

func getAccountView(account common.Address, state xcom.StateDB, blockHash common.Hash, blockNumber uint64) (*AccountView, error) {
	accountView := &AccountView{
		Account:                                 account,
		FreeBalance:                             state.GetBalance(account),
		RestrictingPlanLockedAmount:             big.NewInt(0),
		RestrictingPlanPledgeAmount:             big.NewInt(0),
		DelegationUnLockedFreeBalance:           big.NewInt(0),
		DelegationUnLockedRestrictingPlanAmount: big.NewInt(0),
		DelegationLockedItems:                   make([]DelegationLockedItem, 0),
	}
	// 设置锁仓金
	_, restrictingInfo, err := MonitorInstance().restrictingPlugin.MustGetRestrictingInfoByDecode(state, account)
	if err == nil && &restrictingInfo != nil {
		accountView.RestrictingPlanLockedAmount = restrictingInfo.CachePlanAmount
		accountView.RestrictingPlanPledgeAmount = restrictingInfo.AdvanceAmount
	}

	// 设置委托锁定金
	delegationLocks, err2 := MonitorInstance().stakingPlugin.GetGetDelegationLockCompactInfo(blockHash, blockNumber, account)
	if err2 == nil && delegationLocks != nil {
		accountView.DelegationUnLockedFreeBalance = delegationLocks.Released.ToInt()
		accountView.DelegationUnLockedRestrictingPlanAmount = delegationLocks.RestrictingPlan.ToInt()
		for _, lock := range delegationLocks.Locks {
			lockItem := DelegationLockedItem{
				ExpiredEpoch:          lock.Epoch,
				FreeBalance:           lock.Released.ToInt(),
				RestrictingPlanAmount: lock.RestrictingPlan.ToInt(),
			}
			accountView.DelegationLockedItems = append(accountView.DelegationLockedItems, lockItem)
		}
	}
	return accountView, nil
}

// GetProposalParticipants 获取提案到此区块为止的投票情况，包括：累计投票人数，赞成、反对，弃权的人数
func (api *MonitorAPI) GetProposalParticipants(proposalID, blockHash common.Hash) (*ProposalParticipants, error) {
	proposalParticipants := &ProposalParticipants{0, 0, 0, 0}
	proposal, err := gov.GetProposal(proposalID, monitor.statedb)
	if err != nil {
		return proposalParticipants, err
	} else if proposal == nil {
		return proposalParticipants, gov.ProposalNotFound
	}

	list, err := gov.ListAccuVerifier(blockHash, proposalID)
	if err != nil {
		return proposalParticipants, err
	}
	proposalParticipants.AccuVerifierAccount = uint64(len(list))
	yeas, nays, abstentions, err := gov.TallyVoteValue(proposalID, blockHash)
	if err != nil {
		return proposalParticipants, err
	}
	proposalParticipants.Yeas = yeas
	proposalParticipants.Nays = nays
	proposalParticipants.Abstentions = abstentions
	return proposalParticipants, nil
}

// GetImplicitPPOSTxsByBlockNumber
func (api *MonitorAPI) GetImplicitPPOSTxsByBlockNumber(blockNumber uint64) (*ImplicitPPOSTx, error) {
	log.Debug("GetImplicitPPOSTxsByBlockNumber", "blockNumber", blockNumber)
	dbKey := ImplicitPPOSTxKey.String() + "_" + strconv.FormatUint(blockNumber, 10)
	data, err := MonitorInstance().monitordb.Get([]byte(dbKey))
	if nil != err {
		log.Error("fail to GetImplicitPPOSTxsByBlockNumber", "err", err)
		return nil, err
	}

	if len(data) == 0 { //len(nil)==0
		return nil, nil
	}
	var implicitPPOSTx ImplicitPPOSTx
	ParseJson(data, &implicitPPOSTx)
	return &implicitPPOSTx, nil
}
