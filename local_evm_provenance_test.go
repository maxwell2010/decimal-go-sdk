package decimal

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
)

func TestLocalEVMFixtureProvenance(t *testing.T) {
	raw, err := os.ReadFile("testdata/evm/manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Files map[string]string `json:"files"`
	}
	if err = json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	for name, want := range manifest.Files {
		if filepath.Base(name) != name {
			t.Fatal("invalid fixture manifest path")
		}
		data, err := os.ReadFile(filepath.Join("testdata/evm", name))
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(data)
		if hex.EncodeToString(sum[:]) != want {
			t.Fatal("fixture integrity mismatch", name)
		}
	}
	for name, want := range map[string]string{
		"decimalToken.sol":     "686b1b0fb9f45928f8ed5ebf416b77b6e1ea9ccfa27d106d5acdbb277230e3e3",
		"Checks.official.json": "9504852077e385f2163005fa18e4dc58b4a49005cf68f9b5d5582ad1415bc33b",
	} {
		if manifest.Files[name] != want {
			t.Fatal("pinned official source identity changed", name)
		}
	}
	compiled := loadLocalArtifact(t, "NewToken.compiled.json")
	if len(common.FromHex(compiled.DeployedBytecode)) == 0 {
		t.Fatal("token runtime missing")
	}
	checks := loadLocalArtifact(t, "Checks.official.json")
	if len(common.FromHex(checks.DeployedBytecode)) != 6258 {
		t.Fatal("pinned historical Checks runtime changed")
	}
	embedded, err := abiFiles.ReadFile("abi/Checks.json")
	if err != nil {
		t.Fatal(err)
	}
	var savedABI, fixtureABI any
	if err = json.Unmarshal(embedded, &savedABI); err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(checks.ABI, &fixtureABI); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(savedABI, fixtureABI) {
		t.Fatal("Checks fixture ABI differs from SDK profile")
	}
}
func TestLocalEVMReadTransportDoesNotCommitWrites(t *testing.T) {
	e := newLocalEVM(t)
	artifact := loadLocalArtifact(t, "NewToken.compiled.json")
	owner := common.HexToAddress("0x1001")
	recipient := common.HexToAddress("0x1002")
	address := e.deploy(t, owner, artifact.ABI, common.FromHex(artifact.Bytecode), "Local token", "LOT", big.NewInt(1))
	client := e.client(t)
	token, err := client.Token(address)
	if err != nil {
		t.Fatal(err)
	}
	transfer, err := token.BuildTransfer(recipient, big.NewInt(1))
	if err != nil {
		t.Fatal(err)
	}
	beforeOrigin := e.cfg.Origin
	beforeCode := append([]byte(nil), e.state.GetCode(address)...)
	backend := client.rpc.(*localEVMReadBackend)
	if _, err = backend.CallContract(context.Background(), ethereum.CallMsg{From: owner, To: &address, Data: transfer.Data}, nil); err != nil {
		t.Fatal(err)
	}
	balance, err := token.ReadBalanceOf(context.Background(), recipient)
	if err != nil || balance.Value0.Sign() != 0 {
		t.Fatal("read transport committed a write", balance, err)
	}
	if e.cfg.Origin != beforeOrigin || !bytes.Equal(beforeCode, e.state.GetCode(address)) {
		t.Fatal("read transport mutated execution configuration")
	}
}
