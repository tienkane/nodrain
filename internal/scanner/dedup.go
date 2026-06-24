package scanner

import (
	"sort"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"

	"github.com/tienkane/nodrain/internal/etherscan"
)

// candidate is a unique (token, spender) approval discovered from logs, carrying
// the provenance of the most recent Approval event for that pair.
type candidate struct {
	Token   common.Address
	Spender common.Address
	Block   uint64
	LogIdx  uint64
	TxHash  string
}

// dedupApprovals reduces a wallet's Approval logs to one candidate per (token,
// spender), keeping the most recent event by (blockNumber, logIndex). An earlier
// unlimited grant may have since been lowered or revoked, so only the latest log
// is a meaningful starting point — and the live allowance is re-read regardless.
func dedupApprovals(logs []etherscan.Log) []candidate {
	latest := make(map[string]candidate, len(logs))
	for _, l := range logs {
		if len(l.Topics) < 3 || l.Address == "" {
			continue
		}
		token := common.HexToAddress(l.Address)
		spender := common.HexToAddress(topicToAddress(l.Topics[2]))
		block := hexToUint64(l.BlockNumber)
		logIdx := hexToUint64(l.LogIndex)

		key := strings.ToLower(token.Hex()) + "|" + strings.ToLower(spender.Hex())
		if prev, ok := latest[key]; ok {
			if block < prev.Block || (block == prev.Block && logIdx <= prev.LogIdx) {
				continue
			}
		}
		latest[key] = candidate{
			Token:   token,
			Spender: spender,
			Block:   block,
			LogIdx:  logIdx,
			TxHash:  l.TransactionHash,
		}
	}

	out := make([]candidate, 0, len(latest))
	for _, c := range latest {
		out = append(out, c)
	}
	// Stable, deterministic ordering before classification re-sorts by risk.
	sort.Slice(out, func(i, j int) bool {
		if out[i].Block != out[j].Block {
			return out[i].Block > out[j].Block
		}
		return out[i].LogIdx > out[j].LogIdx
	})
	return out
}

// topicToAddress extracts the 20-byte address from a 32-byte indexed topic.
func topicToAddress(topic string) string {
	h := strings.TrimPrefix(topic, "0x")
	if len(h) < 40 {
		return "0x" + h
	}
	return "0x" + h[len(h)-40:]
}

func hexToUint64(s string) uint64 {
	s = strings.TrimPrefix(strings.TrimSpace(s), "0x")
	if s == "" {
		return 0
	}
	n, err := strconv.ParseUint(s, 16, 64)
	if err != nil {
		return 0
	}
	return n
}
