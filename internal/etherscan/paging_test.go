package etherscan

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func makeFakeLogs(n int, blockHex string) []Log {
	logs := make([]Log, n)
	for i := range logs {
		logs[i] = Log{
			Address:         "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48",
			Topics:          []string{ApprovalTopic0, AddressTopic("0xdead"), AddressTopic("0xbeef")},
			Data:            "0x",
			BlockNumber:     blockHex,
			LogIndex:        fmt.Sprintf("0x%x", i),
			TransactionHash: fmt.Sprintf("0x%064x", i),
		}
	}
	return logs
}

// TestFetchApprovalsPaginatesByBlockRange exercises the multi-page loop: a full
// page (1000 logs, max block 0x64) must trigger a second request that advances
// fromBlock to that highest block; a short second page must stop the loop.
func TestFetchApprovalsPaginatesByBlockRange(t *testing.T) {
	var (
		mu    sync.Mutex
		froms []string
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		from := r.URL.Query().Get("fromBlock")
		mu.Lock()
		froms = append(froms, from)
		mu.Unlock()

		var logs []Log
		if from == "0" {
			logs = makeFakeLogs(logPageSize, "0x64") // full page, highest block = 100
		} else {
			logs = makeFakeLogs(3, "0x65") // partial page => stop
		}
		_ = json.NewEncoder(w).Encode(struct {
			Status  string `json:"status"`
			Message string `json:"message"`
			Result  []Log  `json:"result"`
		}{"1", "OK", logs})
	}))
	defer srv.Close()

	c := New(1, "key", srv.Client())
	c.baseURL = srv.URL
	c.MinInterval = 0

	logs, err := c.FetchApprovals(context.Background(), "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045", nil)
	if err != nil {
		t.Fatalf("FetchApprovals: %v", err)
	}
	if len(logs) != logPageSize+3 {
		t.Errorf("got %d logs, want %d", len(logs), logPageSize+3)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(froms) != 2 {
		t.Fatalf("made %d requests, want 2: %v", len(froms), froms)
	}
	if froms[0] != "0" || froms[1] != "100" {
		t.Errorf("fromBlock sequence = %v, want [0 100]", froms)
	}
}

// TestFetchApprovalsPagesWithinSaturatedBlock confirms that when one block holds
// more than logPageSize logs, paging walks the page number within that block
// instead of skipping the overflow.
func TestFetchApprovalsPagesWithinSaturatedBlock(t *testing.T) {
	var (
		mu   sync.Mutex
		reqs [][2]string // (fromBlock, page)
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		from, page := q.Get("fromBlock"), q.Get("page")
		mu.Lock()
		reqs = append(reqs, [2]string{from, page})
		mu.Unlock()

		var logs []Log
		switch page {
		case "1":
			logs = makeFakeLogs(logPageSize, "0x64") // 1000 logs, all in block 100
		case "2":
			logs = makeFakeLogs(500, "0x64") // rest of block 100, partial => stop
		}
		_ = json.NewEncoder(w).Encode(struct {
			Status  string `json:"status"`
			Message string `json:"message"`
			Result  []Log  `json:"result"`
		}{"1", "OK", logs})
	}))
	defer srv.Close()

	c := New(1, "key", srv.Client())
	c.baseURL = srv.URL
	c.MinInterval = 0

	logs, err := c.FetchApprovals(context.Background(), "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045", nil)
	if err != nil {
		t.Fatalf("FetchApprovals: %v", err)
	}
	// page1@from=0 (1000) + page1@from=100 (1000, re-fetch) + page2@from=100 (500).
	if len(logs) != 2500 {
		t.Errorf("got %d logs, want 2500", len(logs))
	}
	mu.Lock()
	defer mu.Unlock()
	want := [][2]string{{"0", "1"}, {"100", "1"}, {"100", "2"}}
	if len(reqs) != len(want) {
		t.Fatalf("requests = %v, want %v", reqs, want)
	}
	for i := range want {
		if reqs[i] != want[i] {
			t.Errorf("request %d = %v, want %v", i, reqs[i], want[i])
		}
	}
}

// TestFetchApprovalsNoRecords confirms a "No records found" status is a clean
// empty result, not an error.
func TestFetchApprovalsNoRecords(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(struct {
			Status  string `json:"status"`
			Message string `json:"message"`
			Result  []Log  `json:"result"`
		}{"0", "No records found", nil})
	}))
	defer srv.Close()

	c := New(1, "key", srv.Client())
	c.baseURL = srv.URL
	c.MinInterval = 0

	logs, err := c.FetchApprovals(context.Background(), "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045", nil)
	if err != nil {
		t.Fatalf("FetchApprovals: %v", err)
	}
	if len(logs) != 0 {
		t.Errorf("got %d logs, want 0", len(logs))
	}
}
