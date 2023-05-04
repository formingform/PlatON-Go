package monitor

import (
	"encoding/hex"
	"github.com/PlatONnetwork/PlatON-Go/common/hexutil"
	"golang.org/x/crypto/sha3"
	"strings"
)

type ContractType int

const (
	ERC20 ContractType = iota //EnumIndex = 0
	ERC721
	ERC1155
	GENERAL
)

// 定义 MonitorDbKey 类型的方法 String(), 返回字符串。
func (t ContractType) String() string {
	return [...]string{"ERC20", "ERC721", "ERC1155", "GENERAL"}[t-1]
}

var (
	// 返回方法签名的hash, 这个hash将出现在合约的bin中，为了方便比较，返回hash的string(不含0x前缀)
	evmFuncHash = func(funcName string) string {
		prifix := sha3.NewLegacyKeccak256()
		prifix.Write([]byte(funcName))
		bin := hexutil.Encode(prifix.Sum(nil)[:4])
		return bin[2:]
	}
)

type contractBin struct {
	code []byte
	hex  string
}

func (bin *contractBin) Hex() string {
	if len(bin.hex) == 0 {
		bin.hex = hex.EncodeToString(bin.code)
	}
	return bin.hex
}

func (bin contractBin) implements(funcName string) bool {
	return strings.Index(bin.hex, evmFuncHash(funcName)) != -1
}

func (bin contractBin) implementsAll(funcNames ...string) bool {
	for _, funcName := range funcNames {
		if !bin.implements(funcName) {
			return false
		}
	}
	return true
}

func (bin contractBin) implementsAnyOf(funcNames ...string) bool {
	for _, funcName := range funcNames {
		if bin.implements(funcName) {
			return true
		}
	}
	return false
}

func (bin *contractBin) isERC20() bool {
	return bin.implementsAll(
		"totalSupply()",
		"balanceOf(address)",
		"transfer(address,uint256)",
		"transferFrom(address,address,uint256)",
		"approve(address,uint256)",
		"allowance(address,address)")
}

func (bin *contractBin) isERC721() bool {
	return bin.implementsAll(
		"balanceOf(address)",
		"ownerOf(uint256)",
		"approve(address,uint256)",
		"getApproved(uint256)",
		"setApprovalForAll(address,bool)",
		"isApprovedForAll(address,address)",
		"transferFrom(address,address,uint256)",
		"safeTransferFrom(address,address,uint256)",
		"safeTransferFrom(address,address,uint256,bytes)")
}

func (bin *contractBin) isERC1155() bool {
	return bin.implementsAll(
		"safeTransferFrom(address,address,uint256,uint256,bytes)",
		"safeBatchTransferFrom(address,address,uint256[],uint256[],bytes)",
		"balanceOf(address,uint256)",
		"balanceOfBatch(address[],uint256[])",
		"setApprovalForAll(address,bool)",
		"isApprovedForAll(address,address)")
}

func (bin *contractBin) isProxyContract() bool {
	return !bin.isERC20() && !bin.isERC721() && !bin.isERC1155() && bin.implementsAll(
		"fallback()")
}

func (bin *contractBin) getContractType() ContractType {
	if bin.isERC20() {
		return ERC20
	} else if bin.isERC721() {
		return ERC721
	} else if bin.isERC1155() {
		return ERC1155
	} else {
		return GENERAL
	}
}
