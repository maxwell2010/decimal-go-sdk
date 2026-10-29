package decimal

import (
	"context"
	"encoding/json"
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/crypto"
)

// Only sampled official runtime and real factory initialization are used. No
// network state or guessed ownership storage is installed in this fixture.
func TestLocalEVMWeightedSafeOwnerAdministrationAndAuthorization(t *testing.T) {
	ctx := context.Background()
	e := newLocalEVM(t)
	raw, err := os.ReadFile("testdata/evm/safe-runtime.json")
	if err != nil {
		t.Fatal(err)
	}
	var snapshot struct {
		Entries []struct{ Role, Address, Runtime string }
	}
	if err = json.Unmarshal(raw, &snapshot); err != nil {
		t.Fatal(err)
	}
	var impl, factory common.Address
	expected := map[string]string{"safe": "0xf551f56a1c8069a151b5a7e210afecc4febe2c3e2725960ae138adc10eb3e1cc", "factory": "0xe813a7ddfe19f5bb04cc791f075d74945277b1f7e0c5340c038049b4d9cf6a4c"}
	for _, entry := range snapshot.Entries {
		code := common.FromHex(entry.Runtime)
		if crypto.Keccak256Hash(code).Hex() != expected[entry.Role] {
			t.Fatal("unexpected runtime", entry.Role)
		}
		address := common.HexToAddress(entry.Address)
		e.state.SetCode(address, code, tracing.CodeChangeUnspecified)
		if entry.Role == "safe" {
			impl = address
		} else if entry.Role == "factory" {
			factory = address
		}
	}
	if impl == (common.Address{}) || factory == (common.Address{}) {
		t.Fatal("missing runtime")
	}
	client := e.client(t)
	template, err := client.TestnetSafe20261029(impl)
	if err != nil {
		t.Fatal(err)
	}
	a, b, c := testSigner(t, 1), testSigner(t, 2), testSigner(t, 3)
	defer a.Close()
	defer b.Close()
	defer c.Close()
	setup, err := template.BuildSetup([]TestnetSafe20261029SetupArg0{{a.Address(), big.NewInt(2)}, {b.Address(), big.NewInt(1)}}, big.NewInt(3), common.Address{}, nil, common.Address{}, common.Address{}, big.NewInt(0), common.Address{})
	if err != nil {
		t.Fatal(err)
	}
	creation, err := client.BuildSafeFactoryProxy(factory, impl, setup.Data, big.NewInt(10))
	if err != nil {
		t.Fatal(err)
	}
	output, _, err := e.execute(t, a.Address(), creation)
	if err != nil || len(output) != 32 {
		t.Fatal("factory creation", err)
	}
	proxy := common.BytesToAddress(output)
	safe, err := client.TestnetSafe20261029(proxy)
	if err != nil {
		t.Fatal(err)
	}
	assertState := func(nonce, threshold int64, weights map[common.Address]int64) {
		t.Helper()
		n, err := safe.ReadNonce(ctx)
		if err != nil || n.Value0.Cmp(big.NewInt(nonce)) != 0 {
			t.Fatal("nonce", n, err)
		}
		w, err := safe.ReadGetWeightThreshold(ctx)
		if err != nil || w.Value0.Cmp(big.NewInt(threshold)) != 0 {
			t.Fatal("threshold", w, err)
		}
		owners, err := safe.ReadGetOwnersWithWeights(ctx)
		if err != nil || len(owners.Value0) != len(weights) {
			t.Fatal("owners", owners, err)
		}
		for _, owner := range owners.Value0 {
			want, ok := weights[owner.Owner]
			if !ok || owner.Weight.Cmp(big.NewInt(want)) != 0 {
				t.Fatal("owner weight", owners)
			}
		}
	}
	execute := func(call Call, chain int64, fail bool, signers ...*KeySigner) {
		t.Helper()
		nonce, err := safe.ReadNonce(ctx)
		if err != nil {
			t.Fatal(err)
		}
		tx := NewSafeTransaction(call, nonce.Value0)
		hash, err := tx.Hash(big.NewInt(chain), proxy)
		if err != nil {
			t.Fatal(err)
		}
		var signatures [][]byte
		for _, s := range signers {
			sig, err := tx.Sign(ctx, s, big.NewInt(chain), proxy)
			if err != nil {
				t.Fatal(err)
			}
			signatures = append(signatures, sig)
		}
		packed, err := PackSafeEOASignatures(hash, signatures...)
		if err != nil {
			t.Fatal(err)
		}
		execution, err := tx.BuildExecution(safe.Contract, packed)
		if err != nil {
			t.Fatal(err)
		}
		_, logs, err := e.execute(t, a.Address(), execution)
		if (err != nil) != fail {
			t.Fatalf("execution failed=%v error=%v", fail, err)
		}
		if fail && len(logs) != 0 {
			t.Fatal("failure retained logs")
		}
	}
	weights := map[common.Address]int64{a.Address(): 2, b.Address(): 1}
	threshold, err := safe.BuildChangeWeightThreshold(big.NewInt(2))
	if err != nil {
		t.Fatal(err)
	}
	if _, logs, err := e.execute(t, a.Address(), threshold); err == nil || len(logs) != 0 {
		t.Fatal("direct owner administration accepted", err)
	}
	assertState(0, 3, weights)
	execute(threshold, 75, true, a, b)
	assertState(0, 3, weights)
	execute(threshold, 202020, true, a, c)
	assertState(0, 3, weights)
	execute(threshold, 202020, false, a, b)
	assertState(1, 2, weights)
	add, err := safe.BuildAddOwnerWithThreshold(TestnetSafe20261029AddOwnerWithThresholdArg0{c.Address(), big.NewInt(4)}, big.NewInt(6))
	if err != nil {
		t.Fatal(err)
	}
	execute(add, 202020, false, a)
	weights[c.Address()] = 4
	assertState(2, 6, weights)
	owners, err := safe.ReadGetOwnersWithWeights(ctx)
	if err != nil {
		t.Fatal(err)
	}
	previous := common.HexToAddress("0x1")
	found := false
	for _, owner := range owners.Value0 {
		if owner.Owner == b.Address() {
			found = true
			break
		}
		previous = owner.Owner
	}
	if !found {
		t.Fatal("owner absent before removal")
	}
	remove, err := safe.BuildRemoveOwner(previous, b.Address(), big.NewInt(6))
	if err != nil {
		t.Fatal(err)
	}
	execute(remove, 202020, false, a, c)
	delete(weights, b.Address())
	assertState(3, 6, weights)
	invalid, err := safe.BuildChangeWeightThreshold(big.NewInt(7))
	if err != nil {
		t.Fatal(err)
	}
	execute(invalid, 202020, true, a, c)
	assertState(3, 6, weights)
	valid, err := safe.BuildChangeWeightThreshold(big.NewInt(4))
	if err != nil {
		t.Fatal(err)
	}
	execute(valid, 202020, false, a, c)
	assertState(4, 4, weights)
	// Removed owners cannot contribute weight even alongside a remaining owner.
	invalid, err = safe.BuildChangeWeightThreshold(big.NewInt(3))
	if err != nil {
		t.Fatal(err)
	}
	execute(invalid, 202020, true, a, b)
	assertState(4, 4, weights)
	approvedTx := NewSafeTransaction(invalid, big.NewInt(4))
	approvedHash, err := approvedTx.Hash(big.NewInt(202020), proxy)
	if err != nil {
		t.Fatal(err)
	}
	approval, err := safe.BuildApproveHash(TestnetSafe20261029ApproveHashArg0{To: approvedTx.To, Value: approvedTx.Value, Data: approvedTx.Data, Operation: approvedTx.Operation, SafeTxGas: approvedTx.SafeTxGas, BaseGas: approvedTx.BaseGas, GasPrice: approvedTx.GasPrice, GasToken: approvedTx.GasToken, RefundReceiver: approvedTx.RefundReceiver, Nonce: approvedTx.Nonce})
	if err != nil {
		t.Fatal(err)
	}
	if _, logs, err := e.execute(t, b.Address(), approval); err == nil || len(logs) != 0 {
		t.Fatal("removed owner approved hash", err)
	}
	if _, logs, err := e.execute(t, a.Address(), approval); err != nil || len(logs) == 0 {
		t.Fatal("owner approval failed", err)
	}
	approved, err := safe.ReadApprovedHashes(ctx, a.Address(), approvedHash)
	if err != nil || approved.Value0.Cmp(big.NewInt(1)) != 0 {
		t.Fatal("approval not recorded", approved, err)
	}
	removed, err := safe.ReadApprovedHashes(ctx, b.Address(), approvedHash)
	if err != nil || removed.Value0.Sign() != 0 {
		t.Fatal("removed-owner approval recorded", removed, err)
	}
	assertState(4, 4, weights)

}
