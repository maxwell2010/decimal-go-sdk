package decimal

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

// SafeTransaction follows the exact EIP712 field order in the official JS SDK.
type SafeTransaction struct {
	To                           common.Address
	Value                        *big.Int
	Data                         []byte
	Operation                    uint8
	SafeTxGas, BaseGas, GasPrice *big.Int
	GasToken, RefundReceiver     common.Address
	Nonce                        *big.Int
}

func NewSafeTransaction(call Call, nonce *big.Int) SafeTransaction {
	return SafeTransaction{To: call.To, Value: copyInt(call.Value), Data: append([]byte(nil), call.Data...), Nonce: copyInt(nonce), SafeTxGas: new(big.Int), BaseGas: new(big.Int), GasPrice: new(big.Int)}
}
func (s SafeTransaction) TypedData(chainID *big.Int, safe common.Address) (apitypes.TypedData, error) {
	if !uint256(chainID) || chainID.Sign() == 0 || safe == (common.Address{}) || s.Operation > 1 {
		return apitypes.TypedData{}, errors.New("invalid Safe domain or operation")
	}
	for _, n := range []*big.Int{s.Value, s.SafeTxGas, s.BaseGas, s.GasPrice, s.Nonce} {
		if !uint256(n) {
			return apitypes.TypedData{}, errors.New("invalid Safe integer")
		}
	}
	return apitypes.TypedData{Types: apitypes.Types{
		"EIP712Domain": {{Name: "chainId", Type: "uint256"}, {Name: "verifyingContract", Type: "address"}},
		"SafeTx":       {{Name: "to", Type: "address"}, {Name: "value", Type: "uint256"}, {Name: "data", Type: "bytes"}, {Name: "operation", Type: "uint8"}, {Name: "safeTxGas", Type: "uint256"}, {Name: "baseGas", Type: "uint256"}, {Name: "gasPrice", Type: "uint256"}, {Name: "gasToken", Type: "address"}, {Name: "refundReceiver", Type: "address"}, {Name: "nonce", Type: "uint256"}},
	}, PrimaryType: "SafeTx", Domain: apitypes.TypedDataDomain{ChainId: (*math.HexOrDecimal256)(copyInt(chainID)), VerifyingContract: safe.Hex()}, Message: apitypes.TypedDataMessage{
		"to": s.To.Hex(), "value": s.Value.String(), "data": s.Data, "operation": new(big.Int).SetUint64(uint64(s.Operation)).String(), "safeTxGas": s.SafeTxGas.String(), "baseGas": s.BaseGas.String(), "gasPrice": s.GasPrice.String(), "gasToken": s.GasToken.Hex(), "refundReceiver": s.RefundReceiver.Hex(), "nonce": s.Nonce.String(),
	}}, nil
}
func (s SafeTransaction) Hash(chainID *big.Int, safe common.Address) (common.Hash, error) {
	d, e := s.TypedData(chainID, safe)
	if e != nil {
		return common.Hash{}, e
	}
	hash, _, e := apitypes.TypedDataAndHash(d)
	return common.BytesToHash(hash), e
}
func (s SafeTransaction) Sign(ctx context.Context, signer *KeySigner, chainID *big.Int, safe common.Address) ([]byte, error) {
	if signer == nil {
		return nil, errors.New("signer required")
	}
	d, e := s.TypedData(chainID, safe)
	if e != nil {
		return nil, e
	}
	return signer.SignTypedData(ctx, d)
}

// PackSafeEOASignatures recovers signers from this transaction hash, rejects
// duplicates and sorts by numeric address. Contract signatures need a deployment-
// specific encoder and are not accepted as EOA signatures.
func PackSafeEOASignatures(hash common.Hash, signatures ...[]byte) ([]byte, error) {
	if len(signatures) == 0 {
		return nil, errors.New("Safe signatures required")
	}
	type signed struct {
		address common.Address
		sig     []byte
	}
	rows := make([]signed, 0, len(signatures))
	seen := map[common.Address]bool{}
	for _, signature := range signatures {
		if _, e := SplitSignature(signature); e != nil {
			return nil, e
		}
		sig := append([]byte(nil), signature...)
		if sig[64] >= 27 {
			sig[64] -= 27
		}
		pub, e := crypto.SigToPub(hash[:], sig)
		if e != nil {
			return nil, errors.New("cannot recover Safe signer")
		}
		address := crypto.PubkeyToAddress(*pub)
		if seen[address] {
			return nil, errors.New("duplicate Safe signer")
		}
		seen[address] = true
		sig[64] += 27
		rows = append(rows, signed{address, sig})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].address.Cmp(rows[j].address) < 0 })
	out := make([]byte, 0, 65*len(rows))
	for _, r := range rows {
		out = append(out, r.sig...)
	}
	return out, nil
}

type SafeSignature struct {
	Signer common.Address
	Data   []byte
}

// PackSafeSignatures additionally binds every signature to the expected owner.
// The caller must still verify current owners and weight threshold on the Safe.
func PackSafeSignatures(hash common.Hash, signatures ...SafeSignature) ([]byte, error) {
	if len(signatures) == 0 {
		return nil, errors.New("Safe signatures required")
	}
	raw := make([][]byte, 0, len(signatures))
	for _, s := range signatures {
		if _, e := SplitSignature(s.Data); e != nil {
			return nil, e
		}
		b := append([]byte(nil), s.Data...)
		if b[64] >= 27 {
			b[64] -= 27
		}
		pub, e := crypto.SigToPub(hash[:], b)
		if e != nil || crypto.PubkeyToAddress(*pub) != s.Signer {
			return nil, errors.New("Safe signature owner/hash mismatch")
		}
		raw = append(raw, s.Data)
	}
	return PackSafeEOASignatures(hash, raw...)
}

// BuildSafeExecution uses a caller-supplied deployment ABI. Owners, threshold,
// weights and nonce must be checked against that Safe before execution.
func (s SafeTransaction) BuildExecution(contract *Contract, signatures []byte) (Call, error) {
	if contract == nil {
		return Call{}, errors.New("Safe contract required")
	}
	if len(signatures) == 0 {
		return Call{}, errors.New("signatures required")
	}
	if _, e := s.TypedData(big.NewInt(1), contract.Address); e != nil {
		return Call{}, e
	}
	return contract.Build("execTransaction(address,uint256,bytes,uint8,uint256,uint256,uint256,address,address,bytes)", nil, s.To, s.Value, s.Data, s.Operation, s.SafeTxGas, s.BaseGas, s.GasPrice, s.GasToken, s.RefundReceiver, signatures)
}

// BuildSafeFactoryProxy accepts explicit initializer bytes. Decimal uses weighted
// owners, so a standard Gnosis setup(address[]) ABI must not be substituted.
func (c *Client) BuildSafeFactoryProxy(factory, implementation common.Address, initializer []byte, salt *big.Int) (Call, error) {
	if implementation == (common.Address{}) || len(initializer) < 4 || !uint256(salt) {
		return Call{}, errors.New("invalid Safe initializer")
	}
	raw := json.RawMessage(`[{"type":"function","name":"createProxyWithNonce","stateMutability":"nonpayable","inputs":[{"name":"singleton","type":"address"},{"name":"initializer","type":"bytes"},{"name":"saltNonce","type":"uint256"}],"outputs":[{"type":"address"}]}]`)
	f, e := c.ContractWithABI(factory, raw)
	if e != nil {
		return Call{}, e
	}
	return f.Build("createProxyWithNonce(address,bytes,uint256)", nil, implementation, initializer, salt)
}
