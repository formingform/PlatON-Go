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
	ProxyContractKey
)

// 定义 MonitorDbKey 类型的方法 String(), 返回字符串。
func (dbKey MonitorDbKey) String() string {
	return [...]string{"EmbedTransferTx"}[dbKey]
}

type EmbedTransfer struct {
	TxHash common.Hash    `json:"txHash,omitempty"`
	From   common.Address `json:"from,omitempty"`
	To     common.Address `json:"to,omitempty"`
	Amount *big.Int       `json:"amount,omitempty"`
}

type CreatedContract struct {
	ContractType ContractType   `json:"contractType,omitempty"`
	Address      common.Address `json:"address" gencodec:"required"`
	BinHex       string         `json:"binHex,omitempty"`
}

type SuicidedContract struct {
	Address common.Address `json:"address" gencodec:"required"`
}

type ProxyContract struct {
	ProxyAddress     common.Address `json:"proxyAddress,omitempty"`
	ProxyBinHex      string         `json:"proxyBinHex,omitempty"`
	ImplementAddress common.Address `json:"implementAddress" gencodec:"required"`
	ImplementBinHex  string         `json:"implementBinHex,omitempty"`
	ImplementType    ContractType   `json:"implementType,omitempty"`
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
		log.Error("failed to load EmbedTransfer", "err", err)
		return
	}

	var embedTransferList []*EmbedTransfer
	if data != nil {
		err = parseJson(data, &embedTransferList)
		if nil != err {
			log.Error("failed to decode json to EmbedTransfer", "err", err)
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
		log.Error("failed to encode EmbedTransfer to json", "err", err)
		return
	}

	getMonitorDB().PutLevelDB([]byte(dbKey), jsonStr)
	log.Debug("save EmbedTransfer success")
}

func GetEmbedTransfer(blockNumber uint64, txHash common.Hash) []*EmbedTransfer {
	log.Debug("GetEmbedTransfer", "blockNumber", blockNumber, "txHash", txHash.Hex())

	dbKey := EmbedTransferKey.String() + "_" + txHash.String()
	data, err := getMonitorDB().GetLevelDB([]byte(dbKey))
	if nil != err {
		log.Error("failed to load EmbedTransfer", "err", err)
		return nil
	}

	var embedTransferList []*EmbedTransfer
	err = parseJson(data, &embedTransferList)
	if nil != err {
		log.Error("failed to decode json to EmbedTransfer", "err", err)
		return nil
	}
	return embedTransferList
}

func CollectCreatedContract(txHash common.Hash, newContractAddr common.Address, contractCode []byte) {
	dbKey := CreatedContractKey.String() + "_" + txHash.String()
	data, err := getMonitorDB().GetLevelDB([]byte(dbKey))
	if nil != err && err != ErrNotFound {
		log.Error("failed to load CreatedContract", "err", err)
		return
	}
	var createdContractList []*CreatedContract
	if data != nil {
		err = parseJson(data, &createdContractList)
		if nil != err {
			log.Error("failed to decode json to CreatedContract", "err", err)
			return
		}
	}

	contractBin := contractBin{code: contractCode}
	createdContract := new(CreatedContract)
	createdContract.Address = newContractAddr
	createdContract.BinHex = contractBin.Hex()
	createdContract.ContractType = contractBin.getContractType()

	createdContractList = append(createdContractList, createdContract)

	jsonStr, err := toJson(createdContractList)
	if nil != err {
		log.Error("failed to encode CreatedContract to json", "err", err)
		return
	}
	getMonitorDB().PutLevelDB([]byte(dbKey), jsonStr)
	log.Debug("save CreatedContract success")
}

func GetCreatedContract(blockNumber uint64, txHash common.Hash) []*CreatedContract {
	log.Debug("GetCreatedContract", "blockNumber", blockNumber, "txHash", txHash.Hex())

	dbKey := CreatedContractKey.String() + "_" + txHash.String()
	data, err := getMonitorDB().GetLevelDB([]byte(dbKey))
	if nil != err && err != ErrNotFound {
		log.Error("failed to load CreatedContract", "err", err)
		return nil
	}
	var createdContractList []*CreatedContract
	if data != nil {
		err = parseJson(data, &createdContractList)
		if nil != err {
			log.Error("failed to decode json to CreatedContract", "err", err)
			return nil
		}
	}

	log.Debug("get CreatedContract success")
	return createdContractList
}

func CollectSuicidedContract(txHash common.Hash, suicidedContractAddr common.Address) {
	dbKey := SuicidedContractKey.String() + "_" + txHash.String()
	data, err := getMonitorDB().GetLevelDB([]byte(dbKey))
	if nil != err && err != ErrNotFound {
		log.Error("failed to load SuicidedContract", "err", err)
		return
	}

	var suicidedContractList []*SuicidedContract
	if data != nil {
		err = parseJson(data, &suicidedContractList)
		if nil != err {
			log.Error("failed to decode json to SuicidedContract", "err", err)
			return
		}
	}

	suicidedContract := new(SuicidedContract)
	suicidedContract.Address = suicidedContractAddr

	suicidedContractList = append(suicidedContractList, suicidedContract)

	jsonStr, err := toJson(suicidedContractList)
	if nil != err {
		log.Error("failed to encode SuicidedContract to json", "err", err)
		return
	}
	getMonitorDB().PutLevelDB([]byte(dbKey), jsonStr)
	log.Debug("save SuicidedContract success")
}

func GetSuicidedContract(blockNumber uint64, txHash common.Hash) []*SuicidedContract {
	log.Debug("GetSuicidedContract", "blockNumber", blockNumber, "txHash", txHash.Hex())

	dbKey := SuicidedContractKey.String() + "_" + txHash.String()
	data, err := getMonitorDB().GetLevelDB([]byte(dbKey))
	if nil != err && err != ErrNotFound {
		log.Error("failed to load SuicidedContract", "err", err)
		return nil
	}
	var suicidedContractList []*SuicidedContract
	if data != nil {
		err = parseJson(data, &suicidedContractList)
		if nil != err {
			log.Error("failed to decode json to SuicidedContract", "err", err)
			return nil
		}
	}

	log.Debug("get SuicidedContract success")
	return suicidedContractList
}

// 根据交易txHash发现代理关系
func CollectProxyContract(txHash common.Hash, proxyContractAddr, implementContractAddr common.Address, proxyContractCode, implementContractCode []byte) {
	proxyContractBin := contractBin{code: proxyContractCode}
	if len(proxyContractBin.code) > 0 && !proxyContractBin.isProxyContract() {
		log.Debug("found a delegate call, the caller is not a proxy contract", "caller", proxyContractAddr)
		return
	}
	implementContractBin := contractBin{code: implementContractCode}
	if len(implementContractBin.code) > 0 {
		// it's a contract
		implementContractType := implementContractBin.getContractType()
		if implementContractType == GENERAL {
			log.Debug("found a delegate call, the target is a general contract", "target", implementContractCode)
			return
		}
	}

	dbKey := ProxyContractKey.String() + "_" + txHash.String()
	data, err := getMonitorDB().GetLevelDB([]byte(dbKey))
	if nil != err && err != ErrNotFound {
		log.Error("failed to load ProxyContract", "err", err)
		return
	}

	var proxyContractList []*ProxyContract
	if data != nil {
		err = parseJson(data, &proxyContractList)
		if nil != err {
			log.Error("failed to decode json to ProxyContract", "err", err)
			return
		}
	}

	proxyContract := new(ProxyContract)
	proxyContract.ProxyAddress = proxyContractAddr

	proxyContract.ProxyBinHex = proxyContractBin.Hex()
	proxyContract.ImplementAddress = implementContractAddr
	proxyContract.ImplementBinHex = implementContractBin.Hex()
	proxyContract.ImplementType = implementContractBin.getContractType()

	proxyContractList = append(proxyContractList, proxyContract)

	jsonStr, err := toJson(proxyContractList)
	if nil != err {
		log.Error("failed to encode ProxyContract to json", "err", err)
		return
	}
	getMonitorDB().PutLevelDB([]byte(dbKey), jsonStr)
	log.Debug("save ProxyContract success")
}

func GetProxyContract(blockNumber uint64, txHash common.Hash) []*ProxyContract {
	log.Debug("GetProxyContract", "blockNumber", blockNumber, "txHash", txHash.Hex())

	dbKey := ProxyContractKey.String() + "_" + txHash.String()
	data, err := getMonitorDB().GetLevelDB([]byte(dbKey))
	if nil != err && err != ErrNotFound {
		log.Error("failed to load ProxyContract", "err", err)
		return nil
	}
	var proxyContractList []*ProxyContract
	if data != nil {
		err = parseJson(data, &proxyContractList)
		if nil != err {
			log.Error("failed to decode json to ProxyContract", "err", err)
			return nil
		}
	}

	log.Debug("get ProxyContract success")
	return proxyContractList
}
