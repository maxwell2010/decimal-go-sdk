package decimal

import (
	"bytes"
	"context"
	"errors"
	"github.com/ethereum/go-ethereum"
	"math/big"
	"testing"
)

type simulationAuditBackend struct {
	*fakeRPC
	chainHook func()
	read      func(context.Context, ethereum.CallMsg, *big.Int) ([]byte, error)
	reads     int
}

func (b *simulationAuditBackend) ChainID(ctx context.Context) (*big.Int, error) {
	if b.chainHook != nil {
		b.chainHook()
	}
	return b.fakeRPC.ChainID(ctx)
}
func (b *simulationAuditBackend) CallContract(ctx context.Context, msg ethereum.CallMsg, block *big.Int) ([]byte, error) {
	b.reads++
	return b.read(ctx, msg, block)
}
func simulationAuditRunner(t *testing.T, kind string, b *simulationAuditBackend) func(context.Context, Call, *big.Int) ([]byte, error) {
	t.Helper()
	client, err := NewClient(Testnet(), b)
	if err != nil {
		t.Fatal(err)
	}
	contract, err := client.Delegation(testAddress)
	if err != nil {
		t.Fatal(err)
	}
	return func(ctx context.Context, call Call, block *big.Int) ([]byte, error) {
		if kind == "client" {
			return client.Simulate(ctx, testAddress, call, block)
		}
		return contract.Simulate(ctx, testAddress, call, block)
	}
}
func TestSimulationReadCancellationAndOwnership(t *testing.T) {
	for _, kind := range []string{"client", "contract"} {
		t.Run(kind, func(t *testing.T) {
			for _, stage := range []string{"chain", "success", "error"} {
				t.Run(stage, func(t *testing.T) {
					ctx, cancel := context.WithCancel(context.Background())
					defer cancel()
					backend := &simulationAuditBackend{fakeRPC: newFake()}
					if stage == "chain" {
						backend.chainHook = cancel
					}
					backend.read = func(context.Context, ethereum.CallMsg, *big.Int) ([]byte, error) {
						cancel()
						if stage == "error" {
							return nil, errors.New("sensitive backend detail")
						}
						return []byte{1}, nil
					}
					result, err := simulationAuditRunner(t, kind, backend)(ctx, Call{To: testAddress, Value: big.NewInt(0)}, big.NewInt(7))
					if !errors.Is(err, context.Canceled) || result != nil {
						t.Fatal("cancellation lost", result, err)
					}
					if stage == "chain" && backend.reads != 0 {
						t.Fatal("read started after canceled chain check")
					}
				})
			}
			for _, height := range []int64{-1, 0, 7} {
				t.Run(big.NewInt(height).String(), func(t *testing.T) {
					block := big.NewInt(height)
					value := big.NewInt(9)
					data := []byte{1, 2, 3}
					response := []byte{4, 5, 6}
					backend := &simulationAuditBackend{fakeRPC: newFake()}
					backend.chainHook = func() { block.SetInt64(999); value.SetInt64(888); data[0] = 77 }
					backend.read = func(_ context.Context, msg ethereum.CallMsg, selected *big.Int) ([]byte, error) {
						if selected == nil || selected.Int64() != height || msg.Value.Int64() != 9 || !bytes.Equal(msg.Data, []byte{1, 2, 3}) {
							t.Fatal("backend callback changed captured intent", selected, msg)
						}
						selected.SetInt64(555)
						msg.Value.SetInt64(444)
						msg.Data[1] = 33
						return response, nil
					}
					out, err := simulationAuditRunner(t, kind, backend)(context.Background(), Call{To: testAddress, Value: value, Data: data}, block)
					if err != nil || !bytes.Equal(out, response) {
						t.Fatal(out, err)
					}
					if block.Int64() != 999 || value.Int64() != 888 || !bytes.Equal(data, []byte{77, 2, 3}) {
						t.Fatal("backend read borrowed caller memory", block, value, data)
					}
					response[0] = 11
					if out[0] != 4 {
						t.Fatal("returned response aliases backend")
					}
					out[1] = 22
					if response[1] != 5 {
						t.Fatal("caller response mutation reached backend")
					}
				})
			}
			t.Run("latest-nil", func(t *testing.T) {
				backend := &simulationAuditBackend{fakeRPC: newFake()}
				backend.read = func(_ context.Context, _ ethereum.CallMsg, selected *big.Int) ([]byte, error) {
					if selected != nil {
						t.Fatal("nil latest changed")
					}
					return []byte{}, nil
				}
				out, err := simulationAuditRunner(t, kind, backend)(context.Background(), Call{To: testAddress, Value: big.NewInt(0)}, nil)
				if err != nil || out == nil || len(out) != 0 {
					t.Fatal("empty response semantics changed", out, err)
				}
			})
		})
	}
}
func TestSimulationReadKeepsCustomError(t *testing.T) {
	for _, kind := range []string{"client", "contract"} {
		t.Run(kind, func(t *testing.T) {
			b := &simulationAuditBackend{fakeRPC: newFake()}
			b.read = func(context.Context, ethereum.CallMsg, *big.Int) ([]byte, error) { return nil, dataError{} }
			_, err := simulationAuditRunner(t, kind, b)(context.Background(), Call{To: testAddress, Value: big.NewInt(0), Profile: "Delegation"}, big.NewInt(1))
			var reverted *RevertError
			if !errors.As(err, &reverted) || reverted.Name != "InvalidAmount" {
				t.Fatal("revert decoding lost", err)
			}
		})
	}
}
