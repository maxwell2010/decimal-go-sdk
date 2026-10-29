package decimal

import (
	"context"
	"errors"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"reflect"
)

// ReadSafePolicy reads actual Decimal weighted Safe getters at one caller-pinned
// block using the supplied deployment ABI. The JS SDK supports owner weights
// 1..1000; this policy adapter rejects larger values instead of truncating them.
// This snapshot does not itself prove that an owner can authorize a transaction.
func (c *Contract) ReadSafePolicy(ctx context.Context, block *big.Int) (SafePolicy, error) {
	if c == nil || c.client == nil || !uint256(block) {
		return SafePolicy{}, errors.New("client and pinned Safe block required")
	}
	a, e := c.contractABI()
	if e != nil {
		return SafePolicy{}, e
	}
	_, getter, e := method(a, "getOwnersWithWeights()")
	if e != nil || len(getter.Outputs) != 1 || getter.Outputs[0].Type.String() != "(address,uint256)[]" {
		return SafePolicy{}, errors.New("unsupported Safe owner getter ABI")
	}
	values, e := c.ReadAt(ctx, block, "getOwnersWithWeights()")
	if e != nil {
		return SafePolicy{}, e
	}
	if len(values) != 1 {
		return SafePolicy{}, errors.New("invalid Safe owner response")
	}
	rows := reflect.ValueOf(values[0])
	if rows.Kind() != reflect.Slice {
		return SafePolicy{}, errors.New("invalid Safe owners")
	}
	owners := map[common.Address]uint32{}
	for i := 0; i < rows.Len(); i++ {
		row := rows.Index(i)
		if row.Kind() != reflect.Struct || row.NumField() != 2 || !row.Field(0).CanInterface() || !row.Field(1).CanInterface() {
			return SafePolicy{}, errors.New("unsupported Safe owner tuple")
		}
		owner, ok := row.Field(0).Interface().(common.Address)
		if !ok {
			return SafePolicy{}, errors.New("invalid Safe owner type")
		}
		weight, ok := row.Field(1).Interface().(*big.Int)
		if !ok {
			return SafePolicy{}, errors.New("invalid Safe weight type")
		}
		if owner == (common.Address{}) || !uint256(weight) || !weight.IsUint64() || weight.Sign() == 0 || weight.Uint64() > 1000 || owners[owner] != 0 {
			return SafePolicy{}, errors.New("invalid or unsupported Safe owner weight")
		}
		owners[owner] = uint32(weight.Uint64())
	}
	threshold, e := c.ReadAt(ctx, block, "getWeightThreshold()")
	if e != nil {
		return SafePolicy{}, e
	}
	nonce, e := c.ReadAt(ctx, block, "nonce()")
	if e != nil {
		return SafePolicy{}, e
	}
	if len(threshold) != 1 || len(nonce) != 1 {
		return SafePolicy{}, errors.New("invalid Safe policy response")
	}
	limit, ok := threshold[0].(*big.Int)
	if !ok || !uint256(limit) || !limit.IsUint64() || limit.Sign() == 0 {
		return SafePolicy{}, errors.New("invalid Safe threshold")
	}
	n, ok := nonce[0].(*big.Int)
	if !ok || !uint256(n) {
		return SafePolicy{}, errors.New("invalid Safe nonce")
	}
	var total uint64
	for _, w := range owners {
		total += uint64(w)
	}
	if limit.Uint64() > total {
		return SafePolicy{}, errors.New("Safe threshold exceeds owner weight")
	}
	return SafePolicy{Safe: c.Address, ChainID: c.client.ChainID(), Nonce: copyInt(n), Block: copyInt(block), Owners: owners, Threshold: limit.Uint64()}, nil
}
