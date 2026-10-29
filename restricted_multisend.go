package decimal

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"math/big"
	"unicode/utf8"
)

// RestrictedMultisendDeployment is an explicit caller-reviewed, codehash-bound
// deployment. No local C# deployment address is silently made a Go default.
type RestrictedMultisendDeployment struct {
	Address     common.Address
	ChainID     *big.Int
	RuntimeHash common.Hash
}
type RestrictedMultisend struct {
	client     *Client
	deployment RestrictedMultisendDeployment
	contract   *Contract
}

var restrictedMultisendABI = json.RawMessage(`[{"type":"function","name":"sendToken","stateMutability":"nonpayable","inputs":[{"type":"address"},{"type":"address[]"},{"type":"uint256[]"},{"type":"bytes"}],"outputs":[]},{"type":"function","name":"sendTokenWithPermit","stateMutability":"nonpayable","inputs":[{"type":"address"},{"type":"address[]"},{"type":"uint256[]"},{"type":"bytes"},{"type":"uint256"},{"type":"uint8"},{"type":"bytes32"},{"type":"bytes32"}],"outputs":[]}]`)

func (c *Client) RestrictedMultisend(d RestrictedMultisendDeployment) (*RestrictedMultisend, error) {
	if c == nil || d.Address == (common.Address{}) || d.RuntimeHash == (common.Hash{}) || !uint256(d.ChainID) || d.ChainID.Cmp(c.ChainID()) != 0 {
		return nil, errors.New("invalid or cross-network multisend deployment")
	}
	contract, e := c.ContractWithABI(d.Address, restrictedMultisendABI)
	if e != nil {
		return nil, e
	}
	d.ChainID = copyInt(d.ChainID)
	return &RestrictedMultisend{c, d, contract}, nil
}
func (s *RestrictedMultisend) Verify(ctx context.Context) error {
	if s == nil || s.client == nil {
		return errors.New("multisend deployment required")
	}
	code, e := s.client.ContractCode(ctx, s.deployment.Address, nil)
	if e != nil {
		return e
	}
	if len(code) == 0 || crypto.Keccak256Hash(code) != s.deployment.RuntimeHash {
		return errors.New("restricted multisend runtime hash mismatch")
	}
	return nil
}
func (s *RestrictedMultisend) batch(owner, token common.Address, recipients []Recipient, memo string) ([]common.Address, []*big.Int, *big.Int, error) {
	if s == nil || s.client == nil || s.contract == nil {
		return nil, nil, nil, errors.New("multisend deployment required")
	}
	if owner == (common.Address{}) || token == (common.Address{}) || len(recipients) < 1 || len(recipients) > 500 || len([]byte(memo)) > 256 || !utf8.ValidString(memo) {
		return nil, nil, nil, errors.New("invalid restricted multisend batch")
	}
	addresses := make([]common.Address, len(recipients))
	amounts := make([]*big.Int, len(recipients))
	total := new(big.Int)
	for i, r := range recipients {
		if r.Address == (common.Address{}) || r.Address == owner || r.Address == s.deployment.Address || !uint256(r.Amount) || r.Amount.Sign() == 0 {
			return nil, nil, nil, errors.New("invalid restricted multisend recipient")
		}
		addresses[i] = r.Address
		amounts[i] = copyInt(r.Amount)
		total.Add(total, r.Amount)
	}
	if !uint256(total) {
		return nil, nil, nil, errors.New("multisend total overflows uint256")
	}
	return addresses, amounts, total, nil
}
func (s *RestrictedMultisend) BuildToken(owner, token common.Address, recipients []Recipient, memo string) (Call, error) {
	addresses, amounts, _, e := s.batch(owner, token, recipients, memo)
	if e != nil {
		return Call{}, e
	}
	return s.contract.Build("sendToken(address,address[],uint256[],bytes)", nil, token, addresses, amounts, []byte(memo))
}
func (s *RestrictedMultisend) BuildTokenWithPermit(owner, token common.Address, recipients []Recipient, memo string, deadline *big.Int, sig Signature) (Call, error) {
	if sig.V != 27 && sig.V != 28 {
		return Call{}, errors.New("permit recovery ID must be 27 or 28")
	}
	addresses, amounts, _, e := s.batch(owner, token, recipients, memo)
	if e != nil {
		return Call{}, e
	}
	raw := append(append(append([]byte{}, sig.R[:]...), sig.S[:]...), sig.V)
	if _, e = SplitSignature(raw); e != nil {
		return Call{}, e
	}
	if !uint256(deadline) {
		return Call{}, errors.New("invalid permit deadline")
	}
	return s.contract.Build("sendTokenWithPermit(address,address[],uint256[],bytes,uint256,uint8,bytes32,bytes32)", nil, token, addresses, amounts, []byte(memo), deadline, sig.V, sig.R, sig.S)
}

// PrepareTokenWithPermit verifies the sender-bound EIP2612 signature using the
// actual nonce/domain separator, runtime hash and balance. It never broadcasts.
func (s *RestrictedMultisend) PrepareTokenWithPermit(ctx context.Context, owner, token common.Address, recipients []Recipient, memo, name, version string, deadline *big.Int, sig Signature, options TxOptions) (*Prepared, error) {
	call, e := s.BuildTokenWithPermit(owner, token, recipients, memo, deadline, sig)
	if e != nil {
		return nil, e
	}
	if e = s.Verify(ctx); e != nil {
		return nil, e
	}
	_, _, total, e := s.batch(owner, token, recipients, memo)
	if e != nil {
		return nil, e
	}
	t, e := s.client.Token(token)
	if e != nil {
		return nil, e
	}
	balance, e := t.ReadBalanceOf(ctx, owner)
	if e != nil {
		return nil, e
	}
	if balance.Value0.Cmp(total) < 0 {
		return nil, errors.New("insufficient token balance")
	}
	nonce, e := t.ReadNonces(ctx, owner)
	if e != nil {
		return nil, e
	}
	data, e := PermitTypedData(name, version, s.client.ChainID(), token, Permit{Owner: owner, Spender: s.deployment.Address, Value: total, Nonce: nonce.Value0, Deadline: deadline})
	if e != nil {
		return nil, e
	}
	expected, e := typedDomainHash(data)
	if e != nil {
		return nil, e
	}
	actual, e := t.ReadDOMAIN_SEPARATOR(ctx)
	if e != nil {
		return nil, e
	}
	if [32]byte(expected) != actual.Value0 {
		return nil, errors.New("permit domain separator mismatch")
	}
	digest, _, e := apitypes.TypedDataAndHash(data)
	if e != nil {
		return nil, e
	}
	raw := append(append(append([]byte{}, sig.R[:]...), sig.S[:]...), sig.V-27)
	pub, e := crypto.SigToPub(digest, raw)
	if e != nil || crypto.PubkeyToAddress(*pub) != owner {
		return nil, errors.New("permit does not authorize sender")
	}
	return s.client.Prepare(ctx, owner, call, options)
}

// PrepareToken verifies runtime and token balance/allowance before fee preparation.
// The builder's owner must equal the sender. It never signs or submits.
func (s *RestrictedMultisend) PrepareToken(ctx context.Context, owner, token common.Address, recipients []Recipient, memo string, options TxOptions) (*Prepared, error) {
	call, e := s.BuildToken(owner, token, recipients, memo)
	if e != nil {
		return nil, e
	}
	if e = s.Verify(ctx); e != nil {
		return nil, e
	}
	_, _, total, e := s.batch(owner, token, recipients, memo)
	if e != nil {
		return nil, e
	}
	t, e := s.client.Token(token)
	if e != nil {
		return nil, e
	}
	balance, e := t.ReadBalanceOf(ctx, owner)
	if e != nil {
		return nil, e
	}
	allowance, e := t.ReadAllowance(ctx, owner, s.deployment.Address)
	if e != nil {
		return nil, e
	}
	if balance.Value0.Cmp(total) < 0 || allowance.Value0.Cmp(total) < 0 {
		return nil, errors.New("insufficient token balance or allowance")
	}
	return s.client.Prepare(ctx, owner, call, options)
}
