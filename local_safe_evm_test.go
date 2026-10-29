package decimal

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/crypto"
	"math/big"
	"os"
	"testing"
)

func TestLocalEVMOfficialWeightedSafeFactoryAndExecution(t *testing.T) {
	raw, err := os.ReadFile("testdata/evm/safe-runtime.json")
	if err != nil {
		t.Fatal(err)
	}
	manifestRaw, err := os.ReadFile("testdata/evm/safe-manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		SHA   string `json:"snapshot_sha256"`
		Block uint64 `json:"block"`
		Chain uint64 `json:"chain_id"`
		Hash  string `json:"rpc_block_hash"`
	}
	if err = json.Unmarshal(manifestRaw, &manifest); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(raw)
	if hex.EncodeToString(sum[:]) != manifest.SHA || manifest.Block != 22477428 || manifest.Chain != 202020 || manifest.Hash != "0xfb67691493eda26389cac1a122c8a6396192d97b153857d5ed3c87a10cf22d0c" {
		t.Fatal("runtime snapshot provenance mismatch")
	}
	var snapshot struct {
		Entries []struct{ Role, Address, Runtime string }
	}
	if err = json.Unmarshal(raw, &snapshot); err != nil {
		t.Fatal(err)
	}
	e := newLocalEVM(t)
	var impl, factory common.Address
	expected := map[string]string{"safe": "0xf551f56a1c8069a151b5a7e210afecc4febe2c3e2725960ae138adc10eb3e1cc", "factory": "0xe813a7ddfe19f5bb04cc791f075d74945277b1f7e0c5340c038049b4d9cf6a4c"}
	for _, entry := range snapshot.Entries {
		address := common.HexToAddress(entry.Address)
		code := common.FromHex(entry.Runtime)
		if crypto.Keccak256Hash(code).Hex() != expected[entry.Role] {
			t.Fatal("official runtime identity changed", entry.Role)
		}
		e.state.SetCode(address, code, tracing.CodeChangeUnspecified)
		if entry.Role == "safe" {
			impl = address
		} else if entry.Role == "factory" {
			factory = address
		}
	}
	client := e.client(t)
	template, err := client.TestnetSafe20261029(impl)
	if err != nil {
		t.Fatal(err)
	}
	a := testSigner(t, 1)
	defer a.Close()
	b := testSigner(t, 2)
	defer b.Close()
	setup, err := template.BuildSetup([]TestnetSafe20261029SetupArg0{{a.Address(), big.NewInt(2)}, {b.Address(), big.NewInt(1)}}, big.NewInt(3), common.Address{}, nil, common.Address{}, common.Address{}, big.NewInt(0), common.Address{})
	if err != nil {
		t.Fatal(err)
	}
	create, err := client.BuildSafeFactoryProxy(factory, impl, setup.Data, big.NewInt(1))
	if err != nil {
		t.Fatal(err)
	}
	result, logs, err := e.execute(t, a.Address(), create)
	if err != nil {
		t.Fatalf("real factory creation failed: %v %x", err, result)
	}
	if len(result) != 32 || len(logs) == 0 {
		t.Fatal("factory output/event missing")
	}
	proxy := common.BytesToAddress(result)
	safe, err := client.TestnetSafe20261029(proxy)
	if err != nil {
		t.Fatal(err)
	}
	owners, err := safe.ReadGetOwnersWithWeights(context.Background())
	if err != nil || len(owners.Value0) != 2 {
		t.Fatal("weighted setup failed", owners, err)
	}
	wantWeights := map[common.Address]int64{a.Address(): 2, b.Address(): 1}
	for _, entry := range owners.Value0 {
		want, ok := wantWeights[entry.Owner]
		if !ok || entry.Weight.Cmp(big.NewInt(want)) != 0 {
			t.Fatal("owner weights differ", owners)
		}
		delete(wantWeights, entry.Owner)
	}
	if len(wantWeights) != 0 {
		t.Fatal("missing configured weighted owner")
	}
	threshold, err := safe.ReadGetWeightThreshold(context.Background())
	if err != nil || threshold.Value0.Cmp(big.NewInt(3)) != 0 {
		t.Fatal("threshold mismatch", threshold, err)
	}
	e.fund(proxy, big.NewInt(100))
	recipient := common.HexToAddress("0x1005")
	native, err := NativeTransfer(recipient, big.NewInt(9), "")
	if err != nil {
		t.Fatal(err)
	}
	tx := NewSafeTransaction(native, big.NewInt(0))
	hash, err := tx.Hash(big.NewInt(202020), proxy)
	if err != nil {
		t.Fatal(err)
	}
	observedHash, err := safe.ReadGetTransactionHash(context.Background(), tx.To, tx.Value, tx.Data, tx.Operation, tx.SafeTxGas, tx.BaseGas, tx.GasPrice, tx.GasToken, tx.RefundReceiver, tx.Nonce)
	if err != nil || common.Hash(observedHash.Value0) != hash {
		t.Fatal("SDK hash differs from actual Safe EVM hash", observedHash, err)
	}
	one, err := tx.Sign(context.Background(), a, big.NewInt(202020), proxy)
	if err != nil {
		t.Fatal(err)
	}
	two, err := tx.Sign(context.Background(), b, big.NewInt(202020), proxy)
	if err != nil {
		t.Fatal(err)
	}
	partial, err := PackSafeEOASignatures(hash, one)
	if err != nil {
		t.Fatal(err)
	}
	execute, err := tx.BuildExecution(safe.Contract, partial)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = e.execute(t, a.Address(), execute); err == nil {
		t.Fatal("below weight threshold accepted")
	}
	if e.balance(proxy).Cmp(big.NewInt(100)) != 0 || e.balance(recipient).Sign() != 0 {
		t.Fatal("failed execution changed funds")
	}
	complete, err := PackSafeSignatures(hash, SafeSignature{a.Address(), one}, SafeSignature{b.Address(), two})
	if err != nil {
		t.Fatal(err)
	}
	execute, err = tx.BuildExecution(safe.Contract, complete)
	if err != nil {
		t.Fatal(err)
	}
	result, _, err = e.execute(t, a.Address(), execute)
	if err != nil {
		t.Fatalf("weighted execution failed: %v %x", err, result)
	}
	if e.balance(proxy).Cmp(big.NewInt(91)) != 0 || e.balance(recipient).Cmp(big.NewInt(9)) != 0 {
		t.Fatal("safe transfer not executed")
	}
	nonce, err := safe.ReadNonce(context.Background())
	if err != nil || nonce.Value0.Cmp(big.NewInt(1)) != 0 {
		t.Fatal("nonce not advanced", nonce, err)
	}
	if _, _, err = e.execute(t, a.Address(), execute); err == nil {
		t.Fatal("safe replay accepted")
	}
	if e.balance(recipient).Cmp(big.NewInt(9)) != 0 {
		t.Fatal("replay changed funds")
	}
}
