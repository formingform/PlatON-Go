package monitor

import (
	"context"
	"github.com/PlatONnetwork/PlatON-Go/common"
	"github.com/PlatONnetwork/PlatON-Go/common/hexutil"
	"github.com/PlatONnetwork/PlatON-Go/core/state"
	"github.com/PlatONnetwork/PlatON-Go/core/types"
	"github.com/PlatONnetwork/PlatON-Go/log"
	"github.com/PlatONnetwork/PlatON-Go/rpc"
	"github.com/PlatONnetwork/PlatON-Go/x/gov"
	"github.com/PlatONnetwork/PlatON-Go/x/staking"
	"github.com/PlatONnetwork/PlatON-Go/x/xcom"
	"github.com/PlatONnetwork/PlatON-Go/x/xutil"
	"math/big"
	"strconv"
)

// 这个方式暂时未启用，需要scan-agent的改造，用rpc-call代替合约方法调用
type Backend interface {
	CurrentHeader() *types.Header
	CurrentBlock() *types.Block
	HeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Header, error)
	BlockByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Block, error)
	BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error)
	GetReceipts(ctx context.Context, blockHash common.Hash) (types.Receipts, error)
	StateAndHeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*state.StateDB, *types.Header, error)
	StateAndHeaderByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*state.StateDB, *types.Header, error)
	//Eth() *eth.Ethereum
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

// GetReceiptExtsByBlockNumber returns the transaction receipt extend info for the given block number.
func (api *MonitorAPI) GetReceiptExtsByBlockNumber(number rpc.BlockNumber) ([]map[string]interface{}, error) {
	blockNumber := uint64(number)
	if number == rpc.LatestBlockNumber {
		blockNumber = api.b.CurrentBlock().NumberU64()
	}
	log.Info("GetReceiptExtsByBlockNumber", "number", number.Int64(), "blockNumber", blockNumber)

	blockNr := rpc.BlockNumber(blockNumber)
	block, err := api.b.BlockByNumber(context.Background(), blockNr)
	if block == nil {
		return nil, err
	}

	queue := make([]map[string]interface{}, len(block.Transactions()))

	receipts, err := api.b.GetReceipts(context.Background(), block.Hash())
	if err != nil {
		log.Error("rpcGetTransactionByBlock, get receipt error", "receipts:", receipts)
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

		//var signer types.Signer = types.NewEIP155Signer(tx.ChainId())
		//from, _ := types.Sender(signer, tx)
		rb := types.ReceiptBlock{
			Logs: make([]*types.LogBlock, len(receipt.Logs)),
		}
		for logIndex, logsValue := range receipt.Logs {
			tb := &types.LogBlock{
				Address: logsValue.Address,
				Data:    hexutil.Encode(logsValue.Data),
				Index:   logsValue.Index,
				Removed: logsValue.Removed,
				Topics:  logsValue.Topics,
			}
			rb.Logs[logIndex] = tb
		}
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
			"logs":            rb.Logs,
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
		createdContractInfoList := MonitorInstance().GetCreatedContractInfoList(blockNumber, value.Hash())
		if nil == createdContractInfoList {
			fields["contractCreated"] = []*ContractInfo{}
		} else {
			fields["contractCreated"] = createdContractInfoList
		}

		// 把opSuicide操作码销毁的合约地址拿出来，并放入fields["contractSuicided"]
		suicidedContractInfoList := MonitorInstance().GetSuicidedContractInfoList(blockNumber, value.Hash())
		if nil == suicidedContractInfoList {
			fields["contractSuicided"] = []*ContractInfo{}
		} else {
			fields["contractSuicided"] = suicidedContractInfoList
		}

		// 把本交易发现的代理关系拿出来，放入proxyContract
		proxyPatternList := MonitorInstance().GetProxyPatternList(blockNumber, value.Hash())
		if nil == proxyPatternList {
			fields["proxyPatterns"] = []*ProxyPattern{}
		} else {
			fields["proxyPatterns"] = proxyPatternList
		}

		// 把交易中产生的隐式LAT转账返回（如果本身的交易是合约交易才有）
		embedTransferList := MonitorInstance().GetEmbedTransfer(blockNumber, value.Hash())
		if embedTransferList == nil {
			fields["embedTransfers"] = []*EmbedTransfer{}
		} else {
			fields["embedTransfers"] = embedTransferList
		}

		// 把交易中产生的隐式PPOS调用
		implicitPPOSTxList := MonitorInstance().GetImplicitPPOSTx(blockNumber, value.Hash())
		if implicitPPOSTxList == nil {
			fields["implicitPPOSTxs"] = []*ImplicitPPOSTx{}
		} else {
			fields["implicitPPOSTxs"] = implicitPPOSTxList
		}

		queue[key] = fields
	}
	return queue, nil
}

// GetVerifiersByBlockNumber 获取结算周期最后一个块高=blockNumber的验证人（201名单）列表
// 输入参数是上一个结算周期最后一个块高
func (api *MonitorAPI) GetVerifiersByBlockNumber(number rpc.BlockNumber) (*staking.ValidatorExQueue, error) {
	blockNumber := uint64(number)
	if number == rpc.LatestBlockNumber {
		blockNumber = api.b.CurrentBlock().NumberU64()
	}
	log.Info("GetVerifiersByBlockNumber", "number", number.Int64(), "blockNumber", blockNumber)

	// epoch starts from 1
	epoch := xutil.CalculateEpoch(blockNumber)
	dbKey := VerifiersOfEpochKey.String() + strconv.FormatUint(epoch, 10)
	log.Debug("GetVerifiersByBlockNumber", "blockNumber", blockNumber, "dbKey", dbKey)

	data, err := MonitorInstance().monitordb.Get([]byte(dbKey))
	if nil != err {
		log.Error("fail to GetVerifiersByBlockNumber", "blockNumber", blockNumber, "err", err)
		if err == ErrNotFound {
			return nil, nil
		}
		return nil, err
	}

	if len(data) == 0 { //len(nil)==0
		return nil, err
	}
	log.Debug("GetVerifiersByBlockNumber result", "blockNumber", blockNumber, "data:", string(data))

	var validatorExQueue staking.ValidatorExQueue
	common.ParseJson(data, &validatorExQueue)

	return &validatorExQueue, nil

}

// GetValidatorsByBlockNumber 获取共识周期最后一个块高=blockNumber的验证人（25名单）列表
// 输入参数上一个共识轮的最后一个块高
func (api *MonitorAPI) GetValidatorsByBlockNumber(number rpc.BlockNumber) (*staking.ValidatorExQueue, error) {
	blockNumber := uint64(number)
	if number == rpc.LatestBlockNumber {
		blockNumber = api.b.CurrentBlock().NumberU64()
	}
	log.Info("GetValidatorsByBlockNumber", "number", number.Int64(), "blockNumber", blockNumber)

	// epoch starts from 1
	round := uint64(0)
	if blockNumber != round {
		round = xutil.CalculateRound(blockNumber)
	}
	queryNumber := round * xutil.ConsensusSize()
	dbKey := ValidatorsOfEpochKey.String() + strconv.FormatUint(queryNumber, 10)
	log.Debug("GetValidatorsByBlockNumber", "blockNumber", blockNumber, "dbKey", dbKey)

	data, err := MonitorInstance().monitordb.Get([]byte(dbKey))
	if nil != err {
		log.Error("fail to GetValidatorsByBlockNumber", "blockNumber", blockNumber, "err", err)
		if err == ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	if len(data) == 0 { //len(nil)==0
		return nil, nil
	}

	log.Debug("GetValidatorsByBlockNumber result", "blockNumber", blockNumber, "data:", string(data))
	var validators staking.ValidatorExQueue
	common.ParseJson(data, &validators)
	return &validators, nil
}

// 输入的blockNumber是epoch的结束块高，或者是0块高
func (api *MonitorAPI) GetEpochInfoByBlockNumber(number rpc.BlockNumber) (*EpochView, error) {
	blockNumber := uint64(number)
	if number == rpc.LatestBlockNumber {
		blockNumber = api.b.CurrentBlock().NumberU64()
	}
	log.Info("GetEpochInfoByBlockNumber", "number", number.Int64(), "blockNumber", blockNumber)

	var epoch = uint64(1)
	if blockNumber > 0 {
		epoch = xutil.CalculateEpoch(blockNumber)
	}
	dbKey := EpochInfoKey.String() + "_" + strconv.FormatUint(epoch, 10)
	data, err := MonitorInstance().monitordb.Get([]byte(dbKey))
	if nil != err {
		log.Error("fail to GetEpochInfoByBlockNumber", "blockNumber", blockNumber, "epoch", epoch, "err", err)
		if err == ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	if len(data) == 0 { //len(nil)==0
		return nil, nil
	}
	log.Debug("GetEpochInfoByBlockNumber result", "blockNumber", blockNumber, "epoch", epoch, "data:", string(data))

	var view EpochView
	common.ParseJson(data, &view)

	if &view == nil {
		return nil, nil
	}

	if blockNumber == 0 {
		view.NextPackageReward = view.PackageReward
		view.NextStakingReward = view.StakingReward
		view.CurPackageReward = big.NewInt(0)
		view.CurStakingReward = big.NewInt(0)
		return &view, nil
	}

	view.CurPackageReward = view.PackageReward
	view.CurStakingReward = view.StakingReward
	view.NextPackageReward = common.Big0
	view.NextStakingReward = common.Big0

	nextDbKey := EpochInfoKey.String() + "_" + strconv.FormatUint(epoch+1, 10)
	nextData, nextErr := MonitorInstance().monitordb.Get([]byte(nextDbKey))
	if nil != nextErr {
		log.Error("fail to GetEpochInfoByBlockNumber", "blockNumber", blockNumber, "epoch", epoch+1, "err", err)
		if err == ErrNotFound {
			return nil, nil
		}
		return &view, nil
	}
	if len(nextData) > 0 { //len(nil)==0

		log.Debug("GetEpochInfoByBlockNumber result", "blockNumber", blockNumber, "nextEpoch", epoch+1, "nextData:", string(nextData))

		var nextView EpochView
		common.ParseJson(data, &nextView)

		view.NextPackageReward = nextView.PackageReward
		view.NextStakingReward = nextView.StakingReward
	}

	return &view, nil
}

// GetSlashInfoByBlockNumber 选举块高时，查询节点被处罚的信息。输入参数是选举块高
func (api *MonitorAPI) GetSlashInfoByBlockNumber(number rpc.BlockNumber) (*staking.SlashQueue, error) {
	blockNumber := uint64(number)
	if number == rpc.LatestBlockNumber {
		blockNumber = api.b.CurrentBlock().NumberU64()
	}
	log.Info("GetSlashInfoByBlockNumber", "number", number.Int64(), "blockNumber", blockNumber)

	dbKey := SlashKey.String() + "_" + strconv.FormatUint(blockNumber, 10)
	data, err := MonitorInstance().monitordb.Get([]byte(dbKey))
	if nil != err {
		if err == ErrNotFound {
			return nil, nil
		}
		return nil, err
	}

	if len(data) == 0 { //len(nil)==0
		return nil, nil
	}
	var slashQueue staking.SlashQueue
	common.ParseJson(data, &slashQueue)
	return &slashQueue, nil
}

// GetNodeVersion 链上获取当前的所有质押节点版本
func (api *MonitorAPI) GetNodeVersion() (staking.ValidatorExQueue, error) {
	return MonitorInstance().stakingPlugin.GetNodeVersion(api.b.CurrentHeader().Hash())
}

// GetAccountView 链上获取帐号的当前信息，包括：余额，锁仓，委托等
// monitor.getAccountView(["lat14ccm5gxvz7f43dpr809ylwnurj4cn4v24kklyg","lat17warrr67cwplfqpn6aqe9rw406lts54z4zwdzp"],"latest")
// monitor.getAccountView(["lat14ccm5gxvz7f43dpr809ylwnurj4cn4v24kklyg","lat17warrr67cwplfqpn6aqe9rw406lts54z4zwdzp"],4312)
func (api *MonitorAPI) GetAccountView(accounts []common.Address, number rpc.BlockNumber) []*AccountView {
	log.Info("GetAccountView", "accounts", common.ToJson(accounts), "number", number.Int64())

	response := make([]*AccountView, len(accounts))
	header, _ := api.b.HeaderByNumber(context.Background(), number) // latest header should always be available

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
	log.Info("GetProposalParticipants", "proposalID", proposalID.Hex(), "blockHash", blockHash.Hex())
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
