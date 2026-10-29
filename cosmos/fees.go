package cosmos

import (
	"context"
	"errors"
	"math/big"
)

var (
	ErrFeeCycle      = errors.New("Cosmos fee calculation oscillates")
	ErrFeeIterations = errors.New("Cosmos fee calculation iteration limit reached")
	ErrFeeLimit      = errors.New("Cosmos fee exceeds configured maximum or uint256")
	ErrFeeSnapshot   = errors.New("Cosmos fee quote snapshot mismatch")
	ErrFeeBuild      = errors.New("Cosmos fee transaction construction failed")
	ErrFeeQuote      = errors.New("Cosmos commission quote failed")
)

// FeeTxBuilder must rebuild AND sign the exact transaction at the supplied fee,
// keeping messages, chain, signer order, account numbers and sequences fixed.
// No broadcast belongs in this callback. Its mutable fee argument is a copy.
type FeeTxBuilder func(context.Context, *big.Int) ([]byte, error)
type CommissionRequest struct {
	Fee          *big.Int
	SignedBytes  []byte
	SignedLength int
	Snapshot     string
}

// CommissionQuote identifies the pinned parameters/oracle-price snapshot used by
// the external calculator. The SDK supplies no invented network fee rates.
type CommissionQuote struct {
	Fee      *big.Int
	Snapshot string
}
type CommissionQuoter func(context.Context, CommissionRequest) (CommissionQuote, error)
type FeeConvergenceOptions struct {
	InitialFee, MaxFee            *big.Int
	Snapshot                      string
	MaxIterations, MaxSignedBytes int
}
type ConvergedFee struct {
	Fee         *big.Int
	SignedBytes []byte
	Iterations  int
	Snapshot    string
}

func validFee(n *big.Int) bool { return n != nil && n.Sign() >= 0 && n.BitLen() <= 256 }

// ConvergeFee follows node sdk/tx/tx.go's fixed-point computation: sign with F,
// calculate commission from the signed byte length, repeat until quote == F.
// Unlike upstream's unbounded loop it rejects cycles, stale parameter snapshots,
// oversized transactions, fee overflow and iteration exhaustion. On any error
// it returns no partial transaction. The returned bytes were signed at Fee.
func ConvergeFee(ctx context.Context, options FeeConvergenceOptions, build FeeTxBuilder, quote CommissionQuoter) (ConvergedFee, error) {
	empty := ConvergedFee{}
	if e := ctx.Err(); e != nil {
		return empty, e
	}
	if build == nil || quote == nil || options.Snapshot == "" {
		return empty, errors.New("fee builder, commission quoter and pinned snapshot required")
	}
	if !validFee(options.MaxFee) {
		return empty, ErrFeeLimit
	}
	limit := new(big.Int).Set(options.MaxFee)
	fee := big.NewInt(1)
	if options.InitialFee != nil {
		if !validFee(options.InitialFee) {
			return empty, ErrFeeLimit
		}
		fee.Set(options.InitialFee)
	}
	iterations := options.MaxIterations
	if iterations == 0 {
		iterations = 16
	}
	if iterations < 1 || iterations > 1024 {
		return empty, errors.New("fee iteration bound must be 1..1024")
	}
	maxBytes := options.MaxSignedBytes
	if maxBytes == 0 {
		maxBytes = 1 << 20
	}
	if maxBytes < 1 || maxBytes > 64<<20 {
		return empty, errors.New("invalid signed transaction size limit")
	}
	snapshot := options.Snapshot
	seen := map[string]bool{}
	for i := 1; i <= iterations; i++ {
		if e := ctx.Err(); e != nil {
			return empty, e
		}
		if fee.Cmp(limit) > 0 {
			return empty, ErrFeeLimit
		}
		if seen[fee.String()] {
			return empty, ErrFeeCycle
		}
		seen[fee.String()] = true
		raw, e := build(ctx, new(big.Int).Set(fee))
		if ctx.Err() != nil {
			return empty, ctx.Err()
		}
		if e != nil {
			return empty, ErrFeeBuild
		}
		if len(raw) == 0 || len(raw) > maxBytes {
			return empty, errors.New("signed Cosmos transaction size outside configured bound")
		}
		raw = append([]byte(nil), raw...)
		request := CommissionRequest{new(big.Int).Set(fee), append([]byte(nil), raw...), len(raw), snapshot}
		q, e := quote(ctx, request)
		if ctx.Err() != nil {
			return empty, ctx.Err()
		}
		if e != nil {
			return empty, ErrFeeQuote
		}
		if q.Snapshot != snapshot {
			return empty, ErrFeeSnapshot
		}
		if !validFee(q.Fee) || q.Fee.Cmp(limit) > 0 {
			return empty, ErrFeeLimit
		}
		next := new(big.Int).Set(q.Fee)
		if next.Cmp(fee) == 0 {
			return ConvergedFee{new(big.Int).Set(fee), raw, i, snapshot}, nil
		}
		fee = next
	}
	return empty, ErrFeeIterations
}
