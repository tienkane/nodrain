// Package labels resolves spender addresses to human names and flags spenders
// that appear on a known approval-exploit list. It works fully offline from a
// bundled seed map, and optionally enriches unknown spenders from the public
// (MIT-licensed) whois.revoke.cash dataset.
package labels

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/tienkane/nodrain/internal/text"
)

const whoisBaseURL = "https://whois.revoke.cash/generated"

// whoisConcurrency bounds simultaneous whois requests so a large wallet does not
// hammer the static host.
const whoisConcurrency = 8

// Resolver maps spenders to labels, caching results (including misses) in
// memory. It is safe for concurrent use.
type Resolver struct {
	http        *http.Client
	fetchRemote bool

	mu    sync.Mutex
	cache map[string]Label // key: chainID|lowerAddr; zero Label means "known miss"
}

// NewResolver builds a resolver. When fetchRemote is false only the bundled seed
// map is consulted, so resolution is offline and instant.
func NewResolver(httpClient *http.Client, fetchRemote bool) *Resolver {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 8 * time.Second}
	}
	return &Resolver{
		http:        httpClient,
		fetchRemote: fetchRemote,
		cache:       make(map[string]Label),
	}
}

// ResolveAll labels every spender, returning a map keyed by lowercase address.
// Seed hits are resolved instantly; remaining unknowns are fetched from whois
// concurrently (best effort) when remote lookups are enabled. The supplied
// context bounds the remote phase.
func (r *Resolver) ResolveAll(ctx context.Context, chainID int64, spenders []common.Address) map[string]Label {
	out := make(map[string]Label, len(spenders))
	var unknown []common.Address
	queued := make(map[string]bool) // dedup unknowns so one spender fires one fetch

	for _, sp := range spenders {
		low := lowerAddr(sp)
		if _, done := out[low]; done {
			continue
		}
		if l, ok := seedLookup(chainID, low); ok {
			out[low] = l
			continue
		}
		if l, ok := r.cached(chainID, low); ok {
			if l != (Label{}) {
				out[low] = l
			}
			continue
		}
		if !queued[low] {
			queued[low] = true
			unknown = append(unknown, sp)
		}
	}

	if !r.fetchRemote || len(unknown) == 0 {
		return out
	}

	sem := make(chan struct{}, whoisConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	for _, sp := range unknown {
		wg.Add(1)
		sem <- struct{}{}
		go func(sp common.Address) {
			defer wg.Done()
			defer func() { <-sem }()
			l, _ := r.fetchWhois(ctx, chainID, sp)
			r.store(chainID, lowerAddr(sp), l)
			if l != (Label{}) {
				mu.Lock()
				out[lowerAddr(sp)] = l
				mu.Unlock()
			}
		}(sp)
	}
	wg.Wait()
	return out
}

// fetchWhois looks up a spender in the public whois dataset. The address path
// segment must be EIP-55 checksummed. A 404 is a clean "unlabeled" result.
func (r *Resolver) fetchWhois(ctx context.Context, chainID int64, spender common.Address) (Label, error) {
	url := fmt.Sprintf("%s/spenders/%d/%s.json", whoisBaseURL, chainID, spender.Hex())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Label{}, err
	}
	resp, err := r.http.Do(req)
	if err != nil {
		return Label{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return Label{}, nil
	}
	if resp.StatusCode != http.StatusOK {
		return Label{}, fmt.Errorf("whois %s: %s", url, resp.Status)
	}
	var l Label
	if err := json.NewDecoder(resp.Body).Decode(&l); err != nil {
		return Label{}, err
	}
	// Labels come from a remote host; harden them before they can be rendered.
	l.Name = text.Sanitize(l.Name)
	l.Label = text.Sanitize(l.Label)
	return l, nil
}

func (r *Resolver) cached(chainID int64, low string) (Label, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	l, ok := r.cache[cacheKey(chainID, low)]
	return l, ok
}

func (r *Resolver) store(chainID int64, low string, l Label) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cache[cacheKey(chainID, low)] = l
}

func cacheKey(chainID int64, low string) string { return fmt.Sprintf("%d|%s", chainID, low) }

func lowerAddr(a common.Address) string {
	h := a.Hex()
	// common.Address.Hex() is checksummed mixed-case; lowercase for the key.
	return toLowerHex(h)
}

func toLowerHex(s string) string {
	b := []byte(s)
	for i := range b {
		if b[i] >= 'A' && b[i] <= 'Z' {
			b[i] += 'a' - 'A'
		}
	}
	return string(b)
}
