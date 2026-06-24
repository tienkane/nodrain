package labels

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/tienkane/nodrain/internal/text"
)

const (
	exploitListBase = "https://raw.githubusercontent.com/RevokeCash/approval-exploit-list/main"
	exploitCacheTTL = 24 * time.Hour
	exploitFetchers = 8
)

// RiskDB flags spenders that appear on the MIT-licensed RevokeCash
// approval-exploit-list. Lookups are O(1); a nil DB simply flags nothing, which
// is the graceful-degradation path when the list cannot be loaded.
type RiskDB struct {
	// byKey maps "chainID|lowerAddr" to the exploit name.
	byKey map[string]string
}

// exploitEntry is one exploit record from exploits/{slug}.json.
type exploitEntry struct {
	Name      string `json:"name"`
	Addresses []struct {
		ChainID int64  `json:"chainId"`
		Address string `json:"address"`
	} `json:"addresses"`
}

// cachedExploit is the flattened on-disk cache shape.
type cachedExploit struct {
	ChainID int64  `json:"chainId"`
	Address string `json:"address"`
	Name    string `json:"name"`
}

// Lookup returns the exploit name flagging this spender, if any.
func (db *RiskDB) Lookup(chainID int64, addr common.Address) (string, bool) {
	if db == nil || db.byKey == nil {
		return "", false
	}
	name, ok := db.byKey[cacheKey(chainID, lowerAddr(addr))]
	return name, ok
}

// add canonicalises an exploit entry's address through go-ethereum (so the key
// matches Lookup's, which also goes through common.Address) and sanitises the
// remote-sourced name before it can ever be rendered. Malformed addresses are
// skipped rather than inserted under an unreachable key.
func (db *RiskDB) add(e cachedExploit) {
	if !common.IsHexAddress(e.Address) {
		return
	}
	db.byKey[cacheKey(e.ChainID, lowerAddr(common.HexToAddress(e.Address)))] = text.Sanitize(e.Name)
}

// LoadRiskDB returns a populated RiskDB, preferring a fresh on-disk cache and
// otherwise fetching the exploit list. It is best effort: on any failure it
// returns an empty (non-nil) DB and a nil error so a scan never fails because
// the risk list was unreachable.
func LoadRiskDB(ctx context.Context, httpClient *http.Client) *RiskDB {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 8 * time.Second}
	}
	empty := &RiskDB{byKey: map[string]string{}}

	if db := loadExploitCache(); db != nil {
		return db
	}

	entries, complete, err := fetchExploitList(ctx, httpClient)
	if err != nil || len(entries) == 0 {
		return empty
	}

	db := &RiskDB{byKey: make(map[string]string, len(entries))}
	flat := make([]cachedExploit, 0, len(entries))
	for _, e := range entries {
		db.add(e)
		flat = append(flat, e)
	}
	// Only persist a COMPLETE fetch. Caching a partial list (some per-slug
	// fetches failed or the context was cancelled) would make the malicious-
	// spender check silently fail-open for the cache's entire TTL.
	if complete {
		writeExploitCache(flat)
	}
	return db
}

// fetchExploitList pulls index.json then every exploit record concurrently and
// flattens them into (chainID, address, name) tuples. The bool reports whether
// EVERY record was fetched successfully — callers must not persist a partial
// result.
func fetchExploitList(ctx context.Context, c *http.Client) ([]cachedExploit, bool, error) {
	var slugs []string
	if err := getJSON(ctx, c, exploitListBase+"/index.json", &slugs); err != nil {
		return nil, false, err
	}

	var (
		mu       sync.Mutex
		wg       sync.WaitGroup
		out      []cachedExploit
		failures int
		sem      = make(chan struct{}, exploitFetchers)
	)
	for _, slug := range slugs {
		wg.Add(1)
		sem <- struct{}{}
		go func(slug string) {
			defer wg.Done()
			defer func() { <-sem }()
			var e exploitEntry
			if err := getJSON(ctx, c, fmt.Sprintf("%s/exploits/%s.json", exploitListBase, slug), &e); err != nil {
				mu.Lock()
				failures++
				mu.Unlock()
				return
			}
			mu.Lock()
			for _, a := range e.Addresses {
				out = append(out, cachedExploit{ChainID: a.ChainID, Address: a.Address, Name: e.Name})
			}
			mu.Unlock()
		}(slug)
	}
	wg.Wait()
	return out, failures == 0, nil
}

func getJSON(ctx context.Context, c *http.Client, url string, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(v)
}

// --- on-disk cache (best effort; cache errors are ignored) ---

func exploitCachePath() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "nodrain", "exploit-list.json"), nil
}

func loadExploitCache() *RiskDB {
	path, err := exploitCachePath()
	if err != nil {
		return nil
	}
	info, err := os.Stat(path)
	if err != nil || time.Since(info.ModTime()) > exploitCacheTTL {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var flat []cachedExploit
	if err := json.Unmarshal(data, &flat); err != nil || len(flat) == 0 {
		return nil
	}
	db := &RiskDB{byKey: make(map[string]string, len(flat))}
	for _, e := range flat {
		db.add(e)
	}
	return db
}

func writeExploitCache(flat []cachedExploit) {
	path, err := exploitCachePath()
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	data, err := json.Marshal(flat)
	if err != nil {
		return
	}
	_ = os.WriteFile(path, data, 0o644)
}
