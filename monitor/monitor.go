package monitor

import (
	"github.com/PlatONnetwork/PlatON-Go/common"
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
	proxyPatternMapKey
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
	common.ParseJson(data, &embedTransferList)
	return embedTransferList
}

type ContractRef interface {
	Address() common.Address
}

func (m *Monitor) CollectCreatedContractInfo(txHash common.Hash, contractInfo *ContractInfo) {
	log.Debug("CollectCreatedContractInfo", "txHash", txHash.Hex(), "contractInfo", string(common.ToJson(contractInfo)))

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
	common.ParseJson(data, &createdContractInfoList)

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
	common.ParseJson(data, &suicidedContractInfoList)

	suicidedContract := new(ContractInfo)
	suicidedContract.Address = suicidedContractAddr

	suicidedContractInfoList = append(suicidedContractInfoList, suicidedContract)

	json := common.ToJson(suicidedContractInfoList)
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
	common.ParseJson(data, &suicidedContractInfoList)

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
	common.ParseJson(data, &proxyPatternList)
	proxyPatternList = append(proxyPatternList, &ProxyPattern{Proxy: proxyContractInfo, Implementation: implementationContractInfo})

	json := common.ToJson(proxyPatternList)
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
	common.ParseJson(data, &proxyPatternMap)
	proxyPatternMap[proxyContractInfo.Address] = implementationContractInfo.Address

	json = common.ToJson(proxyPatternMap)
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
	common.ParseJson(data, &proxyPatternMap)
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
	common.ParseJson(data, &proxyPatternList)

	log.Debug("get proxy patterns success")
	return proxyPatternList
}
