package staking

import (
	"github.com/PlatONnetwork/AppChain-Go/common"
	"github.com/PlatONnetwork/AppChain-Go/common/hexutil"
	"github.com/PlatONnetwork/AppChain-Go/p2p/discover"
	"math/big"
)

// some consensus round validators or current epoch validators
type ValidatorArraySave struct {
	// the round start blockNumber or epoch start blockNumber
	Start uint64
	// the round end blockNumber or epoch blockNumber
	End uint64
	// the round validators or epoch validators
	Arr ValidatorQueueSave
}

type ValidatorQueueSave []*ValidatorSave
type ValidatorSave struct {
	ValidatorTerm       uint32 // Validator's term in the consensus round
	NodeId              discover.NodeID
	DelegateRewardTotal *big.Int
	DelegateTotal       *big.Int
	StakingBlockNum     uint64
}

type SlashNodeQueue []*SlashNodeData
type SlashNodeData struct {
	// the nodeId will be slashed
	NodeId discover.NodeID
	// the amount of von with slashed
	Amount *big.Int
}

type CandidateVersionQueue []*CandidateVersion
type CandidateVersion struct {
	NodeId         discover.NodeID
	ProgramVersion uint32
}

type Reward struct {
	PackageReward *big.Int
	StakingReward *big.Int
	YearNum       uint32
	YearStartNum  uint64
	YearEndNum    uint64
	RemainEpoch   uint32
	AvgPackTime   uint64
}

type RewardReturn struct {
	CurPackageReward  *hexutil.Big
	CurStakingReward  *hexutil.Big
	NextPackageReward *hexutil.Big
	NextStakingReward *hexutil.Big
	PackageReward     *hexutil.Big
	StakingReward     *hexutil.Big
	YearNum           uint32
	YearStartNum      uint64
	YearEndNum        uint64
	RemainEpoch       uint32
	AvgPackTime       uint64
}

type TransBlockReturnQueue []*TransBlockReturn

type TransBlockReturn struct {
	TxHash     string
	From       common.Address
	To         common.Address
	TransDatas []TransData
}

type TransBlock struct {
	TransHashStr []string
}

type TransInput struct {
	From       []byte
	To         []byte
	TransDatas []TransData
}

type TransData struct {
	Input string
	Code  string
}
