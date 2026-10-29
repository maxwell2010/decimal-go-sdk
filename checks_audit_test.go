package decimal

import (
	"context"
	"errors"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"math/big"
	"testing"
)

type checksCancelBackend struct {
	*fakeRPC
	cancel context.CancelFunc
	stage  string
	reads  int
	log    types.Log
}

func (b *checksCancelBackend) ChainID(ctx context.Context) (*big.Int, error) {
	if b.stage == "chain" {
		b.cancel()
	}
	return b.fakeRPC.ChainID(ctx)
}
func (b *checksCancelBackend) FilterLogs(context.Context, ethereum.FilterQuery) ([]types.Log, error) {
	b.reads++
	if b.stage == "read" {
		b.cancel()
	}
	if b.stage == "error" {
		b.cancel()
		return nil, errors.New("credential must not leak")
	}
	return []types.Log{b.log}, nil
}
func (b *checksCancelBackend) TransactionReceipt(context.Context, common.Hash) (*types.Receipt, error) {
	b.reads++
	if b.stage == "read" {
		b.cancel()
	}
	if b.stage == "error" {
		b.cancel()
		return nil, errors.New("credential must not leak")
	}
	return &types.Receipt{Status: 1, BlockNumber: big.NewInt(50), BlockHash: b.log.BlockHash, TxHash: b.log.TxHash, TransactionIndex: b.log.TxIndex, Logs: []*types.Log{&b.log}}, nil
}
func TestChecksReadersHonorInFlightCancellation(t *testing.T) {
	for _, method := range []string{"history", "receipt"} {
		for _, stage := range []string{"before", "chain", "read", "error"} {
			t.Run(method+"/"+stage, func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				b := &checksCancelBackend{fakeRPC: newFake(), cancel: cancel, stage: stage, log: creationFixture(t)}
				c, e := NewClient(Testnet(), b)
				if e != nil {
					t.Fatal(e)
				}
				if stage == "before" {
					cancel()
				}
				var rows []ChecksCreation
				if method == "history" {
					rows, e = c.ChecksCreationHistory(ctx, testAddress, 49, 50)
				} else {
					rows, e = c.ChecksCreationReceipt(ctx, testAddress, b.log.TxHash)
				}
				if !errors.Is(e, context.Canceled) || rows != nil {
					t.Fatalf("returned results after cancellation: %v %v", rows, e)
				}
				if (stage == "chain" || stage == "before") && b.reads != 0 {
					t.Fatal("started read after cancellation")
				}
			})
		}
	}
}
