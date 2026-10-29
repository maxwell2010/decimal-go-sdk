// Command envelopes prepares unsigned EVM types 0, 1 and 2 entirely in process.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	decimal "github.com/maxwell2010/decimal-go-sdk"
)

// fixtureBackend is not connected to a node. Gas and balances are fixtures,
// not predictions of actual execution or evidence of network type support.
type fixtureBackend struct{}

func (fixtureBackend) ChainID(context.Context) (*big.Int, error)                      { return big.NewInt(202020), nil }
func (fixtureBackend) PendingNonceAt(context.Context, common.Address) (uint64, error) { return 7, nil }
func (fixtureBackend) SuggestGasPrice(context.Context) (*big.Int, error)              { return big.NewInt(10), nil }
func (fixtureBackend) EstimateGas(context.Context, ethereum.CallMsg) (uint64, error) {
	return 21000, nil
}
func (fixtureBackend) BalanceAt(context.Context, common.Address, *big.Int) (*big.Int, error) {
	return new(big.Int).Exp(big.NewInt(10), big.NewInt(24), nil), nil
}
func (fixtureBackend) CodeAt(context.Context, common.Address, *big.Int) ([]byte, error) {
	return []byte{0x00}, nil
}
func (fixtureBackend) CallContract(context.Context, ethereum.CallMsg, *big.Int) ([]byte, error) {
	return nil, errors.New("no contract-call fixture")
}
func (fixtureBackend) SendTransaction(context.Context, *types.Transaction) error {
	panic("example must never submit")
}
func (fixtureBackend) TransactionReceipt(context.Context, common.Hash) (*types.Receipt, error) {
	panic("example must never poll receipts")
}

func prepareExamples(ctx context.Context) ([]*decimal.Prepared, error) {
	client, err := decimal.NewClient(decimal.Testnet(), fixtureBackend{})
	if err != nil {
		return nil, err
	}
	from := common.HexToAddress("0x1000000000000000000000000000000000000001")
	to := common.HexToAddress("0x2000000000000000000000000000000000000002")
	call, err := decimal.NativeTransfer(to, big.NewInt(123), "")
	if err != nil {
		return nil, err
	}
	list := types.AccessList{{Address: to, StorageKeys: []common.Hash{common.HexToHash("0x01")}}}
	options := []decimal.TxOptions{
		{TxType: types.LegacyTxType, MaxGasPrice: big.NewInt(10)},
		{TxType: types.AccessListTxType, AccessList: list, GasPrice: big.NewInt(10)},
		{TxType: types.DynamicFeeTxType, AccessList: list, FeeCap: big.NewInt(20), TipCap: big.NewInt(2), BaseFeeEstimate: big.NewInt(5)},
	}
	var prepared []*decimal.Prepared
	for _, opts := range options {
		p, err := client.Prepare(ctx, from, call, opts)
		if err != nil {
			return nil, err
		}
		prepared = append(prepared, p)
	}
	return prepared, nil
}

func main() {
	prepared, err := prepareExamples(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	for _, p := range prepared {
		tx := p.Transaction
		fmt.Printf("unsigned type=%d nonce=%d gas=%d estimated_fee_wei=%s reserved_fee_wei=%s value_wei=%s\n", tx.Type(), tx.Nonce(), tx.Gas(), p.Fee.EstimatedFee, p.Fee.ReservedFee, tx.Value())
	}
}
