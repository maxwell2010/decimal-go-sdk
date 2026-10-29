package decimal

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// This official upstream mock validates STANDARD EIP2612 execution only. It is
// not a Decimal token fixture or evidence of any public deployment's permit.
func TestLocalEVMStandardPermitProvenance(t *testing.T) {
	raw, err := os.ReadFile("testdata/evm/permit-manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Commit string            `json:"commit"`
		Files  map[string]string `json:"files"`
	}
	if err = json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Commit != "64e48203cecad94f02de9891ecdeed4d629c6dae" || manifest.Files["permit-input.json"] != "46591f61c9000a86c291a2019854657bb585f65ff74bb61db3f1e0f01c8a9081" {
		t.Fatal("official standard fixture pin changed")
	}
	for name, want := range manifest.Files {
		if filepath.Base(name) != name {
			t.Fatal("invalid fixture path")
		}
		data, err := os.ReadFile(filepath.Join("testdata/evm", name))
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(data)
		if hex.EncodeToString(sum[:]) != want {
			t.Fatal("permit fixture hash changed", name)
		}
	}
}
func TestLocalEVMStandardPermitLifecycle(t *testing.T) {
	for _, scenario := range []string{"valid-and-replay", "wrong-name", "wrong-chain", "wrong-contract", "wrong-nonce", "expired"} {
		t.Run(scenario, func(t *testing.T) {
			ctx := context.Background()
			e := newLocalEVM(t)
			artifact := loadLocalArtifact(t, "permit-compiled.json")
			signer := testSigner(t, 1)
			defer signer.Close()
			spender := common.HexToAddress("0x1002")
			address := e.deploy(t, signer.Address(), artifact.ABI, common.FromHex(artifact.Bytecode), "StandardPermit", "SPT", signer.Address(), big.NewInt(1000))
			client := e.client(t)
			token, err := client.Token(address)
			if err != nil {
				t.Fatal(err)
			}
			p := Permit{Owner: signer.Address(), Spender: spender, Value: big.NewInt(17), Nonce: big.NewInt(0), Deadline: big.NewInt(100)}
			sig, err := client.CheckedPermit(ctx, signer, address, "StandardPermit", "1", p)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = client.CheckedPermit(ctx, signer, address, "Wrong", "1", p); err == nil {
				t.Fatal("checked helper ignored actual domain")
			}
			wrongNonce := p
			wrongNonce.Nonce = big.NewInt(1)
			if _, err = client.CheckedPermit(ctx, signer, address, "StandardPermit", "1", wrongNonce); err == nil {
				t.Fatal("checked helper ignored actual nonce")
			}
			if scenario != "valid-and-replay" {
				name := "StandardPermit"
				chain := big.NewInt(202020)
				verifier := address
				signed := p
				switch scenario {
				case "wrong-name":
					name = "Wrong"
				case "wrong-chain":
					chain = big.NewInt(202021)
				case "wrong-contract":
					verifier = common.HexToAddress("0xdead")
				case "wrong-nonce":
					signed.Nonce = big.NewInt(1)
				case "expired":
					p.Deadline = big.NewInt(0)
					signed = p
				}
				typed, err := PermitTypedData(name, "1", chain, verifier, signed)
				if err != nil {
					t.Fatal(err)
				}
				raw, err := signer.SignTypedData(ctx, typed)
				if err != nil {
					t.Fatal(err)
				}
				sig, err = SplitSignature(raw)
				if err != nil {
					t.Fatal(err)
				}
			}
			call, err := token.BuildPermit(p.Owner, p.Spender, p.Value, p.Deadline, sig.V, sig.R, sig.S)
			if err != nil {
				t.Fatal(err)
			}
			output, logs, err := e.execute(t, spender, call)
			valid := scenario == "valid-and-replay"
			if (err == nil) != valid {
				t.Fatalf("permit result %v valid=%v", err, valid)
			}
			nonce, err := token.ReadNonces(ctx, signer.Address())
			if err != nil {
				t.Fatal(err)
			}
			allowance, err := token.ReadAllowance(ctx, signer.Address(), spender)
			if err != nil {
				t.Fatal(err)
			}
			wantNonce := int64(0)
			wantAllowance := int64(0)
			if valid {
				wantNonce = 1
				wantAllowance = 17
			}
			if nonce.Value0.Cmp(big.NewInt(wantNonce)) != 0 || allowance.Value0.Cmp(big.NewInt(wantAllowance)) != 0 {
				t.Fatal("permit state mismatch", nonce, allowance)
			}
			if !valid {
				reason, decodeErr := abi.UnpackRevert(output)
				wantReason := "invalid signature"
				if scenario == "expired" {
					wantReason = "expired deadline"
				}
				if decodeErr != nil || !strings.Contains(reason, wantReason) {
					t.Fatal("unexpected permit rejection", reason, decodeErr)
				}
				if len(logs) != 0 {
					t.Fatal("failed permit retained logs")
				}
				return
			}
			if len(logs) != 1 {
				t.Fatal("missing approval event")
			}
			event, err := token.DecodeLog(*logs[0])
			if err != nil || event.Name != "Approval" {
				t.Fatal("wrong permit event", event, err)
			}
			if _, logs, err = e.execute(t, spender, call); err == nil || len(logs) != 0 {
				t.Fatal("permit replay accepted or retained logs", err)
			}
			nonce, err = token.ReadNonces(ctx, signer.Address())
			if err != nil || nonce.Value0.Cmp(big.NewInt(1)) != 0 {
				t.Fatal("replay mutated nonce", nonce, err)
			}
			transfer, err := token.BuildTransferFrom(signer.Address(), spender, big.NewInt(7))
			if err != nil {
				t.Fatal(err)
			}
			if _, _, err = e.execute(t, spender, transfer); err != nil {
				t.Fatal(err)
			}
			balance, err := token.ReadBalanceOf(ctx, spender)
			if err != nil || balance.Value0.Cmp(big.NewInt(7)) != 0 {
				t.Fatal("permit allowance could not transfer", balance, err)
			}
			allowance, err = token.ReadAllowance(ctx, signer.Address(), spender)
			if err != nil || allowance.Value0.Cmp(big.NewInt(10)) != 0 {
				t.Fatal("permit allowance consumption", allowance, err)
			}
		})
	}
}
