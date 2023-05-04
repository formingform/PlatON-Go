package monitor

import (
	"fmt"
	"github.com/PlatONnetwork/PlatON-Go/common"
	"math/big"
	"testing"
)

func TestCollectEmbedTransfer(t *testing.T) {
	SetDBPath("/home/joey/monitor_db")

	blockNumber := uint64(1000)
	txHash := common.Hash{0x01, 0x02, 0x03}
	from := common.Address{0x01}
	to := common.Address{0x02}
	amount := big.NewInt(1999)

	CollectEmbedTransfer(blockNumber, txHash, from, to, amount)

	txs := GetEmbedTransfer(blockNumber, txHash)
	jsonBytes, err := toJson(txs)
	if err != nil {
		t.Error(err)
	}
	fmt.Println("txs=", string(jsonBytes))
}
