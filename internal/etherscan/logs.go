package etherscan

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// ApprovalTopic0 is keccak256("Approval(address,address,uint256)") — the topic
// every ERC-20 Approval event is indexed under.
const ApprovalTopic0 = "0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925"

// logPageSize is Etherscan's hard cap on getLogs results per query.
const logPageSize = 1000

// maxRateLimitRetries bounds how many times a single page is retried after a
// rate-limit response.
const maxRateLimitRetries = 5

// Log mirrors one entry from logs/getLogs. Every field is a hex string.
type Log struct {
	Address          string   `json:"address"`
	Topics           []string `json:"topics"`
	Data             string   `json:"data"`
	BlockNumber      string   `json:"blockNumber"`
	TimeStamp        string   `json:"timeStamp"`
	LogIndex         string   `json:"logIndex"`
	TransactionHash  string   `json:"transactionHash"`
	TransactionIndex string   `json:"transactionIndex"`
}

// FetchApprovals returns every ERC-20 Approval log emitted with owner as the
// granting party, across all token contracts on the client's chain. It filters
// purely by topics (address omitted: topic0 = Approval, topic1 = padded owner).
//
// It pages primarily by BLOCK RANGE: each request asks for the next 1000 logs
// from a moving fromBlock. This sidesteps Etherscan's page*offset <= 10,000
// window, which would otherwise hard-abort a scan of a wallet with more than
// 10,000 Approval events. When a single block saturates a full page (the highest
// block does not advance), it walks forward by page NUMBER within that block so
// logs beyond the first 1000 in that block are not skipped. The boundary block
// is re-fetched between range steps (cheap), and the caller de-duplicates
// (token,spender) so the overlap is harmless. The progress callback, if
// non-nil, is invoked with the running count after each page.
func (c *Client) FetchApprovals(ctx context.Context, owner string, progress func(fetched int)) ([]Log, error) {
	var all []Log
	var fromBlock uint64
	page := 1
	for {
		logs, err := c.fetchApprovalPage(ctx, owner, fromBlock, page)
		if err != nil {
			if errors.Is(err, errNoRecords) {
				break
			}
			return nil, err
		}
		all = append(all, logs...)
		if progress != nil {
			progress(len(all))
		}
		if len(logs) < logPageSize {
			break // partial page => no more logs
		}
		// A full page means there may be more. Advance fromBlock past the
		// highest block seen and restart paging there. If the highest block has
		// not moved, one block holds >1000 logs, so keep fromBlock and step the
		// page number to read the rest of that block instead of skipping it.
		if next := highestBlock(logs); next > fromBlock {
			fromBlock = next
			page = 1
		} else {
			page++
		}
	}
	return all, nil
}

func (c *Client) fetchApprovalPage(ctx context.Context, owner string, fromBlock uint64, page int) ([]Log, error) {
	params := url.Values{}
	params.Set("module", "logs")
	params.Set("action", "getLogs")
	params.Set("fromBlock", strconv.FormatUint(fromBlock, 10))
	params.Set("toBlock", "latest")
	params.Set("topic0", ApprovalTopic0)
	params.Set("topic0_1_opr", "and")
	params.Set("topic1", AddressTopic(owner))
	params.Set("page", strconv.Itoa(page))
	params.Set("offset", fmt.Sprintf("%d", logPageSize))

	for attempt := 0; ; attempt++ {
		env, err := c.get(ctx, params)
		if err != nil {
			return nil, err
		}
		if env.Status != "1" {
			if cerr := classifyResultError(env.Message, env.Result); cerr != nil {
				if errors.Is(cerr, errRateLimited) && attempt < maxRateLimitRetries {
					if !backoff(ctx, attempt) {
						return nil, ctx.Err()
					}
					continue
				}
				return nil, cerr
			}
			return nil, fmt.Errorf("etherscan getLogs: %s", env.Message)
		}
		var logs []Log
		if err := json.Unmarshal(env.Result, &logs); err != nil {
			return nil, fmt.Errorf("etherscan getLogs: decode result: %w", err)
		}
		return logs, nil
	}
}

// highestBlock returns the largest block number across the logs. Block numbers
// come back as hex strings.
func highestBlock(logs []Log) uint64 {
	var max uint64
	for _, l := range logs {
		s := strings.TrimPrefix(strings.TrimSpace(l.BlockNumber), "0x")
		if n, err := strconv.ParseUint(s, 16, 64); err == nil && n > max {
			max = n
		}
	}
	return max
}

// AddressTopic left-pads a 20-byte address to a 32-byte, lowercase topic value
// as required by getLogs topic1/topic2 filters.
func AddressTopic(addr string) string {
	h := strings.ToLower(strings.TrimPrefix(addr, "0x"))
	if len(h) >= 64 {
		return "0x" + h[len(h)-64:]
	}
	return "0x" + strings.Repeat("0", 64-len(h)) + h
}

// backoff sleeps for an increasing interval, returning false if the context is
// cancelled first.
func backoff(ctx context.Context, attempt int) bool {
	d := time.Duration(attempt+1) * 500 * time.Millisecond
	select {
	case <-time.After(d):
		return true
	case <-ctx.Done():
		return false
	}
}
