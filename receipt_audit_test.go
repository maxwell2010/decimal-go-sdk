package decimal

import (
	"context"
	"errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"math/big"
	"testing"
	"time"
)

type canceledSuccessBackend struct {
	*fakeRPC
	cancel context.CancelFunc
	chain  bool
}

func (b *canceledSuccessBackend) ChainID(ctx context.Context) (*big.Int, error) {
	if b.chain {
		b.cancel()
	}
	return b.fakeRPC.ChainID(ctx)
}
func (b *canceledSuccessBackend) TransactionReceipt(ctx context.Context, h common.Hash) (*types.Receipt, error) {
	b.cancel()
	return b.fakeRPC.TransactionReceipt(ctx, h)
}
func TestReceiptRequiresMinedCoordinates(t *testing.T) {
	tests := map[string]func(*types.Receipt){
		"missing height":  func(r *types.Receipt) { r.BlockNumber = nil },
		"zero height":     func(r *types.Receipt) { r.BlockNumber.SetInt64(0) },
		"negative height": func(r *types.Receipt) { r.BlockNumber.SetInt64(-1) },
		"overflow height": func(r *types.Receipt) { r.BlockNumber.Lsh(big.NewInt(1), 64) },
		"zero hash":       func(r *types.Receipt) { r.BlockHash = common.Hash{} },
		"invalid status":  func(r *types.Receipt) { r.Status = 2 },
		"negative price":  func(r *types.Receipt) { r.EffectiveGasPrice = big.NewInt(-1) },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			f := newFake()
			f.receipt = &types.Receipt{TxHash: common.HexToHash("0x02"), Status: 1, BlockNumber: big.NewInt(1), BlockHash: common.HexToHash("0x01")}
			mutate(f.receipt)
			if m, e := testClient(t, f).WaitReceipt(context.Background(), common.HexToHash("0x02"), time.Millisecond); e == nil || m != nil {
				t.Fatalf("invalid receipt accepted: %v %v", m, e)
			}
		})
	}
}
func TestSuccessfulBackendCannotHideCancellation(t *testing.T) {
	for _, chain := range []bool{false, true} {
		t.Run(map[bool]string{false: "receipt", true: "chain"}[chain], func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			f := newFake()
			f.receipt = &types.Receipt{TxHash: common.HexToHash("0x02"), Status: 1, BlockNumber: big.NewInt(1), BlockHash: common.HexToHash("0x01")}
			c, e := NewClient(Testnet(), &canceledSuccessBackend{f, cancel, chain})
			if e != nil {
				t.Fatal(e)
			}
			if _, e = c.WaitReceipt(ctx, common.HexToHash("0x02"), time.Millisecond); !errors.Is(e, context.Canceled) {
				t.Fatalf("cancellation lost: %v", e)
			}
		})
	}
}
func TestChainGuardInvalidClientAndContext(t *testing.T) {
	for _, c := range []*Client{nil, {}, testClient(t, newFake())} {
		if e := c.checkChain(nil); e == nil {
			t.Fatal("nil context accepted")
		}
	}
	for _, c := range []*Client{nil, {}} {
		if e := c.checkChain(context.Background()); e == nil {
			t.Fatal("invalid client accepted")
		}
	}
}

func TestReceiptRejectsZeroHashBeforeBackend(t *testing.T) {
	f := newFake()
	c := testClient(t, f)
	if m, e := c.WaitReceipt(context.Background(), common.Hash{}, time.Millisecond); m != nil || e == nil {
		t.Fatal("zero hash accepted")
	}
	if f.calls != 0 {
		t.Fatal("invalid request reached backend")
	}
}
