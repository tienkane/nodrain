// Package scanner orchestrates the whole pipeline: discover a wallet's Approval
// history from Etherscan, de-duplicate it to the latest grant per (token,
// spender), re-read the live allowance and token metadata on-chain via
// Multicall3, drop anything spent down to zero, then classify and label what
// remains. It exposes one entry point, Scan, used by both the headless and TUI
// front ends.
package scanner

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/tienkane/nodrain/internal/approval"
	"github.com/tienkane/nodrain/internal/chains"
	"github.com/tienkane/nodrain/internal/etherscan"
	"github.com/tienkane/nodrain/internal/evm"
	"github.com/tienkane/nodrain/internal/labels"
)

// Options configures a scan.
type Options struct {
	Chain        chains.Chain
	Owner        string // 0x-prefixed wallet address
	EtherscanKey string
	RPCURL       string // optional; falls back to the chain's default RPC, then the Etherscan proxy
	UseProxy     bool   // force the Etherscan proxy instead of any RPC
	Labels       bool   // enrich unknown spenders from whois.revoke.cash
	CheckRisk    bool   // flag spenders on the approval-exploit list
	HTTPClient   *http.Client
}

// Progress reports a human-readable pipeline stage. It may be nil.
type Progress func(stage string)

func (p Progress) emit(stage string) {
	if p != nil {
		p(stage)
	}
}

// Scan runs the full pipeline and returns the wallet's active approvals, worst
// first.
func Scan(ctx context.Context, opts Options, progress Progress) ([]approval.Approval, error) {
	if !common.IsHexAddress(opts.Owner) {
		return nil, fmt.Errorf("invalid wallet address: %q", opts.Owner)
	}
	if opts.EtherscanKey == "" {
		return nil, errors.New("missing Etherscan API key")
	}
	owner := common.HexToAddress(opts.Owner)

	httpClient := opts.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	es := etherscan.New(opts.Chain.ID, opts.EtherscanKey, httpClient)

	// Kick off the risk-list load concurrently with log discovery; the two are
	// independent network calls and their latency overlaps.
	riskCh := make(chan *labels.RiskDB, 1)
	go func() {
		if !opts.CheckRisk {
			riskCh <- nil
			return
		}
		riskCh <- labels.LoadRiskDB(ctx, httpClient)
	}()

	progress.emit("Fetching approval history…")
	logs, err := es.FetchApprovals(ctx, owner.Hex(), func(fetched int) {
		progress.emit(fmt.Sprintf("Fetching approval history… (%d events)", fetched))
	})
	if err != nil {
		<-riskCh
		switch {
		case errors.Is(err, etherscan.ErrPlanRequired):
			return nil, fmt.Errorf("%s logs are not on Etherscan's free tier — supply a paid API key (chain %q)", opts.Chain.DisplayName, opts.Chain.Name)
		case errors.Is(err, etherscan.ErrInvalidKey):
			return nil, errors.New("Etherscan rejected the API key — check --etherscan-key / ETHERSCAN_API_KEY (free key at https://etherscan.io/myapikey)")
		}
		return nil, err
	}

	cands := dedupApprovals(logs)
	if len(cands) == 0 {
		<-riskCh
		return nil, nil
	}

	progress.emit(fmt.Sprintf("Reading on-chain state for %d approvals…", len(cands)))
	tokens, pairs := buildBatches(cands)
	meta, allowances, err := opts.readChain(ctx, es, tokens, pairs)
	if err != nil {
		<-riskCh
		return nil, err
	}

	riskDB := <-riskCh

	approvals := classify(opts.Chain.ID, owner, cands, meta, allowances)
	if len(approvals) == 0 {
		return nil, nil
	}

	progress.emit("Labeling spenders…")
	applyLabels(ctx, opts, approvals, riskDB, httpClient)

	sortApprovals(approvals)
	for i := range approvals {
		approvals[i].Finalize()
	}
	return approvals, nil
}

// buildBatches derives the unique token list (for metadata) and the ordered
// (token, spender) pairs (for allowances) from the candidates.
func buildBatches(cands []candidate) ([]common.Address, []evm.AllowancePair) {
	seen := make(map[common.Address]bool, len(cands))
	var tokens []common.Address
	pairs := make([]evm.AllowancePair, len(cands))
	for i, c := range cands {
		if !seen[c.Token] {
			seen[c.Token] = true
			tokens = append(tokens, c.Token)
		}
		pairs[i] = evm.AllowancePair{Token: c.Token, Spender: c.Spender}
	}
	return tokens, pairs
}

// readChain reads token metadata and live allowances, trying each available
// transport in preference order (explicit RPC, chain default RPC, then the
// Etherscan proxy) and falling back on failure.
func (opts Options) readChain(ctx context.Context, es *etherscan.Client, tokens []common.Address, pairs []evm.AllowancePair) (map[common.Address]evm.TokenMeta, []*big.Int, error) {
	var lastErr error
	for _, rf := range opts.readerFactories(es) {
		reader, err := rf.make(ctx)
		if err != nil {
			lastErr = err
			continue
		}
		meta, allowErr := evm.BatchTokenMeta(ctx, reader, tokens)
		var allowances []*big.Int
		if allowErr == nil {
			allowances, allowErr = evm.BatchAllowances(ctx, reader, common.HexToAddress(opts.Owner), pairs)
		}
		reader.Close()
		if allowErr != nil {
			lastErr = fmt.Errorf("%s: %w", rf.name, allowErr)
			continue
		}
		return meta, allowances, nil
	}
	if lastErr == nil {
		lastErr = errors.New("no usable RPC transport")
	}
	return nil, nil, fmt.Errorf("reading on-chain state failed: %w", lastErr)
}

type readerFactory struct {
	name string
	make func(ctx context.Context) (evm.Reader, error)
}

func (opts Options) readerFactories(es *etherscan.Client) []readerFactory {
	var fs []readerFactory
	if !opts.UseProxy {
		if opts.RPCURL != "" {
			url := opts.RPCURL
			fs = append(fs, readerFactory{"rpc", func(ctx context.Context) (evm.Reader, error) { return evm.NewRPCReader(ctx, url) }})
		} else if opts.Chain.DefaultRPC != "" {
			url := opts.Chain.DefaultRPC
			fs = append(fs, readerFactory{"default-rpc", func(ctx context.Context) (evm.Reader, error) { return evm.NewRPCReader(ctx, url) }})
		}
	}
	fs = append(fs, readerFactory{"etherscan-proxy", func(ctx context.Context) (evm.Reader, error) { return evm.NewEtherscanReader(es), nil }})
	return fs
}

// classify turns candidates plus on-chain reads into Approvals, dropping any
// whose live allowance is zero.
func classify(chainID int64, owner common.Address, cands []candidate, meta map[common.Address]evm.TokenMeta, allowances []*big.Int) []approval.Approval {
	out := make([]approval.Approval, 0, len(cands))
	for i, c := range cands {
		amount := allowances[i]
		if !approval.IsActive(amount) {
			continue
		}
		m := meta[c.Token]
		a := approval.Approval{
			ChainID:    chainID,
			Owner:      owner.Hex(),
			Token:      c.Token.Hex(),
			Spender:    c.Spender.Hex(),
			Amount:     amount,
			Symbol:     m.Symbol,
			Decimals:   m.Decimals,
			Unlimited:  approval.IsUnlimited(amount, m.TotalSupply),
			IsPermit2:  evm.IsPermit2(c.Spender),
			LastTxHash: c.TxHash,
			LastBlock:  c.Block,
		}
		if a.Symbol == "" {
			a.Symbol = approval.ShortAddress(a.Token)
		}
		out = append(out, a)
	}
	return out
}

// applyLabels resolves spender names and risk flags onto each approval.
func applyLabels(ctx context.Context, opts Options, approvals []approval.Approval, riskDB *labels.RiskDB, httpClient *http.Client) {
	resolver := labels.NewResolver(httpClient, opts.Labels)

	spenders := make([]common.Address, len(approvals))
	for i, a := range approvals {
		spenders[i] = common.HexToAddress(a.Spender)
	}
	labelMap := resolver.ResolveAll(ctx, opts.Chain.ID, spenders)

	for i := range approvals {
		a := &approvals[i]
		if l, ok := labelMap[strings.ToLower(a.Spender)]; ok {
			a.SpenderName = l.Name
			a.SpenderLabel = l.Label
		}
		if name := riskMatch(riskDB, opts.Chain.ID, a.Spender); name != "" {
			a.Risk = approval.RiskMalicious
			a.Reason = "on approval-exploit list: " + name
		} else if a.Unlimited {
			a.Risk = approval.RiskUnlimited
			if a.IsPermit2 {
				a.Reason = "unlimited Permit2 approval — downstream permits persist after revoking"
			} else {
				a.Reason = "unlimited allowance"
			}
		} else {
			a.Risk = approval.RiskNone
		}
	}
}

func riskMatch(db *labels.RiskDB, chainID int64, spender string) string {
	name, _ := db.Lookup(chainID, common.HexToAddress(spender))
	return name
}

// sortApprovals orders worst first: malicious, then unlimited, then the rest;
// ties broken by token symbol then spender for stable output.
func sortApprovals(approvals []approval.Approval) {
	sort.SliceStable(approvals, func(i, j int) bool {
		a, b := approvals[i], approvals[j]
		if a.Risk != b.Risk {
			return a.Risk > b.Risk
		}
		if a.Symbol != b.Symbol {
			return strings.ToLower(a.Symbol) < strings.ToLower(b.Symbol)
		}
		return a.Spender < b.Spender
	})
}
