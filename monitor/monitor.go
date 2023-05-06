package monitor

import (
	"github.com/PlatONnetwork/PlatON-Go/common"
	"github.com/PlatONnetwork/PlatON-Go/common/json"
	"github.com/PlatONnetwork/PlatON-Go/log"
	"math/big"
)

type MonitorDbKey int

const (
	EmbedTransferKey MonitorDbKey = iota
	CreatedContractKey
	SuicidedContractKey
	ProxyPatternKey
)

// 定义 MonitorDbKey 类型的方法 String(), 返回字符串。
func (dbKey MonitorDbKey) String() string {
	return [...]string{"EmbedTransferTx"}[dbKey]
}

type EmbedTransfer struct {
	TxHash common.Hash    `json:"txHash"`
	From   common.Address `json:"from"`
	To     common.Address `json:"to"`
	Amount *big.Int       `json:"amount"`
}

type ProxyPattern struct {
	Proxy          *contract `json:"proxy"`
	Implementation *contract `json:"implementation"`
}

func toJson(data interface{}) ([]byte, error) {
	if jsonStr, err := json.Marshal(data); err != nil {
		return nil, err
	} else {
		return jsonStr, nil
	}
}

func parseJson(data []byte, v interface{}) error {
	if err := json.Unmarshal(data, v); err != nil {
		return err
	} else {
		return nil
	}
}

func CollectEmbedTransfer(blockNumber uint64, txHash common.Hash, from, to common.Address, amount *big.Int) {
	log.Debug("CollectEmbedTransferTx", "blockNumber", blockNumber, "txHash", txHash.Hex(), "from", from.Bech32(), "to", to.Bech32(), "amount", amount)

	dbKey := EmbedTransferKey.String() + "_" + txHash.String()
	data, err := getMonitorDB().GetLevelDB([]byte(dbKey))
	if nil != err && err != ErrNotFound {
		log.Error("failed to load embed transfers", "err", err)
		return
	}

	var embedTransferList []*EmbedTransfer
	if data != nil {
		err = parseJson(data, &embedTransferList)
		if nil != err {
			log.Error("failed to decode json to embed transfers", "err", err)
			return
		}
	}
	embedTransfer := new(EmbedTransfer)
	embedTransfer.TxHash = txHash
	embedTransfer.From = from
	embedTransfer.To = to
	embedTransfer.Amount = amount

	embedTransferList = append(embedTransferList, embedTransfer)

	jsonStr, err := toJson(embedTransferList)
	if nil != err {
		log.Error("failed to encode embed transfers to json", "err", err)
		return
	}

	getMonitorDB().PutLevelDB([]byte(dbKey), jsonStr)
	log.Debug("save embed transfers success")
}

func GetEmbedTransfer(blockNumber uint64, txHash common.Hash) []*EmbedTransfer {
	log.Debug("GetEmbedTransfer", "blockNumber", blockNumber, "txHash", txHash.Hex())

	dbKey := EmbedTransferKey.String() + "_" + txHash.String()
	data, err := getMonitorDB().GetLevelDB([]byte(dbKey))
	if nil != err {
		log.Error("failed to load embed transfers", "err", err)
		return nil
	}

	var embedTransferList []*EmbedTransfer
	err = parseJson(data, &embedTransferList)
	if nil != err {
		log.Error("failed to decode json to embed transfers", "err", err)
		return nil
	}
	return embedTransferList
}

func CollectCreatedContract(stateDB StateDB, newContractAddr common.Address) {
	dbKey := CreatedContractKey.String() + "_" + stateDB.TxHash().String()
	data, err := getMonitorDB().GetLevelDB([]byte(dbKey))
	if nil != err && err != ErrNotFound {
		log.Error("failed to load created contracts", "err", err)
		return
	}
	var createdContractList []*contract
	if data != nil {
		err = parseJson(data, &createdContractList)
		if nil != err {
			log.Error("failed to decode json to created contracts", "err", err)
			return
		}
	}

	contract := NewContract(newContractAddr, stateDB.GetCode(newContractAddr))
	createdContractList = append(createdContractList, contract)

	jsonStr, err := toJson(createdContractList)
	if nil != err {
		log.Error("failed to encode created contracts to json", "err", err)
		return
	}
	getMonitorDB().PutLevelDB([]byte(dbKey), jsonStr)
	log.Debug("save created contracts success")
}

func GetCreatedContract(blockNumber uint64, txHash common.Hash) []*contract {
	log.Debug("GetCreatedContract", "blockNumber", blockNumber, "txHash", txHash.Hex())

	dbKey := CreatedContractKey.String() + "_" + txHash.String()
	data, err := getMonitorDB().GetLevelDB([]byte(dbKey))
	if nil != err && err != ErrNotFound {
		log.Error("failed to load created contracts", "err", err)
		return nil
	}
	var createdContractList []*contract
	if data != nil {
		err = parseJson(data, &createdContractList)
		if nil != err {
			log.Error("failed to decode json to created contracts", "err", err)
			return nil
		}
	}

	log.Debug("get created contracts success")
	return createdContractList
}

func CollectSuicidedContract(stateDB StateDB, suicidedContractAddr common.Address) {
	dbKey := SuicidedContractKey.String() + "_" + stateDB.TxHash().String()
	data, err := getMonitorDB().GetLevelDB([]byte(dbKey))
	if nil != err && err != ErrNotFound {
		log.Error("failed to load suicided contracts", "err", err)
		return
	}

	var suicidedContractList []*contract
	if data != nil {
		err = parseJson(data, &suicidedContractList)
		if nil != err {
			log.Error("failed to decode json to suicided contracts", "err", err)
			return
		}
	}

	suicidedContract := new(contract)
	suicidedContract.Address = suicidedContractAddr

	suicidedContractList = append(suicidedContractList, suicidedContract)

	jsonStr, err := toJson(suicidedContractList)
	if nil != err {
		log.Error("failed to encode suicided contracts to json", "err", err)
		return
	}
	getMonitorDB().PutLevelDB([]byte(dbKey), jsonStr)
	log.Debug("save suicided contracts success")
}

func GetSuicidedContract(blockNumber uint64, txHash common.Hash) []*contract {
	log.Debug("GetSuicidedContract", "blockNumber", blockNumber, "txHash", txHash.Hex())

	dbKey := SuicidedContractKey.String() + "_" + txHash.String()
	data, err := getMonitorDB().GetLevelDB([]byte(dbKey))
	if nil != err && err != ErrNotFound {
		log.Error("failed to load suicided contracts", "err", err)
		return nil
	}
	var suicidedContractList []*contract
	if data != nil {
		err = parseJson(data, &suicidedContractList)
		if nil != err {
			log.Error("failed to decode json to suicided contracts", "err", err)
			return nil
		}
	}

	log.Debug("get suicided contracts success")
	return suicidedContractList
}

// InspectProxyPattern 根据交易txHash发现代理关系
func InspectProxyPattern(stateDB StateDB, callerAddr, targetAddr common.Address, callerCode, targetCode []byte) {
	caller := NewContract(callerAddr, callerCode)
	target := NewContract(targetAddr, targetCode)

	if !caller.matchProxyPattern() || target.getType() == GENERAL {
		log.Debug("found a delegate call, but not match the proxy pattern", "caller", callerAddr, "target", targetAddr)
		return
	}

	storage := stateDB.GetState(callerAddr, []byte(implSlotZeppelinos))
	implAddr := common.BytesToAddress(storage)
	if implAddr != targetAddr {
		log.Debug("found a delegate call, but not match the proxy pattern", "caller", callerAddr, "target", targetAddr)
		return
	}

	dbKey := ProxyPatternKey.String() + "_" + stateDB.TxHash().String()
	data, err := getMonitorDB().GetLevelDB([]byte(dbKey))
	if nil != err && err != ErrNotFound {
		log.Error("failed to load proxy patterns", "err", err)
		return
	}

	var proxyPatternList []*ProxyPattern
	if data != nil {
		err = parseJson(data, &proxyPatternList)
		if nil != err {
			log.Error("failed to decode json to proxy patterns", "err", err)
			return
		}
	}

	proxyPatternList = append(proxyPatternList, &ProxyPattern{Proxy: caller, Implementation: target})

	jsonStr, err := toJson(proxyPatternList)
	if nil != err {
		log.Error("failed to encode proxy patterns to json", "err", err)
		return
	}
	getMonitorDB().PutLevelDB([]byte(dbKey), jsonStr)
	log.Debug("save proxy patterns success")
}

func GetProxyPattern(blockNumber uint64, txHash common.Hash) []*ProxyPattern {
	log.Debug("GetProxyPattern", "blockNumber", blockNumber, "txHash", txHash.Hex())

	dbKey := ProxyPatternKey.String() + "_" + txHash.String()
	data, err := getMonitorDB().GetLevelDB([]byte(dbKey))
	if nil != err && err != ErrNotFound {
		log.Error("failed to load proxy patterns", "err", err)
		return nil
	}
	var proxyPatternList []*ProxyPattern
	if data != nil {
		err = parseJson(data, &proxyPatternList)
		if nil != err {
			log.Error("failed to decode json to proxy patterns", "err", err)
			return nil
		}
	}

	log.Debug("get proxy patterns success")
	return proxyPatternList
}
