package decimal

import (
	"context"
	"errors"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"math/big"
	"sort"
)

type SafeSignatureKind uint8

const (
	SafeTypedSignature SafeSignatureKind = iota
	SafePersonalSignature
	SafeContractSignature
)

type SafeMixedSignature struct {
	Signer common.Address
	Kind   SafeSignatureKind
	Data   []byte
}

// SafePolicy is an explicit, previously verified deployment snapshot. This SDK
// does not guess owner/weight getters across Safe versions. Block is provenance;
// callers must ensure it is sufficiently fresh before submitting execution.
type SafePolicy struct {
	Safe                  common.Address
	ChainID, Nonce, Block *big.Int
	Owners                map[common.Address]uint32
	Threshold             uint64
}
type SafeContractVerifier func(context.Context, common.Address, common.Hash, []byte) error

// SigningPreimage is the 66-byte EIP712 envelope used by Safe transaction data
// validation. It supports deployment verifiers using the legacy bytes/bytes
// signature interface as well as newer interfaces taking the digest directly.
func (s SafeTransaction) SigningPreimage(chainID *big.Int, safe common.Address) ([]byte, error) {
	d, e := s.TypedData(chainID, safe)
	if e != nil {
		return nil, e
	}
	domain, e := d.HashStruct("EIP712Domain", d.Domain.Map())
	if e != nil {
		return nil, e
	}
	body, e := d.HashStruct("SafeTx", d.Message)
	if e != nil {
		return nil, e
	}
	out := []byte{0x19, 0x01}
	out = append(out, domain...)
	out = append(out, body...)
	return out, nil
}

// PackWeightedSafeSignatures follows official JS execution.ts buildSignatureBytes:
// sorted 65-byte static entries, then unpadded {uint256 length, bytes} tails.
// Every contract signature requires a verifier for that exact deployment's
// signature interface. EIP1271 has incompatible variants; none is assumed here.
func PackWeightedSafeSignatures(ctx context.Context, tx SafeTransaction, p SafePolicy, signatures []SafeMixedSignature, verify SafeContractVerifier) ([]byte, error) {
	if e := ctx.Err(); e != nil {
		return nil, e
	}
	if !uint256(p.Block) || !uint256(p.Nonce) || tx.Nonce == nil || tx.Nonce.Cmp(p.Nonce) != 0 || p.Threshold == 0 {
		return nil, errors.New("invalid or mismatched Safe snapshot")
	}
	hash, e := tx.Hash(p.ChainID, p.Safe)
	if e != nil {
		return nil, e
	}
	owners := make(map[common.Address]uint32, len(p.Owners))
	var total uint64
	for owner, w := range p.Owners {
		if owner == (common.Address{}) || w < 1 || w > 1000 {
			return nil, errors.New("invalid Safe owner/weight")
		}
		owners[owner] = w
		total += uint64(w)
	}
	if p.Threshold > total {
		return nil, errors.New("Safe threshold exceeds owner weight")
	}
	if len(signatures) == 0 {
		return nil, errors.New("Safe signatures required")
	}
	rows := make([]SafeMixedSignature, len(signatures))
	for i, s := range signatures {
		rows[i] = SafeMixedSignature{s.Signer, s.Kind, append([]byte(nil), s.Data...)}
	}
	seen := map[common.Address]bool{}
	var approved uint64
	for i, s := range rows {
		if e := ctx.Err(); e != nil {
			return nil, e
		}
		weight, ok := owners[s.Signer]
		if !ok || seen[s.Signer] {
			return nil, errors.New("unknown or duplicate Safe owner")
		}
		seen[s.Signer] = true
		data := append([]byte(nil), s.Data...)
		switch s.Kind {
		case SafeTypedSignature, SafePersonalSignature:
			if len(data) != 65 {
				return nil, errors.New("invalid Safe EOA signature")
			}
			digest := hash
			if s.Kind == SafePersonalSignature {
				digest = common.BytesToHash(accounts.TextHash(hash[:]))
				if data[64] == 31 || data[64] == 32 {
					data[64] -= 4
				}
			}
			parsed, e := SplitSignature(data)
			if e != nil {
				return nil, e
			}
			data[64] = parsed.V - 27
			pub, e := crypto.SigToPub(digest[:], data)
			if e != nil || crypto.PubkeyToAddress(*pub) != s.Signer {
				return nil, errors.New("Safe signature owner/hash mismatch")
			}
			data[64] += 27
			if s.Kind == SafePersonalSignature {
				data[64] += 4
			}
		case SafeContractSignature:
			if len(data) == 0 || verify == nil {
				return nil, errors.New("contract signature verifier required")
			}
			if e := verify(ctx, s.Signer, hash, append([]byte(nil), data...)); e != nil {
				return nil, errors.New("Safe contract signature rejected")
			}
		default:
			return nil, errors.New("unsupported Safe signature kind")
		}
		rows[i] = SafeMixedSignature{s.Signer, s.Kind, data}
		approved += uint64(weight)
	}
	if e := ctx.Err(); e != nil {
		return nil, e
	}
	if approved < p.Threshold {
		return nil, errors.New("insufficient Safe signature weight")
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Signer.Cmp(rows[j].Signer) < 0 })
	static := make([]byte, 0, 65*len(rows))
	var tails []byte
	for _, s := range rows {
		if s.Kind != SafeContractSignature {
			static = append(static, s.Data...)
			continue
		}
		static = append(static, common.LeftPadBytes(s.Signer.Bytes(), 32)...)
		offset := new(big.Int).SetUint64(uint64(65*len(rows) + len(tails)))
		static = append(static, common.LeftPadBytes(offset.Bytes(), 32)...)
		static = append(static, 0)
		size := new(big.Int).SetUint64(uint64(len(s.Data)))
		tails = append(tails, common.LeftPadBytes(size.Bytes(), 32)...)
		tails = append(tails, s.Data...)
	}
	return append(static, tails...), nil
}
