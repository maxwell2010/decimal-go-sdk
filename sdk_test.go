package decimal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

var testAddress = common.HexToAddress("0x1000000000000000000000000000000000000001")

type fakeRPC struct {
	mu         sync.Mutex
	chain      *big.Int
	balance    *big.Int
	price      *big.Int
	gas        uint64
	code       []byte
	nonce      uint64
	sent       []*types.Transaction
	sendErr    error
	receipt    *types.Receipt
	receiptErr error
	calls      int
	call       func(ethereum.CallMsg, *big.Int) ([]byte, error)
}

func newFake() *fakeRPC {
	return &fakeRPC{chain: big.NewInt(202020), balance: new(big.Int).Exp(big.NewInt(10), big.NewInt(30), nil), price: big.NewInt(10), gas: 21000, code: []byte{1}}
}
func (f *fakeRPC) ChainID(context.Context) (*big.Int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	return copyInt(f.chain), nil
}
func (f *fakeRPC) PendingNonceAt(context.Context, common.Address) (uint64, error) {
	return f.nonce, nil
}
func (f *fakeRPC) SuggestGasPrice(context.Context) (*big.Int, error)             { return copyInt(f.price), nil }
func (f *fakeRPC) EstimateGas(context.Context, ethereum.CallMsg) (uint64, error) { return f.gas, nil }
func (f *fakeRPC) BalanceAt(context.Context, common.Address, *big.Int) (*big.Int, error) {
	return copyInt(f.balance), nil
}
func (f *fakeRPC) CodeAt(context.Context, common.Address, *big.Int) ([]byte, error) {
	return f.code, nil
}
func (f *fakeRPC) CallContract(_ context.Context, m ethereum.CallMsg, b *big.Int) ([]byte, error) {
	if f.call != nil {
		return f.call(m, b)
	}
	return nil, errors.New("no fixture")
}
func (f *fakeRPC) SendTransaction(_ context.Context, tx *types.Transaction) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, tx)
	return f.sendErr
}
func (f *fakeRPC) TransactionReceipt(context.Context, common.Hash) (*types.Receipt, error) {
	if f.receipt == nil && f.receiptErr == nil {
		return nil, ethereum.NotFound
	}
	return f.receipt, f.receiptErr
}
func testClient(t *testing.T, f *fakeRPC) *Client {
	t.Helper()
	c, e := NewClient(Testnet(), f)
	if e != nil {
		t.Fatal(e)
	}
	return c
}
func testSigner(t *testing.T, n int) *KeySigner {
	t.Helper()
	s, e := NewKeySigner(fmt.Sprintf("%064x", n))
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(s.Close)
	return s
}

func TestExactUnits(t *testing.T) {
	for _, s := range []string{"0", "1", "0.000000000000000001", "1234567890123456789012345.123456789012345678"} {
		n, e := ParseUnits(s, 18)
		if e != nil {
			t.Fatal(e)
		}
		if got := FormatUnits(n, 18); got != s {
			t.Fatalf("round trip %s != %s", got, s)
		}
	}
	for _, s := range []string{"-1", "+1", "1e3", "NaN", "1.0000000000000000001", " 1", "1.", ".1", "1..2", new(big.Int).Lsh(big.NewInt(1), 256).String()} {
		if _, e := ParseUnits(s, 18); e == nil {
			t.Fatalf("accepted %q", s)
		}
	}
	if got := FormatUnits(big.NewInt(1200), 3); got != "1.2" {
		t.Fatal(got)
	}
}
func TestOfflineBuildersAndOverloads(t *testing.T) {
	c := &Client{}
	token, _ := c.Token(testAddress)
	call, e := token.BuildTransfer(testAddress, big.NewInt(17))
	if e != nil {
		t.Fatal(e)
	}
	if common.Bytes2Hex(call.Data[:4]) != "a9059cbb" || call.Value.Sign() != 0 {
		t.Fatal("ERC20 transfer selector/value")
	}
	d, _ := c.Delegation(testAddress)
	call, e = d.BuildDelegateDEL(big.NewInt(123), testAddress)
	if e != nil || call.Value.Cmp(big.NewInt(123)) != 0 {
		t.Fatal("delegate value", e)
	}
	if _, e = token.BuildTransfer(testAddress, big.NewInt(-1)); e == nil {
		t.Fatal("negative amount")
	}
	if _, e = token.BuildTransfer(testAddress, nil); e == nil {
		t.Fatal("nil amount")
	}
	if _, e = token.Build("approve(address,uint256)", big.NewInt(1), testAddress, big.NewInt(1)); e == nil {
		t.Fatal("nonpayable with value")
	}
	if _, e = token.Build("mint(address,uint256)", nil, testAddress, big.NewInt(1)); e == nil {
		t.Fatal("reserve token cannot mint")
	}
	nft, _ := c.ERC1155(testAddress)
	if _, e = nft.BuildSafeBatchTransferFrom(testAddress, testAddress, []*big.Int{big.NewInt(1)}, []*big.Int{big.NewInt(2)}, nil); e != nil {
		t.Fatal(e)
	}
	center, _ := c.TokenCenter(testAddress)
	_, e = center.BuildCreateToken(big.NewInt(1000), TokenCenterCreateTokenArg0{InitialMint: big.NewInt(1), MinTotalSupply: big.NewInt(1), MaxTotalSupply: big.NewInt(100), Creator: testAddress, Crr: 50, Identity: "id", Symbol: "TST", Name: "test"})
	if e != nil {
		t.Fatal(e)
	}
}
func TestFeeQuoteAndBroadcastSeparation(t *testing.T) {
	f := newFake()
	c := testClient(t, f)
	s := testSigner(t, 1)
	call, _ := NativeTransfer(testAddress, big.NewInt(77), "memo")
	p, e := c.Prepare(context.Background(), s.Address(), call, TxOptions{})
	if e != nil {
		t.Fatal(e)
	}
	if p.Fee.GasLimit != 23100 || p.Fee.EstimatedFee.Int64() != 210000 || p.Fee.ReservedFee.Int64() != 231000 || p.Fee.RequiredBalance.Int64() != 231077 {
		t.Fatal("fee math")
	}
	if len(f.sent) != 0 {
		t.Fatal("Prepare broadcast")
	}
	signed, e := s.SignTransaction(context.Background(), p.Transaction, c.ChainID())
	if e != nil {
		t.Fatal(e)
	}
	from, e := types.Sender(types.LatestSignerForChainID(c.ChainID()), signed)
	if e != nil || from != s.Address() {
		t.Fatal("signature")
	}
	if string(signed.Data()) != "memo" {
		t.Fatal("memo lost")
	}
	h, e := c.Broadcast(context.Background(), signed)
	if e != nil || h != signed.Hash() || len(f.sent) != 1 {
		t.Fatal("broadcast", e)
	}
}
func TestPreflightRejectsWithoutBroadcast(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*fakeRPC, *TxOptions)
	}{
		{"chain", func(f *fakeRPC, o *TxOptions) { f.chain = big.NewInt(75) }},
		{"price cap", func(f *fakeRPC, o *TxOptions) { o.MaxGasPrice = big.NewInt(9) }},
		{"fee cap", func(f *fakeRPC, o *TxOptions) { o.MaxFee = big.NewInt(1) }},
		{"insufficient balance", func(f *fakeRPC, o *TxOptions) { f.balance = big.NewInt(1) }},
		{"gas under estimate", func(f *fakeRPC, o *TxOptions) { o.GasLimit = 20000 }},
		{"bad multiplier", func(f *fakeRPC, o *TxOptions) { o.GasMultiplierBPS = 9999 }},
		{"no code", func(f *fakeRPC, o *TxOptions) { f.code = nil }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFake()
			o := TxOptions{}
			tc.change(f, &o)
			c := testClient(t, f)
			s := testSigner(t, 1)
			d, _ := c.Delegation(testAddress)
			call, _ := d.BuildDelegateDEL(big.NewInt(1), testAddress)
			if _, e := c.Send(context.Background(), s, call, o); e == nil {
				t.Fatal("accepted")
			}
			if len(f.sent) > 0 {
				t.Fatal("broadcast")
			}
		})
	}
}
func TestConcurrentAutomaticNonces(t *testing.T) {
	f := newFake()
	c := testClient(t, f)
	s := testSigner(t, 1)
	call, _ := NativeTransfer(testAddress, big.NewInt(1), "")
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, e := c.Send(context.Background(), s, call, TxOptions{}); e != nil {
				t.Error(e)
			}
		}()
	}
	wg.Wait()
	if len(f.sent) != 16 {
		t.Fatal("missing sends")
	}
	seen := map[uint64]bool{}
	for _, tx := range f.sent {
		if seen[tx.Nonce()] {
			t.Fatal("duplicate nonce")
		}
		seen[tx.Nonce()] = true
	}
	if !seen[0] || !seen[15] {
		t.Fatal("nonces not sequential")
	}
}
func TestUnknownOutcomeIsNeverRetried(t *testing.T) {
	f := newFake()
	f.sendErr = errors.New("timeout with credential data that must not leak")
	c := testClient(t, f)
	s := testSigner(t, 1)
	call, _ := NativeTransfer(testAddress, big.NewInt(1), "")
	r, e := c.Send(context.Background(), s, call, TxOptions{})
	var unknown *SubmissionUnknownError
	if !errors.As(e, &unknown) || r.Hash != unknown.Hash || len(f.sent) != 1 {
		t.Fatal("lost ambiguous result")
	}
	if strings.Contains(e.Error(), "credential") {
		t.Fatal("RPC error leaked")
	}
	f.sendErr = nil
	r, e = c.Send(context.Background(), s, call, TxOptions{})
	if e != nil || r.Transaction.Nonce() != 1 {
		t.Fatal("unknown nonce reused")
	}
}
func TestReceiptsAndActualFee(t *testing.T) {
	f := newFake()
	f.receipt = &types.Receipt{TxHash: common.HexToHash("0x02"), Status: 0, BlockNumber: big.NewInt(1), BlockHash: common.HexToHash("0x01"), GasUsed: 21000, EffectiveGasPrice: big.NewInt(7)}
	c := testClient(t, f)
	r, e := c.WaitReceipt(context.Background(), common.HexToHash("0x02"), time.Millisecond)
	var reverted *RevertedError
	if !errors.As(e, &reverted) || r.ActualFee.Int64() != 147000 {
		t.Fatal("reverted receipt counted successful")
	}
	f.receipt.Status = 1
	f.receipt.EffectiveGasPrice = nil
	r, e = c.WaitReceipt(context.Background(), common.HexToHash("0x02"), time.Millisecond)
	if e != nil || r.ActualFee != nil {
		t.Fatal("missing actual fee fabricated")
	}
	f.receipt = nil
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
	defer cancel()
	if _, e = c.WaitReceipt(ctx, common.HexToHash("0x02"), time.Millisecond); !errors.Is(e, context.DeadlineExceeded) {
		t.Fatal("receipt ignores cancellation")
	}
}
func TestWalletDerivationAndClose(t *testing.T) {
	// Public BIP39 test vector, not a funded wallet or user secret.
	words := strings.Repeat("abandon ", 11) + "about"
	s, e := SignerFromMnemonic(words, "", "")
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	if s.Address() != common.HexToAddress("0x9858EfFD232B4033E47d90003D41EC34EcaEda94") {
		t.Fatal("BIP44 path mismatch")
	}
	j, _ := json.Marshal(s)
	if strings.Contains(string(j), "key") {
		t.Fatal("key in JSON")
	}
	s.Close()
	if _, e = s.SignDigest(context.Background(), common.Hash{}); e == nil {
		t.Fatal("closed signer usable")
	}
}
func TestSafeSignaturesAndDomain(t *testing.T) {
	s1 := testSigner(t, 1)
	s2 := testSigner(t, 2)
	call, _ := NativeTransfer(testAddress, big.NewInt(1), "")
	safe := NewSafeTransaction(call, big.NewInt(7))
	h, e := safe.Hash(big.NewInt(202020), testAddress)
	if e != nil {
		t.Fatal(e)
	}
	a, e := safe.Sign(context.Background(), s1, big.NewInt(202020), testAddress)
	if e != nil {
		t.Fatal(e)
	}
	b, _ := safe.Sign(context.Background(), s2, big.NewInt(202020), testAddress)
	p, e := PackSafeEOASignatures(h, b, a)
	if e != nil || len(p) != 130 {
		t.Fatal("packing", e)
	}
	if _, e = PackSafeEOASignatures(h, a, a); e == nil {
		t.Fatal("duplicate signer")
	}
	h2, _ := safe.Hash(big.NewInt(75), testAddress)
	if h == h2 {
		t.Fatal("Safe domain missing chain")
	}
	decoded := append([]byte(nil), p[:65]...)
	decoded[64] -= 27
	pub, e := crypto.SigToPub(h[:], decoded)
	if e != nil {
		t.Fatal(e)
	}
	first := crypto.PubkeyToAddress(*pub)
	if first.Cmp(s1.Address()) > 0 || first.Cmp(s2.Address()) > 0 {
		t.Fatal("signatures unsorted")
	}
}
func TestFrozenWithdrawalUsesContractState(t *testing.T) {
	f := FrozenWithdrawal{State: DelegationGetFrozenStakeOutput0{Stake: DelegationGetFrozenStakeOutput0Stake{Amount: big.NewInt(1)}, FreezeType: 1, FreezeStatus: 1, UnfreezeTimestamp: big.NewInt(10)}}
	if !f.Active() || !f.Matured(11) {
		t.Fatal("active withdrawal missing")
	}
	f.State.FreezeStatus = 2
	if f.Active() || f.Matured(11) {
		t.Fatal("completed withdrawal active")
	}
	f.State.FreezeStatus = 1
	f.State.UnfreezeTimestamp = big.NewInt(0)
	if f.Matured(11) {
		t.Fatal("zero timestamp matured")
	}
}
func TestRegistryUsesNetworkAndTupleRead(t *testing.T) {
	f := newFake()
	c := testClient(t, f)
	a, _ := loadABI("ContractCenter")
	f.call = func(m ethereum.CallMsg, _ *big.Int) ([]byte, error) {
		if *m.To != Testnet().ContractCenter {
			t.Error("wrong registry network")
		}
		return a.Methods["getContractAddress"].Outputs.Pack(testAddress)
	}
	address, e := c.ResolveSystemContract(context.Background(), "delegation")
	if e != nil || address != testAddress {
		t.Fatal("resolve", e)
	}
	d, _ := c.Delegation(testAddress)
	abi, _ := loadABI("Delegation")
	tuple := DelegationGetFrozenStakeOutput0{Stake: DelegationGetFrozenStakeOutput0Stake{Validator: testAddress, Delegator: testAddress, Token: testAddress, Amount: big.NewInt(3), TokenId: new(big.Int), TokenType: 1, HoldTimestamp: new(big.Int)}, FreezeStatus: 1, FreezeType: 1, UnfreezeTimestamp: big.NewInt(123)}
	f.call = func(m ethereum.CallMsg, block *big.Int) ([]byte, error) {
		if block == nil || block.Int64() != 42 {
			t.Error("snapshot block not passed")
		}
		return abi.Methods["getFrozenStake"].Outputs.Pack(tuple)
	}
	rows, e := d.FrozenWithdrawals(context.Background(), testAddress, []*big.Int{big.NewInt(7), big.NewInt(7)}, big.NewInt(42))
	if e != nil || len(rows) != 1 || rows[0].State.Stake.Amount.Int64() != 3 {
		t.Fatal("tuple read", e)
	}
}
func TestMultisendAndAllowanceSafety(t *testing.T) {
	c := testClient(t, newFake())
	call, e := c.BuildMultisendDEL([]Recipient{{testAddress, big.NewInt(2)}, {testAddress, big.NewInt(3)}})
	if e != nil || call.Value.Int64() != 5 {
		t.Fatal("multisend", e)
	}
	if _, e = c.BuildMultisendERC20(); e == nil {
		t.Fatal("unsafe ERC20 aggregate")
	}
	a, _ := NativeTransfer(c.cfg.Multicall, big.NewInt(0), "")
	if _, e = c.PlanERC20Action(context.Background(), testAddress, testAddress, c.cfg.Multicall, big.NewInt(1), a); e == nil {
		t.Fatal("unsafe spender approval")
	}
}
