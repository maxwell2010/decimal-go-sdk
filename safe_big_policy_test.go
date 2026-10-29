package decimal

import (
	"context"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"testing"
)

func TestBigSafeWeightsAndSignatureAuthorization(t *testing.T) {
	signer := testSigner(t, 1)
	call, _ := NativeTransfer(testAddress, new(big.Int), "")
	tx := NewSafeTransaction(call, big.NewInt(7))
	weight := new(big.Int).Lsh(big.NewInt(1), 100)
	p := BigSafePolicy{Safe: testAddress, ChainID: big.NewInt(202020), Nonce: big.NewInt(7), Block: big.NewInt(42), Owners: map[common.Address]*big.Int{signer.Address(): weight}, Threshold: copyInt(weight)}
	sig, e := tx.Sign(context.Background(), signer, p.ChainID, p.Safe)
	if e != nil {
		t.Fatal(e)
	}
	input := []SafeMixedSignature{{signer.Address(), SafeTypedSignature, sig}}
	packed, e := PackBigWeightedSafeSignatures(context.Background(), tx, p, input, nil)
	if e != nil || len(packed) != 65 {
		t.Fatal("uint256 weight truncated", e)
	}
	p.Threshold = new(big.Int).Lsh(big.NewInt(1), 256)
	if _, e = PackBigWeightedSafeSignatures(context.Background(), tx, p, input, nil); e == nil {
		t.Fatal("threshold overflow accepted")
	}
	p.Threshold = copyInt(weight)
	if _, e = PackBigWeightedSafeSignatures(context.Background(), tx, p, append(input, input[0]), nil); e == nil {
		t.Fatal("duplicate owner counted")
	}
}
func TestBigSafePolicyCallbackMutationAndSumOverflow(t *testing.T) {
	owner := common.HexToAddress("0x01")
	call, _ := NativeTransfer(testAddress, new(big.Int), "")
	tx := NewSafeTransaction(call, new(big.Int))
	weight := new(big.Int).Lsh(big.NewInt(1), 200)
	p := BigSafePolicy{Safe: testAddress, ChainID: big.NewInt(202020), Nonce: new(big.Int), Block: big.NewInt(42), Owners: map[common.Address]*big.Int{owner: weight}, Threshold: copyInt(weight)}
	input := []SafeMixedSignature{{owner, SafeContractSignature, []byte{1, 2}}}
	packed, e := PackBigWeightedSafeSignatures(context.Background(), tx, p, input, func(context.Context, common.Address, common.Hash, []byte) error {
		weight.SetInt64(0)
		p.Threshold.SetInt64(0)
		input[0].Data[0] = 99
		return nil
	})
	if e != nil || packed[len(packed)-2] != 1 {
		t.Fatal("mutable policy/signature aliases", e)
	}
	max := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))
	p.Owners = map[common.Address]*big.Int{owner: max, testAddress: big.NewInt(1)}
	p.Threshold = big.NewInt(1)
	if _, e = PackBigWeightedSafeSignatures(context.Background(), tx, p, input, nil); e == nil {
		t.Fatal("total weight overflow")
	}
}
func TestReadBigSafePolicyKeepsUint256(t *testing.T) {
	f := newFake()
	client := testClient(t, f)
	contract, e := client.Contract("TestnetSafe20261029", testAddress)
	if e != nil {
		t.Fatal(e)
	}
	a, _ := loadABI("TestnetSafe20261029")
	weight := new(big.Int).Lsh(big.NewInt(1), 120)
	block := big.NewInt(42)
	f.call = func(msg ethereum.CallMsg, b *big.Int) ([]byte, error) {
		if b == nil || b.Cmp(block) != 0 {
			t.Fatal("unpinned policy read")
		}
		m, e := a.MethodById(msg.Data[:4])
		if e != nil {
			return nil, e
		}
		switch m.RawName {
		case "getOwnersWithWeights":
			return m.Outputs.Pack([]struct {
				Owner  common.Address
				Weight *big.Int
			}{{testAddress, weight}})
		case "getWeightThreshold":
			return m.Outputs.Pack(weight)
		default:
			return m.Outputs.Pack(big.NewInt(7))
		}
	}
	policy, e := contract.ReadBigSafePolicy(context.Background(), block)
	if e != nil || policy.Owners[testAddress].Cmp(weight) != 0 || policy.Threshold.Cmp(weight) != 0 {
		t.Fatal("policy read", e)
	}
	weight.SetInt64(0)
	block.SetInt64(0)
	if policy.Threshold.BitLen() != 121 || policy.Block.Int64() != 42 {
		t.Fatal("policy retains callback aliases")
	}
}
