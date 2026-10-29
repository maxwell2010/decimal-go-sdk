package decimal

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/crypto"
	"math/big"
	"os"
	"strings"
	"testing"
)

func TestLocalEVMObservedCurrentChecksRuntimeSigning(t *testing.T) {
	raw, err := os.ReadFile("testdata/evm/checks-current-runtime.json")
	if err != nil {
		t.Fatal(err)
	}
	manifestRaw, err := os.ReadFile("testdata/evm/checks-current-manifest.json")
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
		t.Fatal("observed runtime provenance changed")
	}
	var snap struct{ Address, Runtime string }
	if err = json.Unmarshal(raw, &snap); err != nil {
		t.Fatal(err)
	}
	code := common.FromHex(snap.Runtime)
	if len(code) != 6318 || crypto.Keccak256Hash(code).Hex() != "0x78da094992fb84bf17517277dcfc9c67ab36ba535f8879b5fd5b5614ac397bf4" {
		t.Fatal("observed runtime identity changed")
	}
	if common.HexToAddress(snap.Address) != common.HexToAddress("0xBACA5A733002F5323aBE4c83C91B87212E7A3ca3") {
		t.Fatal("unexpected implementation address")
	}
	e := newLocalEVM(t)
	address := common.HexToAddress(snap.Address)
	e.state.SetCode(address, code, tracing.CodeChangeUnspecified)
	creator := common.HexToAddress("0x1001")
	receiver := common.HexToAddress("0x1002")
	stranger := common.HexToAddress("0x1003")
	e.fund(creator, big.NewInt(1000))
	checks, err := e.client(t).Checks(address)
	if err != nil {
		t.Fatal(err)
	}
	artifact := loadLocalArtifact(t, "Checks.official.json")
	parsed, err := abi.JSON(strings.NewReader(string(artifact.ABI)))
	if err != nil {
		t.Fatal(err)
	}
	signer, err := EVMCheckSignerFromPassword("public current-runtime fixture")
	if err != nil {
		t.Fatal(err)
	}
	defer signer.Close()
	create, err := checks.BuildCreateChecksDEL(big.NewInt(100), []common.Address{signer.Address()}, big.NewInt(100), big.NewInt(1000), big.NewInt(0))
	if err != nil {
		t.Fatal(err)
	}
	result, logs, err := e.execute(t, creator, create)
	if err != nil {
		t.Fatalf("observed runtime create failed %v %x", err, result)
	}
	values, err := parsed.Unpack("createChecksDEL", result)
	if err != nil {
		t.Fatal(err)
	}
	hashes := values[0].([][32]byte)
	if len(hashes) != 1 || len(logs) == 0 || e.balance(address).Cmp(big.NewInt(100)) != 0 {
		t.Fatal("creation output/custody absent")
	}
	hash := common.Hash(hashes[0])
	foundCreate := false
	for _, log := range logs {
		if len(log.Topics) > 0 && log.Topics[0] == parsed.Events["CreatedCheck"].ID {
			fields, decodeErr := parsed.Events["CreatedCheck"].Inputs.Unpack(log.Data)
			if decodeErr != nil || len(fields) != 2 || common.Hash(fields[0].([32]byte)) != hash {
				t.Fatal("creation event mismatch", fields, decodeErr)
			}
			foundCreate = true
		}
	}
	if !foundCreate {
		t.Fatal("missing actual CreatedCheck event")
	}

	signature, err := SignEVMCheckRedemption(context.Background(), signer, hash, receiver)
	if err != nil {
		t.Fatal(err)
	}
	redeem, err := checks.BuildRedeemChecks([][]byte{signature}, hashes)
	if err != nil {
		t.Fatal(err)
	}
	var digests []common.Hash
	e.cfg.EVMConfig.Tracer = &tracing.Hooks{OnEnter: func(depth int, typ byte, from, to common.Address, input []byte, gas uint64, value *big.Int) {
		if to == common.HexToAddress("0x1") && len(input) == 128 {
			digests = append(digests, common.BytesToHash(input[:32]))
		}
	}}
	result, rejected, err := e.execute(t, stranger, redeem)
	if err == nil || len(rejected) != 0 || e.balance(address).Cmp(big.NewInt(100)) != 0 || e.balance(stranger).Sign() != 0 {
		t.Fatal("stranger proof accepted or changed funds", err)
	}
	result, logs, err = e.execute(t, receiver, redeem)
	if err != nil {
		t.Fatalf("recipient-bound redemption rejected: %v %x traces=%v", err, result, digests)
	}
	foundRedeem := false
	for _, log := range logs {
		if len(log.Topics) > 0 && log.Topics[0] == parsed.Events["RedeemCheck"].ID {
			fields, decodeErr := parsed.Events["RedeemCheck"].Inputs.Unpack(log.Data)
			if decodeErr != nil || common.Hash(fields[0].([32]byte)) != hash || fields[1].(uint8) != 1 {
				t.Fatal("redemption event mismatch", fields, decodeErr)
			}
			foundRedeem = true
		}
	}
	if !foundRedeem {
		t.Fatal("missing actual RedeemCheck event")
	}
	if len(digests) != 2 || digests[0] != EVMCheckRedemptionDigest(hash, stranger) || digests[1] != EVMCheckRedemptionDigest(hash, receiver) {
		t.Fatal("unexpected observed signature digests", digests)
	}
	if len(logs) == 0 || e.balance(receiver).Cmp(big.NewInt(100)) != 0 || e.balance(address).Sign() != 0 {
		t.Fatal("redemption funds/events missing")
	}
	if _, logs, err = e.execute(t, receiver, redeem); err == nil || len(logs) != 0 || e.balance(receiver).Cmp(big.NewInt(100)) != 0 {
		t.Fatal("replay accepted or retained effects", err)
	}
}
