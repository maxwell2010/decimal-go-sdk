package decimal

import (
	"context"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"math/big"
	"testing"
)

func TestExplicitTypedEnvelopesAndFeeReserve(t *testing.T) {
	for _, kind := range []uint8{types.AccessListTxType, types.DynamicFeeTxType} {
		t.Run(string(rune('0'+kind)), func(t *testing.T) {
			f := newFake()
			c := testClient(t, f)
			s := testSigner(t, 1)
			call, _ := NativeTransfer(testAddress, big.NewInt(7), "typed")
			opts := TxOptions{TxType: kind, AccessList: types.AccessList{{Address: testAddress, StorageKeys: []common.Hash{common.HexToHash("0x01")}}}}
			if kind == types.DynamicFeeTxType {
				opts.FeeCap = big.NewInt(100)
				opts.TipCap = big.NewInt(3)
				opts.BaseFeeEstimate = big.NewInt(10)
			} else {
				opts.GasPrice = big.NewInt(10)
			}
			p, e := c.Prepare(context.Background(), s.Address(), call, opts)
			if e != nil {
				t.Fatal(e)
			}
			if kind == types.DynamicFeeTxType {
				if p.Fee.GasPrice != nil || p.Fee.ReservedFee.Int64() != 2310000 || p.Fee.EstimatedFee.Int64() != 273000 {
					t.Fatal("cap reserve/effective estimate conflated")
				}
			}
			result, e := c.Send(context.Background(), s, call, opts)
			if e != nil || result.Transaction.Type() != kind {
				t.Fatal("typed send", e)
			}
			if !sameAccessList(result.Transaction.AccessList(), opts.AccessList) {
				t.Fatal("access list lost")
			}
			from, e := types.Sender(types.LatestSignerForChainID(c.ChainID()), result.Transaction)
			if e != nil || from != s.Address() {
				t.Fatal("typed signature")
			}
		})
	}
}
func TestDynamicFeeWithoutBaseHasNoFabricatedEstimate(t *testing.T) {
	c := testClient(t, newFake())
	call, _ := NativeTransfer(testAddress, big.NewInt(1), "")
	p, e := c.Prepare(context.Background(), testAddress, call, TxOptions{TxType: 2, FeeCap: big.NewInt(10), TipCap: big.NewInt(1)})
	if e != nil || p.Fee.EstimatedFee != nil || p.Fee.EstimatedEffectiveGasPrice != nil || p.Fee.ReservedFee == nil {
		t.Fatal("unknown effective price estimated", e)
	}
}
func TestTypedEnvelopeValidation(t *testing.T) {
	for _, o := range []TxOptions{
		{TxType: 3}, {AccessList: types.AccessList{{Address: testAddress}}}, {TipCap: big.NewInt(1)},
		{TxType: 2, FeeCap: big.NewInt(10)}, {TxType: 2, FeeCap: big.NewInt(10), TipCap: big.NewInt(11)},
		{TxType: 2, FeeCap: big.NewInt(10), TipCap: big.NewInt(1), GasPrice: big.NewInt(1)},
		{TxType: 2, FeeCap: big.NewInt(10), TipCap: big.NewInt(1), BaseFeeEstimate: big.NewInt(11)},
		{TxType: 2, FeeCap: big.NewInt(10), TipCap: big.NewInt(1), MaxGasPrice: big.NewInt(9)},
		{TxType: 2, FeeCap: big.NewInt(10), TipCap: big.NewInt(1), MaxFee: big.NewInt(1)},
	} {
		c := testClient(t, newFake())
		call, _ := NativeTransfer(testAddress, big.NewInt(1), "")
		if _, e := c.Prepare(context.Background(), testAddress, call, o); e == nil {
			t.Fatal("invalid type or caps accepted")
		}
	}
	f := newFake()
	f.balance = big.NewInt(23100)
	c := testClient(t, f)
	call, _ := NativeTransfer(testAddress, big.NewInt(1), "")
	if _, e := c.Prepare(context.Background(), testAddress, call, TxOptions{TxType: 2, FeeCap: big.NewInt(100), TipCap: big.NewInt(1)}); e == nil {
		t.Fatal("cap balance reserve bypassed")
	}
}

type typedMutatingSigner struct {
	key *KeySigner
	tip bool
}

func (s typedMutatingSigner) Address() common.Address { return s.key.Address() }
func (s typedMutatingSigner) SignTransaction(ctx context.Context, tx *types.Transaction, id *big.Int) (*types.Transaction, error) {
	tip := tx.GasTipCap()
	list := cloneAccessList(tx.AccessList())
	if s.tip {
		tip.Add(tip, big.NewInt(1))
	} else {
		list[0].StorageKeys[0] = common.HexToHash("0xffff")
	}
	modified := types.NewTx(&types.DynamicFeeTx{ChainID: id, Nonce: tx.Nonce(), To: tx.To(), Value: tx.Value(), Gas: tx.Gas(), GasFeeCap: tx.GasFeeCap(), GasTipCap: tip, Data: tx.Data(), AccessList: list})
	return s.key.SignTransaction(ctx, modified, id)
}
func TestSignerCannotChangeTypedFeeOrAccessList(t *testing.T) {
	for _, tip := range []bool{true, false} {
		f := newFake()
		c := testClient(t, f)
		call, _ := NativeTransfer(testAddress, big.NewInt(1), "")
		opts := TxOptions{TxType: 2, FeeCap: big.NewInt(100), TipCap: big.NewInt(1), AccessList: types.AccessList{{Address: testAddress, StorageKeys: []common.Hash{common.Hash{}}}}}
		if _, e := c.Send(context.Background(), typedMutatingSigner{testSigner(t, 1), tip}, call, opts); e == nil || len(f.sent) != 0 {
			t.Fatal("typed intent mutation escaped")
		}
	}
}
