package monitor

import (
	"github.com/PlatONnetwork/AppChain-Go/common/json"
	"github.com/PlatONnetwork/AppChain-Go/log"
)

func toJson(obj interface{}) []byte {
	if obj == nil {
		return []byte{}
	}
	bs, err := json.Marshal(obj)
	if err != nil {
		log.Error("cannot marshal object", "err", err)
		return []byte{}
	} else {
		return bs
	}

}

func parseJson(bs []byte, objRefer interface{}) {
	if len(bs) == 0 {
		return
	}
	err := json.Unmarshal(bs, objRefer)
	if err != nil {
		log.Error("cannot unmarshal to object", "err", err)
	}
}
