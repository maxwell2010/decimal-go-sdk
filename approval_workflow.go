package decimal

import (
	"context"
	"errors"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

type AllowancePlan struct {
	Approvals []Call
	Action    Call
}

// PlanERC20Allowance supports tokens requiring a confirmed zero allowance before
// a new nonzero approval. Each approval is an independent EOA transaction. Wait
// for its successful receipt before preparing the next step. Re-read allowance
// before the action if other processes can modify it. No step is auto-executed.
func (c *Client) PlanERC20Allowance(ctx context.Context, token, owner, spender common.Address, amount *big.Int, action Call, resetExisting bool) (AllowancePlan, error) {
	if c == nil || !uint256(amount) || !uint256(action.Value) || action.To != spender || owner == (common.Address{}) || spender == (common.Address{}) {
		return AllowancePlan{}, errors.New("invalid allowance action")
	}
	if spender == c.cfg.Multicall {
		return AllowancePlan{}, errors.New("generic Multicall must not receive ERC20 allowance")
	}
	t, e := c.Token(token)
	if e != nil {
		return AllowancePlan{}, e
	}
	allowance, e := t.ReadAllowance(ctx, owner, spender)
	if e != nil {
		return AllowancePlan{}, e
	}
	action.Value = copyInt(action.Value)
	action.Data = append([]byte(nil), action.Data...)
	plan := AllowancePlan{Action: action, Approvals: []Call{}}
	if allowance.Value0.Cmp(amount) >= 0 {
		return plan, nil
	}
	if resetExisting && allowance.Value0.Sign() > 0 {
		zero, e := t.BuildApprove(spender, new(big.Int))
		if e != nil {
			return AllowancePlan{}, e
		}
		plan.Approvals = append(plan.Approvals, zero)
	}
	approval, e := t.BuildApprove(spender, amount)
	if e != nil {
		return AllowancePlan{}, e
	}
	plan.Approvals = append(plan.Approvals, approval)
	return plan, nil
}
