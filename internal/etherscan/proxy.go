package etherscan

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// proxyResponse covers both shapes the proxy module can return: the JSON-RPC
// reply ({result|error}) on success, and the standard {status,message,result}
// envelope that the gateway emits for rate limits and plan/key errors.
type proxyResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Result  string `json:"result"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// EthCall performs a read-only eth_call through Etherscan's proxy module and
// returns the raw result hex ("0x…"). This is the zero-config fallback used
// when no dedicated RPC endpoint is configured. It is rate-limited like every
// other Etherscan call, so callers should prefer batching (Multicall3) to keep
// the number of proxy calls small.
func (c *Client) EthCall(ctx context.Context, to, data string) (string, error) {
	params := url.Values{}
	params.Set("module", "proxy")
	params.Set("action", "eth_call")
	params.Set("to", to)
	params.Set("data", data)
	params.Set("tag", "latest")

	for attempt := 0; ; attempt++ {
		body, err := c.rawGet(ctx, params)
		if err != nil {
			return "", err
		}
		var r proxyResponse
		if err := json.Unmarshal(body, &r); err != nil {
			return "", fmt.Errorf("etherscan eth_call: decode: %w (body=%s)", err, snippet(body))
		}

		// Detect an error in either shape: a JSON-RPC error object, the
		// status="0" envelope, or a result that isn't call-output hex.
		var apiErr string
		switch {
		case r.Error != nil:
			apiErr = r.Error.Message
		case r.Status == "0":
			apiErr = strings.TrimSpace(r.Message + " " + r.Result)
		case r.Result != "" && !strings.HasPrefix(r.Result, "0x"):
			apiErr = r.Result
		}

		if apiErr != "" {
			cerr := classifyResultError(apiErr, nil)
			if errors.Is(cerr, errRateLimited) && attempt < maxRateLimitRetries {
				if !backoff(ctx, attempt) {
					return "", ctx.Err()
				}
				continue
			}
			if cerr != nil && !errors.Is(cerr, errNoRecords) {
				return "", cerr // ErrPlanRequired / ErrInvalidKey
			}
			return "", fmt.Errorf("etherscan eth_call: %s", apiErr)
		}
		return r.Result, nil
	}
}
