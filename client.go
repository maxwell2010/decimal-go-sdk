package decimal

import (
	"context"
	"errors"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

// Backend allows deterministic tests, hardware signers and custom transports.
type Backend interface {
	ChainID(context.Context) (*big.Int, error)
	PendingNonceAt(context.Context, common.Address) (uint64, error)
	SuggestGasPrice(context.Context) (*big.Int, error)
	EstimateGas(context.Context, ethereum.CallMsg) (uint64, error)
	BalanceAt(context.Context, common.Address, *big.Int) (*big.Int, error)
	CodeAt(context.Context, common.Address, *big.Int) ([]byte, error)
	CallContract(context.Context, ethereum.CallMsg, *big.Int) ([]byte, error)
	SendTransaction(context.Context, *types.Transaction) error
	TransactionReceipt(context.Context, common.Hash) (*types.Receipt, error)
}

type nonceState struct {
	gate chan struct{}
	next uint64
	used bool
}
type Client struct {
	rpc    Backend
	cfg    Config
	nonces sync.Map
	close  func()
}

func NewClient(cfg Config, backend Backend) (*Client, error) {
	if e := cfg.validate(); e != nil {
		return nil, e
	}
	if backend == nil {
		return nil, errors.New("RPC backend required")
	}
	cfg.ChainID = copyInt(cfg.ChainID)
	return &Client{rpc: backend, cfg: cfg}, nil
}
func Dial(ctx context.Context, cfg Config) (*Client, error) {
	if e := cfg.validate(); e != nil {
		return nil, e
	}
	if !validURL(cfg.RPCURL) {
		return nil, errors.New("invalid RPC URL")
	}
	rpc, e := ethclient.DialContext(ctx, cfg.RPCURL)
	if e != nil {
		return nil, errors.New("RPC connection failed")
	}
	c, e := NewClient(cfg, rpc)
	if e != nil {
		rpc.Close()
		return nil, e
	}
	c.close = rpc.Close
	if e = c.checkChain(ctx); e != nil {
		c.Close()
		return nil, e
	}
	return c, nil
}
func (c *Client) Close() {
	if c.close != nil {
		c.close()
	}
}
func (c *Client) ChainID() *big.Int { return copyInt(c.cfg.ChainID) }
func (c *Client) checkChain(ctx context.Context) error {
	if e := ctx.Err(); e != nil {
		return e
	}
	id, e := c.rpc.ChainID(ctx)
	if e != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return errors.New("RPC chain ID unavailable")
	}
	if id == nil || id.Cmp(c.cfg.ChainID) != 0 {
		return errors.New("RPC chain ID mismatch")
	}
	return nil
}

// ResolveSystemContract reads the registry; child addresses are never inherited across networks.
func (c *Client) ResolveSystemContract(ctx context.Context, slug string) (common.Address, error) {
	if slug == "" {
		return common.Address{}, errors.New("contract slug required")
	}
	r, e := c.Contract("ContractCenter", c.cfg.ContractCenter)
	if e != nil {
		return common.Address{}, e
	}
	v, e := r.Read(ctx, "getContractAddress(string)", slug)
	if e != nil {
		return common.Address{}, e
	}
	if len(v) != 1 {
		return common.Address{}, errors.New("invalid registry response")
	}
	address, ok := v[0].(common.Address)
	if !ok || address == (common.Address{}) {
		return common.Address{}, errors.New("unregistered system contract")
	}
	code, e := c.rpc.CodeAt(ctx, address, nil)
	if e != nil || len(code) == 0 {
		return common.Address{}, errors.New("system contract has no code")
	}
	return address, nil
}
func (c *Client) Balance(ctx context.Context, address common.Address) (*big.Int, error) {
	if e := c.checkChain(ctx); e != nil {
		return nil, e
	}
	n, e := c.rpc.BalanceAt(ctx, address, nil)
	if e != nil {
		return nil, rpcFailure(ctx, "balance unavailable")
	}
	return n, nil
}

func rpcFailure(ctx context.Context, message string) error {
	if e := ctx.Err(); e != nil {
		return e
	}
	return errors.New(message)
}

// TxOptions defaults to legacy pricing. Typed envelopes require explicit opt-in
// and a compatible deployment; support is never inferred from a missing base fee.
// GasMultiplierBPS defaults to 11000.
type TxOptions struct {
	// TxType is explicit: 0 legacy (default), 1 EIP2930, 2 EIP1559.
	TxType                          uint8
	AccessList                      types.AccessList
	FeeCap, TipCap, BaseFeeEstimate *big.Int
	Nonce                           *uint64
	GasLimit                        uint64
	GasMultiplierBPS                uint64
	GasPrice, MaxGasPrice, MaxFee   *big.Int
}
type FeeQuote struct {
	// Dynamic-fee EstimatedFee is nil without an explicit BaseFeeEstimate.
	FeeCap, TipCap, BaseFeeEstimate, EstimatedEffectiveGasPrice *big.Int
	EstimatedGas, GasLimit                                      uint64
	GasPrice, EstimatedFee, ReservedFee, Value, RequiredBalance *big.Int
}
type Prepared struct {
	Transaction *types.Transaction
	From        common.Address
	Fee         FeeQuote
}

func (c *Client) Prepare(ctx context.Context, from common.Address, call Call, opts TxOptions) (*Prepared, error) {
	return c.prepare(ctx, from, call, opts)
}
func (c *Client) prepare(ctx context.Context, from common.Address, call Call, opts TxOptions) (*Prepared, error) {
	if call.deployment != nil {
		if call.To != call.deployment.Proxy {
			return nil, errors.New("proxy transaction target changed")
		}
		block, e := c.BlockNumber(ctx)
		if e != nil {
			return nil, e
		}
		if e = c.verifyDeployment(ctx, *call.deployment, block); e != nil {
			return nil, e
		}
	}
	for _, p := range []**big.Int{&opts.GasPrice, &opts.MaxGasPrice, &opts.MaxFee, &opts.FeeCap, &opts.TipCap, &opts.BaseFeeEstimate} {
		if *p != nil {
			*p = copyInt(*p)
		}
	}
	if opts.TxType > types.DynamicFeeTxType {
		return nil, errors.New("unsupported transaction type")
	}
	if opts.TxType == types.LegacyTxType && len(opts.AccessList) > 0 {
		return nil, errors.New("legacy transaction cannot carry an access list")
	}
	if opts.TxType == types.DynamicFeeTxType {
		if opts.GasPrice != nil || !uint256(opts.FeeCap) || opts.FeeCap.Sign() == 0 || !uint256(opts.TipCap) || opts.FeeCap.Cmp(opts.TipCap) < 0 {
			return nil, errors.New("explicit valid dynamic fee cap/tip required; gasPrice is incompatible")
		}
		if opts.BaseFeeEstimate != nil && (!uint256(opts.BaseFeeEstimate) || opts.BaseFeeEstimate.Cmp(opts.FeeCap) > 0) {
			return nil, errors.New("invalid base fee estimate or fee cap below base fee")
		}
	} else if opts.FeeCap != nil || opts.TipCap != nil || opts.BaseFeeEstimate != nil {
		return nil, errors.New("dynamic fee fields require transaction type 2")
	}
	accessList := cloneAccessList(opts.AccessList)
	if e := c.checkProfileNetwork(call.Profile); e != nil {
		return nil, e
	}
	if !uint256(call.Value) {
		return nil, errors.New("invalid transaction value")
	}
	call.Value = copyInt(call.Value)
	call.Data = append([]byte(nil), call.Data...)
	if e := c.checkChain(ctx); e != nil {
		return nil, e
	}
	if call.Contract {
		code, e := c.rpc.CodeAt(ctx, call.To, nil)
		if e != nil || len(code) == 0 {
			return nil, rpcFailure(ctx, "target contract has no code")
		}
	}
	price := copyInt(opts.GasPrice)
	if opts.TxType == types.DynamicFeeTxType {
		price = copyInt(opts.FeeCap)
	} else if opts.GasPrice == nil {
		p, e := c.rpc.SuggestGasPrice(ctx)
		if e != nil {
			return nil, rpcFailure(ctx, "gas price unavailable")
		}
		price = copyInt(p)
	}
	if !uint256(price) || price.Sign() == 0 {
		return nil, errors.New("invalid gas price")
	}
	if opts.MaxGasPrice != nil && (!uint256(opts.MaxGasPrice) || price.Cmp(opts.MaxGasPrice) > 0) {
		return nil, errors.New("gas price exceeds limit")
	}
	for _, limit := range []*big.Int{opts.MaxFee} {
		if limit != nil && !uint256(limit) {
			return nil, errors.New("invalid fee limit")
		}
	}
	msg := ethereum.CallMsg{From: from, To: &call.To, Value: copyInt(call.Value), Data: append([]byte(nil), call.Data...), GasPrice: copyInt(price), AccessList: cloneAccessList(accessList)}
	quote := FeeQuote{GasPrice: copyInt(price), EstimatedEffectiveGasPrice: copyInt(price)}
	if opts.TxType == types.DynamicFeeTxType {
		msg.GasPrice = nil
		msg.GasFeeCap = copyInt(opts.FeeCap)
		msg.GasTipCap = copyInt(opts.TipCap)
		quote.GasPrice = nil
		quote.FeeCap = copyInt(opts.FeeCap)
		quote.TipCap = copyInt(opts.TipCap)
		quote.EstimatedEffectiveGasPrice = nil
		if opts.BaseFeeEstimate != nil {
			quote.BaseFeeEstimate = copyInt(opts.BaseFeeEstimate)
			effective := new(big.Int).Add(quote.BaseFeeEstimate, quote.TipCap)
			if effective.Cmp(quote.FeeCap) > 0 {
				effective.Set(quote.FeeCap)
			}
			quote.EstimatedEffectiveGasPrice = effective
		}
	}
	estimated, e := c.rpc.EstimateGas(ctx, msg)
	if e != nil {
		return nil, rpcFailure(ctx, "gas estimation failed; transaction not signed")
	}
	if estimated == 0 {
		return nil, errors.New("invalid gas estimate")
	}
	bps := opts.GasMultiplierBPS
	if bps == 0 {
		bps = 11000
	}
	if bps < 10000 || bps > 100000 {
		return nil, errors.New("invalid gas multiplier")
	}
	buffer := new(big.Int).Mul(new(big.Int).SetUint64(estimated), new(big.Int).SetUint64(bps))
	buffer.Add(buffer, big.NewInt(9999)).Div(buffer, big.NewInt(10000))
	if !buffer.IsUint64() {
		return nil, errors.New("gas limit overflow")
	}
	gas := buffer.Uint64()
	if opts.GasLimit != 0 {
		gas = opts.GasLimit
	}
	if gas < estimated {
		return nil, errors.New("gas limit below estimate")
	}
	reserved := new(big.Int).Mul(new(big.Int).SetUint64(gas), price)
	if opts.MaxFee != nil && reserved.Cmp(opts.MaxFee) > 0 {
		return nil, errors.New("reserved fee exceeds limit")
	}
	required := new(big.Int).Add(reserved, call.Value)
	balance, e := c.rpc.BalanceAt(ctx, from, nil)
	if e != nil || balance == nil {
		return nil, rpcFailure(ctx, "balance unavailable")
	}
	if balance.Cmp(required) < 0 {
		return nil, errors.New("insufficient DEL for value and reserved fee")
	}
	nonce := uint64(0)
	if opts.Nonce != nil {
		nonce = *opts.Nonce
	} else {
		nonce, e = c.rpc.PendingNonceAt(ctx, from)
		if e != nil {
			return nil, rpcFailure(ctx, "pending nonce unavailable")
		}
	}
	if nonce == ^uint64(0) {
		return nil, errors.New("nonce exhausted")
	}
	var payload types.TxData = &types.LegacyTx{Nonce: nonce, To: &call.To, Value: copyInt(call.Value), Gas: gas, GasPrice: price, Data: append([]byte(nil), call.Data...)}
	if opts.TxType == types.AccessListTxType {
		payload = &types.AccessListTx{ChainID: c.ChainID(), Nonce: nonce, To: &call.To, Value: copyInt(call.Value), Gas: gas, GasPrice: price, Data: append([]byte(nil), call.Data...), AccessList: accessList}
	}
	if opts.TxType == types.DynamicFeeTxType {
		payload = &types.DynamicFeeTx{ChainID: c.ChainID(), Nonce: nonce, To: &call.To, Value: copyInt(call.Value), Gas: gas, GasFeeCap: copyInt(quote.FeeCap), GasTipCap: copyInt(quote.TipCap), Data: append([]byte(nil), call.Data...), AccessList: accessList}
	}
	quote.EstimatedGas = estimated
	quote.GasLimit = gas
	quote.ReservedFee = reserved
	quote.Value = copyInt(call.Value)
	quote.RequiredBalance = required
	if quote.EstimatedEffectiveGasPrice != nil {
		quote.EstimatedFee = new(big.Int).Mul(new(big.Int).SetUint64(estimated), quote.EstimatedEffectiveGasPrice)
	}
	return &Prepared{Transaction: types.NewTx(payload), From: from, Fee: quote}, nil
}

// SubmissionUnknownError means the RPC may have accepted the signed transaction.
// Reconcile Hash before resubmitting. No alternate endpoint or fee retry is used.
type SubmissionUnknownError struct{ Hash common.Hash }

func (e *SubmissionUnknownError) Error() string {
	return "transaction submission outcome unknown; reconcile local hash " + e.Hash.Hex()
}

type Submission struct {
	Hash        common.Hash
	Transaction *types.Transaction
	Fee         FeeQuote
}

// Send serializes automatic nonce allocation per sender within this Client.
// Separate processes/clients must coordinate nonces externally. Explicit nonces
// are caller-managed. Prepare does not reserve a nonce.
func (c *Client) Send(ctx context.Context, signer Signer, call Call, opts TxOptions) (*Submission, error) {
	if e := ctx.Err(); e != nil {
		return nil, e
	}
	if signer == nil {
		return nil, errors.New("signer required")
	}
	if ready, ok := signer.(interface{ Ready() error }); ok {
		if e := ready.Ready(); e != nil {
			return nil, e
		}
	}
	address := signer.Address()
	entry, _ := c.nonces.LoadOrStore(address, &nonceState{gate: make(chan struct{}, 1)})
	state := entry.(*nonceState)
	select {
	case state.gate <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	defer func() { <-state.gate }()
	if e := ctx.Err(); e != nil {
		return nil, e
	}
	automatic := opts.Nonce == nil
	if automatic {
		pending, e := c.rpc.PendingNonceAt(ctx, address)
		if e != nil {
			return nil, rpcFailure(ctx, "pending nonce unavailable")
		}
		if state.used && state.next > pending {
			pending = state.next
		}
		if pending == ^uint64(0) {
			return nil, errors.New("nonce exhausted")
		}
		opts.Nonce = &pending
	}
	p, e := c.prepare(ctx, address, call, opts)
	if e != nil {
		return nil, e
	}
	// The external signer must not get a mutable alias to the reference intent
	// (Transaction.Data exposes its backing bytes). Preserve a serialized copy.
	intentBytes, e := p.Transaction.MarshalBinary()
	if e != nil {
		return nil, errors.New("cannot serialize unsigned intent")
	}
	intent := new(types.Transaction)
	if e = intent.UnmarshalBinary(intentBytes); e != nil {
		return nil, errors.New("cannot snapshot unsigned intent")
	}
	signed, e := signer.SignTransaction(ctx, p.Transaction, copyInt(c.cfg.ChainID))
	if e != nil {
		return nil, e
	}
	if e = c.validateSigned(signed); e != nil {
		return nil, e
	}
	if !sameIntent(intent, signed) {
		return nil, errors.New("signer changed transaction intent")
	}
	from, e := types.Sender(types.LatestSignerForChainID(c.cfg.ChainID), signed)
	if e != nil || from != address {
		return nil, errors.New("signer changed sender")
	}
	if e = c.checkChain(ctx); e != nil {
		return nil, e
	}
	if e = ctx.Err(); e != nil {
		return nil, e
	}
	if automatic {
		state.used = true
		state.next = signed.Nonce() + 1
	}
	result := &Submission{signed.Hash(), signed, p.Fee}
	if e = c.rpc.SendTransaction(ctx, signed); e != nil {
		return result, &SubmissionUnknownError{signed.Hash()}
	}
	return result, nil
}
func sameIntent(a, b *types.Transaction) bool {
	if a == nil || b == nil || a.To() == nil || b.To() == nil {
		return false
	}
	return a.Type() == b.Type() && *a.To() == *b.To() && a.Nonce() == b.Nonce() && a.Gas() == b.Gas() && a.Value().Cmp(b.Value()) == 0 && a.GasPrice().Cmp(b.GasPrice()) == 0 && a.GasTipCap().Cmp(b.GasTipCap()) == 0 && a.GasFeeCap().Cmp(b.GasFeeCap()) == 0 && sameAccessList(a.AccessList(), b.AccessList()) && string(a.Data()) == string(b.Data())
}

func cloneAccessList(input types.AccessList) types.AccessList {
	if input == nil {
		return nil
	}
	out := make(types.AccessList, len(input))
	for i, row := range input {
		out[i] = types.AccessTuple{Address: row.Address, StorageKeys: append([]common.Hash(nil), row.StorageKeys...)}
	}
	return out
}
func sameAccessList(a, b types.AccessList) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Address != b[i].Address || len(a[i].StorageKeys) != len(b[i].StorageKeys) {
			return false
		}
		for j := range a[i].StorageKeys {
			if a[i].StorageKeys[j] != b[i].StorageKeys[j] {
				return false
			}
		}
	}
	return true
}
func (c *Client) validateSigned(tx *types.Transaction) error {
	if tx == nil || !tx.Protected() || tx.ChainId().Cmp(c.cfg.ChainID) != 0 {
		return errors.New("signed chain ID mismatch")
	}
	if tx.Nonce() == ^uint64(0) {
		return errors.New("nonce exhausted")
	}
	if _, e := types.Sender(types.LatestSignerForChainID(c.cfg.ChainID), tx); e != nil {
		return errors.New("invalid transaction signature")
	}
	return nil
}

// Broadcast submits an already signed transaction exactly once.
func (c *Client) Broadcast(ctx context.Context, tx *types.Transaction) (common.Hash, error) {
	if e := c.validateSigned(tx); e != nil {
		return common.Hash{}, e
	}
	if e := c.checkChain(ctx); e != nil {
		return common.Hash{}, e
	}
	h := tx.Hash()
	if e := c.rpc.SendTransaction(ctx, tx); e != nil {
		return h, &SubmissionUnknownError{h}
	}
	return h, nil
}

type Mined struct {
	Receipt   *types.Receipt
	ActualFee *big.Int
}
type RevertedError struct{ Hash common.Hash }

func (e *RevertedError) Error() string { return "transaction reverted: " + e.Hash.Hex() }
func (c *Client) WaitReceipt(ctx context.Context, hash common.Hash, poll time.Duration) (*Mined, error) {
	if poll <= 0 {
		poll = time.Second
	}
	if e := c.checkChain(ctx); e != nil {
		return nil, e
	}
	timer := time.NewTicker(poll)
	defer timer.Stop()
	for {
		r, e := c.rpc.TransactionReceipt(ctx, hash)
		if e == nil && r != nil {
			if r.TxHash != hash {
				return nil, errors.New("receipt transaction hash mismatch")
			}
			var fee *big.Int
			if r.EffectiveGasPrice != nil {
				fee = new(big.Int).Mul(new(big.Int).SetUint64(r.GasUsed), r.EffectiveGasPrice)
			}
			m := &Mined{r, fee}
			if r.Status != types.ReceiptStatusSuccessful {
				return m, &RevertedError{hash}
			}
			return m, nil
		}
		if e != nil && !errors.Is(e, ethereum.NotFound) {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			return nil, errors.New("receipt RPC failed")
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}
