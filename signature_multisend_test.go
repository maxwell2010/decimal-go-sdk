package decimal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"math/big"
	"testing"
)

func TestExplicitContractSignatureInterfaces(t *testing.T) {
	for _, kind := range []ContractSignatureInterface{ERC1271Bytes32, SafeLegacyBytes} {
		t.Run(fmt.Sprint(kind), func(t *testing.T) {
			f := newFake()
			c := testClient(t, f)
			typ := "bytes32"
			magic := [4]byte{0x16, 0x26, 0xba, 0x7e}
			if kind == SafeLegacyBytes {
				typ = "bytes"
				magic = [4]byte{0x20, 0xc1, 0x3b, 0x0b}
			}
			raw := json.RawMessage(fmt.Sprintf(`[{"type":"function","name":"isValidSignature","stateMutability":"view","inputs":[{"type":%q},{"type":"bytes"}],"outputs":[{"type":"bytes4"}]}]`, typ))
			contract, _ := c.ContractWithABI(testAddress, raw)
			a, _ := contract.contractABI()
			preimage := []byte("message")
			hash := crypto.Keccak256Hash(preimage)
			f.call = func(m ethereum.CallMsg, block *big.Int) ([]byte, error) {
				if block == nil || block.Int64() != 42 {
					t.Fatal("signature call not pinned")
				}
				method, e := a.MethodById(m.Data[:4])
				if e != nil {
					return nil, e
				}
				values, e := method.Inputs.Unpack(m.Data[4:])
				if e != nil {
					return nil, e
				}
				if kind == SafeLegacyBytes {
					if string(values[0].([]byte)) != "message" {
						t.Fatal("legacy preimage")
					}
				} else if values[0].([32]byte) != [32]byte(hash) {
					t.Fatal("modern digest")
				}
				return method.Outputs.Pack(magic)
			}
			verifier, e := c.ContractSignatureVerifier(raw, big.NewInt(42), kind, preimage)
			if e != nil {
				t.Fatal(e)
			}
			raw[0] = 'X'
			preimage[0] = 'X'
			if e = verifier(context.Background(), testAddress, hash, []byte{1}); e != nil {
				t.Fatal(e)
			}
			magic = [4]byte{0xff}
			if e = verifier(context.Background(), testAddress, hash, []byte{1}); e == nil {
				t.Fatal("wrong magic accepted")
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			if e = verifier(ctx, testAddress, hash, []byte{1}); !errors.Is(e, context.Canceled) {
				t.Fatal("cancel", e)
			}
		})
	}
}

func TestRestrictedMultisendAllowanceAndPermit(t *testing.T) {
	f := newFake()
	c := testClient(t, f)
	s := testSigner(t, 1)
	deployment := RestrictedMultisendDeployment{Address: common.HexToAddress("0x2000000000000000000000000000000000000002"), ChainID: c.ChainID(), RuntimeHash: crypto.Keccak256Hash(f.code)}
	batch, e := c.RestrictedMultisend(deployment)
	if e != nil {
		t.Fatal(e)
	}
	token := testAddress
	recipients := []Recipient{{common.HexToAddress("0x3000000000000000000000000000000000000003"), big.NewInt(10)}}
	data, e := PermitTypedData("Fixture", "1", c.ChainID(), token, Permit{Owner: s.Address(), Spender: deployment.Address, Value: big.NewInt(10), Nonce: new(big.Int), Deadline: big.NewInt(9999999999)})
	if e != nil {
		t.Fatal(e)
	}
	domain, _ := typedDomainHash(data)
	ta, _ := loadABI("Token")
	allowance := big.NewInt(100)
	f.call = func(msg ethereum.CallMsg, _ *big.Int) ([]byte, error) {
		m, e := ta.MethodById(msg.Data[:4])
		if e != nil {
			return nil, e
		}
		switch m.RawName {
		case "balanceOf":
			return m.Outputs.Pack(big.NewInt(100))
		case "allowance":
			return m.Outputs.Pack(allowance)
		case "nonces":
			return m.Outputs.Pack(new(big.Int))
		case "DOMAIN_SEPARATOR":
			return m.Outputs.Pack([32]byte(domain))
		}
		return nil, errors.New("unexpected call")
	}
	prepared, e := batch.PrepareToken(context.Background(), s.Address(), token, recipients, "memo", TxOptions{})
	if e != nil || prepared.Transaction.To() == nil || *prepared.Transaction.To() != deployment.Address {
		t.Fatal("allowance route", e)
	}
	raw, e := s.SignTypedData(context.Background(), data)
	if e != nil {
		t.Fatal(e)
	}
	sig, e := SplitSignature(raw)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = batch.PrepareTokenWithPermit(context.Background(), s.Address(), token, recipients, "memo", "Fixture", "1", big.NewInt(9999999999), sig, TxOptions{}); e != nil {
		t.Fatal("permit route", e)
	}
	if _, e = batch.PrepareTokenWithPermit(context.Background(), common.HexToAddress("0x09"), token, recipients, "memo", "Fixture", "1", big.NewInt(9999999999), sig, TxOptions{}); e == nil {
		t.Fatal("permit accepted for another sender")
	}
	sig.V -= 27
	if _, e = batch.BuildTokenWithPermit(s.Address(), token, recipients, "", big.NewInt(1), sig); e == nil {
		t.Fatal("invalid permit v accepted")
	}
	allowance.SetInt64(0)
	if _, e = batch.PrepareToken(context.Background(), s.Address(), token, recipients, "", TxOptions{}); e == nil {
		t.Fatal("missing allowance")
	}
	f.code = []byte{2}
	if e = batch.Verify(context.Background()); e == nil {
		t.Fatal("runtime mismatch")
	}
	if len(f.sent) != 0 {
		t.Fatal("multisend broadcast")
	}
}
