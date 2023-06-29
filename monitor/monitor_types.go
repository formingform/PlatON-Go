package monitor

import (
	"github.com/PlatONnetwork/PlatON-Go/common"
	"math/big"
)

type ContractRef interface {
	Address() common.Address
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

// 隐含的PPOS交易。
// 通常来说，PPOS交易是有用户直接发给相应内置合约的。
// 不过，用户自己部署的智能合约，也可以调用内置合约。这样，区块中的交易，看上去是个普通合约调用，但实际上，隐式的调用了PPOS合约（包括多次调用），产生了隐式的PPOS交易（多条），SCAN需要知道这些隐式PPOS交易的相关信息
type PPOSTx struct {
	From   common.Address
	To     common.Address
	Input  []byte
	Result []byte
}

type ImplicitPPOSTx struct {
	//key=原始交易hash
	//value=PPOSTx
	PPOSTxMap map[common.Hash][]*PPOSTx
}
