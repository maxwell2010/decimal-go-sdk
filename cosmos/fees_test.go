package cosmos

import (
	"context"
	"errors"
	"math/big"
	"testing"
)

func feeOptions() FeeConvergenceOptions {
	return FeeConvergenceOptions{MaxFee: big.NewInt(10000), Snapshot: "block-42-price-and-params"}
}
func feeBuilder(_ context.Context, fee *big.Int) ([]byte, error) {
	return []byte("signed-fee=" + fee.String()), nil
}
func TestConvergeFeeSignedLengthAndSnapshot(t *testing.T) {
	opts := feeOptions()
	calls := 0
	result, e := ConvergeFee(context.Background(), opts, feeBuilder, func(_ context.Context, r CommissionRequest) (CommissionQuote, error) {
		calls++
		if len(r.SignedBytes) != r.SignedLength {
			t.Fatal("length mismatch")
		}
		n := int64(r.SignedLength)
		r.Fee.SetInt64(999)
		r.SignedBytes[0] = 0
		opts.MaxFee.SetInt64(0)
		return CommissionQuote{big.NewInt(n), r.Snapshot}, nil
	})
	if e != nil || result.Iterations != 3 || calls != 3 || string(result.SignedBytes) != "signed-fee=13" || result.Fee.Int64() != 13 {
		t.Fatal(result, e)
	}
}
func TestConvergeFeeFailureModes(t *testing.T) {
	for _, mode := range []string{"cycle", "overflow", "maxfee", "snapshot", "quote-error", "build-error", "iterations", "size"} {
		t.Run(mode, func(t *testing.T) {
			opts := feeOptions()
			build := FeeTxBuilder(feeBuilder)
			expected := ErrFeeCycle
			quote := CommissionQuoter(func(_ context.Context, r CommissionRequest) (CommissionQuote, error) {
				n := big.NewInt(2)
				if r.Fee.Int64() == 2 {
					n.SetInt64(1)
				}
				return CommissionQuote{n, r.Snapshot}, nil
			})
			switch mode {
			case "overflow":
				expected = ErrFeeLimit
				quote = func(_ context.Context, r CommissionRequest) (CommissionQuote, error) {
					return CommissionQuote{new(big.Int).Lsh(big.NewInt(1), 256), r.Snapshot}, nil
				}
			case "maxfee":
				expected = ErrFeeLimit
				opts.MaxFee = big.NewInt(1)
			case "snapshot":
				expected = ErrFeeSnapshot
				quote = func(context.Context, CommissionRequest) (CommissionQuote, error) {
					return CommissionQuote{big.NewInt(1), "stale"}, nil
				}
			case "quote-error":
				expected = ErrFeeQuote
				quote = func(context.Context, CommissionRequest) (CommissionQuote, error) {
					return CommissionQuote{}, errors.New("sensitive backend error")
				}
			case "build-error":
				expected = ErrFeeBuild
				build = func(context.Context, *big.Int) ([]byte, error) { return nil, errors.New("sensitive signer error") }
			case "iterations":
				expected = ErrFeeIterations
				opts.MaxIterations = 1
			case "size":
				expected = nil
				opts.MaxSignedBytes = 1
			}
			result, e := ConvergeFee(context.Background(), opts, build, quote)
			if e == nil || (expected != nil && !errors.Is(e, expected)) {
				t.Fatal(mode, e)
			}
			if result.Fee != nil || result.SignedBytes != nil {
				t.Fatal("partial transaction escaped")
			}
		})
	}
}
func TestConvergeFeeCancellationAndBuilderMutation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	opts := feeOptions()
	build := func(_ context.Context, n *big.Int) ([]byte, error) { n.SetInt64(55); cancel(); return []byte{1}, nil }
	if _, e := ConvergeFee(ctx, opts, build, func(context.Context, CommissionRequest) (CommissionQuote, error) {
		t.Fatal("quote after cancellation")
		return CommissionQuote{}, nil
	}); !errors.Is(e, context.Canceled) {
		t.Fatal(e)
	}
	result, e := ConvergeFee(context.Background(), opts, func(_ context.Context, n *big.Int) ([]byte, error) { n.SetInt64(99); return []byte{1}, nil }, func(_ context.Context, r CommissionRequest) (CommissionQuote, error) {
		return CommissionQuote{big.NewInt(1), r.Snapshot}, nil
	})
	if e != nil || result.Fee.Int64() != 1 {
		t.Fatal("builder mutated iteration fee", e)
	}
}
