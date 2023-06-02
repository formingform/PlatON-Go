package plugin

import (
	"github.com/PlatONnetwork/AppChain-Go/log"
	"github.com/PlatONnetwork/AppChain-Go/x/staking"
	"github.com/PlatONnetwork/AppChain-Go/x/xutil"
	"strconv"
)

func (sk *StakingPlugin) GetHistoryVerifierList(blockNumber uint64) (staking.ValidatorExQueue, error) {

	i := uint64(0)
	if blockNumber != i {
		i = xutil.CalculateEpoch(blockNumber)
	}

	queryNumber := i * xutil.CalcBlocksEachEpoch()
	numStr := strconv.FormatUint(queryNumber, 10)
	log.Debug("wow,GetHistoryVerifierList query number:", "num string", numStr)
	data, err := STAKING_DB.HistoryDB.Get([]byte(VerifierName + numStr))
	if nil != err {
		return nil, err
	}
	var verifierList staking.ValidatorArraySave
	err = rlp.DecodeBytes(data, &verifierList)
	if nil != err {
		return nil, err
	}
	log.Debug("wow,GetHistoryVerifierList", verifierList)

	queue := make(staking.ValidatorExQueue, len(verifierList.Arr))

	var candidateHexQueue staking.CandidateHexQueue

	if queryNumber == 0 {
		data, err := STAKING_DB.HistoryDB.Get([]byte(InitNodeName + numStr))
		if nil != err {
			return nil, err
		}

		err = rlp.DecodeBytes(data, &candidateHexQueue)
		if nil != err {
			return nil, err
		}
		log.Debug("wow,GetHistoryVerifierList candidateHexQueue", candidateHexQueue)
	}
	for i, v := range verifierList.Arr {

		valEx := &staking.ValidatorEx{
			NodeId:              v.NodeId,
			ValidatorTerm:       v.ValidatorTerm,
			DelegateRewardTotal: (*hexutil.Big)(v.DelegateRewardTotal),
			DelegateTotal:       (*hexutil.Big)(v.DelegateTotal),
			StakingBlockNum:     v.StakingBlockNum,
		}
		if queryNumber == 0 {
			for _, vc := range candidateHexQueue {
				if vc.NodeId == v.NodeId {
					valEx.BenefitAddress = vc.BenefitAddress
					valEx.StakingAddress = vc.StakingAddress
					valEx.Website = vc.Website
					valEx.Description = vc.Description
					valEx.ExternalId = vc.ExternalId
					valEx.NodeName = vc.NodeName
					break
				}
			}
		}
		queue[i] = valEx
	}

	return queue, nil
}
