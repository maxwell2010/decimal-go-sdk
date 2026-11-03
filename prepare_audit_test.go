package decimal

import (
	"context"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"testing"
)

type prepareAuditBackend struct {
	*fakeRPC
	cancel context.CancelFunc
	target *common.Address
	before func()
}

func (b *prepareAuditBackend) ChainID(ctx context.Context) (*big.Int, error) {
	if b.before != nil {
		b.before()
	}
	return b.fakeRPC.ChainID(ctx)
}
func (b *prepareAuditBackend) EstimateGas(ctx context.Context, msg ethereum.CallMsg) (uint64, error) {
	if b.target != nil {
		*msg.To = *b.target
	}
	return 21000, nil
}
func (b *prepareAuditBackend) PendingNonceAt(context.Context, common.Address) (uint64, error) {
	if b.cancel != nil {
		b.cancel()
	}
	return 0, nil
}
func TestPrepareOwnsEstimateTarget(t *testing.T) {
	evil := common.HexToAddress("0x2222")
	b := &prepareAuditBackend{fakeRPC: newFake(), target: &evil}
	c, e := NewClient(Testnet(), b)
	if e != nil {
		t.Fatal(e)
	}
	wanted := common.HexToAddress("0x1111")
	call, _ := NativeTransfer(wanted, big.NewInt(1), "")
	p, e := c.Prepare(context.Background(), testAddress, call, TxOptions{})
	if e != nil {
		t.Fatal(e)
	}
	if *p.Transaction.To() != wanted {
		t.Fatal("backend changed prepared target", p.Transaction.To())
	}
}
func TestPrepareCancellationAfterNonceResponse(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	b := &prepareAuditBackend{fakeRPC: newFake(), cancel: cancel}
	c, _ := NewClient(Testnet(), b)
	call, _ := NativeTransfer(testAddress, big.NewInt(1), "")
	p, e := c.Prepare(ctx, testAddress, call, TxOptions{})
	if e != context.Canceled || p != nil {
		t.Fatal("canceled nonce response returned successful Prepared", p != nil, e)
	}
}

func TestPrepareSnapshotsExplicitNonceBeforeBackend(t *testing.T) {
	nonce := uint64(3)
	price := big.NewInt(10)
	value := big.NewInt(2)
	data := []byte{1, 2}
	b := &prepareAuditBackend{fakeRPC: newFake(), before: func() { nonce = 99; price.SetInt64(888); value.SetInt64(777); data[0] = 9 }}
	c, _ := NewClient(Testnet(), b)
	p, e := c.Prepare(context.Background(), testAddress, Call{To: testAddress, Value: value, Data: data}, TxOptions{Nonce: &nonce, GasPrice: price})
	if e != nil {
		t.Fatal(e)
	}
	if p.Transaction.Nonce() != 3 || p.Transaction.Value().Int64() != 2 || p.Transaction.GasPrice().Int64() != 10 || p.Transaction.Data()[0] != 1 {
		t.Fatal("backend mutation changed intent")
	}
	if _, e := c.Prepare(context.Background(), testAddress, Call{To: testAddress}, TxOptions{}); e == nil {
		t.Fatal("nil value validation relaxed")
	}
}
