package decimal

import (
	"bytes"
	"context"
	"errors"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"math/big"
	"testing"
)

func TestWeightedMixedSafeSignatures(t *testing.T) {
	signer := testSigner(t, 1)
	call, _ := NativeTransfer(testAddress, big.NewInt(1), "")
	tx := NewSafeTransaction(call, big.NewInt(7))
	owner := common.HexToAddress("0x0000000000000000000000000000000000000001")
	policy := SafePolicy{Safe: testAddress, ChainID: big.NewInt(202020), Nonce: big.NewInt(7), Block: big.NewInt(42), Owners: map[common.Address]uint32{signer.Address(): 2, owner: 3}, Threshold: 5}
	sig, e := tx.Sign(context.Background(), signer, policy.ChainID, policy.Safe)
	if e != nil {
		t.Fatal(e)
	}
	hash, _ := tx.Hash(policy.ChainID, policy.Safe)
	input := []SafeMixedSignature{{signer.Address(), SafeTypedSignature, sig}, {owner, SafeContractSignature, []byte{0xab, 0xcd}}}
	count := 0
	verify := func(_ context.Context, got common.Address, h common.Hash, data []byte) error {
		count++
		if got != owner || h != hash || !bytes.Equal(data, []byte{0xab, 0xcd}) {
			t.Fatal("verifier inputs")
		}
		data[0] = 0
		return nil
	}
	packed, e := PackWeightedSafeSignatures(context.Background(), tx, policy, input, verify)
	if e != nil {
		t.Fatal(e)
	}
	if count != 1 || len(packed) != 164 || new(big.Int).SetBytes(packed[32:64]).Int64() != 130 || packed[64] != 0 || new(big.Int).SetBytes(packed[130:162]).Int64() != 2 || !bytes.Equal(packed[162:], []byte{0xab, 0xcd}) {
		t.Fatal("JS dynamic offsets/length layout mismatch")
	}
	if input[1].Data[0] != 0xab {
		t.Fatal("signature input mutated")
	}
	if _, e = PackWeightedSafeSignatures(context.Background(), tx, policy, input, nil); e == nil {
		t.Fatal("unverified contract signature")
	}
	if _, e = PackWeightedSafeSignatures(context.Background(), tx, policy, input, func(context.Context, common.Address, common.Hash, []byte) error { return errors.New("bad") }); e == nil {
		t.Fatal("rejected contract counted")
	}
	if _, e = PackWeightedSafeSignatures(context.Background(), tx, policy, input[:1], verify); e == nil {
		t.Fatal("insufficient weight")
	}
	policy.Nonce = big.NewInt(8)
	if _, e = PackWeightedSafeSignatures(context.Background(), tx, policy, input, verify); e == nil {
		t.Fatal("stale nonce")
	}
}
func TestSafePersonalAndSnapshot(t *testing.T) {
	s := testSigner(t, 1)
	call, _ := NativeTransfer(testAddress, big.NewInt(1), "")
	tx := NewSafeTransaction(call, big.NewInt(0))
	p := SafePolicy{Safe: testAddress, ChainID: big.NewInt(202020), Nonce: new(big.Int), Block: big.NewInt(1), Owners: map[common.Address]uint32{s.Address(): 1}, Threshold: 1}
	hash, _ := tx.Hash(p.ChainID, p.Safe)
	preimage, e := tx.SigningPreimage(p.ChainID, p.Safe)
	if e != nil || len(preimage) != 66 || crypto.Keccak256Hash(preimage) != hash {
		t.Fatal("Safe signing preimage", e)
	}
	digest := common.BytesToHash(accounts.TextHash(hash[:]))
	sig, e := s.SignDigest(context.Background(), digest)
	if e != nil {
		t.Fatal(e)
	}
	packed, e := PackWeightedSafeSignatures(context.Background(), tx, p, []SafeMixedSignature{{s.Address(), SafePersonalSignature, sig}}, nil)
	if e != nil || packed[64] < 31 || packed[64] > 32 {
		t.Fatal("personal signature", e)
	}
	if _, e = PackWeightedSafeSignatures(context.Background(), tx, p, []SafeMixedSignature{{s.Address(), SafeTypedSignature, sig}}, nil); e == nil {
		t.Fatal("personal signature accepted as typed")
	}
}

func TestMultipleContractSafeOffsetsAndSnapshot(t *testing.T) {
	owner1 := common.HexToAddress("0x01")
	owner2 := common.HexToAddress("0x02")
	call, _ := NativeTransfer(testAddress, new(big.Int), "")
	tx := NewSafeTransaction(call, new(big.Int))
	p := SafePolicy{Safe: testAddress, ChainID: big.NewInt(202020), Nonce: new(big.Int), Block: big.NewInt(1), Owners: map[common.Address]uint32{owner1: 1, owner2: 1}, Threshold: 2}
	signatures := []SafeMixedSignature{{owner2, SafeContractSignature, []byte{2, 3, 4}}, {owner1, SafeContractSignature, []byte{1}}}
	packed, e := PackWeightedSafeSignatures(context.Background(), tx, p, signatures, func(context.Context, common.Address, common.Hash, []byte) error {
		delete(p.Owners, owner2)
		signatures[0].Data[0] = 99
		return nil
	})
	if e != nil {
		t.Fatal(e)
	}
	if len(packed) != 198 || new(big.Int).SetBytes(packed[32:64]).Int64() != 130 || new(big.Int).SetBytes(packed[97:129]).Int64() != 163 || !bytes.Equal(packed[195:], []byte{2, 3, 4}) {
		t.Fatal("multiple dynamic offsets or input snapshot")
	}
}
func TestAllowanceResetSteps(t *testing.T) {
	f := newFake()
	c := testClient(t, f)
	tokenABI, _ := loadABI("Token")
	spender := common.HexToAddress("0x2000000000000000000000000000000000000002")
	f.call = func(ethereum.CallMsg, *big.Int) ([]byte, error) {
		return tokenABI.Methods["allowance"].Outputs.Pack(big.NewInt(3))
	}
	action, _ := NativeTransfer(spender, new(big.Int), "action")
	plan, e := c.PlanERC20Allowance(context.Background(), testAddress, testAddress, spender, big.NewInt(10), action, true)
	if e != nil {
		t.Fatal(e)
	}
	if len(plan.Approvals) != 2 {
		t.Fatal("missing zero approval")
	}
	for i, expected := range []int64{0, 10} {
		v, e := tokenABI.Methods["approve"].Inputs.Unpack(plan.Approvals[i].Data[4:])
		if e != nil || v[1].(*big.Int).Int64() != expected {
			t.Fatal("allowance step", e)
		}
	}
	action.Data[0] = 'x'
	if string(plan.Action.Data) != "action" || len(f.sent) != 0 {
		t.Fatal("plan mutated or sent")
	}
	plan, e = c.PlanERC20Allowance(context.Background(), testAddress, testAddress, spender, big.NewInt(2), action, true)
	if e != nil || len(plan.Approvals) != 0 {
		t.Fatal("unneeded approval", e)
	}
}

func TestReadSafePolicyPinsAllGetters(t *testing.T) {
	f := newFake()
	client := testClient(t, f)
	contract, e := client.Contract("TestnetSafe20261029", testAddress)
	if e != nil {
		t.Fatal(e)
	}
	a, e := loadABI("TestnetSafe20261029")
	if e != nil {
		t.Fatal(e)
	}
	weight := big.NewInt(3)
	calls := 0
	f.call = func(msg ethereum.CallMsg, block *big.Int) ([]byte, error) {
		calls++
		if block == nil || block.Int64() != 42 {
			t.Fatal("unpinned Safe read")
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
			return m.Outputs.Pack(big.NewInt(2))
		case "nonce":
			return m.Outputs.Pack(big.NewInt(7))
		}
		return nil, errors.New("unexpected getter")
	}
	policy, e := contract.ReadSafePolicy(context.Background(), big.NewInt(42))
	if e != nil || policy.Owners[testAddress] != 3 || policy.Threshold != 2 || policy.Nonce.Int64() != 7 || calls != 3 {
		t.Fatal("Safe snapshot", e)
	}
	weight = big.NewInt(1001)
	if _, e = contract.ReadSafePolicy(context.Background(), big.NewInt(42)); e == nil {
		t.Fatal("unsupported weight truncated")
	}
}
