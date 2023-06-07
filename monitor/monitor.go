package monitor

import (
	"github.com/PlatONnetwork/AppChain-Go/common"
	"github.com/PlatONnetwork/AppChain-Go/common/hexutil"
	"github.com/PlatONnetwork/AppChain-Go/core/state"
	"github.com/PlatONnetwork/AppChain-Go/log"
	"github.com/PlatONnetwork/AppChain-Go/x/staking"
	"github.com/PlatONnetwork/AppChain-Go/x/xutil"
	"math/big"
	"strconv"
	"sync"
)

type MonitorDbKey int

const (
	UnusualTransferTxKey MonitorDbKey = iota
	CreatedContractKey
	SuicidedContractKey
	ProxyPatternKey
	proxyPatternMapKey
	ValidatorKey
	VerifierKey
	EpochInfoKey
	SlashKey
	ImplicitPPOSTxKey
)

// 定义 MonitorDbKey 类型的方法 String(), 返回字符串。
func (dbKey MonitorDbKey) String() string {
	return [...]string{
		"UnusualTransferTxKey",
		"CreatedContractKey",
		"SuicidedContractKey",
		"ProxyPatternKey",
		"proxyPatternMapKey",
		"ValidatorKey",
		"VerifierKey",
		"EpochInfoKey",
		"YearKey",
		"InitNodeKey",
		"SlashKey",
		"ImplicitPPOSTxKey",
	}[dbKey]
}

type Monitor struct {
	statedb           *state.StateDB
	monitordb         *monitorDB
	stakingPlugin     Intf_stakingPlugin
	restrictingPlugin Intf_restrictingPlugin
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

func (m *Monitor) SetStakingPlugin(pluginImpl Intf_stakingPlugin) {
	monitor.stakingPlugin = pluginImpl
}
func (m *Monitor) SetRestrictingPlugin(pluginImpl Intf_restrictingPlugin) {
	monitor.restrictingPlugin = pluginImpl
}

func (m *Monitor) CollectUnusualTransferTx(blockNumber uint64, txHash common.Hash, from, to common.Address, amount *big.Int) {
	log.Debug("CollectUnusualTransferTx", "blockNumber", blockNumber, "txHash", txHash.Hex(), "from", from.Bech32(), "to", to.Bech32(), "amount", amount)

	dbKey := UnusualTransferTxKey.String() + "_" + txHash.String()
	data, err := m.monitordb.Get([]byte(dbKey))
	if nil != err && err != ErrNotFound {
		log.Error("failed to load unusual transfers", "err", err)
		return
	}

	var unusualTransferTxList []*UnusualTransferTx
	ParseJson(data, &unusualTransferTxList)

	unusualTransferTx := new(UnusualTransferTx)
	unusualTransferTx.TxHash = txHash
	unusualTransferTx.From = from
	unusualTransferTx.To = to
	unusualTransferTx.Amount = amount

	unusualTransferTxList = append(unusualTransferTxList, unusualTransferTx)

	json := ToJson(unusualTransferTxList)
	if len(json) > 0 {
		m.monitordb.Put([]byte(dbKey), json)
		log.Debug("save unusual transfers success")
	}

}

func (m *Monitor) GetUnusualTransferTx(blockNumber uint64, txHash common.Hash) []*UnusualTransferTx {
	log.Debug("GetUnusualTransferTx", "blockNumber", blockNumber, "txHash", txHash.Hex())

	dbKey := UnusualTransferTxKey.String() + "_" + txHash.String()
	data, err := m.monitordb.Get([]byte(dbKey))
	if nil != err {
		log.Error("failed to load unusual transfers", "err", err)
		return nil
	}

	var unusualTransferTxList []*UnusualTransferTx
	ParseJson(data, &unusualTransferTxList)
	return unusualTransferTxList
}

func (m *Monitor) CollectCreatedContractInfo(txHash common.Hash, contractInfo *ContractInfo) {
	log.Debug("CollectCreatedContractInfo", "txHash", txHash.Hex(), "contractInfo", string(ToJson(contractInfo)))

	dbKey := CreatedContractKey.String() + "_" + txHash.String()
	data, err := m.monitordb.Get([]byte(dbKey))
	if nil != err && err != ErrNotFound {
		log.Error("failed to load created contracts", "err", err)
		return
	}
	var createdContractInfoList []*ContractInfo
	ParseJson(data, &createdContractInfoList)

	createdContractInfoList = append(createdContractInfoList, contractInfo)

	json := ToJson(createdContractInfoList)
	if len(json) > 0 {
		m.monitordb.Put([]byte(dbKey), json)
		log.Debug("save created contracts success")
	}

}

func (m *Monitor) GetCreatedContracts(blockNumber uint64, txHash common.Hash) []*ContractInfo {
	log.Debug("GetCreatedContract", "blockNumber", blockNumber, "txHash", txHash.Hex())

	dbKey := CreatedContractKey.String() + "_" + txHash.String()
	data, err := m.monitordb.Get([]byte(dbKey))
	if nil != err && err != ErrNotFound {
		log.Error("failed to load created contracts", "err", err)
		return nil
	}
	var createdContractInfoList []*ContractInfo
	ParseJson(data, &createdContractInfoList)

	log.Debug("get created contracts success")
	return createdContractInfoList
}

func (m *Monitor) CollectSuicidedContract(txHash common.Hash, suicidedContractAddr common.Address) {
	dbKey := SuicidedContractKey.String() + "_" + txHash.String()
	data, err := m.monitordb.Get([]byte(dbKey))
	if nil != err && err != ErrNotFound {
		log.Error("failed to load suicided contracts", "err", err)
		return
	}

	var suicidedContractInfoList []*ContractInfo
	ParseJson(data, &suicidedContractInfoList)

	suicidedContract := new(ContractInfo)
	suicidedContract.Address = suicidedContractAddr

	suicidedContractInfoList = append(suicidedContractInfoList, suicidedContract)

	json := ToJson(suicidedContractInfoList)
	if len(json) > 0 {
		m.monitordb.Put([]byte(dbKey), json)
		log.Debug("save suicided contracts success")
	}

}

func (m *Monitor) GetSuicidedContracts(blockNumber uint64, txHash common.Hash) []*ContractInfo {
	log.Debug("GetSuicidedContract", "blockNumber", blockNumber, "txHash", txHash.Hex())

	dbKey := SuicidedContractKey.String() + "_" + txHash.String()
	data, err := m.monitordb.Get([]byte(dbKey))
	if nil != err && err != ErrNotFound {
		log.Error("failed to load suicided contracts", "err", err)
		return nil
	}
	var suicidedContractInfoList []*ContractInfo
	ParseJson(data, &suicidedContractInfoList)

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
	ParseJson(data, &proxyPatternList)
	proxyPatternList = append(proxyPatternList, &ProxyPattern{Proxy: proxyContractInfo, Implementation: implementationContractInfo})

	json := ToJson(proxyPatternList)
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
	ParseJson(data, &proxyPatternMap)
	proxyPatternMap[proxyContractInfo.Address] = implementationContractInfo.Address

	json = ToJson(proxyPatternMap)
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
	ParseJson(data, &proxyPatternMap)
	if value, exist := proxyPatternMap[self]; exist {
		if value == target {
			return true
		}
	}
	return false
}

func (m *Monitor) GetProxyPatterns(blockNumber uint64, txHash common.Hash) []*ProxyPattern {
	log.Debug("GetProxyPattern", "blockNumber", blockNumber, "txHash", txHash.Hex())

	dbKey := ProxyPatternKey.String() + "_" + txHash.String()
	data, err := m.monitordb.Get([]byte(dbKey))
	if nil != err && err != ErrNotFound {
		log.Error("failed to load proxy patterns", "err", err)
		return nil
	}
	var proxyPatternList []*ProxyPattern
	ParseJson(data, &proxyPatternList)

	log.Debug("get proxy patterns success")
	return proxyPatternList
}

func (m *Monitor) CollectionEpochInfo(epoch uint64, newBlockReward, epochTotalStakingReward *big.Int, chainAge uint32, yearStartBlockNumber uint64, remainEpochThisYear uint32, avgPackTime uint64) {
	view := EpochView{
		PackageReward:     newBlockReward,
		StakingReward:     epochTotalStakingReward,
		ChainAge:          chainAge + 1, // ChainAge starts from 1
		YearStartBlockNum: yearStartBlockNumber,
		YearEndBlockNum:   yearStartBlockNumber + uint64(remainEpochThisYear)*xutil.CalcBlocksEachEpoch(),
		RemainEpoch:       remainEpochThisYear,
		AvgPackTime:       avgPackTime,
	}
	json := ToJson(view)
	dbKey := EpochInfoKey.String() + "_" + strconv.FormatUint(epoch, 10)
	m.monitordb.Put([]byte(dbKey), json)
	log.Debug("CollectionEpochInfo", "data", string(json))
}

// 保存25名单(包含详细信息）
// epoch轮数从1开始，key的组成是：ValidatorKey+每个epoch轮的开始块高
func (m *Monitor) CollectValidators(blockHash common.Hash, blockNumber uint64, validators *staking.ValidatorArray) {
	//获取所有质押节点（包含详细信息，25中的提出退出节点，不再存在此列表中）
	currentCandidate, err := m.stakingPlugin.GetCandidateList(blockHash, blockNumber)
	if nil != err {
		log.Error("Failed to GetCandidateList", "blockHash", blockHash.Hex(), "blockNumber", blockNumber, "err", err)
		return
	}
	validatorExQueue := make(staking.ValidatorExQueue, len(validators.Arr))
	for k, v := range validators.Arr {
		validatorExQueue[k] = &staking.ValidatorEx{
			ValidatorTerm:   v.ValidatorTerm,
			NodeId:          v.NodeId,
			StakingBlockNum: v.StakingBlockNum,
			ProgramVersion:  v.ProgramVersion,
		}
		var notInCadidateList = true
		// 给ValidatorEx补充详细信息
		for _, cv := range currentCandidate {
			if cv.NodeId == v.NodeId {
				validatorExQueue[k].DelegateRewardTotal = cv.DelegateRewardTotal
				validatorExQueue[k].DelegateTotal = cv.DelegateTotal
				validatorExQueue[k].BenefitAddress = cv.BenefitAddress
				validatorExQueue[k].StakingAddress = cv.StakingAddress
				validatorExQueue[k].Website = cv.Website
				validatorExQueue[k].Description = cv.Description
				validatorExQueue[k].ExternalId = cv.ExternalId
				validatorExQueue[k].NodeName = cv.NodeName
				notInCadidateList = false
				break
			}
		}
		if notInCadidateList {
			// 不在currentCandidate的verifier，需要额外补充详细信息
			nodeIdAddr, err := xutil.NodeId2Addr(v.NodeId)
			if nil != err {
				log.Error("Failed to NodeId2Addr: parse current nodeId is failed", "err", err)
			}
			can, err := m.stakingPlugin.GetCandidateInfo(blockHash, nodeIdAddr)
			if err != nil || can == nil {
				log.Error("Failed to Query Current Round candidate info on stakingPlugin Confirmed When Settletmetn block",
					"blockHash", blockHash.Hex(), "blockNumber", blockNumber, "err", err)
				log.Debug("Failed get can :", can)
			} else {
				validatorExQueue[k].DelegateRewardTotal = (*hexutil.Big)(can.DelegateRewardTotal)
				validatorExQueue[k].DelegateTotal = (*hexutil.Big)(can.DelegateTotal)
				validatorExQueue[k].BenefitAddress = can.BenefitAddress
				validatorExQueue[k].StakingAddress = can.StakingAddress
				validatorExQueue[k].Website = can.Website
				validatorExQueue[k].Description = can.Description
				validatorExQueue[k].ExternalId = can.ExternalId
				validatorExQueue[k].NodeName = can.NodeName
				validatorExQueue[k].ProgramVersion = can.ProgramVersion
			}
		}
	}
	json := ToJson(validatorExQueue)

	epoch := xutil.CalculateEpoch(blockNumber)

	epochStartBlockNumber := (epoch - 1) * xutil.CalcBlocksEachEpoch()
	dbKey := ValidatorKey.String() + strconv.FormatUint(epochStartBlockNumber, 10)

	m.monitordb.Put([]byte(dbKey), json)
	log.Debug("wow,insert verifier history", "blockNumber", blockNumber, "blockHash", blockHash.String(), "dbKey", dbKey)
	return
}

// 保存201名单（包含详细信息）
// epoch轮数从1开始，key的组成是：VerifierKey+每个epoch轮的开始块高
func (m *Monitor) CollectVerifiers(blockHash common.Hash, blockNumber uint64, queryStartNotIrr bool) {
	//获取201名单（只包含必要信息）
	verifiers, err := m.stakingPlugin.GetVerifierArray(blockHash, blockNumber, queryStartNotIrr)
	if nil != err {
		log.Error("Failed to Query Current Round verifiers on stakingPlugin Confirmed When Settletmetn block",
			"blockHash", blockHash.Hex(), "blockNumber", blockNumber, "err", err)
		return
	}
	//获取所有质押节点（包含详细信息，201中的提出退出节点，不再存在此列表中）
	currentCandidate, err := m.stakingPlugin.GetCandidateList(blockHash, blockNumber)
	if nil != err {
		log.Error("Failed to Query Current Round candidate on stakingPlugin Confirmed When Settletmetn block",
			"blockHash", blockHash.Hex(), "blockNumber", blockNumber, "err", err)
		return
	}
	validatorExQueue := make(staking.ValidatorExQueue, len(verifiers.Arr))
	for k, v := range verifiers.Arr {
		validatorExQueue[k] = &staking.ValidatorEx{
			ValidatorTerm:   v.ValidatorTerm,
			NodeId:          v.NodeId,
			StakingBlockNum: v.StakingBlockNum,
			ProgramVersion:  v.ProgramVersion,
		}
		var notInCadidateList = true
		// 给ValidatorEx补充详细信息
		for _, cv := range currentCandidate {
			if cv.NodeId == v.NodeId {
				validatorExQueue[k].DelegateRewardTotal = cv.DelegateRewardTotal
				validatorExQueue[k].DelegateTotal = cv.DelegateTotal
				validatorExQueue[k].BenefitAddress = cv.BenefitAddress
				validatorExQueue[k].StakingAddress = cv.StakingAddress
				validatorExQueue[k].Website = cv.Website
				validatorExQueue[k].Description = cv.Description
				validatorExQueue[k].ExternalId = cv.ExternalId
				validatorExQueue[k].NodeName = cv.NodeName
				notInCadidateList = false
				break
			}
		}
		if notInCadidateList {
			// 不在currentCandidate的verifier，需要额外补充详细信息
			nodeIdAddr, err := xutil.NodeId2Addr(v.NodeId)
			if nil != err {
				log.Error("Failed to NodeId2Addr: parse current nodeId is failed", "err", err)
			}
			can, err := m.stakingPlugin.GetCandidateInfo(blockHash, nodeIdAddr)
			if err != nil || can == nil {
				log.Error("Failed to Query Current Round candidate info on stakingPlugin Confirmed When Settletmetn block",
					"blockHash", blockHash.Hex(), "blockNumber", blockNumber, "err", err)
				log.Debug("Failed get can :", can)
			} else {
				validatorExQueue[k].DelegateRewardTotal = (*hexutil.Big)(can.DelegateRewardTotal)
				validatorExQueue[k].DelegateTotal = (*hexutil.Big)(can.DelegateTotal)
				validatorExQueue[k].BenefitAddress = can.BenefitAddress
				validatorExQueue[k].StakingAddress = can.StakingAddress
				validatorExQueue[k].Website = can.Website
				validatorExQueue[k].Description = can.Description
				validatorExQueue[k].ExternalId = can.ExternalId
				validatorExQueue[k].NodeName = can.NodeName
				validatorExQueue[k].ProgramVersion = can.ProgramVersion
			}
		}
	}
	json := ToJson(validatorExQueue)

	epoch := xutil.CalculateEpoch(blockNumber)

	epochStartBlockNumber := (epoch - 1) * xutil.CalcBlocksEachEpoch()
	dbKey := VerifierKey.String() + strconv.FormatUint(epochStartBlockNumber, 10)

	m.monitordb.Put([]byte(dbKey), json)
	log.Debug("wow,insert verifier history", "blockNumber", blockNumber, "blockHash", blockHash.String(), "dbKey", dbKey)
	return
}

// 收集隐式的ppos交易数据
func (m *Monitor) CollectImplicitPPOSTx(blockNumber uint64, txHash common.Hash, from, to common.Address, input, result []byte) {
	dbKey := ImplicitPPOSTxKey.String() + "_" + strconv.FormatUint(blockNumber, 10)
	data, err := m.monitordb.Get([]byte(dbKey))
	if nil != err && err != ErrNotFound {
		log.Error("failed to load data from local db", "err", err)
		return
	}

	var implicitPPOSTx *ImplicitPPOSTx
	if len(data) == 0 {
		implicitPPOSTx = &ImplicitPPOSTx{ContractTxMap: make(map[common.Hash][]*ContractTx)}
	} else {
		ParseJson(data, &implicitPPOSTx)
	}
	contractTx := &ContractTx{From: from, To: to, Input: input, Result: result}
	implicitPPOSTx.ContractTxMap[txHash] = append(implicitPPOSTx.ContractTxMap[txHash], contractTx)
	json := ToJson(implicitPPOSTx)
	if len(json) > 0 {
		m.monitordb.Put([]byte(dbKey), json)
	}
}

func (m *Monitor) CollectSlashInfo(electionBlockNumber uint64, slashQueue staking.SlashQueue) {
	if slashQueue == nil || len(slashQueue) == 0 {
		return
	}
	dbKey := SlashKey.String() + "_" + strconv.FormatUint(electionBlockNumber, 10)
	json := ToJson(slashQueue)

	err := m.monitordb.Put([]byte(dbKey), json)
	if nil != err && err != ErrNotFound {
		log.Error("failed to CollectSlashInfo", "electionBlockNumber", electionBlockNumber, "err", err)
		return
	}
}
