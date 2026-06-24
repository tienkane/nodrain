package evm

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestDecodeSymbolBytes32(t *testing.T) {
	// MKR-style: "MKR" packed into a right-padded bytes32.
	var b [32]byte
	copy(b[:], "MKR")
	if got := DecodeSymbol(b[:]); got != "MKR" {
		t.Errorf("bytes32 symbol = %q, want MKR", got)
	}
}

func TestDecodeSymbolString(t *testing.T) {
	// ABI string "USDC": offset(32) | length(4) | data.
	out := make([]byte, 0, 96)
	offset := make([]byte, 32)
	offset[31] = 0x20
	length := make([]byte, 32)
	length[31] = 4
	data := make([]byte, 32)
	copy(data, "USDC")
	out = append(out, offset...)
	out = append(out, length...)
	out = append(out, data...)
	if got := DecodeSymbol(out); got != "USDC" {
		t.Errorf("string symbol = %q, want USDC", got)
	}
}

func TestDecodeSymbolStripsControlChars(t *testing.T) {
	var b [32]byte
	copy(b[:], "BAD\x07\x01TKN")
	if got := DecodeSymbol(b[:]); got != "BADTKN" {
		t.Errorf("symbol = %q, want BADTKN (control chars stripped)", got)
	}
}

func TestDecodeUint8(t *testing.T) {
	word := make([]byte, 32)
	word[31] = 6
	if d, ok := DecodeUint8(word); !ok || d != 6 {
		t.Errorf("DecodeUint8 = %d, %v; want 6, true", d, ok)
	}
	if _, ok := DecodeUint8([]byte{0x06}); ok {
		t.Error("DecodeUint8 of short input should fail")
	}
}

func TestRevokeCalldataSelector(t *testing.T) {
	spender := common.HexToAddress("0x1111111254EEB25477B68fb85Ed929f73A960582")
	data, err := RevokeCalldata(spender)
	if err != nil {
		t.Fatal(err)
	}
	// approve(address,uint256) selector.
	wantSel, _ := hex.DecodeString("095ea7b3")
	if !bytes.Equal(data[:4], wantSel) {
		t.Errorf("selector = %x, want 095ea7b3", data[:4])
	}
	if len(data) != 4+32+32 {
		t.Errorf("calldata len = %d, want 68", len(data))
	}
	// The amount word must be all zeros (revoke).
	if !bytes.Equal(data[36:68], make([]byte, 32)) {
		t.Errorf("amount word not zero: %x", data[36:68])
	}
}

func TestIsPermit2(t *testing.T) {
	if !IsPermit2(common.HexToAddress("0x000000000022D473030F116dDEE9F6B43aC78BA3")) {
		t.Error("canonical Permit2 not recognised")
	}
	if IsPermit2(common.HexToAddress("0xdead")) {
		t.Error("non-Permit2 flagged as Permit2")
	}
}
