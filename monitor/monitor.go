package monitor

import (
	"bytes"
	"github.com/PlatONnetwork/PlatON-Go/common"
	"github.com/PlatONnetwork/PlatON-Go/common/hexutil"
	"github.com/PlatONnetwork/PlatON-Go/core/state"
	"github.com/PlatONnetwork/PlatON-Go/log"
	"math/big"
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

/*type ProxyInfo struct {
	Proxy            common.Address `json:"proxy,omitempty"`
	Implementation   common.Address `json:"implementation,omitempty"`
	TokenName        string         `json:"tokenName,omitempty"`
	TokenSymbol      string         `json:"tokenSymbol,omitempty"`
	TokenDecimals    uint8          `json:"tokenDecimals,omitempty"`
	TokenTotalSupply *big.Int       `json:"tokenTotalSupply,omitempty"`
}*/

func (m *Monitor) CollectEmbedTransfer(blockNumber uint64, txHash common.Hash, from, to common.Address, amount *big.Int) {
	log.Debug("CollectEmbedTransferTx", "blockNumber", blockNumber, "txHash", txHash.String(), "from", from.Bech32(), "to", to.Bech32(), "amount", amount)

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
		log.Error("failed to load embed transfers", "err", err)
		return nil
	}

	var embedTransferList []*EmbedTransfer
	common.ParseJson(data, &embedTransferList)
	return embedTransferList
}

func (m *Monitor) CollectCreatedContractInfo(txHash common.Hash, contractInfo *ContractInfo) {
	log.Debug("CollectCreatedContractInfo", "txHash", txHash.String(), "contractInfo", string(common.ToJson(contractInfo)))

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
	if nil != err && err != ErrNotFound {
		log.Error("failed to load created contracts", "err", err)
		return nil
	}
	var createdContractInfoList []*ContractInfo
	common.ParseJson(data, &createdContractInfoList)

	log.Debug("GetCreatedContract success", "txHash", txHash.String(), "json", string(data))
	return createdContractInfoList
}

func (m *Monitor) CollectSuicidedContractInfo(txHash common.Hash, suicidedContractAddr common.Address) {
	log.Debug("CollectSuicidedContractInfo", "txHash", txHash.String(), "suicidedContractAddr", suicidedContractAddr.String())

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
	if nil != err && err != ErrNotFound {
		log.Error("failed to load suicided contracts", "err", err)
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

	log.Debug("CollectProxyPattern save proxy relation flag success", "txHash", txHash.String(), "proxy", proxyContractInfo.Address.String(), "implementation", implementationContractInfo.Address.String())

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

func (m *Monitor) IsProxied(proxy, impl common.Address) bool {
	flagDbKey := proxyPatternFlagKey.String() + "_" + proxy.String() + "_" + impl.String()
	flagBytes, err := m.monitordb.Get([]byte(flagDbKey))
	if err == ErrNotFound {
		return false
	}

	if nil != err && err != ErrNotFound {
		log.Error("failed to load proxy flag", "err", err)
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
	if nil != err && err != ErrNotFound {
		log.Error("failed to load proxy patterns", "err", err)
		return nil
	}
	var proxyPatternList []*ProxyPattern
	common.ParseJson(data, &proxyPatternList)

	log.Debug("GetProxyPatternList success", "txHash", txHash.String(), "json", string(data))
	return proxyPatternList
}

// 收集隐式的ppos交易数据
// 新方式（暂时未启用
func (m *Monitor) CollectImplicitPPOSTx(blockNumber uint64, txHash common.Hash, from, to common.Address, input, result []byte) {
	log.Debug("CollectImplicitPPOSTx", "blockNumber", blockNumber, "txHash", txHash.String(), "from", from.String(), "to", to.String(), "input", hexutil.Encode(input), "result", hexutil.Encode(result))

	dbKey := ImplicitPPOSTxKey.String() + "_" + txHash.String()
	data, err := m.monitordb.Get([]byte(dbKey))
	if nil != err && err != ErrNotFound {
		log.Error("failed to load data from local db", "err", err)
		return
	}

	var implicitPPOSTx *ImplicitPPOSTx
	if len(data) == 0 {
		implicitPPOSTx = &ImplicitPPOSTx{PPOSTxMap: make(map[common.Hash][]*PPOSTx)}
	} else {
		common.ParseJson(data, &implicitPPOSTx)
	}
	contractTx := &PPOSTx{From: from, To: to, Input: input, Result: result}
	implicitPPOSTx.PPOSTxMap[txHash] = append(implicitPPOSTx.PPOSTxMap[txHash], contractTx)
	json := common.ToJson(implicitPPOSTx)
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
	if nil != err && err != ErrNotFound {
		log.Error("failed to load data from local db", "err", err)
		return nil
	}

	var implicitPPOSTxList []*ImplicitPPOSTx
	common.ParseJson(data, &implicitPPOSTxList)

	log.Debug("GetImplicitPPOSTx success", "txHash", txHash.String(), "json", string(data))
	return implicitPPOSTxList
}
