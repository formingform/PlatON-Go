package monitor

import (
	"github.com/PlatONnetwork/PlatON-Go/common"
	"github.com/PlatONnetwork/PlatON-Go/x/restricting"
	"github.com/PlatONnetwork/PlatON-Go/x/staking"
	"github.com/PlatONnetwork/PlatON-Go/x/xcom"
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
type ImplicitPPOSTx struct {
	From common.Address `json:"from"`
	To   common.Address `json:"to"`
	//FnCode uint16         `json:"fnCode"`
	//FnParams   []interface{}  `json:"fnParams"`
	InputHex   string `json:"inputHex"`
	LogDataHex string `json:"logDataHex"`
	//ErrCode  uint16         `json:"errCode"` //内置合约执成功时为0，其它为业务错误代码

}

type Intf_stakingPlugin interface {
	GetNextValList(blockHash common.Hash, blockNumber uint64, isCommit bool) (*staking.ValidatorArray, error)
	GetCurrValList(common.Hash, uint64, bool) (*staking.ValidatorArray, error)
	GetVerifierArray(common.Hash, uint64, bool) (*staking.ValidatorArray, error)
	GetCandidateList(common.Hash, uint64) (staking.CandidateHexQueue, error)
	GetCandidateInfo(common.Hash, *big.Int) (*staking.Candidate, error)
	GetNodeVersion(blockHash common.Hash) (staking.ValidatorExQueue, error)
	GetGetDelegationLockCompactInfo(blockHash common.Hash, blockNumber uint64, delAddr common.Address) (*staking.DelegationLockHex, error)
}
type Intf_restrictingPlugin interface {
	MustGetRestrictingInfoByDecode(state xcom.StateDB, account common.Address) ([]byte, restricting.RestrictingInfo, *common.BizError)
}
