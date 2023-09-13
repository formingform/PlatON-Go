package monitor

import (
	"bytes"
	"encoding/binary"
	"github.com/PlatONnetwork/PlatON-Go/common"
	"github.com/PlatONnetwork/PlatON-Go/common/hexutil"
	"github.com/PlatONnetwork/PlatON-Go/core/rawdb"
	"github.com/PlatONnetwork/PlatON-Go/core/state"
	"github.com/PlatONnetwork/PlatON-Go/core/types"
	"github.com/PlatONnetwork/PlatON-Go/log"
	"github.com/PlatONnetwork/PlatON-Go/x/staking"
	"github.com/PlatONnetwork/PlatON-Go/x/xcom"
	"github.com/PlatONnetwork/PlatON-Go/x/xutil"
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
	proxyPatternFlagKey
	ImplicitPPOSTxKey

	ValidatorsOfEpochKey
	VerifiersOfEpochKey
	EpochInfoKey
	SlashKey
)

// 定义 MonitorDbKey 类型的方法 String(), 返回字符串。
func (dbKey MonitorDbKey) String() string {
	return [...]string{
		"EmbedTransferKey",
		"CreatedContractKey",
		"SuicidedContractKey",
		"ProxyPatternKey",
		"proxyPatternFlagKey",
		"ImplicitPPOSTxKey",

		"ValidatorsOfEpochKey",
		"VerifiersOfEpochKey",
		"EpochInfoKey",
		"SlashKey",
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
func InitMonitorForUnitTest(monitorDbFullPath string) {
	onceMonitor.Do(func() {
		dbFullPath = monitorDbFullPath
		db := rawdb.NewMemoryDatabase()
		statedb, _ := state.New(common.Hash{}, state.NewDatabaseWithConfig(db, nil))

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
func (m *Monitor) Monitordb() *monitorDB {
	return monitor.monitordb
}

/*type ProxyInfo struct {
	Proxy            common.Address `json:"proxy,omitempty"`
	Implementation   common.Address `json:"implementation,omitempty"`
	TokenName        string         `json:"tokenName,omitempty"`
	TokenSymbol      string         `json:"tokenSymbol,omitempty"`
	TokenDecimals    uint8          `json:"tokenDecimals,omitempty"`
	TokenTotalSupply *big.Int       `json:"tokenTotalSupply,omitempty"`
}*/

/*
*
收集合约内置的转账交易（不包括初始交易是带value的合约调用）
1. 合约内部的lat转账
2. 合约自杀时的lat转账
*/
func (m *Monitor) CollectEmbedTransfer(blockNumber uint64, txHash common.Hash, from, to common.Address, amount *big.Int) {
	log.Info("CollectEmbedTransferTx", "blockNumber", blockNumber, "txHash", txHash.String(), "from", from.Bech32(), "to", to.Bech32(), "amount", amount)

	dbKey := EmbedTransferKey.String() + "_" + txHash.String()
	data, err := m.monitordb.Get([]byte(dbKey))
	if nil != err && err != ErrNotFound {
		log.Error("failed to load embed transfers", "err", err)
		return
	}

	var embedTransferList []*EmbedTransfer
	common.ParseJson(data, &embedTransferList)

	embedTransfer := new(EmbedTransfer)
	embedTransfer.TxHash = txHash
	embedTransfer.From = from
	embedTransfer.To = to
	embedTransfer.Amount = amount

	embedTransferList = append(embedTransferList, embedTransfer)

	json := common.ToJson(embedTransferList)
	if len(json) > 0 {
		m.monitordb.Put([]byte(dbKey), json)
		log.Info("save embed transfers success")
	}
	log.Info("CollectEmbedTransfer success", "txHash", txHash.String(), "json", string(json))
}

func (m *Monitor) GetEmbedTransfer(blockNumber uint64, txHash common.Hash) []*EmbedTransfer {
	log.Debug("GetEmbedTransfer", "blockNumber", blockNumber, "txHash", txHash.String())

	dbKey := EmbedTransferKey.String() + "_" + txHash.String()
	data, err := m.monitordb.Get([]byte(dbKey))

	if nil != err {
		if err == ErrNotFound {
			log.Debug("GetEmbedTransfer success: no data")
		} else {
			log.Error("GetEmbedTransfer failed", "err", err)
		}
		return nil
	}

	var embedTransferList []*EmbedTransfer
	common.ParseJson(data, &embedTransferList)
	return embedTransferList
}

func (m *Monitor) CollectCreatedContractInfo(txHash common.Hash, contractInfo *ContractInfo) {
	log.Info("CollectCreatedContractInfo", "txHash", txHash.String(), "contractInfo", string(common.ToJson(contractInfo)))

	dbKey := CreatedContractKey.String() + "_" + txHash.String()
	data, err := m.monitordb.Get([]byte(dbKey))
	if nil != err && err != ErrNotFound {
		log.Error("failed to load created contracts", "err", err)
		return
	}
	var createdContractInfoList []*ContractInfo
	common.ParseJson(data, &createdContractInfoList)

	createdContractInfoList = append(createdContractInfoList, contractInfo)

	json := common.ToJson(createdContractInfoList)
	if len(json) > 0 {
		m.monitordb.Put([]byte(dbKey), json)
	}
	log.Info("CollectCreatedContractInfo success", "txHash", txHash.String(), "json", string(json))
}

func (m *Monitor) GetCreatedContractInfoList(blockNumber uint64, txHash common.Hash) []*ContractInfo {
	log.Debug("GetCreatedContract", "blockNumber", blockNumber, "txHash", txHash.String())

	dbKey := CreatedContractKey.String() + "_" + txHash.String()
	data, err := m.monitordb.Get([]byte(dbKey))
	if nil != err {
		if err == ErrNotFound {
			log.Debug("GetCreatedContract success: no data")
		} else {
			log.Error("GetCreatedContract failed", "err", err)
		}
		return nil
	}
	var createdContractInfoList []*ContractInfo
	common.ParseJson(data, &createdContractInfoList)

	log.Debug("GetCreatedContract success", "txHash", txHash.String(), "json", string(data))
	return createdContractInfoList
}

func (m *Monitor) CollectSuicidedContractInfo(txHash common.Hash, suicidedContractAddr common.Address) {
	log.Info("CollectSuicidedContractInfo", "txHash", txHash.String(), "suicidedContractAddr", suicidedContractAddr.String())

	dbKey := SuicidedContractKey.String() + "_" + txHash.String()
	data, err := m.monitordb.Get([]byte(dbKey))
	if nil != err && err != ErrNotFound {
		log.Error("failed to load suicided contracts", "err", err)
		return
	}

	var suicidedContractInfoList []*ContractInfo
	common.ParseJson(data, &suicidedContractInfoList)

	suicidedContract := new(ContractInfo)
	suicidedContract.Address = suicidedContractAddr

	suicidedContractInfoList = append(suicidedContractInfoList, suicidedContract)

	json := common.ToJson(suicidedContractInfoList)
	if len(json) > 0 {
		m.monitordb.Put([]byte(dbKey), json)
	}
	log.Info("CollectSuicidedContractInfo success", "txHash", txHash.String(), "json", string(json))
}

func (m *Monitor) GetSuicidedContractInfoList(blockNumber uint64, txHash common.Hash) []*ContractInfo {
	log.Debug("GetSuicidedContract", "blockNumber", blockNumber, "txHash", txHash.String())

	dbKey := SuicidedContractKey.String() + "_" + txHash.String()
	data, err := m.monitordb.Get([]byte(dbKey))
	if nil != err {
		if err == ErrNotFound {
			log.Debug("GetSuicidedContract success: no data")
		} else {
			log.Error("GetSuicidedContract failed", "err", err)
		}
		return nil
	}
	var suicidedContractInfoList []*ContractInfo
	common.ParseJson(data, &suicidedContractInfoList)

	log.Debug("GetSuicidedContract success", "txHash", txHash.String(), "json", string(data))
	return suicidedContractInfoList
}

// CollectProxyPattern 根据交易txHash发现代理关系
func (m *Monitor) CollectProxyPattern(txHash common.Hash, proxyContractInfo, implementationContractInfo *ContractInfo) {
	// 检查是否发现过此代理关系, 以proxy address为key即可
	// === to save the proxy map to local db

	if m.IsProxied(proxyContractInfo.Address, implementationContractInfo.Address) {
		return
	} else {
		flagDbKey := proxyPatternFlagKey.String() + "_" + proxyContractInfo.Address.String() + "_" + implementationContractInfo.Address.String()
		m.monitordb.Put([]byte(flagDbKey), []byte{0x01})
	}

	log.Info("CollectProxyPattern", "txHash", txHash.String(), "proxy", proxyContractInfo.Address.String(), "implementation", implementationContractInfo.Address.String())

	// 收集当前当前交易发现的代理关系
	dbKey := ProxyPatternKey.String() + "_" + txHash.String()
	data, err := m.monitordb.Get([]byte(dbKey))
	if nil != err && err != ErrNotFound {
		log.Error("failed to load proxy patterns", "err", err)
		return
	}

	var proxyPatternList []*ProxyPattern
	common.ParseJson(data, &proxyPatternList)
	proxyPatternList = append(proxyPatternList, &ProxyPattern{Proxy: proxyContractInfo, Implementation: implementationContractInfo})

	json := common.ToJson(proxyPatternList)
	if len(json) > 0 {
		m.monitordb.Put([]byte(dbKey), json)
	}
	log.Info("CollectProxyPattern success", "txHash", txHash.String(), "json", string(json))
}

// epoch切换时，收集下一个epoch的相关信息
func (m *Monitor) CollectionNextEpochInfo(nextEpoch uint64, newBlockReward, epochTotalStakingReward *big.Int, chainAge uint32, yearStartBlockNumber uint64, remainEpochThisYear uint32, avgPackTime uint64) {
	view := EpochView{
		PackageReward:     newBlockReward,          //出块奖励
		StakingReward:     epochTotalStakingReward, //总的质押奖励
		ChainAge:          chainAge + 1,            // ChainAge starts from 1
		YearStartBlockNum: yearStartBlockNumber,
		YearEndBlockNum:   yearStartBlockNumber + uint64(remainEpochThisYear)*xutil.CalcBlocksEachEpoch(),
		RemainEpoch:       remainEpochThisYear,
		AvgPackTime:       avgPackTime,
	}
	json := common.ToJson(view)
	dbKey := EpochInfoKey.String() + "_" + strconv.FormatUint(nextEpoch, 10)
	m.monitordb.Put([]byte(dbKey), json)
	log.Debug("CollectionNextEpochInfo", "data", string(json))
}

func (m *Monitor) IsProxied(proxy, impl common.Address) bool {
	flagDbKey := proxyPatternFlagKey.String() + "_" + proxy.String() + "_" + impl.String()
	flagBytes, err := m.monitordb.Get([]byte(flagDbKey))
	if nil != err {
		if err == ErrNotFound {
			log.Debug("IsProxied success: no data")
		} else {
			log.Error("IsProxied failed", "err", err)
		}
		return false
	}

	if len(flagBytes) > 0 && bytes.Equal(flagBytes, []byte{0x01}) {
		return true
	} else {
		return false
	}
}

func (m *Monitor) GetProxyPatternList(blockNumber uint64, txHash common.Hash) []*ProxyPattern {
	log.Debug("GetProxyPattern", "blockNumber", blockNumber, "txHash", txHash.String())

	dbKey := ProxyPatternKey.String() + "_" + txHash.String()
	data, err := m.monitordb.Get([]byte(dbKey))
	if nil != err {
		if err == ErrNotFound {
			log.Debug("GetProxyPatternList success: no data")
		} else {
			log.Error("GetProxyPatternList failed", "err", err)
		}
		return nil
	}
	var proxyPatternList []*ProxyPattern
	common.ParseJson(data, &proxyPatternList)

	log.Debug("GetProxyPatternList success", "txHash", txHash.String(), "json", string(data))
	return proxyPatternList
}

// 收集隐式的ppos交易数据
// 新方式（暂时未启用
func (m *Monitor) CollectImplicitPPOSTx(blockNumber uint64, txHash common.Hash, from, to common.Address, input []byte, ret []byte, itsLog *types.Log) {
	errCode := binary.BigEndian.Uint16(ret)
	inputHex := hexutil.Encode(input)
	log.Info("CollectImplicitPPOSTx", "blockNumber", blockNumber, "txHash", txHash.String(), "from", from.String(), "to", to.String(), "input", inputHex, "errCode", errCode)
	dbKey := ImplicitPPOSTxKey.String() + "_" + txHash.String()
	data, err := m.monitordb.Get([]byte(dbKey))
	if nil != err && err != ErrNotFound {
		log.Error("failed to load data from local db", "err", err)
		return
	}

	newElement := ImplicitPPOSTx{From: from, To: to, InputHex: inputHex, LogDataHex: hexutil.Encode(itsLog.Data)}

	var implicitPPOSTxs []*ImplicitPPOSTx
	if len(data) > 0 {
		common.ParseJson(data, &implicitPPOSTxs)
	}
	implicitPPOSTxs = append(implicitPPOSTxs, &newElement)

	json := common.ToJson(implicitPPOSTxs)
	if len(json) > 0 {
		m.monitordb.Put([]byte(dbKey), json)
	}
	log.Info("CollectImplicitPPOSTx success", "txHash", txHash.String(), "json", string(json))
}

// 收集隐式的ppos交易数据
func (m *Monitor) GetImplicitPPOSTx(blockNumber uint64, txHash common.Hash) []*ImplicitPPOSTx {
	log.Debug("GetImplicitPPOSTx", "blockNumber", blockNumber, "txHash", txHash.String())

	dbKey := ImplicitPPOSTxKey.String() + "_" + txHash.String()
	data, err := m.monitordb.Get([]byte(dbKey))
	if nil != err {
		if err == ErrNotFound {
			log.Debug("GetImplicitPPOSTx success: no data")
		} else {
			log.Error("GetImplicitPPOSTx failed", "err", err)
		}
		return nil
	}

	var implicitPPOSTxList []*ImplicitPPOSTx
	common.ParseJson(data, &implicitPPOSTxList)

	log.Debug("GetImplicitPPOSTx success", "txHash", txHash.String(), "json", string(data))
	return implicitPPOSTxList
}

func (m *Monitor) CollectSlashInfo(electionBlockNumber uint64, slashQueue staking.SlashQueue) {
	log.Info("CollectSlashInfo", "blockNumber", electionBlockNumber, "slashQueue", string(common.ToJson(slashQueue)))

	if slashQueue == nil || len(slashQueue) == 0 {
		return
	}
	dbKey := SlashKey.String() + "_" + strconv.FormatUint(electionBlockNumber, 10)
	json := common.ToJson(slashQueue)

	err := m.monitordb.Put([]byte(dbKey), json)
	if nil != err && err != ErrNotFound {
		log.Error("failed to CollectSlashInfo", "electionBlockNumber", electionBlockNumber, "err", err)
		return
	}
}

func (m *Monitor) CollectInitVerifiers(blockHash common.Hash, blockNumber uint64, queryStartNotIrr bool) {
	//获取201名单（只包含必要信息）
	verifiers, err := m.stakingPlugin.GetVerifierArray(blockHash, blockNumber, queryStartNotIrr)
	if nil != err {
		log.Error("Failed to Query Current Round verifiers on stakingPlugin Confirmed When Settletmetn block",
			"blockHash", blockHash.Hex(), "blockNumber", blockNumber, "err", err)
		return
	}

	log.Info("CollectInitVerifiers:", "blockNumber", blockNumber, "size", len(verifiers.Arr), "verifiers", string(common.ToJson(verifiers)))

	log.Debug("CollectInitVerifiers:", "size", len(verifiers.Arr), "data:", common.ToJson(verifiers))
	validatorExQueue, err := m.convertToValidatorExQueue(blockHash, blockNumber, verifiers)
	if nil != err {
		log.Error("failed to convertToValidatorExQueue", "blockHash", blockHash.Hex(), "blockNumber", blockNumber, "err", err)
		return
	}
	json := common.ToJson(validatorExQueue)

	dbKey := VerifiersOfEpochKey.String() + strconv.FormatUint(1, 10)

	m.monitordb.Put([]byte(dbKey), json)
	log.Debug("success to CollectInitVerifiers", "blockNumber", blockNumber, "blockHash", blockHash.String(), "dbKey", dbKey)
	return
}

func (m *Monitor) CollectInitValidators(blockHash common.Hash, blockNumber uint64, queryStartNotIrr bool) {
	curValidators, err := m.stakingPlugin.GetCurrValList(blockHash, blockNumber, queryStartNotIrr)
	if nil != err {
		log.Error("Failed to CollectInitEpochValidators", "blockNumber", blockHash, "blockHash", blockNumber, "err", err)
		return
	}

	log.Info("CollectInitValidators:", "blockNumber", blockNumber, "size", len(curValidators.Arr), "validators", string(common.ToJson(curValidators)))
	validatorExQueue, err := m.convertToValidatorExQueue(blockHash, blockNumber, curValidators)
	if nil != err {
		log.Error("Failed to convertToValidatorExQueue", "blockNumber", blockNumber, "blockHash", blockHash.Hex(), "err", err)
		return
	}
	json := common.ToJson(validatorExQueue)

	dbKey := ValidatorsOfEpochKey.String() + strconv.FormatUint(0, 10)

	m.monitordb.Put([]byte(dbKey), json)
	log.Debug("success to CollectInitEpochValidators", "blockNumber", blockNumber, "blockHash", blockHash.String(), "dbKey", dbKey)
	return
}

// CollectNextEpochVerifiers 保存201名单（包含详细信息）
// CollectNextEpochVerifiers 在上个epoch的结束块高上，收集新的Verifiers名单（201名单，包含详细信息）
// epoch轮数从1开始，key的组成是：VerifiersOfEpochKey+每个epoch轮的开始块高
func (m *Monitor) CollectNextEpochVerifiers(blockHash common.Hash, blockNumber uint64, queryStartNotIrr bool) {
	//获取201名单（只包含必要信息）
	verifiers, err := m.stakingPlugin.GetVerifierArray(blockHash, blockNumber, queryStartNotIrr)
	if nil != err {
		log.Error("Failed to Query Current Round verifiers on stakingPlugin Confirmed When Settletmetn block",
			"blockHash", blockHash.Hex(), "blockNumber", blockNumber, "err", err)
		return
	}
	log.Info("CollectNextEpochVerifiers:", "blockNumber", blockNumber, "size", len(verifiers.Arr), "verifiers", string(common.ToJson(verifiers)))

	validatorExQueue, err := m.convertToValidatorExQueue(blockHash, blockNumber, verifiers)
	if nil != err {
		log.Error("failed to convertToValidatorExQueue", "blockNumber", blockNumber, "blockHash", blockHash.Hex(), "err", err)
		return
	}
	json := common.ToJson(validatorExQueue)

	nextEpoch := xutil.CalculateEpoch(blockNumber) + 1
	if blockNumber == 1 {
		nextEpoch = 1
	}
	dbKey := VerifiersOfEpochKey.String() + strconv.FormatUint(nextEpoch, 10)

	m.monitordb.Put([]byte(dbKey), json)
	log.Debug("success to CollectNextEpochVerifiers", "blockNumber", blockNumber, "blockHash", blockHash.String(), "dbKey", dbKey)
	return
}

// CollectNextEpochValidators 在上个epoch的结束块高上，收集新的validator名单（25名单，包含详细信息）
// epoch轮数从1开始，key的组成是：ValidatorsOfEpochKey+每个epoch轮的开始块高
func (m *Monitor) CollectNextEpochValidators(blockHash common.Hash, blockNumber uint64, queryStartNotIrr bool) {
	nextValidators, err := m.stakingPlugin.GetNextValList(blockHash, blockNumber, queryStartNotIrr)
	if nil != err {
		log.Error("Failed to CollectNextEpochValidators", "blockNumber", blockNumber, "blockHash", blockHash.Hex(), "err", err)
		return
	}
	log.Info("CollectNextEpochValidators:", "blockNumber", blockNumber, "size", len(nextValidators.Arr), "validators", string(common.ToJson(nextValidators)))

	validatorExQueue, err := m.convertToValidatorExQueue(blockHash, blockNumber, nextValidators)
	if nil != err {
		log.Error("Failed to convertToValidatorExQueue", "blockNumber", blockNumber, "blockHash", blockHash.Hex(), "err", err)
		return
	}
	json := common.ToJson(validatorExQueue)

	dbKey := ValidatorsOfEpochKey.String() + strconv.FormatUint(blockNumber+xcom.ElectionDistance(), 10)

	m.monitordb.Put([]byte(dbKey), json)
	log.Debug("success to CollectNextEpochValidators", "blockNumber", blockNumber, "blockHash", blockHash.String(), "dbKey", dbKey)
	return
}

func (m *Monitor) convertToValidatorExQueue(blockHash common.Hash, blockNumber uint64, validatorList *staking.ValidatorArray) (staking.ValidatorExQueue, error) {
	//获取所有质押节点（包含详细信息，25中的提出退出节点，不再存在此列表中）
	currentCandidate, err := m.stakingPlugin.GetCandidateList(blockHash, blockNumber)
	if nil != err {
		log.Error("Failed to GetCandidateList", "blockHash", blockHash.Hex(), "blockNumber", blockNumber, "err", err)
		return nil, err
	}
	validatorExQueue := make(staking.ValidatorExQueue, len(validatorList.Arr))
	for k, v := range validatorList.Arr {
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
				/*validatorExQueue[k].DelegateRewardTotal = cv.DelegateRewardTotal
				validatorExQueue[k].DelegateTotal = cv.DelegateTotal
				validatorExQueue[k].BenefitAddress = cv.BenefitAddress
				validatorExQueue[k].StakingAddress = cv.StakingAddress
				validatorExQueue[k].Website = cv.Website
				validatorExQueue[k].Description = cv.Description
				validatorExQueue[k].ExternalId = cv.ExternalId
				validatorExQueue[k].NodeName = cv.NodeName*/
				//validatorExQueue[k].StakingAddress = cv.StakingAddress
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
				/*validatorExQueue[k].DelegateRewardTotal = (*hexutil.Big)(can.DelegateRewardTotal)
				validatorExQueue[k].DelegateTotal = (*hexutil.Big)(can.DelegateTotal)
				validatorExQueue[k].BenefitAddress = can.BenefitAddress
				validatorExQueue[k].StakingAddress = can.StakingAddress
				validatorExQueue[k].Website = can.Website
				validatorExQueue[k].Description = can.Description
				validatorExQueue[k].ExternalId = can.ExternalId
				validatorExQueue[k].NodeName = can.NodeName*/
				validatorExQueue[k].StakingAddress = can.StakingAddress
				validatorExQueue[k].ProgramVersion = can.ProgramVersion
			}
		}
	}
	return validatorExQueue, nil
}
