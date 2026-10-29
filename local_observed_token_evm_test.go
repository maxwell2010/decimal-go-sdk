package decimal

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/crypto"
)

// This uses sampled Decimal implementation code, initialized through its real
// public ABI in empty local state. It does not copy live proxy storage.
func TestLocalEVMObservedDecimalReservelessPermit(t *testing.T) {
	raw, err := os.ReadFile("testdata/evm/token-runtime-snapshot.json")
	if err != nil {
		t.Fatal(err)
	}
	meta, err := os.ReadFile("testdata/evm/token-runtime-manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	var m struct {
		SHA   string `json:"snapshot_sha256"`
		Chain uint64 `json:"chain_id"`
		Block uint64 `json:"block"`
		Hash  string `json:"rpc_block_hash"`
	}
	if err = json.Unmarshal(meta, &m); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(raw)
	if hex.EncodeToString(sum[:]) != m.SHA || m.Chain != 202020 || m.Block != 22477428 || m.Hash != "0xfb67691493eda26389cac1a122c8a6396192d97b153857d5ed3c87a10cf22d0c" {
		t.Fatal("runtime provenance changed")
	}
	var snapshot struct {
		Entries []struct{ Role, Address, Runtime string }
	}
	if err = json.Unmarshal(raw, &snapshot); err != nil {
		t.Fatal(err)
	}
	expected := map[string]string{"reserve": "0x67ef16857177e434b860586efd9ff7cf2c57c1adf4ba50bd9310f42f9c8c2cf1", "reserveless": "0x67e10916520447faad79c557c719042ba1e12b95ca57e1c18b7a5ecfaca4c1fd"}
	e := newLocalEVM(t)
	var target common.Address
	for _, row := range snapshot.Entries {
		code := common.FromHex(row.Runtime)
		if crypto.Keccak256Hash(code).Hex() != expected[row.Role] {
			t.Fatal("runtime hash mismatch", row.Role)
		}
		if row.Role == "reserveless" {
			target = common.HexToAddress(row.Address)
			if target != common.HexToAddress("0x9D3e4E9953B031F2c89Da77c0F2abaD169eDd43F") {
				t.Fatal("implementation address changed")
			}
			e.state.SetCode(target, code, tracing.CodeChangeUnspecified)
		}
	}
	if target == (common.Address{}) {
		t.Fatal("reserveless runtime missing")
	}
	signer := testSigner(t, 1)
	defer signer.Close()
	spender := common.HexToAddress("0x1002")
	client := e.client(t)
	token, err := client.TestnetReservelessToken20261029(target)
	if err != nil {
		t.Fatal(err)
	}
	init, err := token.BuildInitialize(nil, "ObservedDecimal", "ODL", signer.Address(), true, true, big.NewInt(1000), big.NewInt(10000), "local official runtime")
	if err != nil {
		t.Fatal(err)
	}
	output, _, err := e.execute(t, signer.Address(), init)
	if err != nil {
		reason, _ := abi.UnpackRevert(output)
		t.Fatalf("real initialize rejected: %v %x %s", err, output, reason)
	}
	p := Permit{Owner: signer.Address(), Spender: spender, Value: big.NewInt(17), Nonce: big.NewInt(0), Deadline: big.NewInt(100)}
	sig, err := client.CheckedPermit(context.Background(), signer, target, "ObservedDecimal", "1", p)
	if err != nil {
		t.Fatal("actual domain/nonce guard", err)
	}
	call, err := token.BuildPermit(signer.Address(), spender, p.Value, p.Deadline, sig.V, sig.R, sig.S)
	if err != nil {
		t.Fatal(err)
	}
	output, logs, err := e.execute(t, spender, call)
	if err != nil {
		reason, _ := abi.UnpackRevert(output)
		t.Fatalf("actual permit rejected: %v %x %s", err, output, reason)
	}
	if len(logs) != 1 {
		t.Fatal("permit Approval missing")
	}
	parsed, err := loadABI("TestnetReservelessToken20261029")
	if err != nil {
		t.Fatal(err)
	}
	if len(logs[0].Topics) != 3 || logs[0].Topics[0] != parsed.Events["Approval"].ID || common.BytesToAddress(logs[0].Topics[1].Bytes()) != signer.Address() || common.BytesToAddress(logs[0].Topics[2].Bytes()) != spender || new(big.Int).SetBytes(logs[0].Data).Cmp(big.NewInt(17)) != 0 {
		t.Fatal("actual Approval contents differ")
	}

	allowance, err := token.ReadAllowance(context.Background(), signer.Address(), spender)
	if err != nil || allowance.Value0.Cmp(big.NewInt(17)) != 0 {
		t.Fatal("allowance", allowance, err)
	}
	nonce, err := token.ReadNonces(context.Background(), signer.Address())
	if err != nil || nonce.Value0.Cmp(big.NewInt(1)) != 0 {
		t.Fatal("nonce", nonce, err)
	}
	if _, logs, err = e.execute(t, spender, call); err == nil || len(logs) != 0 {
		t.Fatal("permit replay accepted", err)
	}
	transfer, err := token.BuildTransferFrom(signer.Address(), spender, big.NewInt(7))
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = e.execute(t, spender, transfer); err != nil {
		t.Fatal(err)
	}
	balance, err := token.ReadBalanceOf(context.Background(), spender)
	if err != nil || balance.Value0.Cmp(big.NewInt(7)) != 0 {
		t.Fatal("permit spend", balance, err)
	}
	allowance, err = token.ReadAllowance(context.Background(), signer.Address(), spender)
	if err != nil || allowance.Value0.Cmp(big.NewInt(10)) != 0 {
		t.Fatal("spent allowance", allowance, err)
	}
	// A previously initialized implementation cannot be taken over again.
	if _, logs, err = e.execute(t, spender, init); err == nil || len(logs) != 0 {
		t.Fatal("reinitialization accepted", err)
	}
	for _, scenario := range []string{"wrong-domain", "wrong-chain", "expired"} {
		t.Run(scenario, func(t *testing.T) {
			name := "ObservedDecimal"
			chain := big.NewInt(202020)
			proof := p
			proof.Nonce = big.NewInt(1)
			switch scenario {
			case "wrong-domain":
				name = "Wrong"
			case "wrong-chain":
				chain = big.NewInt(202021)
			case "expired":
				proof.Deadline = big.NewInt(0)
			}
			typed, err := PermitTypedData(name, "1", chain, target, proof)
			if err != nil {
				t.Fatal(err)
			}
			rawsig, err := signer.SignTypedData(context.Background(), typed)
			if err != nil {
				t.Fatal(err)
			}
			parts, err := SplitSignature(rawsig)
			if err != nil {
				t.Fatal(err)
			}
			invalid, err := token.BuildPermit(signer.Address(), spender, proof.Value, proof.Deadline, parts.V, parts.R, parts.S)
			if err != nil {
				t.Fatal(err)
			}
			if _, logs, err = e.execute(t, spender, invalid); err == nil || len(logs) != 0 {
				t.Fatal("invalid permit accepted", err)
			}
			nonce, err := token.ReadNonces(context.Background(), signer.Address())
			if err != nil || nonce.Value0.Cmp(big.NewInt(1)) != 0 {
				t.Fatal("invalid permit changed nonce", nonce, err)
			}
			allowance, err := token.ReadAllowance(context.Background(), signer.Address(), spender)
			if err != nil || allowance.Value0.Cmp(big.NewInt(10)) != 0 {
				t.Fatal("invalid permit changed allowance", allowance, err)
			}
		})
	}

}
