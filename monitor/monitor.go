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
	Proxy          *ContractInfo `json:"proxy"`
	Implementation *ContractInfo `json:"implementation"`
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

type ContractRef interface {
	Address() common.Address
}

func CollectCreatedContractInfo(txHash common.Hash, contractInfo *ContractInfo) {
	dbKey := CreatedContractKey.String() + "_" + txHash.String()
	data, err := getMonitorDB().GetLevelDB([]byte(dbKey))
	if nil != err && err != ErrNotFound {
		log.Error("failed to load created contracts", "err", err)
		return
	}
	var createdContractInfoList []*ContractInfo
	if data != nil {
		err = parseJson(data, &createdContractInfoList)
		if nil != err {
			log.Error("failed to decode json to created contracts", "err", err)
			return
		}
	}

	createdContractInfoList = append(createdContractInfoList, contractInfo)

	jsonStr, err := toJson(createdContractInfoList)
	if nil != err {
		log.Error("failed to encode created contracts to json", "err", err)
		return
	}
	getMonitorDB().PutLevelDB([]byte(dbKey), jsonStr)
	log.Debug("save created contracts success")
}

func GetCreatedContractInfo(blockNumber uint64, txHash common.Hash) []*ContractInfo {
	log.Debug("GetCreatedContract", "blockNumber", blockNumber, "txHash", txHash.Hex())

	dbKey := CreatedContractKey.String() + "_" + txHash.String()
	data, err := getMonitorDB().GetLevelDB([]byte(dbKey))
	if nil != err && err != ErrNotFound {
		log.Error("failed to load created contracts", "err", err)
		return nil
	}
	var createdContractInfoList []*ContractInfo
	if data != nil {
		err = parseJson(data, &createdContractInfoList)
		if nil != err {
			log.Error("failed to decode json to created contracts", "err", err)
			return nil
		}
	}

	log.Debug("get created contracts success")
	return createdContractInfoList
}

func CollectSuicidedContract(txHash common.Hash, suicidedContractAddr common.Address) {
	dbKey := SuicidedContractKey.String() + "_" + txHash.String()
	data, err := getMonitorDB().GetLevelDB([]byte(dbKey))
	if nil != err && err != ErrNotFound {
		log.Error("failed to load suicided contracts", "err", err)
		return
	}

	var suicidedContractInfoList []*ContractInfo
	if data != nil {
		err = parseJson(data, &suicidedContractInfoList)
		if nil != err {
			log.Error("failed to decode json to suicided contracts", "err", err)
			return
		}
	}

	suicidedContract := new(ContractInfo)
	suicidedContract.Address = suicidedContractAddr

	suicidedContractInfoList = append(suicidedContractInfoList, suicidedContract)

	jsonStr, err := toJson(suicidedContractInfoList)
	if nil != err {
		log.Error("failed to encode suicided contracts to json", "err", err)
		return
	}
	getMonitorDB().PutLevelDB([]byte(dbKey), jsonStr)
	log.Debug("save suicided contracts success")
}

func GetSuicidedContract(blockNumber uint64, txHash common.Hash) []*ContractInfo {
	log.Debug("GetSuicidedContract", "blockNumber", blockNumber, "txHash", txHash.Hex())

	dbKey := SuicidedContractKey.String() + "_" + txHash.String()
	data, err := getMonitorDB().GetLevelDB([]byte(dbKey))
	if nil != err && err != ErrNotFound {
		log.Error("failed to load suicided contracts", "err", err)
		return nil
	}
	var suicidedContractInfoList []*ContractInfo
	if data != nil {
		err = parseJson(data, &suicidedContractInfoList)
		if nil != err {
			log.Error("failed to decode json to suicided contracts", "err", err)
			return nil
		}
	}

	log.Debug("get suicided contracts success")
	return suicidedContractInfoList
}

// CollectProxyPattern 根据交易txHash发现代理关系
func CollectProxyPattern(txHash common.Hash, proxyContractInfo, implementationContractInfo *ContractInfo) {
	dbKey := ProxyPatternKey.String() + "_" + txHash.String()
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

	proxyPatternList = append(proxyPatternList, &ProxyPattern{Proxy: proxyContractInfo, Implementation: implementationContractInfo})

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
