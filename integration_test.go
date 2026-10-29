package decimal

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestRPCTransportSigningEndToEnd(t *testing.T) {
	var sent *types.Transaction
	counts := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     json.RawMessage   `json:"id"`
			Method string            `json:"method"`
			Params []json.RawMessage `json:"params"`
		}
		if e := json.NewDecoder(r.Body).Decode(&req); e != nil {
			t.Error(e)
			return
		}
		counts[req.Method]++
		var result any
		switch req.Method {
		case "eth_chainId":
			result = "0x31524"
		case "eth_getTransactionCount":
			result = "0x0"
		case "eth_gasPrice":
			result = "0xa"
		case "eth_estimateGas":
			result = "0x5208"
		case "eth_getBalance":
			result = "0x3635c9adc5dea00000"
		case "eth_sendRawTransaction":
			var raw string
			_ = json.Unmarshal(req.Params[0], &raw)
			tx := new(types.Transaction)
			if e := tx.UnmarshalBinary(common.FromHex(raw)); e != nil {
				t.Error(e)
			}
			sent = tx
			result = tx.Hash().Hex()
		default:
			t.Errorf("unexpected RPC %s", req.Method)
			result = nil
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": result})
	}))
	defer server.Close()
	cfg := Testnet()
	cfg.RPCURL = server.URL
	c, e := Dial(context.Background(), cfg)
	if e != nil {
		t.Fatal(e)
	}
	defer c.Close()
	s := testSigner(t, 1)
	call, _ := NativeTransfer(testAddress, big.NewInt(9), "test memo")
	result, e := c.Send(context.Background(), s, call, TxOptions{})
	if e != nil {
		t.Fatal(e)
	}
	if sent == nil || sent.Hash() != result.Hash || string(sent.Data()) != "test memo" || sent.ChainId().Cmp(big.NewInt(202020)) != 0 || counts["eth_sendRawTransaction"] != 1 {
		t.Fatal("wire transaction differs from intent")
	}
}
func TestAPIRequestRoutesAndFailureIsolation(t *testing.T) {
	paths := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		switch {
		case strings.HasSuffix(r.URL.Path, "bad"):
			http.Error(w, "secret-bearing body", 500)
		case strings.HasSuffix(r.URL.Path, "redirect"):
			http.Redirect(w, r, "https://example.invalid", 302)
		default:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"Result":{"abi":[]},"ok":true}`))
		}
	}))
	defer server.Close()
	a, e := NewAPI(server.URL+"/api/v1/", nil)
	if e != nil {
		t.Fatal(e)
	}
	var out map[string]any
	if e = a.WalletStakes(context.Background(), testAddress, &out); e != nil {
		t.Fatal(e)
	}
	if paths[0] != "/api/v1/validators/wallet/"+strings.ToLower(testAddress.Hex())+"/stakes/coins" {
		t.Fatal(paths)
	}
	for _, path := range []string{"../escape", "%2e%2e/escape", "https://example.com/", "//example.com", "/root", "x?secret=1", "x#fragment"} {
		if e = a.Get(context.Background(), path, nil, &out); e == nil {
			t.Fatalf("unsafe path %s", path)
		}
	}
	for _, path := range []string{"bad", "redirect"} {
		if e = a.Get(context.Background(), path, nil, &out); e == nil || strings.Contains(e.Error(), "secret") {
			t.Fatal("failure isolation")
		}
	}
	if _, e = (Page{Limit: 501}).Query(); e == nil {
		t.Fatal("unbounded pagination")
	}
	q, e := (Page{Limit: 10, Offset: 20}).Query()
	if e != nil || q.Get("offset") != "20" {
		t.Fatal("page query")
	}
	if e = a.Get(context.Background(), "test", url.Values{"x": {"a b"}}, &out); e != nil {
		t.Fatal(e)
	}
}
func TestDeploymentABIAndReferenceOperation(t *testing.T) {
	c := &Client{}
	raw := json.RawMessage(`[{"type":"function","name":"mint","stateMutability":"nonpayable","inputs":[{"name":"to","type":"address"},{"name":"amount","type":"uint256"}],"outputs":[]},{"type":"function","name":"small","stateMutability":"nonpayable","inputs":[{"name":"value","type":"uint24"}],"outputs":[]}]`)
	contract, e := c.ContractWithABI(testAddress, raw)
	if e != nil {
		t.Fatal(e)
	}
	call, e := contract.BuildReferenceOperation("mintTokenReserveless", nil, testAddress, big.NewInt(7))
	if e != nil || len(call.Data) != 68 {
		t.Fatal("deployment mint", e)
	}
	if _, e = contract.BuildNamed("small", nil, new(big.Int).Lsh(big.NewInt(1), 24)); e == nil {
		t.Fatal("uint24 overflow")
	}
	if _, e = contract.BuildNamed("small", nil, big.NewInt(100)); e != nil {
		t.Fatal(e)
	}
	token, _ := c.Token(testAddress)
	if _, e = token.BuildReferenceOperation("updateTokenMinTotalSupply", nil, big.NewInt(1)); e == nil {
		t.Fatal("fabricated absent ABI")
	}
	if _, e = token.BuildReferenceOperation("transferToken", nil, testAddress, big.NewInt(1)); e != nil {
		t.Fatal("reference token transfer", e)
	}
	if len(ReferenceOperations()) != 93 {
		t.Fatal("reference catalog includes commented methods")
	}
}

func TestFrozenTupleReadAndOwnerMismatch(t *testing.T) {
	f := newFake()
	c := testClient(t, f)
	d, _ := c.Delegation(testAddress)
	a, _ := loadABI("Delegation")
	tuple := DelegationGetFrozenStakeOutput0{Stake: DelegationGetFrozenStakeOutput0Stake{Validator: testAddress, Delegator: testAddress, Token: testAddress, Amount: big.NewInt(3), TokenId: new(big.Int), HoldTimestamp: new(big.Int)}, FreezeStatus: 1, FreezeType: 1, UnfreezeTimestamp: big.NewInt(123)}
	f.call = func(_ ethereum.CallMsg, _ *big.Int) ([]byte, error) {
		return a.Methods["getFrozenStake"].Outputs.Pack(tuple)
	}
	r, e := d.ReadGetFrozenStake(context.Background(), big.NewInt(1))
	if e != nil || r.Value0.Stake.Amount.Int64() != 3 {
		t.Fatal("typed tuple conversion", e)
	}
	if _, e = d.FrozenWithdrawals(context.Background(), common.Address{}, []*big.Int{big.NewInt(1)}, big.NewInt(42)); e == nil {
		t.Fatal("wrong wallet accepted")
	}
}
func TestSafeMatchesCSharpOfficialJSVector(t *testing.T) {
	call, _ := NativeTransfer(common.HexToAddress("0x2222222222222222222222222222222222222222"), big.NewInt(123), "")
	call.Data = common.FromHex("0xdeadbeef")
	s := NewSafeTransaction(call, big.NewInt(7))
	address := common.HexToAddress("0x1111111111111111111111111111111111111111")
	h, e := s.Hash(big.NewInt(202020), address)
	if e != nil || h.Hex() != "0x7fa98d43d883784f0a11456fc76bbf398389bad8ed326e30929f102b8b4fe6fd" {
		t.Fatal("cross SDK EIP712 mismatch", e)
	}
	signer, e := NewKeySigner(strings.Repeat("01", 32))
	if e != nil {
		t.Fatal(e)
	}
	defer signer.Close()
	sig, e := s.Sign(context.Background(), signer, big.NewInt(202020), address)
	if e != nil {
		t.Fatal(e)
	}
	if common.Bytes2Hex(sig) != "a50db00f04b398a3d3fc7aebbee71c52f71c574d3374e62e9a8e7c99ff418edc23ab29c20eb033286a85bb7e7f755a41e953c058060e3927600523c5c85003b61b" {
		t.Fatal("cross SDK signature mismatch")
	}
	if _, e = PackSafeSignatures(h, SafeSignature{signer.Address(), sig}); e != nil {
		t.Fatal(e)
	}
	other, _ := s.Hash(big.NewInt(75), address)
	if _, e = PackSafeSignatures(other, SafeSignature{signer.Address(), sig}); e == nil {
		t.Fatal("wrong domain accepted")
	}
}

type maliciousSigner struct{ base *KeySigner }

func (s maliciousSigner) Address() common.Address { return s.base.Address() }
func (s maliciousSigner) SignTransaction(ctx context.Context, tx *types.Transaction, id *big.Int) (*types.Transaction, error) {
	modified := types.NewTx(&types.LegacyTx{Nonce: tx.Nonce(), To: tx.To(), Value: big.NewInt(999), Gas: tx.Gas(), GasPrice: tx.GasPrice(), Data: tx.Data()})
	return s.base.SignTransaction(ctx, modified, id)
}
func TestSignerCannotChangeIntent(t *testing.T) {
	f := newFake()
	c := testClient(t, f)
	s := maliciousSigner{testSigner(t, 1)}
	call, _ := NativeTransfer(testAddress, big.NewInt(1), "")
	if _, e := c.Send(context.Background(), s, call, TxOptions{}); e == nil || len(f.sent) != 0 {
		t.Fatal("mutated intent broadcast")
	}
}
func TestClosedSignerFailsBeforeRPC(t *testing.T) {
	f := newFake()
	c := testClient(t, f)
	s := testSigner(t, 1)
	s.Close()
	call, _ := NativeTransfer(testAddress, big.NewInt(1), "")
	if _, e := c.Send(context.Background(), s, call, TxOptions{}); e == nil || f.calls != 0 {
		t.Fatal("closed signer touched RPC")
	}
}

type dataError struct{}

func (dataError) Error() string { return "RPC error with secret" }
func (dataError) ErrorData() any {
	a, _ := loadABI("Delegation")
	err := a.Errors["InvalidAmount"]
	return err.ID.Hex()[:10]
}
func TestCustomRevertAndEvents(t *testing.T) {
	a, _ := loadABI("Delegation")
	e := decodeRPCError(dataError{}, a)
	var revert *RevertError
	if !errors.As(e, &revert) || revert.Name != "InvalidAmount" {
		t.Fatal("custom revert", e)
	}
	c := &Client{}
	token, _ := c.Token(testAddress)
	ta, _ := loadABI("Token")
	event := ta.Events["Transfer"]
	payload, e := event.Inputs.NonIndexed().Pack(big.NewInt(12))
	if e != nil {
		t.Fatal(e)
	}
	log := types.Log{Address: testAddress, Topics: []common.Hash{event.ID, common.BytesToHash(testAddress.Bytes()), common.BytesToHash(testAddress.Bytes())}, Data: payload}
	decoded, e := token.DecodeLog(log)
	if e != nil || decoded.Name != "Transfer" {
		t.Fatal("event decode", e)
	}
	if _, ok := decoded.Values["value"].(*big.Int); !ok {
		t.Fatal("event amount")
	}
}

func TestFrozenBatchUsesOneCallAndPinnedBlock(t *testing.T) {
	f := newFake()
	c := testClient(t, f)
	d, _ := c.Delegation(testAddress)
	a, _ := loadABI("Delegation")
	calls := 0
	state := DelegationGetFrozenStakesOutput0{Stake: DelegationGetFrozenStakesOutput0Stake{Validator: testAddress, Delegator: testAddress, Token: testAddress, Amount: big.NewInt(3), TokenId: new(big.Int), HoldTimestamp: new(big.Int)}, FreezeStatus: 1, FreezeType: 1, UnfreezeTimestamp: big.NewInt(10)}
	f.call = func(m ethereum.CallMsg, b *big.Int) ([]byte, error) {
		calls++
		if b == nil || b.Int64() != 42 {
			t.Error("batch snapshot lost")
		}
		if string(m.Data[:4]) != string(a.Methods["getFrozenStakes"].ID) {
			t.Error("single getter used")
		}
		return a.Methods["getFrozenStakes"].Outputs.Pack([]DelegationGetFrozenStakesOutput0{state, state})
	}
	rows, e := d.FrozenWithdrawalsBatch(context.Background(), testAddress, []*big.Int{big.NewInt(7), big.NewInt(8)}, big.NewInt(42))
	if e != nil || len(rows) != 2 || calls != 1 {
		t.Fatal("batch read", e)
	}
}
func TestWithdrawalIndexEmptyIsDifferentFromUnavailable(t *testing.T) {
	response := `{"Result":{"withdrawals_authoritative":true,"withdrawals_fresh":true,"withdrawals_checked_block":42,"withdrawals":[]}}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(response)) }))
	defer server.Close()
	api, _ := NewAPI(server.URL, nil)
	result, e := api.WithdrawalIndex(context.Background(), "staking", big.NewInt(42))
	if e != nil || len(result.Entries) != 0 || result.Block.Int64() != 42 {
		t.Fatal("authoritative empty", e)
	}
	if _, e = api.WithdrawalIndex(context.Background(), "staking", big.NewInt(43)); e == nil {
		t.Fatal("stale block accepted")
	}
	response = `{"Result":{"withdrawals":[]}}`
	if _, e = api.WithdrawalIndex(context.Background(), "staking", nil); e == nil {
		t.Fatal("unavailable became empty")
	}
}
