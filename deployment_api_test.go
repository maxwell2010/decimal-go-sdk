package decimal

import (
	"context"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestVersionedProfileCannotBePreparedOnMainnet(t *testing.T) {
	f := newFake()
	f.chain = big.NewInt(75)
	main, e := NewClient(Mainnet(), f)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = main.TestnetNFTCenter20261029(testAddress); e == nil {
		t.Fatal("testnet profile accepted on mainnet")
	}
	offline := &Client{}
	contract, _ := offline.TestnetNFTCenter20261029(testAddress)
	meta := TestnetNFTCenter20261029CreateDRC721Arg0{Creator: testAddress, Symbol: "TEST", Name: "test", ContractURI: "ipfs://test"}
	call, e := contract.BuildCreateDRC721(meta)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = main.Prepare(context.Background(), testAddress, call, TxOptions{}); e == nil || f.calls != 0 {
		t.Fatal("cross-network versioned call reached RPC")
	}
}

func TestLoadDeploymentRejectsProxyOnlyABI(t *testing.T) {
	body := `{"Result":{"abi":[{"type":"constructor","stateMutability":"nonpayable","inputs":[{"name":"implementation","type":"address"}]},{"type":"fallback","stateMutability":"payable"}]}}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(body)) }))
	defer server.Close()
	api, e := NewAPI(server.URL, nil)
	if e != nil {
		t.Fatal(e)
	}
	client := testClient(t, newFake())
	if _, e = client.LoadDeploymentContract(context.Background(), api, testAddress); e == nil {
		t.Fatal("proxy-only ABI returned as operational contract")
	}
	// Interface retrieval remains useful for auditing; it must not masquerade as
	// an implementation ABI or automatically invent proxy admin selectors.
	if _, e = api.DeploymentABI(context.Background(), testAddress); e != nil {
		t.Fatal(e)
	}
	body = `{"Result":{"abi":[{"type":"function","name":"owner","stateMutability":"view","inputs":[],"outputs":[{"type":"address"}]}]}}`
	if _, e = client.LoadDeploymentContract(context.Background(), api, testAddress); e != nil {
		t.Fatal("callable deployment interface rejected", e)
	}
}
