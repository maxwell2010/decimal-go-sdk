package decimal

import (
	"context"
	"errors"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"math/big"
	"reflect"
	"testing"
	"time"
)

type chainMutatingSigner struct{ key *KeySigner }

type dataMutatingSigner struct{ key *KeySigner }

func (s dataMutatingSigner) Address() common.Address { return s.key.Address() }
func (s dataMutatingSigner) SignTransaction(ctx context.Context, tx *types.Transaction, id *big.Int) (*types.Transaction, error) {
	tx.Data()[0] ^= 0xff
	return s.key.SignTransaction(ctx, tx, id)
}
func TestExternalSignerCannotMutateDataInPlace(t *testing.T) {
	f := newFake()
	c := testClient(t, f)
	s := dataMutatingSigner{testSigner(t, 1)}
	call, _ := NativeTransfer(testAddress, big.NewInt(1), "memo")
	if _, e := c.Send(context.Background(), s, call, TxOptions{}); e == nil || len(f.sent) != 0 {
		t.Fatal("in-place calldata mutation was broadcast")
	}
}

func (s chainMutatingSigner) Address() common.Address { return s.key.Address() }
func (s chainMutatingSigner) SignTransaction(ctx context.Context, tx *types.Transaction, id *big.Int) (*types.Transaction, error) {
	id.SetInt64(75)
	return s.key.SignTransaction(ctx, tx, id)
}
func TestExternalSignerCannotMutateConfiguredChain(t *testing.T) {
	f := newFake()
	c := testClient(t, f)
	s := chainMutatingSigner{testSigner(t, 1)}
	call, _ := NativeTransfer(testAddress, big.NewInt(1), "")
	if _, e := c.Send(context.Background(), s, call, TxOptions{}); e == nil {
		t.Fatal("wrong chain signer accepted")
	}
	if c.ChainID().Int64() != 202020 || len(f.sent) != 0 {
		t.Fatal("signer mutated client chain")
	}
}

func TestCanceledSendDoesNotWaitForNonceOwner(t *testing.T) {
	f := newFake()
	c := testClient(t, f)
	s := testSigner(t, 1)
	state := &nonceState{gate: make(chan struct{}, 1)}
	state.gate <- struct{}{}
	c.nonces.Store(s.Address(), state)
	call, _ := NativeTransfer(testAddress, big.NewInt(1), "")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	done := make(chan error, 1)
	go func() { _, e := c.Send(ctx, s, call, TxOptions{}); done <- e }()
	select {
	case e := <-done:
		if !errors.Is(e, context.DeadlineExceeded) {
			t.Fatal(e)
		}
	case <-time.After(time.Second):
		<-state.gate
		t.Fatal("send blocked behind unrelated context")
	}
	<-state.gate
	if f.calls != 0 {
		t.Fatal("canceled queued sender touched RPC")
	}
}

func TestReceiptCannotCompleteAnotherHash(t *testing.T) {
	f := newFake()
	f.receipt = &types.Receipt{Status: 1, TxHash: common.HexToHash("0x01")}
	c := testClient(t, f)
	if _, e := c.WaitReceipt(context.Background(), common.HexToHash("0x02"), time.Millisecond); e == nil {
		t.Fatal("wrong receipt accepted")
	}
}
func TestCanceledPreparePreservesContextError(t *testing.T) {
	f := newFake()
	c := testClient(t, f)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	call, _ := NativeTransfer(testAddress, big.NewInt(1), "")
	if _, e := c.Prepare(ctx, testAddress, call, TxOptions{}); !errors.Is(e, context.Canceled) {
		t.Fatal(e)
	}
	if f.calls != 0 {
		t.Fatal("canceled preflight queried RPC")
	}
}
func TestUnexportedABITupleFieldReturnsError(t *testing.T) {
	typ, e := abi.NewType("tuple", "", []abi.ArgumentMarshaling{{Name: "owner", Type: "address"}})
	if e != nil {
		t.Fatal(e)
	}
	v := struct {
		owner common.Address `abi:"owner"`
	}{testAddress}
	if _, e = normalizeArgument(typ, reflect.ValueOf(v)); e == nil {
		t.Fatal("unexported field accepted")
	}
}
