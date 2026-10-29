package decimal

import (
	"context"
	"encoding/json"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"testing"
)

func TestContractSignatureVerifierPreservesExplicitSafeCaller(t *testing.T) {
	raw := json.RawMessage(`[{"type":"function","name":"isValidSignature","stateMutability":"view","inputs":[{"type":"bytes32"},{"type":"bytes"}],"outputs":[{"type":"bytes4"}]}]`)
	f := newFake()
	client := testClient(t, f)
	safe := common.HexToAddress("0x2000000000000000000000000000000000000002")
	verifier, e := client.ContractSignatureVerifierFrom(raw, big.NewInt(123), ERC1271Bytes32, nil, safe)
	if e != nil {
		t.Fatal(e)
	}
	f.call = func(msg ethereum.CallMsg, block *big.Int) ([]byte, error) {
		if msg.From != safe || *msg.To != testAddress || block.Int64() != 123 {
			t.Error("signature call context changed")
		}
		out := make([]byte, 32)
		copy(out, []byte{0x16, 0x26, 0xba, 0x7e})
		return out, nil
	}
	if e = verifier(context.Background(), testAddress, common.Hash{}, []byte{1}); e != nil {
		t.Fatal(e)
	}
	if len(f.sent) != 0 {
		t.Fatal("signature verification broadcast")
	}
}
