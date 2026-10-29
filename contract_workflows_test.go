package decimal

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWithdrawalIndexRejectsMissingEntries(t *testing.T) {
	response := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(response)) }))
	defer server.Close()
	api, _ := NewAPI(server.URL, nil)
	for _, entries := range []string{"", `,"withdrawals":null`} {
		response = `{"Result":{"withdrawals_authoritative":true,"withdrawals_fresh":true,"withdrawals_checked_block":42` + entries + `}}`
		if _, e := api.WithdrawalIndex(context.Background(), "staking", nil); e == nil {
			t.Fatal("missing/null interpreted as authoritative empty")
		}
	}
	for _, n := range []string{`"+1"`, `"-0"`, `" 1"`, `1.5`, `1e3`, `null`} {
		if _, e := jsonInteger(json.RawMessage(n)); e == nil {
			t.Fatalf("noncanonical index accepted: %s", n)
		}
	}
}
func TestCheckedNFTPermitValidatesNonceOwnerDomain(t *testing.T) {
	for _, kind := range []string{"ERC721", "ERC1155"} {
		t.Run(kind, func(t *testing.T) {
			f := newFake()
			c := testClient(t, f)
			s := testSigner(t, 1)
			collection, _ := c.Contract(kind, testAddress)
			a, _ := loadABI(kind)
			var id *big.Int
			if kind == "ERC721" {
				id = big.NewInt(7)
			}
			nonce := big.NewInt(3)
			deadline := big.NewInt(100)
			d, _ := NFTPermitTypedData("NFT", "1", c.ChainID(), testAddress, testAddress, id, nonce, deadline)
			domain, _ := typedDomainHash(d)
			badDomain := false
			wrongOwner := false
			f.call = func(m ethereum.CallMsg, _ *big.Int) ([]byte, error) {
				method, e := a.MethodById(m.Data[:4])
				if e != nil {
					return nil, e
				}
				switch method.RawName {
				case "ownerOf":
					owner := s.Address()
					if wrongOwner {
						owner = common.Address{}
					}
					return method.Outputs.Pack(owner)
				case "nonces":
					return method.Outputs.Pack(nonce)
				case "DOMAIN_SEPARATOR":
					v := [32]byte(domain)
					if badDomain {
						v[0] ^= 1
					}
					return method.Outputs.Pack(v)
				}
				return nil, errors.New("unexpected")
			}
			if _, e := c.CheckedNFTPermit(context.Background(), s, collection, "NFT", "1", testAddress, id, nonce, deadline); e != nil {
				t.Fatal(e)
			}
			if _, e := c.CheckedNFTPermit(context.Background(), s, collection, "NFT", "1", testAddress, id, big.NewInt(4), deadline); e == nil {
				t.Fatal("stale nonce accepted")
			}
			badDomain = true
			if _, e := c.CheckedNFTPermit(context.Background(), s, collection, "NFT", "1", testAddress, id, nonce, deadline); e == nil {
				t.Fatal("wrong domain accepted")
			}
			badDomain = false
			if id != nil {
				wrongOwner = true
				if _, e := c.CheckedNFTPermit(context.Background(), s, collection, "NFT", "1", testAddress, id, nonce, deadline); e == nil {
					t.Fatal("wrong owner accepted")
				}
			}
		})
	}
}
func TestSafeExecutionRejectsInvalidOperation(t *testing.T) {
	c := &Client{}
	contract, _ := c.ContractWithABI(testAddress, json.RawMessage(`[]`))
	call, _ := NativeTransfer(testAddress, big.NewInt(1), "")
	tx := NewSafeTransaction(call, big.NewInt(0))
	tx.Operation = 2
	if _, e := tx.BuildExecution(contract, []byte{1}); e == nil {
		t.Fatal("invalid operation accepted")
	}
}
