package main

import (
	"github.com/ethereum/go-ethereum/common"
	"testing"
)

func TestAllUnifiedNFTExamplesRemainOffline(t *testing.T) {
	calls, err := buildExamples()
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 15 {
		t.Fatal("missing user action or approval example")
	}
	seen := map[string]bool{}
	for i, call := range calls {
		if seen[call.Signature] {
			t.Fatal("duplicate example", call.Signature)
		}
		seen[call.Signature] = true
		target := common.HexToAddress("0x1001")
		if i >= 13 {
			target = common.HexToAddress("0x1003")
		}
		if call.To != target || len(call.Data) < 4 || call.Value.Sign() != 0 {
			t.Fatal("invalid offline intent", i)
		}
	}
}
