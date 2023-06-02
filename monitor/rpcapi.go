package monitor

import (
	"encoding/hex"
	"github.com/PlatONnetwork/AppChain-Go/common"
	"github.com/PlatONnetwork/AppChain-Go/common/hexutil"
	"github.com/PlatONnetwork/AppChain-Go/log"
	"github.com/PlatONnetwork/AppChain-Go/rlp"
	"github.com/PlatONnetwork/AppChain-Go/rpc"
	"github.com/PlatONnetwork/AppChain-Go/x/gov"
	"github.com/PlatONnetwork/AppChain-Go/x/plugin"
	"github.com/PlatONnetwork/AppChain-Go/x/restricting"
	"github.com/PlatONnetwork/AppChain-Go/x/staking"
	"github.com/PlatONnetwork/AppChain-Go/x/xcom"
	"github.com/PlatONnetwork/AppChain-Go/x/xutil"
	"math/big"
	"strconv"
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
func (m *Monitor) APIs() []rpc.API {
	return []rpc.API{
		{
			Namespace: "monitor",
			Version:   "1.0",
			Service:   NewMonitorAPI(monitor),
			Public:    true,
		},
	}
}

func (m *Monitor) GetHistoryVerifierList(blockNumber *big.Int) (staking.ValidatorExQueue, error) {
	i := uint64(0)
	if blockNumber.Uint64() != i {
		i = xutil.CalculateEpoch(blockNumber.Uint64())
	}

	queryNumber := i * xutil.CalcBlocksEachEpoch()
	numStr := strconv.FormatUint(queryNumber, 10)
	log.Debug("wow,GetHistoryVerifierList query number:", "num string", numStr)
	data, err := STAKING_DB.HistoryDB.Get([]byte(VerifierName + numStr))
	if nil != err {
		return nil, err
	}
	var verifierList staking.ValidatorArraySave
	err = rlp.DecodeBytes(data, &verifierList)
	if nil != err {
		return nil, err
	}
	log.Debug("wow,GetHistoryVerifierList", verifierList)

	queue := make(staking.ValidatorExQueue, len(verifierList.Arr))

	var candidateHexQueue staking.CandidateHexQueue

	if queryNumber == 0 {
		data, err := STAKING_DB.HistoryDB.Get([]byte(InitNodeName + numStr))
		if nil != err {
			return nil, err
		}

		err = rlp.DecodeBytes(data, &candidateHexQueue)
		if nil != err {
			return nil, err
		}
		log.Debug("wow,GetHistoryVerifierList candidateHexQueue", candidateHexQueue)
	}
	for i, v := range verifierList.Arr {

		valEx := &staking.ValidatorEx{
			NodeId:              v.NodeId,
			ValidatorTerm:       v.ValidatorTerm,
			DelegateRewardTotal: (*hexutil.Big)(v.DelegateRewardTotal),
			DelegateTotal:       (*hexutil.Big)(v.DelegateTotal),
			StakingBlockNum:     v.StakingBlockNum,
		}
		if queryNumber == 0 {
			for _, vc := range candidateHexQueue {
				if vc.NodeId == v.NodeId {
					valEx.BenefitAddress = vc.BenefitAddress
					valEx.StakingAddress = vc.StakingAddress
					valEx.Website = vc.Website
					valEx.Description = vc.Description
					valEx.ExternalId = vc.ExternalId
					valEx.NodeName = vc.NodeName
					break
				}
			}
		}
		queue[i] = valEx
	}

	return queue, nil
}

func (m *Monitor) GetHistoryValidatorList(blockNumber *big.Int) (staking.ValidatorExQueue, error) {
	i := uint64(0)
	if blockNumber.Uint64() != i {
		i = xutil.CalculateRound(blockNumber.Uint64())
	}
	queryNumber := i * xutil.ConsensusSize()
	numStr := strconv.FormatUint(queryNumber, 10)
	log.Debug("wow,GetHistoryValidatorList query number:", "num string", numStr)
	data, err := STAKING_DB.HistoryDB.Get([]byte(ValidatorName + numStr))
	if nil != err {
		return nil, err
	}
	var validatorArr staking.ValidatorArraySave
	err = rlp.DecodeBytes(data, &validatorArr)
	if nil != err {
		return nil, err
	}
	log.Debug("wow,GetHistoryValidatorList", validatorArr)
	queue := make(staking.ValidatorExQueue, len(validatorArr.Arr))
	var candidateHexQueue staking.CandidateHexQueue

	if queryNumber == 0 {
		data, err := STAKING_DB.HistoryDB.Get([]byte(InitNodeName + numStr))
		if nil != err {
			return nil, err
		}

		err = rlp.DecodeBytes(data, &candidateHexQueue)
		if nil != err {
			return nil, err
		}
		log.Debug("wow,GetHistoryValidatorList candidateHexQueue", candidateHexQueue)
	}
	for i, v := range validatorArr.Arr {

		valEx := &staking.ValidatorEx{
			NodeId:              v.NodeId,
			ValidatorTerm:       v.ValidatorTerm,
			DelegateRewardTotal: (*hexutil.Big)(v.DelegateRewardTotal),
		}
		if queryNumber == 0 {
			for _, vc := range candidateHexQueue {
				if vc.NodeId == v.NodeId {
					valEx.BenefitAddress = vc.BenefitAddress
					valEx.StakingAddress = vc.StakingAddress
					valEx.Website = vc.Website
					valEx.Description = vc.Description
					valEx.ExternalId = vc.ExternalId
					valEx.NodeName = vc.NodeName
					break
				}
			}
		}
		queue[i] = valEx
	}
	return queue, nil
}

func (m *Monitor) GetHistoryReward(blockNumber *big.Int) (staking.RewardReturn, error) {
	i := uint64(0)
	if blockNumber.Uint64() != i {
		i = xutil.CalculateEpoch(blockNumber.Uint64())
	}

	queryNumber := i * xutil.CalcBlocksEachEpoch()
	numStr := strconv.FormatUint(queryNumber, 10)
	log.Debug("wow,GetHistoryReward query number:", "num string", numStr)
	data, err := STAKING_DB.HistoryDB.Get([]byte(RewardName + numStr))
	var reward staking.Reward
	var rewardReturn staking.RewardReturn
	if nil != err {
		return rewardReturn, err
	}

	err = rlp.DecodeBytes(data, &reward)
	if nil != err {
		return rewardReturn, err
	}
	log.Debug("wow,GetHistoryReward reward:", "PackageReward", reward.PackageReward, "StakingReward", reward.StakingReward)

	// 查询当前结算周期出块及质押经理
	curPackageReward := big.NewInt(0)
	curStakingReward := big.NewInt(0)
	if i > 0 {
		numStr := strconv.FormatUint(queryNumber-xutil.CalcBlocksEachEpoch(), 10)
		log.Debug("wow,GetCurHistoryReward query number:", "num string", numStr)
		data, err := STAKING_DB.HistoryDB.Get([]byte(RewardName + numStr))
		var curReward staking.Reward
		if nil != err {
			return rewardReturn, err
		}

		err = rlp.DecodeBytes(data, &curReward)
		if nil != err {
			return rewardReturn, err
		}
		log.Debug("wow,GetCurHistoryReward reward:", "PackageReward", curReward.PackageReward, "StakingReward", curReward.StakingReward)

		curPackageReward = curReward.PackageReward
		curStakingReward = curReward.StakingReward
	}

	rewardReturn = staking.RewardReturn{
		CurPackageReward:  (*hexutil.Big)(curPackageReward),
		CurStakingReward:  (*hexutil.Big)(curStakingReward),
		NextPackageReward: (*hexutil.Big)(reward.PackageReward),
		NextStakingReward: (*hexutil.Big)(reward.StakingReward),
		PackageReward:     (*hexutil.Big)(reward.PackageReward),
		StakingReward:     (*hexutil.Big)(reward.StakingReward),
		YearNum:           reward.YearNum,
		YearStartNum:      reward.YearStartNum,
		YearEndNum:        reward.YearEndNum,
		RemainEpoch:       reward.RemainEpoch,
		AvgPackTime:       reward.AvgPackTime,
	}
	log.Debug("wow,GetHistoryReward rewardReturn:", "PackageReward", rewardReturn.PackageReward, "StakingReward", rewardReturn.StakingReward)
	log.Debug("wow,GetHistoryReward", rewardReturn)

	return rewardReturn, nil
}

func (m *Monitor) GetHistoryLowRateSlashList(blockNumber *big.Int) (staking.SlashNodeQueue, error) {
	numStr := strconv.FormatUint(blockNumber.Uint64(), 10)
	log.Debug("wow,GetSlashData query number:", "num string", numStr)
	data, err := STAKING_DB.HistoryDB.Get([]byte(SlashName + numStr))
	if nil != err {
		return nil, err
	}
	var slashNodeQueue staking.SlashNodeQueue
	err = rlp.DecodeBytes(data, &slashNodeQueue)
	if nil != err {
		return nil, err
	}
	snq := make(staking.SlashNodeQueue, len(slashNodeQueue))
	for i, v := range slashNodeQueue {
		snq[i] = &staking.SlashNodeData{
			NodeId: v.NodeId,
			Amount: v.Amount,
		}
	}
	log.Debug("wow,GetSlashData", "snq", snq)
	return snq, nil
}

func (monitor *Monitor) GetNodeVersion(blockHash common.Hash) (staking.CandidateVersionQueue, error) {
	iter := plugin.StakingInstance().GetStakingDB().IteratorCandidatePowerByBlockHash(blockHash, 0)
	if err := iter.Error(); nil != err {
		return nil, err
	}
	defer iter.Release()

	queue := make(staking.CandidateVersionQueue, 0)

	count := 0

	for iter.Valid(); iter.Next(); {

		count++

		log.Debug("GetNodeVersion: iter", "key", hex.EncodeToString(iter.Key()))

		addrSuffix := iter.Value()
		can, err := plugin.StakingInstance().GetStakingDB().GetCandidateStoreWithSuffix(blockHash, addrSuffix)
		if nil != err {
			return nil, err
		}

		canVersion := &staking.CandidateVersion{
			NodeId:         can.NodeId,
			ProgramVersion: can.ProgramVersion,
		}
		queue = append(queue, canVersion)
	}
	log.Debug("GetNodeVersion: loop count", "count", count)

	return queue, nil
}

func (m *Monitor) GetRestrictingBalance(accounts []common.Address, blockHash common.Hash, blockNumber *big.Int) []restricting.BalanceResult {
	resposne := make([]restricting.BalanceResult, len(accounts))

	for idx, address := range accounts {
		result, err := getRestrictingBalance(address, monitor.statedb, blockHash, blockNumber.Uint64())
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

func getRestrictingBalance(account common.Address, state xcom.StateDB, blockHash common.Hash, blockNumber uint64) (restricting.BalanceResult, error) {

	log.Debug("begin to GetRestrictingBalance", "account", account.String())

	var (
		result restricting.BalanceResult
	)
	result.Account = account
	result.FreeBalance = (*hexutil.Big)(state.GetBalance(account))
	result.LockBalance = (*hexutil.Big)(big.NewInt(0))
	result.PledgeBalance = (*hexutil.Big)(big.NewInt(0))
	result.DLFreeBalance = (*hexutil.Big)(big.NewInt(0))
	result.DLRestrictingBalance = (*hexutil.Big)(big.NewInt(0))
	result.Locks = make([]restricting.DelegationLockPeriodResult, 0)
	// 设置锁仓金
	_, info, err := plugin.RestrictingInstance().MustGetRestrictingInfoByDecode(state, account)
	if err == nil {
		result.LockBalance = (*hexutil.Big)(info.CachePlanAmount)
		result.PledgeBalance = (*hexutil.Big)(info.AdvanceAmount)
	}

	// 设置委托锁定金
	if gov.Gte130VersionState(state) {
		var (
			dLock  restricting.DelegationLockPeriodResult
			dLocks []restricting.DelegationLockPeriodResult
		)
		locks, err := plugin.StakingInstance().GetGetDelegationLockCompactInfo(blockHash, blockNumber, account)

		if err == nil {
			result.DLFreeBalance = locks.Released
			result.DLRestrictingBalance = locks.RestrictingPlan
			for _, lock := range locks.Locks {
				dLock.Epoch = lock.Epoch
				dLock.Released = lock.Released
				dLock.RestrictingPlan = lock.RestrictingPlan
				dLocks = append(dLocks, dLock)
			}

			if len(dLocks) > 0 {
				result.Locks = dLocks
			}
		}
		log.Debug("end to GetRestrictingBalance", "locks", locks)
	}
	log.Debug("end to GetRestrictingBalance", "GetRestrictingBalance", result)

	return result, nil
}

func (m *Monitor) GetProposalParticipants(proposalID, blockHash common.Hash) (accuVerifierAccount uint64, yeas uint64, nays uint64, abstentions uint64, err error) {
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

func (m *Monitor) GetPPosInvokeInfo(blockNumber *big.Int) (staking.TransBlockReturnQueue, error) {
	numStr := strconv.FormatUint(blockNumber.Uint64(), 10)
	log.Debug("wow,GetTransData query number:", "num string", numStr)
	blockKey := TransBlockName + numStr
	transKey := TransHashName + numStr

	blockData, err := STAKING_DB.HistoryDB.Get([]byte(blockKey))
	if nil != err {
		return nil, err
	}
	var transBlock staking.TransBlock
	err = rlp.DecodeBytes(blockData, &transBlock)
	if nil != err {
		return nil, err
	}

	transDataQuene := make(staking.TransBlockReturnQueue, len(transBlock.TransHashStr))

	for i, v := range transBlock.TransHashStr {

		transInputBytes, err := STAKING_DB.HistoryDB.Get([]byte(transKey + v))
		if nil != err {
			log.Error("get transData error", err)
			continue
		}
		var transInput staking.TransInput
		err = rlp.DecodeBytes(transInputBytes, &transInput)
		if nil != err {
			return nil, err
		}
		transDataQuene[i] = &staking.TransBlockReturn{
			TxHash:     v,
			From:       common.BytesToAddress(transInput.From),
			To:         common.BytesToAddress(transInput.To),
			TransDatas: transInput.TransDatas,
		}
	}
	log.Debug("wow,GetTransData", "transDataQuene", transDataQuene)
	return transDataQuene, nil
}
