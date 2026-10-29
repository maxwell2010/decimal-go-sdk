package decimal

import (
	"bytes"
	"context"
	"encoding/json"
	"math/big"
	"os"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
)

// Execute the pinned official implementation in memory. This is not a proxy
// initialization, a mined receipt, or evidence of public deployment acceptance.
func TestLocalEVMOfficialChecksCreationAndSigningProfileMismatch(t *testing.T) {
	raw, err := os.ReadFile("testdata/evm/Checks.official.json")
	if err != nil {
		t.Fatal(err)
	}
	var artifact struct {
		ABI              json.RawMessage `json:"abi"`
		Bytecode         string          `json:"bytecode"`
		DeployedBytecode string          `json:"deployedBytecode"`
	}
	if err = json.Unmarshal(raw, &artifact); err != nil {
		t.Fatal(err)
	}
	parsed, err := abi.JSON(strings.NewReader(string(artifact.ABI)))
	if err != nil {
		t.Fatal(err)
	}
	e := newLocalEVM(t)
	creator := common.HexToAddress("0x1001")
	receiver := common.HexToAddress("0x1002")
	stranger := common.HexToAddress("0x1003")
	e.fund(creator, big.NewInt(1000000))
	address := e.deploy(t, creator, artifact.ABI, common.FromHex(artifact.Bytecode))
	if !bytes.Equal(e.state.GetCode(address), common.FromHex(artifact.DeployedBytecode)) {
		t.Fatal("Checks deployed runtime differs from pinned artifact")
	}
	checks, err := e.client(t).Checks(address)
	if err != nil {
		t.Fatal(err)
	}
	secret, err := EVMCheckSignerFromPassword("local public Checks fixture")
	if err != nil {
		t.Fatal(err)
	}
	defer secret.Close()
	nonce, err := checks.ReadNonces(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	amount := big.NewInt(100)
	create, err := checks.BuildCreateChecksDEL(amount, []common.Address{secret.Address()}, amount, big.NewInt(1000), nonce.Value0)
	if err != nil {
		t.Fatal(err)
	}
	output, logs, err := e.execute(t, creator, create)
	if err != nil {
		t.Fatal("official create DEL", err)
	}
	values, err := parsed.Unpack("createChecksDEL", output)
	if err != nil {
		t.Fatal(err)
	}
	hashes := values[0].([][32]byte)
	if len(hashes) != 1 {
		t.Fatal("unexpected hashes", hashes)
	}
	hash := common.Hash(hashes[0])
	seen := false
	for _, log := range logs {
		if len(log.Topics) > 0 && log.Topics[0] == parsed.Events["CreatedCheck"].ID {
			fields, err := parsed.Events["CreatedCheck"].Inputs.Unpack(log.Data)
			if err != nil {
				t.Fatal(err)
			}
			if common.Hash(fields[0].([32]byte)) != hash {
				t.Fatal("returned/emitted check hash differ")
			}
			seen = true
		}
	}
	if !seen || e.balance(address).Cmp(amount) != 0 {
		t.Fatal("creation event or funding absent")
	}

	var recoveredDigests []common.Hash
	e.cfg.EVMConfig.Tracer = &tracing.Hooks{OnEnter: func(depth int, typ byte, from, to common.Address, input []byte, gas uint64, value *big.Int) {
		if to == common.HexToAddress("0x1") && len(input) == 128 {
			recoveredDigests = append(recoveredDigests, common.BytesToHash(input[:32]))
		}
	}}
	signature, err := SignEVMCheckRedemption(context.Background(), secret, hash, receiver)
	if err != nil {
		t.Fatal(err)
	}
	redeem, err := checks.BuildRedeemChecks([][]byte{signature}, hashes)
	if err != nil {
		t.Fatal(err)
	}
	// The bytecode shipped beside the JS ABI is a different signing generation
	// from the current JS wrapper. Preserve this incompatibility as evidence.
	for _, caller := range []common.Address{stranger, receiver} {
		result, rejectedLogs, callErr := e.execute(t, caller, redeem)
		expected := parsed.Errors["InvalidSignature"].ID
		if len(rejectedLogs) != 0 || callErr == nil || len(result) != 4 || string(result) != string(expected[:4]) {
			t.Fatalf("expected artifact InvalidSignature, got %v %x", callErr, result)
		}
		if e.balance(address).Cmp(amount) != 0 || e.balance(receiver).Sign() != 0 || e.balance(stranger).Sign() != 0 {
			t.Fatal("rejected proof changed funds")
		}
	}
	checkOnly := common.BytesToHash(accounts.TextHash(hash[:]))
	if len(recoveredDigests) != 2 || recoveredDigests[0] != checkOnly || recoveredDigests[1] != checkOnly || checkOnly == EVMCheckRedemptionDigest(hash, receiver) {
		t.Fatal("artifact signing generation observation changed", recoveredDigests)
	}
	// Test owner-only rejection without manufacturing an initialization route.
	ownerChange, err := checks.BuildTransferOwnership(stranger)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = e.execute(t, creator, ownerChange); err == nil {
		t.Fatal("uninitialized implementation accepted owner-only operation")
	}
	// The incompatible proof is rejected before expiry handling. No expiry,
	// successful redemption, or replay acceptance is inferred for this artifact.
}

// This historical artifact accepts a check-only personal signature. This test
// deliberately does not expose that generation as a public SDK signing helper
// or infer the protocol used by any public deployment.
func TestLocalEVMHistoricalChecksArtifactRedemptionAndExpiry(t *testing.T) {
	for _, expired := range []bool{false, true} {
		t.Run(map[bool]string{false: "redeem-and-replay", true: "expired"}[expired], func(t *testing.T) {
			artifact := loadLocalArtifact(t, "Checks.official.json")
			parsed, err := abi.JSON(strings.NewReader(string(artifact.ABI)))
			if err != nil {
				t.Fatal(err)
			}
			e := newLocalEVM(t)
			creator := common.HexToAddress("0x1001")
			receiver := common.HexToAddress("0x1002")
			e.fund(creator, big.NewInt(1000000))
			address := e.deploy(t, creator, artifact.ABI, common.FromHex(artifact.Bytecode))
			checks, err := e.client(t).Checks(address)
			if err != nil {
				t.Fatal(err)
			}
			secret, err := EVMCheckSignerFromPassword("local public historical Checks fixture")
			if err != nil {
				t.Fatal(err)
			}
			defer secret.Close()
			nonce, err := checks.ReadNonces(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			amount := big.NewInt(100)
			create, err := checks.BuildCreateChecksDEL(amount, []common.Address{secret.Address()}, amount, big.NewInt(1000), nonce.Value0)
			if err != nil {
				t.Fatal(err)
			}
			output, _, err := e.execute(t, creator, create)
			if err != nil {
				t.Fatal(err)
			}
			values, err := parsed.Unpack("createChecksDEL", output)
			if err != nil {
				t.Fatal(err)
			}
			hashes := values[0].([][32]byte)
			if len(hashes) != 1 {
				t.Fatal("unexpected hash count")
			}
			hash := common.Hash(hashes[0])
			signature, err := secret.SignDigest(context.Background(), common.BytesToHash(accounts.TextHash(hash[:])))
			if err != nil {
				t.Fatal(err)
			}
			signature[64] += 27
			redeem, err := checks.BuildRedeemChecks([][]byte{signature}, hashes)
			if err != nil {
				t.Fatal(err)
			}
			if expired {
				e.block(1001)
				result, rejectedLogs, callErr := e.execute(t, receiver, redeem)
				expected := parsed.Errors["ExpiredDueBlock"].ID
				if len(rejectedLogs) != 0 || callErr == nil || len(result) != 4 || string(result) != string(expected[:4]) {
					t.Fatalf("expiry: %v %x", callErr, result)
				}
				if e.balance(address).Cmp(amount) != 0 || e.balance(receiver).Sign() != 0 {
					t.Fatal("expiry changed funds")
				}
				return
			}
			_, redemptionLogs, err := e.execute(t, receiver, redeem)
			if err != nil {
				t.Fatalf("historical redeem failed: %v", err)
			}
			found := false
			for _, log := range redemptionLogs {
				if len(log.Topics) > 0 && log.Topics[0] == parsed.Events["RedeemCheck"].ID {
					fields, unpackErr := parsed.Events["RedeemCheck"].Inputs.Unpack(log.Data)
					if unpackErr != nil || len(fields) != 2 || common.Hash(fields[0].([32]byte)) != hash || fields[1].(uint8) != 1 {
						t.Fatal("redemption event mismatch", fields, unpackErr)
					}
					found = true
				}
			}
			if !found {
				t.Fatal("missing actual RedeemCheck event")
			}
			if e.balance(receiver).Cmp(amount) != 0 || e.balance(address).Sign() != 0 {
				t.Fatal("historical redemption balances mismatch")
			}
			result, rejectedLogs, callErr := e.execute(t, receiver, redeem)
			expected := parsed.Errors["InvalidStatus"].ID
			if len(rejectedLogs) != 0 || callErr == nil || len(result) != 36 || string(result[:4]) != string(expected[:4]) || new(big.Int).SetBytes(result[4:]).Uint64() != 1 {
				t.Fatalf("replay: %v %x", callErr, result)
			}
			if e.balance(receiver).Cmp(amount) != 0 {
				t.Fatal("replay changed balance")
			}
		})
	}
}
