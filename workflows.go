package decimal

import (
	"context"
	"errors"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"math/big"
	"reflect"
)

// ApprovalPlan always separates EOA approvals from the actual action. Execute
// the approval, wait for success, then re-estimate/send the action.
type ApprovalPlan struct {
	Approval *Call
	Action   Call
}

func (c *Client) PlanERC20Action(ctx context.Context, token, owner, spender common.Address, amount *big.Int, action Call) (ApprovalPlan, error) {
	if !uint256(amount) || action.To != spender || owner == (common.Address{}) || spender == (common.Address{}) {
		return ApprovalPlan{}, errors.New("invalid action or allowance amount")
	}
	if spender == c.cfg.Multicall {
		return ApprovalPlan{}, errors.New("generic Multicall must not receive ERC20 allowance")
	}
	t, e := c.Token(token)
	if e != nil {
		return ApprovalPlan{}, e
	}
	allowance, e := t.ReadAllowance(ctx, owner, spender)
	if e != nil {
		return ApprovalPlan{}, e
	}
	plan := ApprovalPlan{Action: action}
	if allowance.Value0.Cmp(amount) < 0 {
		a, e := t.BuildApprove(spender, amount)
		if e != nil {
			return ApprovalPlan{}, e
		}
		plan.Approval = &a
	}
	return plan, nil
}

type Recipient struct {
	Address common.Address
	Amount  *big.Int
}

func (c *Client) BuildMultisendDEL(recipients []Recipient) (Call, error) {
	if len(recipients) == 0 {
		return Call{}, errors.New("recipients required")
	}
	if len(recipients) == 1 {
		return NativeTransfer(recipients[0].Address, recipients[0].Amount, "")
	}
	total := new(big.Int)
	calls := make([]MulticallAggregateArg0, len(recipients))
	for i, r := range recipients {
		if !uint256(r.Amount) || r.Amount.Sign() == 0 {
			return Call{}, errors.New("positive recipient amount required")
		}
		total.Add(total, r.Amount)
		calls[i] = MulticallAggregateArg0{Target: r.Address, Value: copyInt(r.Amount), CallData: []byte{}}
	}
	if !uint256(total) {
		return Call{}, errors.New("total exceeds uint256")
	}
	contract, e := c.Multicall(c.cfg.Multicall)
	if e != nil {
		return Call{}, e
	}
	return contract.BuildAggregate(total, calls)
}

// ERC20 multisend via the arbitrary-call Multicall is intentionally unavailable.
// A sender-restricted deployment can be used explicitly with ContractWithABI.
func (c *Client) BuildMultisendERC20() (Call, error) {
	return Call{}, errors.New("ERC20 Multicall spender route disabled; use a verified sender-restricted contract")
}

// CheckedPermit validates chain, owner, nonce and the token's actual EIP712 domain
// separator before signing. No hardcoded token version or maximum deadline.
func (c *Client) CheckedPermit(ctx context.Context, signer *KeySigner, token common.Address, name, version string, p Permit) (Signature, error) {
	if signer == nil || p.Owner != signer.Address() {
		return Signature{}, errors.New("permit owner mismatch")
	}
	if e := c.checkChain(ctx); e != nil {
		return Signature{}, e
	}
	t, e := c.Token(token)
	if e != nil {
		return Signature{}, e
	}
	n, e := t.ReadNonces(ctx, p.Owner)
	if e != nil {
		return Signature{}, e
	}
	if p.Nonce == nil || n.Value0.Cmp(p.Nonce) != 0 {
		return Signature{}, errors.New("permit nonce mismatch")
	}
	d, e := PermitTypedData(name, version, c.cfg.ChainID, token, p)
	if e != nil {
		return Signature{}, e
	}
	expected, e := typedDomainHash(d)
	if e != nil {
		return Signature{}, e
	}
	actual, e := t.ReadDOMAIN_SEPARATOR(ctx)
	if e != nil {
		return Signature{}, e
	}
	if actual.Value0 != [32]byte(expected) {
		return Signature{}, errors.New("permit domain separator mismatch")
	}
	sig, e := signer.SignTypedData(ctx, d)
	if e != nil {
		return Signature{}, e
	}
	return SplitSignature(sig)
}

// FrozenWithdrawal preserves contract state. A matured timestamp never means completed.
type FrozenWithdrawal struct {
	Index *big.Int
	State DelegationGetFrozenStakeOutput0
}

func (f FrozenWithdrawal) Active() bool {
	return f.State.FreezeType == 1 && f.State.FreezeStatus == 1 && f.State.Stake.Amount != nil && f.State.Stake.Amount.Sign() > 0
}
func (f FrozenWithdrawal) Matured(unixSeconds int64) bool {
	return f.Active() && f.State.UnfreezeTimestamp != nil && f.State.UnfreezeTimestamp.Sign() > 0 && f.State.UnfreezeTimestamp.IsInt64() && f.State.UnfreezeTimestamp.Int64() <= unixSeconds
}
func convertFrozen(value any) *DelegationGetFrozenStakeOutput0 {
	return abi.ConvertType(value, new(DelegationGetFrozenStakeOutput0)).(*DelegationGetFrozenStakeOutput0)
}
func typedDomainHash(data apitypes.TypedData) (common.Hash, error) {
	b, e := data.HashStruct("EIP712Domain", data.Domain.Map())
	return common.BytesToHash(b), e
}

// FrozenWithdrawals verifies caller-provided authoritative indexes at one block.
// It never interprets API unavailability as an empty wallet. Index enumeration
// belongs to an authoritative indexer; this method does not invent missing IDs.
func (d *Delegation) FrozenWithdrawals(ctx context.Context, wallet common.Address, indexes []*big.Int, block *big.Int) ([]FrozenWithdrawal, error) {
	if block == nil || block.Sign() < 0 {
		return nil, errors.New("explicit snapshot block required")
	}
	rows := make([]FrozenWithdrawal, 0, len(indexes))
	seen := map[string]bool{}
	for _, index := range indexes {
		if !uint256(index) {
			return nil, errors.New("invalid frozen stake index")
		}
		if seen[index.String()] {
			continue
		}
		seen[index.String()] = true
		v, e := d.ReadAt(ctx, block, "getFrozenStake(uint256)", index)
		if e != nil {
			return nil, e
		}
		state := *convertFrozen(v[0])
		row := FrozenWithdrawal{copyInt(index), state}
		if row.Active() && state.Stake.Delegator != wallet {
			return nil, errors.New("frozen stake belongs to another wallet")
		}
		if row.Active() {
			rows = append(rows, row)
		}
	}
	return rows, nil
}

// FrozenWithdrawalsBatch verifies one page of indexes in one eth_call at a pinned
// block. Use pages of at most 500 entries to bound RPC response/calldata size.
// Deployments without getFrozenStakes can explicitly use FrozenWithdrawals.
func (d *Delegation) FrozenWithdrawalsBatch(ctx context.Context, wallet common.Address, indexes []*big.Int, block *big.Int) ([]FrozenWithdrawal, error) {
	if block == nil || block.Sign() < 0 {
		return nil, errors.New("explicit snapshot block required")
	}
	if len(indexes) > 500 {
		return nil, errors.New("frozen stake page exceeds 500")
	}
	unique := make([]*big.Int, 0, len(indexes))
	seen := map[string]bool{}
	for _, id := range indexes {
		if !uint256(id) {
			return nil, errors.New("invalid frozen index")
		}
		if !seen[id.String()] {
			unique = append(unique, copyInt(id))
			seen[id.String()] = true
		}
	}
	if len(unique) == 0 {
		return []FrozenWithdrawal{}, nil
	}
	values, e := d.ReadAt(ctx, block, "getFrozenStakes(uint256[])", unique)
	if e != nil {
		return nil, e
	}
	if len(values) != 1 {
		return nil, errors.New("invalid frozen batch response")
	}
	rows := reflect.ValueOf(values[0])
	if rows.Kind() != reflect.Slice || rows.Len() != len(unique) {
		return nil, errors.New("frozen batch response size mismatch")
	}
	result := make([]FrozenWithdrawal, 0, rows.Len())
	for i := 0; i < rows.Len(); i++ {
		state := *convertFrozen(rows.Index(i).Interface())
		row := FrozenWithdrawal{unique[i], state}
		if row.Active() && state.Stake.Delegator != wallet {
			return nil, errors.New("frozen stake belongs to another wallet")
		}
		if row.Active() {
			result = append(result, row)
		}
	}
	return result, nil
}
