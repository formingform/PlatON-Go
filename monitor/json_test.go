package monitor

import (
	"github.com/PlatONnetwork/AppChain-Go/common"
	"github.com/PlatONnetwork/AppChain-Go/common/hexutil"
	"github.com/PlatONnetwork/AppChain-Go/rlp"
	"github.com/PlatONnetwork/AppChain-Go/x/staking"
	"testing"
)

func TestRlpEncode(t *testing.T) {
	stakingReward := staking.Reward{
		PackageReward: common.Big3,
		StakingReward: common.Big32,
		YearNum:       4 + 1,
		YearStartNum:  0,
		YearEndNum:    1,
		RemainEpoch:   4,
		AvgPackTime:   2 * 1000,
	}
	dataReward, _ := rlp.EncodeToBytes(stakingReward)
	t.Logf("dataReward:{}", hexutil.Encode(dataReward))
}

func TestJsonEncode(t *testing.T) {
	stakingReward := staking.Reward{
		PackageReward: common.Big3,
		StakingReward: common.Big32,
		YearNum:       4 + 1,
		YearStartNum:  0,
		YearEndNum:    1,
		RemainEpoch:   4,
		AvgPackTime:   2 * 1000,
	}
	dataReward := toJson(stakingReward)
	t.Logf("dataReward:{}", hexutil.Encode(dataReward))
}
