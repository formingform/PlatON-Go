package monitor

import (
	"github.com/PlatONnetwork/AppChain-Go/common"
	"github.com/PlatONnetwork/AppChain-Go/core/state"
	"github.com/PlatONnetwork/AppChain-Go/core/types"
	"github.com/PlatONnetwork/AppChain-Go/log"
	"github.com/PlatONnetwork/AppChain-Go/p2p/discover"
	"github.com/PlatONnetwork/AppChain-Go/rlp"
	"github.com/PlatONnetwork/AppChain-Go/x/plugin"
	"github.com/PlatONnetwork/AppChain-Go/x/staking"
	"github.com/PlatONnetwork/AppChain-Go/x/xcom"
	"github.com/PlatONnetwork/AppChain-Go/x/xutil"
	"math"
	"math/big"
	"strconv"
	"sync"
)

type MonitorDbKey int

const (
	EmbedTransferKey MonitorDbKey = iota
	CreatedContractKey
	SuicidedContractKey
	ProxyPatternKey
	proxyPatternMapKey
	ValidatorKey
	VerifierKey
	RewardKey
	YearKey
	InitNodeKey
	SlashKey
	TransBlockKey
	TransHashKey
)

type Monitor struct {
	statedb   *state.StateDB
	monitordb *monitorDB
}

var (
	onceMonitor sync.Once
	monitor     *Monitor
)

func InitMonitor(statedb *state.StateDB) {
	onceMonitor.Do(func() {
		if levelDB, err := openLevelDB(16, 500); err != nil {
			log.Crit("init monitor db fail", "err", err)
		} else {
			dbInstance := &monitorDB{path: dbFullPath, levelDB: levelDB, closed: false}
			monitor = &Monitor{statedb: statedb, monitordb: dbInstance}
		}

	})
}

func MonitorInstance() *Monitor {
	return monitor
}

// 定义 MonitorDbKey 类型的方法 String(), 返回字符串。
func (dbKey MonitorDbKey) String() string {
	return [...]string{"EmbedTransferTx", "CreatedContractKey", "SuicidedContractKey", "ProxyPatternKey", "proxyPatternMapKey"}[dbKey]
}

type EmbedTransfer struct {
	TxHash common.Hash    `json:"txHash,-"`
	From   common.Address `json:"from,omitempty"`
	To     common.Address `json:"to,omitempty"`
	Amount *big.Int       `json:"amount,omitempty"`
}

type ProxyPattern struct {
	Proxy          *ContractInfo `json:"proxy,omitempty"`
	Implementation *ContractInfo `json:"implementation,omitempty"`
}

func (m *Monitor) CollectEmbedTransfer(blockNumber uint64, txHash common.Hash, from, to common.Address, amount *big.Int) {
	log.Debug("CollectEmbedTransferTx", "blockNumber", blockNumber, "txHash", txHash.Hex(), "from", from.Bech32(), "to", to.Bech32(), "amount", amount)

	dbKey := EmbedTransferKey.String() + "_" + txHash.String()
	data, err := m.monitordb.Get([]byte(dbKey))
	if nil != err && err != ErrNotFound {
		log.Error("failed to load embed transfers", "err", err)
		return
	}

	var embedTransferList []*EmbedTransfer
	parseJson(data, &embedTransferList)

	embedTransfer := new(EmbedTransfer)
	embedTransfer.TxHash = txHash
	embedTransfer.From = from
	embedTransfer.To = to
	embedTransfer.Amount = amount

	embedTransferList = append(embedTransferList, embedTransfer)

	json := toJson(embedTransferList)
	if len(json) > 0 {
		m.monitordb.Put([]byte(dbKey), json)
		log.Debug("save embed transfers success")
	}

}

func (m *Monitor) GetEmbedTransfer(blockNumber uint64, txHash common.Hash) []*EmbedTransfer {
	log.Debug("GetEmbedTransfer", "blockNumber", blockNumber, "txHash", txHash.Hex())

	dbKey := EmbedTransferKey.String() + "_" + txHash.String()
	data, err := m.monitordb.Get([]byte(dbKey))
	if nil != err {
		log.Error("failed to load embed transfers", "err", err)
		return nil
	}

	var embedTransferList []*EmbedTransfer
	parseJson(data, &embedTransferList)
	return embedTransferList
}

type ContractRef interface {
	Address() common.Address
}

func (m *Monitor) CollectCreatedContractInfo(txHash common.Hash, contractInfo *ContractInfo) {
	log.Debug("CollectCreatedContractInfo", "txHash", txHash.Hex(), "contractInfo", string(toJson(contractInfo)))

	dbKey := CreatedContractKey.String() + "_" + txHash.String()
	data, err := m.monitordb.Get([]byte(dbKey))
	if nil != err && err != ErrNotFound {
		log.Error("failed to load created contracts", "err", err)
		return
	}
	var createdContractInfoList []*ContractInfo
	parseJson(data, &createdContractInfoList)

	createdContractInfoList = append(createdContractInfoList, contractInfo)

	json := toJson(createdContractInfoList)
	if len(json) > 0 {
		m.monitordb.Put([]byte(dbKey), json)
		log.Debug("save created contracts success")
	}

}

func (m *Monitor) GetCreatedContractInfoList(blockNumber uint64, txHash common.Hash) []*ContractInfo {
	log.Debug("GetCreatedContract", "blockNumber", blockNumber, "txHash", txHash.Hex())

	dbKey := CreatedContractKey.String() + "_" + txHash.String()
	data, err := m.monitordb.Get([]byte(dbKey))
	if nil != err && err != ErrNotFound {
		log.Error("failed to load created contracts", "err", err)
		return nil
	}
	var createdContractInfoList []*ContractInfo
	parseJson(data, &createdContractInfoList)

	log.Debug("get created contracts success")
	return createdContractInfoList
}

func (m *Monitor) CollectSuicidedContractInfo(txHash common.Hash, suicidedContractAddr common.Address) {
	dbKey := SuicidedContractKey.String() + "_" + txHash.String()
	data, err := m.monitordb.Get([]byte(dbKey))
	if nil != err && err != ErrNotFound {
		log.Error("failed to load suicided contracts", "err", err)
		return
	}

	var suicidedContractInfoList []*ContractInfo
	parseJson(data, &suicidedContractInfoList)

	suicidedContract := new(ContractInfo)
	suicidedContract.Address = suicidedContractAddr

	suicidedContractInfoList = append(suicidedContractInfoList, suicidedContract)

	json := toJson(suicidedContractInfoList)
	if len(json) > 0 {
		m.monitordb.Put([]byte(dbKey), json)
		log.Debug("save suicided contracts success")
	}

}

func (m *Monitor) GetSuicidedContractInfoList(blockNumber uint64, txHash common.Hash) []*ContractInfo {
	log.Debug("GetSuicidedContract", "blockNumber", blockNumber, "txHash", txHash.Hex())

	dbKey := SuicidedContractKey.String() + "_" + txHash.String()
	data, err := m.monitordb.Get([]byte(dbKey))
	if nil != err && err != ErrNotFound {
		log.Error("failed to load suicided contracts", "err", err)
		return nil
	}
	var suicidedContractInfoList []*ContractInfo
	parseJson(data, &suicidedContractInfoList)

	log.Debug("get suicided contracts success")
	return suicidedContractInfoList
}

// CollectProxyPattern 根据交易txHash发现代理关系
func (m *Monitor) CollectProxyPattern(txHash common.Hash, proxyContractInfo, implementationContractInfo *ContractInfo) {
	dbKey := ProxyPatternKey.String() + "_" + txHash.String()
	data, err := m.monitordb.Get([]byte(dbKey))
	if nil != err && err != ErrNotFound {
		log.Error("failed to load proxy patterns", "err", err)
		return
	}

	var proxyPatternList []*ProxyPattern
	parseJson(data, &proxyPatternList)
	proxyPatternList = append(proxyPatternList, &ProxyPattern{Proxy: proxyContractInfo, Implementation: implementationContractInfo})

	json := toJson(proxyPatternList)
	if len(json) > 0 {
		m.monitordb.Put([]byte(dbKey), json)
		log.Debug("save proxy patterns success")
	}

	// === to save the proxy map to local db

	dbMapKey := proxyPatternMapKey.String()
	data, err = m.monitordb.Get([]byte(dbMapKey))
	if nil != err && err != ErrNotFound {
		log.Error("failed to load proxy map", "err", err)
		return
	}

	var proxyPatternMap = make(map[common.Address]common.Address)
	parseJson(data, &proxyPatternMap)
	proxyPatternMap[proxyContractInfo.Address] = implementationContractInfo.Address

	json = toJson(proxyPatternMap)
	if len(json) > 0 {
		m.monitordb.Put([]byte(dbMapKey), json)
		log.Debug("save proxy map success")
	}
}

func (m *Monitor) IsProxied(self, target common.Address) bool {
	dbMapKey := proxyPatternMapKey.String()
	data, err := m.monitordb.Get([]byte(dbMapKey))
	if nil != err && err != ErrNotFound {
		log.Error("failed to load proxy map", "err", err)
		return false
	}

	var proxyPatternMap map[common.Address]common.Address
	parseJson(data, &proxyPatternMap)
	if value, exist := proxyPatternMap[self]; exist {
		if value == target {
			return true
		}
	}
	return false
}

func (m *Monitor) GetProxyPatternList(blockNumber uint64, txHash common.Hash) []*ProxyPattern {
	log.Debug("GetProxyPattern", "blockNumber", blockNumber, "txHash", txHash.Hex())

	dbKey := ProxyPatternKey.String() + "_" + txHash.String()
	data, err := m.monitordb.Get([]byte(dbKey))
	if nil != err && err != ErrNotFound {
		log.Error("failed to load proxy patterns", "err", err)
		return nil
	}
	var proxyPatternList []*ProxyPattern
	parseJson(data, &proxyPatternList)

	log.Debug("get proxy patterns success")
	return proxyPatternList
}

func (m *Monitor) CollectVerifiers(blockNumber uint64, txHash common.Hash) []*ProxyPattern {
	log.Debug("GetProxyPattern", "blockNumber", blockNumber, "txHash", txHash.Hex())

	dbKey := ProxyPatternKey.String() + "_" + txHash.String()
	data, err := m.monitordb.Get([]byte(dbKey))
	if nil != err && err != ErrNotFound {
		log.Error("failed to load proxy patterns", "err", err)
		return nil
	}
	var proxyPatternList []*ProxyPattern
	parseJson(data, &proxyPatternList)

	log.Debug("get proxy patterns success")
	return proxyPatternList
}

func (m *Monitor) SetReward(block *types.Block, numStr string) error {
	//set reward history
	packageReward, err := plugin.LoadNewBlockReward(block.Hash(), plugin.StakingInstance().GetStakingDB().GetDB())
	if nil != err {
		log.Error("Failed to LoadNewBlockReward on stakingPlugin Confirmed When Settletmetn block", "err", err)
		return err
	}
	stakingReward, err := plugin.LoadStakingReward(block.Hash(), plugin.StakingInstance().GetStakingDB().GetDB())
	if nil != err {
		log.Error("Failed to LoadStakingReward on stakingPlugin Confirmed When Settletmetn block", "err", err)
		return err
	}
	yearNum, err := plugin.LoadChainYearNumber(block.Hash(), plugin.StakingInstance().GetStakingDB().GetDB())
	if nil != err {
		log.Error("Failed to LoadChainYearNumber on stakingPlugin Confirmed When Settletmetn block", "err", err)
		return err
	}
	var reward staking.Reward
	if numStr == "0" {
		reward = staking.Reward{
			PackageReward: packageReward,
			StakingReward: stakingReward,
			YearNum:       yearNum + 1,
			YearStartNum:  0,
			YearEndNum:    xutil.CalcBlocksEachYear(),
			RemainEpoch:   uint32(xutil.EpochsPerYear()),
			AvgPackTime:   xcom.Interval() * 1000,
		}
		numberStart, err := rlp.EncodeToBytes(uint64(0))
		if nil != err {
			log.Error("Failed to EncodeToBytes on stakingPlugin Confirmed When Settletmetn block", "err", err)
			return err
		}
		m.monitordb.Put([]byte(YearKey.String()+"1"), numberStart)
	} else {
		incIssuanceTime, err := xcom.LoadIncIssuanceTime(block.Hash(), plugin.StakingInstance().GetStakingDB().GetDB())
		if nil != err {
			log.Error("Failed to LoadIncIssuanceTime on stakingPlugin Confirmed When Settletmetn block", "err", err)
			return err
		}
		number, err := xcom.LoadIncIssuanceNumber(block.Hash(), plugin.StakingInstance().GetStakingDB().GetDB())
		if nil != err {
			log.Error("Failed to LoadIncIssuanceTime on stakingPlugin Confirmed When Settletmetn block", "err", err)
			return err
		}

		avgPackTime, err := xcom.LoadCurrentAvgPackTime()
		if nil != err {
			log.Error("Failed to LoadAvgPackTime on stakingPlugin Confirmed When Settletmetn block", "err", err)
			return err
		}
		epochBlocks := xutil.CalcBlocksEachEpoch()
		remainTime := incIssuanceTime - int64(block.Header().Time)
		remainEpoch := 1
		remainBlocks := math.Ceil(float64(remainTime) / float64(avgPackTime))
		if remainBlocks > float64(epochBlocks) {
			remainEpoch = int(math.Ceil(remainBlocks / float64(epochBlocks)))
		}
		//get the num of year
		blocks := block.Number().Uint64() + uint64(remainEpoch)*epochBlocks
		if number != 0 && block.Number().Uint64()%number == 0 {
			yearTemp := strconv.FormatUint(uint64(yearNum+1), 10)
			numberStart, err := rlp.EncodeToBytes(number)
			if nil != err {
				log.Error("mygod,Failed to EncodeToBytes on stakingPlugin Confirmed When Settletmetn block", "err", err)
				return err
			}
			m.monitordb.Put([]byte(YearKey.String()+yearTemp), numberStart)
			log.Debug("set yearNum", "yearTemp", yearTemp, "number", block.Number())
		}
		if number == blocks {
			yearTemp := strconv.FormatUint(uint64(yearNum+1), 10)
			data, err := m.monitordb.Get([]byte(YearKey.String() + yearTemp))
			if nil != err {
				log.Error("mygod,get YearName error", "key", YearKey.String()+yearTemp, "err", err)
			}
			err = rlp.DecodeBytes(data, &number)
			if nil != err {
				log.Error("mygod,DecodeBytes YearName error", "key", YearKey.String()+yearTemp, "err", err)
			}
		}
		log.Debug("LoadNewBlockReward and LoadStakingReward", "packageReward", packageReward, "stakingReward", stakingReward, "hash", block.Hash(), "block number", block.Number(),
			"blocks", blocks, "number", number)
		reward = staking.Reward{
			PackageReward: packageReward,
			StakingReward: stakingReward,
			YearNum:       yearNum + 1,
			YearStartNum:  number,
			YearEndNum:    blocks,
			RemainEpoch:   uint32(remainEpoch),
			AvgPackTime:   avgPackTime,
		}
	}
	log.Debug("staking.Reward ,LoadNewBlockReward and LoadStakingReward", "packageReward", reward.PackageReward, "stakingReward", reward.StakingReward, "hash", block.Hash(), "number", block.Number())
	dataReward, err := rlp.EncodeToBytes(reward)
	if nil != err {
		log.Error("Failed to EncodeToBytes on stakingPlugin Confirmed When Settletmetn block", "err", err)
		return err
	}
	m.monitordb.Put([]byte(RewardKey.String()+numStr), dataReward)
	log.Debug("wow,insert rewardName history :", dataReward)
	return nil
}

func (m *Monitor) SetValidator(block *types.Block, numStr string, nodeId discover.NodeID) (bool, map[discover.NodeID]struct{}, error) {
	var isCurr bool
	currMap := make(map[discover.NodeID]struct{})
	current, err := plugin.StakingInstance().GetCurrValList(block.Hash(), block.NumberU64(), plugin.QueryStartNotIrr)
	if nil != err {
		log.Error("Failed to Query Current Round validators on stakingPlugin Confirmed When Election block",
			"blockNumber", block.Number().Uint64(), "blockHash", block.Hash().TerminalString(), "err", err)
		return isCurr, currMap, err
	}
	currentValidatorArray := &staking.ValidatorArraySave{
		Start: current.Start,
		End:   current.End,
	}
	vQSave := make(staking.ValidatorQueueSave, len(current.Arr))
	for k, v := range current.Arr {
		currMap[v.NodeId] = struct{}{}
		if nodeId == v.NodeId {
			isCurr = true
		}
		vQSave[k] = &staking.ValidatorSave{
			ValidatorTerm: v.ValidatorTerm,
			NodeId:        v.NodeId,
		}
	}
	currentValidatorArray.Arr = vQSave
	data, err := rlp.EncodeToBytes(currentValidatorArray)
	if nil != err {
		log.Error("Failed to EncodeToBytes on stakingPlugin Confirmed When Election block", "err", err)
		return isCurr, currMap, err
	}

	m.monitordb.Put([]byte(ValidatorKey.String()+numStr), data)
	log.Debug("wow,insert validator history", "blockNumber", block.Number(), "blockHash", block.Hash().String(), "insertNum", ValidatorKey.String()+numStr)
	log.Debug("wow,insert validator history", "currentValidatorArray", currentValidatorArray)
	return isCurr, currMap, nil
}

func (m *Monitor) SetVerifier(block *types.Block, numStr string) error {
	current, err := plugin.StakingInstance().GetVerifierArray(block.Hash(), block.NumberU64(), plugin.QueryStartNotIrr)
	if nil != err {
		log.Error("Failed to Query Current Round verifiers on stakingPlugin Confirmed When Settletmetn block",
			"blockHash", block.Hash().Hex(), "blockNumber", block.Number().Uint64(), "err", err)
		return err
	}

	currentCandidate, error := plugin.StakingInstance().GetCandidateList(block.Hash(), block.NumberU64())
	if nil != error {
		log.Error("Failed to Query Current Round candidate on stakingPlugin Confirmed When Settletmetn block",
			"blockHash", block.Hash().Hex(), "blockNumber", block.Number().Uint64(), "err", error)
		return error
	}
	currentValidatorArray := &staking.ValidatorArraySave{
		Start: current.Start,
		End:   current.End,
	}
	vQSave := make(staking.ValidatorQueueSave, len(current.Arr))
	for k, v := range current.Arr {
		vQSave[k] = &staking.ValidatorSave{
			ValidatorTerm:   v.ValidatorTerm,
			NodeId:          v.NodeId,
			StakingBlockNum: v.StakingBlockNum,
		}
		var isCurrent = false
		for _, cv := range currentCandidate {
			if cv.NodeId == v.NodeId {
				vQSave[k].DelegateRewardTotal = cv.DelegateRewardTotal.ToInt()
				vQSave[k].DelegateTotal = cv.DelegateTotal.ToInt()
				isCurrent = true
				break
			}
		}
		if !isCurrent {
			nodeIdAddr, err := xutil.NodeId2Addr(v.NodeId)
			if nil != err {
				log.Error("Failed to NodeId2Addr: parse current nodeId is failed", "err", err)
			}
			can, err := plugin.StakingInstance().GetCandidateInfo(block.Hash(), nodeIdAddr)
			if err != nil || can == nil {
				log.Error("Failed to Query Current Round candidate info on stakingPlugin Confirmed When Settletmetn block",
					"blockHash", block.Hash().Hex(), "blockNumber", block.Number().Uint64(), "err", err)
				log.Debug("Failed get can :", can)
			} else {
				vQSave[k].DelegateRewardTotal = can.DelegateRewardTotal
				vQSave[k].DelegateTotal = can.DelegateTotal
			}
		}

	}
	currentValidatorArray.Arr = vQSave
	data, err := rlp.EncodeToBytes(currentValidatorArray)
	if nil != err {
		log.Error("Failed to EncodeToBytes on stakingPlugin Confirmed When Settletmetn block", "err", err)
		return err
	}
	m.monitordb.Put([]byte(VerifierKey.String()+numStr), data)
	log.Debug("wow,insert verifier history", "blockNumber", block.Number(), "blockHash", block.Hash().String(), "insertNum", VerifierKey.String()+numStr)
	log.Debug("wow,insert verifier history :", currentValidatorArray)

	if numStr == "0" {
		dataCandidate, err := rlp.EncodeToBytes(currentCandidate)
		if nil != err {
			log.Error("Failed to EncodeToBytes on stakingPlugin Confirmed When Settletmetn block", "err", err)
			return err
		}
		m.monitordb.Put([]byte(InitNodeKey.String()+"0"), dataCandidate)
		log.Debug("wow,insert candidate  0:", currentCandidate)
	}
	return nil
}
