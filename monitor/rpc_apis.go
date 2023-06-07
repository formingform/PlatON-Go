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

// GetExtReceiptsByBlock returns the transaction receipt for the given block number.
func (api *MonitorAPI) GetExtReceipts(blockNumber uint64) ([]map[string]interface{}, error) {
	blockNr := rpc.BlockNumber(blockNumber)
	block, err := api.b.BlockByNumber(nil, blockNr)
	if block == nil {
		return nil, err
	}

	queue := make([]map[string]interface{}, len(block.Transactions()))

	receipts, err := api.b.GetReceipts(nil, block.Hash())
	if err != nil {
		log.Error("GetExtReceipts, get receipt error", "receipts:", receipts)
	}

	for key, value := range block.Transactions() {
		//tx, blockHash, blockNumber, index := rawdb.ReadTransaction(s.b.ChainDb(), value.Hash())
		//if tx == nil {
		//	log.Error("rpcGetTransactionByBlock, get tx error","blockHash:",blockHash,"blockNumber:",blockNumber,"index:",index)
		//	continue
		//}
		if len(receipts) <= int(key) {
			log.Error("rpcGetTransactionByBlock, get receipt length error", "receipts:", receipts, "index:", key)
			continue
		}
		receipt := receipts[key]

		fields := map[string]interface{}{
			//"blockHash":         blockHash,
			//"blockNumber":       hexutil.Uint64(blockNumber),
			"transactionHash":  value.Hash(),
			"transactionIndex": hexutil.Uint64(key),
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
		createdContractInfoList := MonitorInstance().GetCreatedContracts(block.NumberU64(), value.Hash())
		if nil == createdContractInfoList {
			fields["contractCreated"] = []*ContractInfo{}
		} else {
			fields["contractCreated"] = createdContractInfoList
		}

		// 把opSuicide操作码销毁的合约地址拿出来，并放入fields["contractSuicided"]
		suicidedContractInfoList := MonitorInstance().GetSuicidedContracts(block.NumberU64(), value.Hash())
		if nil == suicidedContractInfoList {
			fields["contractSuicided"] = []*ContractInfo{}
		} else {
			fields["contractSuicided"] = suicidedContractInfoList
		}

		// 把本交易发现的代理关系拿出来，放入proxyContract
		proxyPatternList := MonitorInstance().GetProxyPatterns(block.NumberU64(), value.Hash())
		if nil == proxyPatternList {
			fields["proxyPatterns"] = []*ProxyPattern{}
		} else {
			fields["proxyPatterns"] = proxyPatternList
		}

		// 把交易中产生的隐式LAT转账返回（如果本身的交易是合约交易才有）
		embedTransferList := MonitorInstance().GetUnusualTransferTx(blockNumber, value.Hash())
		if embedTransferList == nil {
			fields["embedTransfer"] = []*UnusualTransferTx{}
		} else {
			fields["embedTransfer"] = embedTransferList
		}
		queue[key] = fields
	}
	return queue, nil
}

func (api *MonitorAPI) GetHistoryVerifiers(blockNumber uint64) (staking.ValidatorExQueue, error) {
	// epoch starts from 1
	epoch := xutil.CalculateEpoch(blockNumber)
	//todo: 直接用epoch做key
	epochStartBlockNumber := (epoch - 1) * xutil.CalcBlocksEachEpoch()
	dbKey := VerifierKey.String() + strconv.FormatUint(epochStartBlockNumber, 10)

	data, err := MonitorInstance().monitordb.Get([]byte(dbKey))
	if nil != err && err != ErrNotFound {
		log.Error("failed to GetHistoryVerifiers", "blockNumber", blockNumber, "err", err)
		return nil, err
	}

	if len(data) == 0 { //len(nil)==0
		return nil, err
	}
	var validatorExQueue staking.ValidatorExQueue
	ParseJson(data, &validatorExQueue)

	return validatorExQueue, nil
}

func (api *MonitorAPI) GetHistoryValidators(blockNumber uint64) staking.ValidatorExQueue {
	// epoch starts from 1
	epoch := xutil.CalculateEpoch(blockNumber)

	epochStartBlockNumber := (epoch - 1) * xutil.CalcBlocksEachEpoch()
	dbKey := ValidatorKey.String() + strconv.FormatUint(epochStartBlockNumber, 10)

	data, err := MonitorInstance().monitordb.Get([]byte(dbKey))
	if nil != err && err != ErrNotFound {
		log.Error("failed to GetHistoryValidators", "blockNumber", blockNumber, "err", err)
		return nil
	}
	if len(data) == 0 { //len(nil)==0
		return nil
	}
	var validators staking.ValidatorExQueue
	ParseJson(data, &validators)
	return validators
}

func (api *MonitorAPI) GetHistoryEpochInfo(blockNumber uint64) *EpochView {
	epoch := xutil.CalculateEpoch(blockNumber)
	dbKey := EpochInfoKey.String() + "_" + strconv.FormatUint(epoch, 10)
	data, err := MonitorInstance().monitordb.Get([]byte(dbKey))
	if nil != err && err != ErrNotFound {
		log.Error("failed to GetHistoryEpochInfo", "blockNumber", blockNumber, "epoch", epoch, "err", err)
		return nil
	}
	if len(data) == 0 { //len(nil)==0
		return nil
	}
	var view *EpochView
	ParseJson(data, &view)

	if view == nil {
		return nil
	}

	if epoch > 2 {
		dbKey := EpochInfoKey.String() + "_" + strconv.FormatUint(epoch-1, 10)
		data, err := MonitorInstance().monitordb.Get([]byte(dbKey))
		if nil != err && err != ErrNotFound {
			log.Error("failed to GetHistoryEpochInfo", "blockNumber", blockNumber, "epoch", epoch-1, "err", err)
			return nil
		}
		if len(data) > 0 { //len(nil)==0
			var curView *EpochView
			ParseJson(data, curView)

			view.CurPackageReward = curView.PackageReward
			view.CurStakingReward = curView.StakingReward
		}
	}
	return view
}

func (api *MonitorAPI) GetSlashInfo(electionBlockNumber uint64) staking.SlashQueue {
	dbKey := SlashKey.String() + "_" + strconv.FormatUint(electionBlockNumber, 10)
	data, err := MonitorInstance().monitordb.Get([]byte(dbKey))
	if nil != err {
		return nil
	}
	var slashQueue staking.SlashQueue
	err = rlp.DecodeBytes(data, &slashQueue)
	if nil != err {
		return nil
	}
	return slashQueue
}

func (api *MonitorAPI) GetNodeVersion(blockHash common.Hash) (staking.ValidatorExQueue, error) {
	return MonitorInstance().stakingPlugin.GetNodeVersion(blockHash)
}

func (api *MonitorAPI) GetAccountView(accounts []common.Address, blockHash common.Hash, blockNumber uint64) []*AccountView {
	response := make([]*AccountView, len(accounts))
	for idx, address := range accounts {
		accountView, err := getAccountView(address, monitor.statedb, blockHash, blockNumber)
		if err != nil {
			log.Error("getRestrictingBalance err", "account:", address, "err", err)
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
		FreeBalance:                             (*hexutil.Big)(state.GetBalance(account)),
		RestrictingPlanLockedAmount:             (*hexutil.Big)(big.NewInt(0)),
		RestrictingPlanPledgeAmount:             (*hexutil.Big)(big.NewInt(0)),
		DelegationUnLockedFreeBalance:           (*hexutil.Big)(big.NewInt(0)),
		DelegationUnLockedRestrictingPlanAmount: (*hexutil.Big)(big.NewInt(0)),
		DelegationLockedItems:                   make([]DelegationLockedItem, 0),
	}
	// 设置锁仓金
	_, info, err := MonitorInstance().restrictingPlugin.MustGetRestrictingInfoByDecode(state, account)
	if err != nil {
		log.Error("failed to MustGetRestrictingInfoByDecode", "account", account.String(), "err", err)
		return nil, err
	}
	accountView.RestrictingPlanLockedAmount = (*hexutil.Big)(info.CachePlanAmount)
	accountView.RestrictingPlanPledgeAmount = (*hexutil.Big)(info.AdvanceAmount)

	// 设置委托锁定金
	locks, err2 := MonitorInstance().stakingPlugin.GetGetDelegationLockCompactInfo(blockHash, blockNumber, account)
	if err2 != nil {
		log.Error("failed to MustGetRestrictingInfoByDecode", "account", account.String(), "err", err2)
		return nil, err2
	}
	accountView.DelegationUnLockedFreeBalance = locks.Released
	accountView.DelegationUnLockedRestrictingPlanAmount = locks.RestrictingPlan
	for _, lock := range locks.Locks {
		lockItem := DelegationLockedItem{
			ExpiredEpoch:          lock.Epoch,
			FreeBalance:           lock.Released,
			RestrictingPlanAmount: lock.RestrictingPlan,
		}
		accountView.DelegationLockedItems = append(accountView.DelegationLockedItems, lockItem)
	}
	return accountView, nil
}

func (api *MonitorAPI) GetProposalParticipants(proposalID, blockHash common.Hash) (accuVerifierAccount, yeas, nays, abstentions uint64, err error) {
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

func (api *MonitorAPI) GetImplicitPPOSTx(blockNumber uint64) (*ImplicitPPOSTx, error) {
	log.Debug("GetImplicitPPOSTx", "blockNumber", blockNumber)
	dbKey := ImplicitPPOSTxKey.String() + "_" + strconv.FormatUint(blockNumber, 10)
	data, err := MonitorInstance().monitordb.Get([]byte(dbKey))
	if nil != err && err != ErrNotFound {
		log.Error("failed to load data from local db", "err", err)
		return nil, err
	}

	if len(data) == 0 { //len(nil)==0
		return nil, err
	}
	var implicitPPOSTx *ImplicitPPOSTx
	ParseJson(data, &implicitPPOSTx)
	return implicitPPOSTx, nil
}
