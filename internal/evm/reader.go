package evm

import (
	"context"
	"fmt"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/tienkane/nodrain/internal/etherscan"
)

// Reader performs read-only eth_calls. Two implementations exist — a real RPC
// node and the Etherscan proxy — so the rest of the package is transport
// agnostic.
type Reader interface {
	// Call executes a view call to `to` with `data` against the latest block.
	Call(ctx context.Context, to common.Address, data []byte) ([]byte, error)
	// MaxBatch caps the inner calls per Multicall3 aggregate3 for this
	// transport. RPC nodes take large POST batches; the Etherscan proxy puts
	// the whole calldata in a GET URL, so it must stay far smaller.
	MaxBatch() int
	// Close releases any underlying connection.
	Close()
}

// rpcReader calls a dedicated JSON-RPC endpoint via go-ethereum's ethclient. It
// is the fast path: Multicall3 collapses hundreds of reads into a few calls.
type rpcReader struct {
	client *ethclient.Client
}

// NewRPCReader dials an Ethereum JSON-RPC endpoint.
func NewRPCReader(ctx context.Context, rpcURL string) (Reader, error) {
	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return nil, fmt.Errorf("dial rpc %q: %w", rpcURL, err)
	}
	return &rpcReader{client: client}, nil
}

func (r *rpcReader) Call(ctx context.Context, to common.Address, data []byte) ([]byte, error) {
	return r.client.CallContract(ctx, ethereum.CallMsg{To: &to, Data: data}, nil)
}

func (r *rpcReader) MaxBatch() int { return 150 }

func (r *rpcReader) Close() { r.client.Close() }

// etherscanReader routes eth_calls through the Etherscan proxy module. It needs
// no RPC URL, but every call is rate-limited, so it leans hard on Multicall3
// batching to keep the call count low.
type etherscanReader struct {
	client *etherscan.Client
}

// NewEtherscanReader wraps an Etherscan client as a Reader.
func NewEtherscanReader(c *etherscan.Client) Reader {
	return &etherscanReader{client: c}
}

func (r *etherscanReader) Call(ctx context.Context, to common.Address, data []byte) ([]byte, error) {
	resHex, err := r.client.EthCall(ctx, to.Hex(), hexutil.Encode(data))
	if err != nil {
		return nil, err
	}
	if resHex == "" || resHex == "0x" {
		return nil, nil
	}
	return hexutil.Decode(resHex)
}

// MaxBatch keeps the aggregate3 calldata small enough to fit in the proxy's GET
// query string (~10 allowance calls stays well under an 8 KB URL limit).
func (r *etherscanReader) MaxBatch() int { return 10 }

func (r *etherscanReader) Close() {}
