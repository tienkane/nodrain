package evm

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// Multicall3 is deployed at the same deterministic address on every chain
// nodrain supports.
var Multicall3 = common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11")

// Call3 mirrors Multicall3's input tuple. Field order and types must match the
// ABI components exactly.
type Call3 struct {
	Target       common.Address
	AllowFailure bool
	CallData     []byte
}

// Result mirrors Multicall3's output tuple.
type Result struct {
	Success    bool
	ReturnData []byte
}

// aggregate3 packs and executes one Multicall3 batch (no chunking).
func aggregate3(ctx context.Context, r Reader, calls []Call3) ([]Result, error) {
	data, err := multicall3ABI.Pack("aggregate3", calls)
	if err != nil {
		return nil, fmt.Errorf("pack aggregate3: %w", err)
	}
	raw, err := r.Call(ctx, Multicall3, data)
	if err != nil {
		return nil, fmt.Errorf("multicall3 call: %w", err)
	}
	if len(raw) == 0 {
		return nil, fmt.Errorf("multicall3 at %s returned no data (is it deployed on this chain?)", Multicall3.Hex())
	}
	var out []Result
	if err := multicall3ABI.UnpackIntoInterface(&out, "aggregate3", raw); err != nil {
		return nil, fmt.Errorf("unpack aggregate3: %w", err)
	}
	return out, nil
}

// Aggregate3 runs calls through Multicall3 in chunks sized for the reader's
// transport and returns the results in the same order. Every inner call uses
// allowFailure, so one reverting token never sinks the batch.
func Aggregate3(ctx context.Context, r Reader, calls []Call3) ([]Result, error) {
	chunkSize := r.MaxBatch()
	if chunkSize < 1 {
		chunkSize = 1
	}
	out := make([]Result, 0, len(calls))
	for start := 0; start < len(calls); start += chunkSize {
		end := min(start+chunkSize, len(calls))
		res, err := aggregate3(ctx, r, calls[start:end])
		if err != nil {
			return nil, err
		}
		if len(res) != end-start {
			return nil, fmt.Errorf("multicall3: expected %d results, got %d", end-start, len(res))
		}
		out = append(out, res...)
	}
	return out, nil
}

// TokenMeta is a token's display metadata. Found is false when even the symbol
// could not be read (e.g. the address is not an ERC-20).
type TokenMeta struct {
	Symbol      string
	Decimals    uint8
	TotalSupply *big.Int
	Found       bool
}

// BatchTokenMeta reads symbol/decimals/totalSupply for each unique token in one
// batched round of Multicall3 calls.
func BatchTokenMeta(ctx context.Context, r Reader, tokens []common.Address) (map[common.Address]TokenMeta, error) {
	symData, _ := erc20ABI.Pack("symbol")
	decData, _ := erc20ABI.Pack("decimals")
	supData, _ := erc20ABI.Pack("totalSupply")

	calls := make([]Call3, 0, len(tokens)*3)
	for _, t := range tokens {
		calls = append(calls,
			Call3{Target: t, AllowFailure: true, CallData: symData},
			Call3{Target: t, AllowFailure: true, CallData: decData},
			Call3{Target: t, AllowFailure: true, CallData: supData},
		)
	}

	results, err := Aggregate3(ctx, r, calls)
	if err != nil {
		return nil, err
	}

	meta := make(map[common.Address]TokenMeta, len(tokens))
	for i, t := range tokens {
		sym := results[i*3]
		dec := results[i*3+1]
		sup := results[i*3+2]

		m := TokenMeta{Decimals: 18}
		if sym.Success {
			if s := DecodeSymbol(sym.ReturnData); s != "" {
				m.Symbol = s
				m.Found = true
			}
		}
		if dec.Success {
			if d, ok := DecodeUint8(dec.ReturnData); ok {
				m.Decimals = d
			}
		}
		if sup.Success {
			if v, ok := DecodeUint256(sup.ReturnData); ok {
				m.TotalSupply = v
			}
		}
		meta[t] = m
	}
	return meta, nil
}

// AllowancePair identifies one allowance to read.
type AllowancePair struct {
	Token   common.Address
	Spender common.Address
}

// BatchAllowances reads the live allowance(owner, spender) for each pair in
// batched Multicall3 calls, returning amounts in the same order. A failed or
// empty read yields zero.
func BatchAllowances(ctx context.Context, r Reader, owner common.Address, pairs []AllowancePair) ([]*big.Int, error) {
	calls := make([]Call3, len(pairs))
	for i, p := range pairs {
		data, err := erc20ABI.Pack("allowance", owner, p.Spender)
		if err != nil {
			return nil, fmt.Errorf("pack allowance: %w", err)
		}
		calls[i] = Call3{Target: p.Token, AllowFailure: true, CallData: data}
	}

	results, err := Aggregate3(ctx, r, calls)
	if err != nil {
		return nil, err
	}

	out := make([]*big.Int, len(pairs))
	for i, res := range results {
		if !res.Success {
			out[i] = big.NewInt(0)
			continue
		}
		if v, ok := DecodeUint256(res.ReturnData); ok {
			out[i] = v
		} else {
			out[i] = big.NewInt(0)
		}
	}
	return out, nil
}
