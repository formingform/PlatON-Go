package monitor

import (
	"github.com/PlatONnetwork/AppChain-Go/common"
	"math/big"
	"testing"
)

func TestJsonEncode(t *testing.T) {
	epochView := EpochView{
		PackageReward:     new(big.Int).SetUint64(uint64(12312312312312312)),
		StakingReward:     common.Big32,
		ChainAge:          4 + 1,
		YearStartBlockNum: 0,
		YearEndBlockNum:   1,
		RemainEpoch:       4,
		AvgPackTime:       2 * 1000,
	}
	dataReward := ToJson(epochView)
	t.Logf("dataReward:%s", string(dataReward))
}

func TestGetReceiptExt(t *testing.T) {
	fields := map[string]interface{}{
		"transactionHash":  common.Hash{0x01},
		"transactionIndex": int64(12),

		"gasUsed": uint64(12),

		"contractAddress": nil,
	}
	t.Logf("dataReward:%s", string(ToJson(fields)))
}
func TestAccountView(t *testing.T) {
	accountView := &AccountView{
		Account:     common.Address{0x1},
		FreeBalance: new(big.Int).SetUint64(uint64(491292)),
		// 锁仓锁定的余额
		RestrictingPlanLockedAmount: new(big.Int).SetUint64(uint64(491292)),
		// 锁仓欠释放的余额
		RestrictingPlanPledgeAmount: new(big.Int).SetUint64(uint64(491292)),
		// 锁定结束的委托金，资金来源是用户账户余额
		DelegationUnLockedFreeBalance: new(big.Int).SetUint64(uint64(491292)),
		// 锁定结束的委托金，资金来源是锁仓计划。用户来领取委托金时，一部分可以直接释放到用户账户；一部分可能重新回到锁仓计划中
		DelegationUnLockedRestrictingPlanAmount: new(big.Int).SetUint64(uint64(491292)),
		// 委托冻结冻结中明细
		DelegationLockedItems: make([]DelegationLockedItem, 0),
	}

	lockItem := DelegationLockedItem{
		ExpiredEpoch:          uint32(12),
		FreeBalance:           new(big.Int).SetUint64(uint64(491292)),
		RestrictingPlanAmount: new(big.Int).SetUint64(uint64(491292)),
	}
	accountView.DelegationLockedItems = append(accountView.DelegationLockedItems, lockItem)
	t.Logf("accountView:%s", string(ToJson(accountView)))
}
