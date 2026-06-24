package evm

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// RevokeCalldata returns the calldata for approve(spender, 0). The transaction
// that carries it must target the TOKEN contract, not the spender. This is the
// only "write" payload nodrain produces, and it is emitted unsigned for the user
// to submit from their own wallet.
func RevokeCalldata(spender common.Address) ([]byte, error) {
	return erc20ABI.Pack("approve", spender, big.NewInt(0))
}

// RevokeCalldataHex is RevokeCalldata as a 0x-prefixed hex string.
func RevokeCalldataHex(spender common.Address) (string, error) {
	data, err := RevokeCalldata(spender)
	if err != nil {
		return "", err
	}
	return hexutil.Encode(data), nil
}
