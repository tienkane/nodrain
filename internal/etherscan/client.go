// Package etherscan is a small client for the Etherscan V2 unified multichain
// API. One key and a chainid query parameter address 60+ EVM chains through a
// single base URL. nodrain uses two endpoints: logs/getLogs to discover a
// wallet's historical Approval events, and (optionally) proxy/eth_call to read
// live state when no dedicated RPC endpoint is configured.
package etherscan

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// BaseURL is the single V2 endpoint; the network is selected with ?chainid=.
const BaseURL = "https://api.etherscan.io/v2/api"

// Client talks to one chain. It serialises requests and throttles them to stay
// under the free tier's rate limit.
type Client struct {
	APIKey  string
	ChainID int64
	HTTP    *http.Client

	// MinInterval is the minimum spacing between requests. The free tier allows
	// ~3 req/s; ~350ms keeps a safe margin.
	MinInterval time.Duration

	// baseURL is the API endpoint; overridable in tests.
	baseURL  string
	lastCall time.Time
}

// New builds a client with sensible defaults.
func New(chainID int64, apiKey string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &Client{
		APIKey:      apiKey,
		ChainID:     chainID,
		HTTP:        httpClient,
		MinInterval: 350 * time.Millisecond,
		baseURL:     BaseURL,
	}
}

// ErrPlanRequired is returned when the configured key's plan does not cover the
// requested chain (Etherscan moved Base/Optimism/BSC off the free tier).
var ErrPlanRequired = errors.New("etherscan: this chain requires a paid API plan")

// ErrInvalidKey is returned when Etherscan rejects the API key.
var ErrInvalidKey = errors.New("etherscan: invalid API key")

// envelope is the standard Etherscan response wrapper. result is left raw
// because its shape depends on the action.
type envelope struct {
	Status  string          `json:"status"`
	Message string          `json:"message"`
	Result  json.RawMessage `json:"result"`
}

// rawGet performs a throttled GET against the V2 API with the given query
// parameters (chainid and apikey are added automatically) and returns the
// response body. The proxy (JSON-RPC) and standard (envelope) endpoints decode
// it differently, so this stays format-agnostic.
func (c *Client) rawGet(ctx context.Context, params url.Values) ([]byte, error) {
	c.throttle(ctx)

	params.Set("chainid", strconv.FormatInt(c.ChainID, 10))
	params.Set("apikey", c.APIKey)

	endpoint := c.baseURL + "?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("etherscan: http %d: %s", resp.StatusCode, snippet(body))
	}
	return body, nil
}

// get performs a request and decodes the standard Etherscan envelope. A non-"1"
// status is surfaced as an error unless the caller recognises it (e.g. "No
// records found").
func (c *Client) get(ctx context.Context, params url.Values) (*envelope, error) {
	body, err := c.rawGet(ctx, params)
	if err != nil {
		return nil, err
	}
	var env envelope
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("etherscan: decode response: %w (body=%s)", err, snippet(body))
	}
	return &env, nil
}

// throttle blocks until MinInterval has elapsed since the previous call, or the
// context is cancelled.
func (c *Client) throttle(ctx context.Context) {
	if c.MinInterval <= 0 {
		return
	}
	wait := c.MinInterval - time.Since(c.lastCall)
	if wait > 0 {
		select {
		case <-time.After(wait):
		case <-ctx.Done():
		}
	}
	c.lastCall = time.Now()
}

// classifyResultError inspects an Etherscan error message and maps the
// well-known ones to typed errors so callers can react (plan gating, rate-limit
// retry) instead of string-matching themselves.
func classifyResultError(message string, result json.RawMessage) error {
	msg := strings.ToLower(message + " " + string(result))
	switch {
	case strings.Contains(msg, "no records found"):
		return errNoRecords
	case strings.Contains(msg, "rate limit"):
		return errRateLimited
	case strings.Contains(msg, "invalid api key"),
		strings.Contains(msg, "invalid apikey"):
		return ErrInvalidKey
	case strings.Contains(msg, "api plan"),
		strings.Contains(msg, "not authorized"),
		strings.Contains(msg, "upgrade"),
		strings.Contains(msg, "subscription"):
		return ErrPlanRequired
	default:
		return nil
	}
}

var (
	errNoRecords   = errors.New("etherscan: no records found")
	errRateLimited = errors.New("etherscan: rate limited")
)

func snippet(b []byte) string {
	const max = 200
	s := strings.TrimSpace(string(b))
	if len(s) > max {
		return s[:max] + "…"
	}
	return s
}
