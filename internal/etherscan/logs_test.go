package etherscan

import "testing"

func TestAddressTopic(t *testing.T) {
	got := AddressTopic("0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045")
	want := "0x000000000000000000000000d8da6bf26964af9d7eed9e03e53415d37aa96045"
	if got != want {
		t.Errorf("AddressTopic = %q, want %q", got, want)
	}
	if len(got) != 66 { // 0x + 64 hex chars
		t.Errorf("topic length = %d, want 66", len(got))
	}
}

func TestApprovalTopic0(t *testing.T) {
	// keccak256("Approval(address,address,uint256)")
	const want = "0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925"
	if ApprovalTopic0 != want {
		t.Errorf("ApprovalTopic0 = %q, want %q", ApprovalTopic0, want)
	}
}

func TestHighestBlock(t *testing.T) {
	logs := []Log{
		{BlockNumber: "0x10"},  // 16
		{BlockNumber: "0x1a4"}, // 420
		{BlockNumber: "0xff"},  // 255
		{BlockNumber: ""},      // ignored
		{BlockNumber: "0x1a3"}, // 419
	}
	if got := highestBlock(logs); got != 420 {
		t.Errorf("highestBlock = %d, want 420", got)
	}
	if got := highestBlock(nil); got != 0 {
		t.Errorf("highestBlock(nil) = %d, want 0", got)
	}
}
