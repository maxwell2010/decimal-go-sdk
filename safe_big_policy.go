package decimal

import (
	"context"
	"errors"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"reflect"
)

// BigSafePolicy preserves the actual Safe ABI's uint256 weights and threshold.
// All numeric inputs are copied before verification callbacks are invoked.
type BigSafePolicy struct {
	Safe                  common.Address
	ChainID, Nonce, Block *big.Int
	Owners                map[common.Address]*big.Int
	Threshold             *big.Int
}

func snapshotBigSafePolicy(p BigSafePolicy) (BigSafePolicy, error) {
	if p.Safe == (common.Address{}) || !uint256(p.ChainID) || p.ChainID.Sign() == 0 || !uint256(p.Nonce) || !uint256(p.Block) || !uint256(p.Threshold) || p.Threshold.Sign() == 0 {
		return BigSafePolicy{}, errors.New("invalid uint256 Safe policy")
	}
	result := BigSafePolicy{Safe: p.Safe, ChainID: copyInt(p.ChainID), Nonce: copyInt(p.Nonce), Block: copyInt(p.Block), Threshold: copyInt(p.Threshold), Owners: map[common.Address]*big.Int{}}
	total := new(big.Int)
	for owner, weight := range p.Owners {
		if owner == (common.Address{}) || !uint256(weight) || weight.Sign() == 0 {
			return BigSafePolicy{}, errors.New("invalid uint256 Safe owner weight")
		}
		result.Owners[owner] = copyInt(weight)
		total.Add(total, weight)
		if !uint256(total) {
			return BigSafePolicy{}, errors.New("Safe total weight overflows uint256")
		}
	}
	if result.Threshold.Cmp(total) > 0 {
		return BigSafePolicy{}, errors.New("Safe threshold exceeds total weight")
	}
	return result, nil
}

// PackBigWeightedSafeSignatures validates full-width weighted authorization, then
// reuses the same owner-bound EOA/contract verification and wire packer. No owner
// weight or threshold is cast to a machine integer.
func PackBigWeightedSafeSignatures(ctx context.Context, tx SafeTransaction, policy BigSafePolicy, signatures []SafeMixedSignature, verify SafeContractVerifier) ([]byte, error) {
	if e := ctx.Err(); e != nil {
		return nil, e
	}
	p, e := snapshotBigSafePolicy(policy)
	if e != nil {
		return nil, e
	}
	if !uint256(tx.Nonce) || tx.Nonce.Cmp(p.Nonce) != 0 {
		return nil, errors.New("Safe nonce mismatch")
	}
	rows := make([]SafeMixedSignature, len(signatures))
	seen := map[common.Address]bool{}
	approved := new(big.Int)
	for i, s := range signatures {
		weight, ok := p.Owners[s.Signer]
		if !ok || seen[s.Signer] {
			return nil, errors.New("unknown or duplicate Safe owner")
		}
		seen[s.Signer] = true
		approved.Add(approved, weight)
		rows[i] = SafeMixedSignature{s.Signer, s.Kind, append([]byte(nil), s.Data...)}
	}
	if approved.Cmp(p.Threshold) < 0 {
		return nil, errors.New("insufficient Safe signature weight")
	}
	// A synthetic equal-weight policy is only used AFTER full-width threshold
	// validation; the shared packer performs cryptographic checks, not big-weight
	// arithmetic. This keeps the wire format and signature checks identical.
	owners := make(map[common.Address]uint32, len(p.Owners))
	for owner := range p.Owners {
		owners[owner] = 1
	}
	return PackWeightedSafeSignatures(ctx, tx, SafePolicy{Safe: p.Safe, ChainID: p.ChainID, Nonce: p.Nonce, Block: p.Block, Owners: owners, Threshold: 1}, rows, verify)
}

func (c *Contract) ReadBigSafePolicy(ctx context.Context, block *big.Int) (BigSafePolicy, error) {
	if c == nil || c.client == nil || !uint256(block) {
		return BigSafePolicy{}, errors.New("client and pinned Safe block required")
	}
	a, e := c.contractABI()
	if e != nil {
		return BigSafePolicy{}, e
	}
	_, getter, e := method(a, "getOwnersWithWeights()")
	if e != nil || len(getter.Outputs) != 1 || getter.Outputs[0].Type.String() != "(address,uint256)[]" {
		return BigSafePolicy{}, errors.New("unsupported Safe owner getter ABI")
	}
	values, e := c.ReadAt(ctx, block, "getOwnersWithWeights()")
	if e != nil {
		return BigSafePolicy{}, e
	}
	if len(values) != 1 {
		return BigSafePolicy{}, errors.New("invalid Safe owner response")
	}
	rows := reflect.ValueOf(values[0])
	if rows.Kind() != reflect.Slice {
		return BigSafePolicy{}, errors.New("invalid Safe owner list")
	}
	owners := map[common.Address]*big.Int{}
	for i := 0; i < rows.Len(); i++ {
		row := rows.Index(i)
		if row.Kind() != reflect.Struct || row.NumField() != 2 || !row.Field(0).CanInterface() || !row.Field(1).CanInterface() {
			return BigSafePolicy{}, errors.New("invalid Safe owner tuple")
		}
		owner, ok := row.Field(0).Interface().(common.Address)
		if !ok {
			return BigSafePolicy{}, errors.New("invalid Safe owner")
		}
		weight, ok := row.Field(1).Interface().(*big.Int)
		if !ok || owners[owner] != nil {
			return BigSafePolicy{}, errors.New("invalid or duplicate Safe owner weight")
		}
		owners[owner] = copyInt(weight)
	}
	threshold, e := c.ReadAt(ctx, block, "getWeightThreshold()")
	if e != nil {
		return BigSafePolicy{}, e
	}
	nonce, e := c.ReadAt(ctx, block, "nonce()")
	if e != nil {
		return BigSafePolicy{}, e
	}
	if len(threshold) != 1 || len(nonce) != 1 {
		return BigSafePolicy{}, errors.New("invalid Safe policy response")
	}
	limit, ok := threshold[0].(*big.Int)
	if !ok {
		return BigSafePolicy{}, errors.New("invalid Safe threshold")
	}
	n, ok := nonce[0].(*big.Int)
	if !ok {
		return BigSafePolicy{}, errors.New("invalid Safe nonce")
	}
	return snapshotBigSafePolicy(BigSafePolicy{Safe: c.Address, ChainID: c.client.ChainID(), Nonce: n, Block: block, Owners: owners, Threshold: limit})
}
